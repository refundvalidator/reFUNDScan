package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
    osmoExplorerAccount = "https://www.mintscan.io/osmosis/address/"
    gravExplorerAccount = "https://www.mintscan.io/gravity-bridge/address/"
)

// Places a string in HTML bold brackets
func mkBold(msg string) string{
    return fmt.Sprintf("<b>%s</b>",msg)
}

// Returns and HTML formatted hyperlink for an account when given a wallet or validator address
func mkAccountLink(addr string) string{
    switch addr[:len(config.Bech32Prefix + "val")]{
    case config.Bech32Prefix + "val":
        return fmt.Sprintf("<a href=\"%s%s\">%s</a>",config.explorerValidators,addr,getAccountName(addr))
    }
    switch addr[:3]{
    case "osm":
        return fmt.Sprintf("<a href=\"%s%s\">%s</a>",osmoExplorerAccount,addr,getAccountName(addr))
    case "gra":
        return fmt.Sprintf("<a href=\"%s%s\">%s</a>",gravExplorerAccount,addr,getAccountName(addr))
    default:
        return fmt.Sprintf("<a href=\"%s%s\">%s</a>",config.explorerAccount,addr,getAccountName(addr))
    }
}

// Returns a HTML formatted hyprlink for a transaction when given a TX Hash with an amount
func mkTranscationLink(hash string, amount string) string {
    return fmt.Sprintf("<a href=\"%s%s\">%s</a>",config.explorerTx,hash,denomToAmount(amount))
}

// When given a transaction hash
// Searches rest endpoints for a memo on the transaction, if not available returns an empty string
func getMemo(hash string) string {
    var tx TxResponse
    err := getData(config.restTx + hash, &tx)
    if err != nil {
        log.Println("Failed to get TX rest response: ", err)
        return ""
    }
    return tx.Tx.Body.Memo
}

// When given a wallet or validator address, returns the name associated with the wallet, if it has one
// Otherwise returns a truncated version of the wallet address
func getAccountName(msg string) string {

    // Known account names
    named := map[string][]string{
        "BitForex 🏦": {"und18mcmhkq6fmhu9hpy3sx5cugqwv6z0wrz7nn5d7", ""},
        "Poloniex 🏦" : {"und186slma7kkxlghwc3hzjr9gkqwhefhln5pw5k26",""},
        "ProBit 🏦" : {"und1jkhkllr3ws3uxclawn4kpuuglffg327wvfg8r9",""},
        "DigiFinex 🏦" : {"und1xnrruk9qlgnmh8qxcz9ypfezj45qk96v2rgnzk",""},
        "All Unjailed Delegations" : {"und1fl48vsnmsdzcv85q5d2q4z5ajdha8yu3j7wxl3",""},
        "Burn Address 🔥" : {"und1qqqqqqqqqqqqqqqqqqqqqqqqqqqqph4djz5txt",""},
        "Unbonding/Jailed Delegations" : {"und1tygms3xhhs3yv487phx3dw4a95jn7t7lx7jhf9",""},
        "Locked eFUND" : {"und1nwt6chnk0efe8ngwa5y63egmdumht6arlvluh3",""},
        "wFUND" : {"und12k2pyuylm9t7ugdvz67h9pg4gmmvhn5vcrzmhj",""},
        "Foundation Wallet #1\n( Cold Wallet ) 🏛️" : {"und1fxnqz9evaug5m4xuh68s62qg9f5xe2vzsj44l8",""},
        "Foundation Wallet #2 🏛️" : {"und1pyqttnfyqujh4hvjhcx45mz8svptp6f40n4u3p",""},
        "Foundation Wallet #3 🏛️" : {"und1hdn830wndtquqxzaz3rds7r7hqgpsg5q9ggxpk",""},
        "Foundation Wallet #4 🏛️" : {"und1cwhkh2ag8w2lf3ngd509wzy43ljxkkn3qe3q4z",""},
        "The Arbitrager 🕵️" : {"und1d268glh7hns5p6p4yxeuhqqg5v7aygn84u336u",""},
    }
    // Convert undval to und1 addresses and append to map
    for _, val := range vals.Validators {
        _, data, err := bech32.Decode(val.OperatorAddress)
        if err != nil {
            log.Println("Could not decode bech32 address") 
        }
        addr, err := bech32.Encode(config.Bech32Prefix,data)
        if err != nil {
            log.Println("Could not encode bech32 address")
        }
        named[val.Description.Moniker] = []string{addr, val.OperatorAddress}
    }
    // Check if name matches wallet or val addr
    for key, val := range named {
        if val[0] == msg || val[1] == msg {
            return key
        }
    }

    // Check ICNS for name
    var icns ICNSResponse
    query := fmt.Sprintf(`{ "primary_name": { "address": "%s" }}`, msg)
    b64 := base64.StdEncoding.EncodeToString([]byte(query))
    err := getData(config.ICNSUrl + "/cosmwasm/wasm/v1/contract/osmo1xk0s8xgktn9x5vwcgtjdxqzadg88fgn33p8u9cnpdxwemvxscvast52cdd/smart/" + b64, &icns)
    if err != nil {
        log.Println("Failed to get ICNS response ", err)
    }
    if icns.Data.Name != "" {
        return icns.Data.Name + " (ICNS)"
    }

    // Return truncated addr if the addr isnt in the named map
    return fmt.Sprintf("%s...%s",msg[:7],msg[len(msg)-7:])
}

func denomsToAmount() func(string) string{
    var total float64
    return func(msg string) string {
        var amount string
        var denom string

        switch msg[len(msg)-len(config.Denom):] {
        case config.Denom:
            denom = config.Denom
            amount = msg[:len(msg)-len(config.Denom)]
        default:
            // Other IBC denoms such as ibc/xxxx
            // IBC denom hash is always 64 chars + 4 chars for the ibc/
            denom = msg[len(msg)-68:]
            amount = msg[:len(msg)-68]
        }
        numericalAmount, _ := strconv.ParseFloat(amount, 64)
        total += numericalAmount
        return fmt.Sprintf("%f%s",total,denom)
    }
}

// Converts the denom to the formatted amount
// E.G. 1000000000nund becomes 1.00 FUND
func denomToAmount(msg string) string {
    var amount string
    var denom string

    fmt.Println(msg)
    switch msg[len(msg)-len(config.Denom):] {
    case config.Denom:
        denom = config.Denom
        amount = msg[:len(msg)-len(config.Denom)]
    default:
        // Other IBC denoms such as ibc/xxxx
        // IBC denom hash is always 64 chars + 4 chars for the ibc/
        denom = msg[len(msg)-68:]
        amount = msg[:len(msg)-68]
    }

    numericalAmount, _ := strconv.ParseFloat(amount, 64)
    // This will format the numbers in human readable form E.G. 1000 FUND should become 1,000 FUND
    formatter := message.NewPrinter(language.English)

    switch denom {
    case config.Denom:
        // Fund
        exp, _ := strconv.ParseFloat("1" + strings.Repeat("0",config.Exponent), 64)
        numericalAmount = math.Round((numericalAmount/exp)*100)/100
        return formatter.Sprintf("%.2f %s ($%.2f USD)", numericalAmount, config.Coin ,(cg.MarketData.CurrentPrice.USD * numericalAmount))
    case "ibc/ED07A3391A112B175915CD8FAF43A2DA8E4790EDE12566649D0C2F97716B8518":
        // Osmo
        numericalAmount = math.Round((numericalAmount/1000000)*100)/100
        return formatter.Sprintf("%.2f OSMO", numericalAmount)
    case "ibc/C950356239AD2A205DE09FDF066B1F9FF19A7CA7145EA48A5B19B76EE47E52F7":
        // Grav
        numericalAmount = math.Round((numericalAmount/1000000)*100)/100
        return formatter.Sprintf("%.2f GRAV", numericalAmount)
    default:
        return "Unknown IBC"
    }
}

