package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/fatih/color"
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
        return fmt.Sprintf("<a href=\"%s%s\">%s</a>",config.ExplorerValidator,addr,getAccountName(addr))
    }
    switch addr[:3]{
    case "osm":
        return fmt.Sprintf("<a href=\"%s%s\">%s</a>",osmoExplorerAccount,addr,getAccountName(addr))
    case "gra":
        return fmt.Sprintf("<a href=\"%s%s\">%s</a>",gravExplorerAccount,addr,getAccountName(addr))
    default:
        return fmt.Sprintf("<a href=\"%s%s\">%s</a>",config.ExplorerAccount,addr,getAccountName(addr))
    }
}

// Returns a HTML formatted hyprlink for a transaction when given a TX Hash with an amount
func mkTranscationLink(hash string, amount string) string {
    return fmt.Sprintf("<a href=\"%s%s\">%s</a>",config.ExplorerTx,hash,denomToAmount(amount))
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
    return tx.Tx.Body.Memo
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
            return name.Name
        }
    }

    // Check if name matches wallet or val addr
    for key, val := range names {
        if val[0] == msg || val[1] == msg {
            return key
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
        return icns.Data.PrimaryName
    }

    // Return truncated addr if the addr isnt in the named map
    return fmt.Sprintf("%s...%s",msg[:7],msg[len(msg)-7:])
}

func denomsToAmount() func(string) string{
    var total float64
    return func(msg string) string {
        var amount string
        var denom string
        var index int

        switch msg[len(msg)-len(config.Denom):] {
        case config.Denom:
            denom = config.Denom
            amount = msg[:len(msg)-len(config.Denom)]
        default:
            // Other IBC denoms such as ibc/xxxx
            // IBC denom hash is always 64 chars + 4 chars for the ibc/
            for i, c := range msg{
                if _, err := strconv.Atoi(string(c)); err == nil {
                    amount += string(c)
                    index = i
                } else {
                    break
                }           
            }
            denom = msg[index:]
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
    var index int

    switch msg[len(msg)-len(config.Denom):] {
    case config.Denom:
        denom = config.Denom
        amount = msg[:len(msg)-len(config.Denom)]
    default:
        // Other IBC denoms such as ibc/xxxx
        // IBC denom hash is always 64 chars + 4 chars for the ibc/
        for i, c := range msg{
            if _, err := strconv.Atoi(string(c)); err == nil {
                amount += string(c)
               index = i
            } else {
                break
            }           
        }
        denom = msg[index:]
        // denom = msg[len(msg)-68:]
        // amount = msg[:len(msg)-68]
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

// Checks if the message is allowed to send based on the whitelist/blacklist rules defined
func isAllowedMessage (config MessageConfig, msg string) bool {
    switch config.Filter {
    case "blacklist":
        for _, str := range config.WhiteBlackList {
            if strings.Contains(msg, str) {
                log.Println(color.YellowString("Filtered Message! Message contained blacklisted item: " + str))
                return false
            }
        }
        return true
    case "whitelist":
        for _, str := range config.WhiteBlackList {
            if strings.Contains(msg, str) {
                return true
            }
        }
        log.Println(color.YellowString("Filtered Message! Message did not contain any whitelisted item"))
        return false
    default:
        return true
    }
}
