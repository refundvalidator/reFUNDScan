package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
    "regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/btcsuite/btcutil/bech32"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)


// TODO:
// Remove the need for these pre-defined URLs for universal links
var (
    osmoExplorerAccount = "https://www.mintscan.io/osmosis/address/"
    gravExplorerAccount = "https://www.mintscan.io/gravity-bridge/address/"
)

// Returns and MD formatted hyperlink for an account when given a wallet or validator address
func mkAccountLink(addr string) string{
    switch addr[:len(config.Bech32Prefix + "val")]{
    case config.Bech32Prefix + "val":
        return fmt.Sprintf("[%s](%s%s)",getAccountName(addr),config.ExplorerValidator,addr)
    }
    switch addr[:3]{
    case "osm":
        return fmt.Sprintf("[%s](%s%s)",getAccountName(addr),osmoExplorerAccount,addr)
    case "gra":
        return fmt.Sprintf("[%s](%s%s)",getAccountName(addr),gravExplorerAccount,addr)
    default:
        return fmt.Sprintf("[%s](%s%s)",getAccountName(addr),config.ExplorerAccount,addr)
    }
}

// Returns a MD formatted hyprlink for a transaction when given a TX Hash with an amount
func mkTranscationLink(hash string, amount string) string {
    return fmt.Sprintf("[%s](%s%s)", denomToAmount(amount), config.ExplorerTx,hash)
}

// When given a transaction hash
// Searches rest endpoints for a memo on the transaction, if not available returns an empty string
func getMemo(hash string) string {
    var tx TxResponse
    err := getData(config.RestTx + hash, &tx)
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
        addr, err := bech32.Encode(config.Bech32Prefix,data)
        if err != nil {
            log.Println(color.YellowString("Could not encode bech32 address"))
            continue
        }
        names[val.Description.Moniker] = []string{addr, val.OperatorAddress}
    }

    // Check if name matches named wallet from config
    for _, name := range config.Named {
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
    err := getData(config.ICNSAccount + b64, &icns)
    if err != nil {
        log.Println(color.YellowString("Failed to get ICNS response ", err))
    }
    if icns.Data.PrimaryName != "" {
        return removeForbiddenChars(icns.Data.PrimaryName)
    }

    // Return truncated addr if the addr isnt in the named map
    return fmt.Sprintf("%s...%s",msg[:7],msg[len(msg)-7:])
}

// TODO: Split up this functinon, and create a config file entry 
// to set custom IBC's
// Also, setup predefined IBC's using the chains' assetlist
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
func denomToAmount(msg string) string {
    amount, denom := splitAmountDenom(msg)
    // This will format the numbers in human readable form E.G.
    // 1000 FUND should become 1,000 FUND
    formatter := message.NewPrinter(language.English)
    if denom == config.Denom {
        exp, _ := strconv.ParseFloat("1" + strings.Repeat("0",config.Exponent), 64)
        amount = math.Round((amount/exp)*100)/100
        return formatter.Sprintf("%.2f %s (%.2f %s)", amount, config.Coin ,(*config.CurrencyAmount * amount), config.Currency)
    } else if denom[:4] == "ibc/" {
       amount, denom, err := getIBC(amount ,denom[4:]) 
        if err != nil {
            return "Unknown IBC"
        }
       return formatter.Sprintf("%.2f %s", amount, denom)
    } else {
        return "Unknown IBC"
    }
}
// Removes any MD incompatible charactres from a string
func removeForbiddenChars(msg string) string {
	msg = regexp.MustCompile(`[\[\]\(\)*]`).ReplaceAllString(msg, "")
    return msg
}
