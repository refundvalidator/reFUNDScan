package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
    "regexp"
	"strconv"
	"strings"
    "time"
    "math/rand"

	"github.com/fatih/color"
	"github.com/btcsuite/btcutil/bech32"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)


// Returns and MD formatted hyperlink for an account when given a wallet or validator address
func mkAccountLink(addr string) string{
    if addr[:len(config.Chain.Prefix + "val")] == config.Chain.Prefix + "val"{
        return fmt.Sprintf("[%s](%s%s)",getAccountName(addr),config.Explorer.Validator,addr)
    } else {
        for _, chain := range(config.OtherChains) {
            if chain.Prefix == addr[:len(chain.Prefix)] {
                url := config.Explorer.Base + chain.ExplorerPath + "/account/" + addr
                return fmt.Sprintf("[%s](%s)",getAccountName(addr),url)
            }
        }
        return fmt.Sprintf("[%s](%s%s)",getAccountName(addr),config.Explorer.Account, addr)
    }
}

// Returns a MD formatted hyprlink for a transaction when given a TX Hash with an amount
func mkTranscationLink(hash string, amount string) string {
    return fmt.Sprintf("[%s](%s%s)", denomToAmount(amount), config.Explorer.TX,hash)
}

// When given a transaction hash
// Searches rest endpoints for a memo on the transaction, if not available returns an empty string
func getMemo(hash string) string {
    var tx TxResponse
    err := getData(config.Connections.Rest + "cosmos/tx/v1beta1/txs/" + hash, &tx)
    if err != nil {
        log.Println(color.YellowString("Failed to get TX rest response: ", err))
        return ""
    }
    return removeForbiddenChars(tx.Tx.Body.Memo)
}

// When given a wallet or validator address, returns the name associated with the wallet, if it has one
// Otherwise returns a truncated version of the wallet address
func getAccountName(msg string) string {
    // Known account names
    names := map[string][]string{}
    // Convert undval to und1 addresses and append to map
    for _, val := range vals.Validators {
        _, data, err := bech32.Decode(val.OperatorAddress)
        if err != nil {
            log.Println(color.YellowString("Could not decode bech32 address"))
            continue
        }
        addr, err := bech32.Encode(config.Chain.Prefix,data)
        if err != nil {
            log.Println(color.YellowString("Could not encode bech32 address"))
            continue
        }
        names[val.Description.Moniker] = []string{addr, val.OperatorAddress}
    }

    // Check if name matches named wallet from config
    for _, name := range config.Config.AddressesConfig.Addresses {
        if name.Addr == msg {
            return removeForbiddenChars(name.Name)
        }
    }

    // Check if name matches wallet or val addr
    for key, val := range names {
        if val[0] == msg || val[1] == msg {
            return removeForbiddenChars(key)
        }
    }

    // Check ICNS for name
    var icns ICNSResponse
    query := fmt.Sprintf(`{ "icns_names": { "address": "%s" }}`, msg)
    b64 := base64.StdEncoding.EncodeToString([]byte(query))
    err := getData(config.Connections.ICNS +
        "cosmwasm/wasm/v1/contract/osmo1xk0s8xgktn9x5vwcgtjdxqzadg88fgn33p8u9cnpdxwemvxscvast52cdd/smart/" +
        b64, &icns)
    if err != nil {
        log.Println(color.YellowString("Failed to get ICNS response ", err))
    }
    if icns.Data.PrimaryName != "" {
        return removeForbiddenChars(icns.Data.PrimaryName)
    }

    // Return truncated addr if the addr isnt in the named map
    return fmt.Sprintf("%s...%s",msg[:7],msg[len(msg)-7:])
}

func denomTotaler() func(string) string{
    var total float64
    return func(msg string) string {
        amount, denom := splitAmountDenom(msg)
        total += amount
        return fmt.Sprintf("%.0f%s",total,denom)
    }
}

// Converts the denom to the formatted amount
// E.G. 1000000000nund becomes 1.00 FUND
// TODO Find a way to add currency amounts to IBC's, without overloading the CoinGecko API
func denomToAmount(msg string) string {
    amount, denom := splitAmountDenom(msg)
    // This will format the numbers in human readable form E.G.
    // 1000 FUND should become 1,000 FUND
    formatter := message.NewPrinter(language.English)
    if denom == config.Chain.Denom {
        exp, _ := strconv.ParseFloat("1" + strings.Repeat("0",config.Chain.Exponent), 64)
        amount = math.Round((amount/exp)*100)/100
        return formatter.Sprintf("%.2f %s (%.2f %s)", amount, config.Chain.DisplayName ,(*config.Chain.CoinGeckoData.Price * amount), config.Currency)
    } else if denom[:4] == "ibc/" {
        amount, denom, err := getIBC(amount ,denom[4:]) 
        if err != nil {
            return "Unknown IBC"
        }
        var price float64
        for i := range config.OtherChains {
            chain := &config.OtherChains[i]
            if chain.DisplayName == denom {
                // Only query for data we need, and start auto refreshing the data
                // Sleeps for a random amount of time, to wait for the response
                // This is needed to prevent overloading the CoinGecko API
                if !chain.CoinGeckoData.Active {
                    chain.CoinGeckoData.Active = true
                    url := "https://api.coingecko.com/api/v3/coins/" + chain.CoinGeckoData.ID
                    // Random amount of time to stagger the messages, prevent all the API Requests from hitting at once.
                    time.Sleep(time.Duration(rand.Intn(60-10+1)+10) * time.Second)
                    go autoRefresh(url, &chain.CoinGeckoData.Data)
                    // Wait for the data query
                    time.Sleep(10 * time.Second)
                }
                price = *chain.CoinGeckoData.Price
            }
        }
        if price == 0 {
            return formatter.Sprintf("%.2f %s (%s %s)", amount, denom,"?", config.Currency)
        }
        return formatter.Sprintf("%.2f %s (%.2f %s)", amount, denom,(price * amount), config.Currency)
    } else {
        return "Unknown IBC"
    }
}
// Removes any MD incompatible charactres from a string
func removeForbiddenChars(msg string) string {
	msg = regexp.MustCompile(`[\[\]\(\)*]`).ReplaceAllString(msg, "")
    return msg
}

func ensureTrailingSlash(str *string) {
    if !strings.HasSuffix(*str, "/") {
        *str += "/" 
    }
}
func ensureNoSpaces(str *string) {
    *str = strings.ReplaceAll(*str," ","-")
}
