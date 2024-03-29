package main

import (
	"log"
	"strconv"
	"strings"
    "fmt"
    "math"
    "errors"

	"github.com/fatih/color"
)

// Splits and amount like 1000nund, and returns 1000 and nund
func splitAmountDenom(amount string) (float64, string){
    var index int
    var amnt string
    for i, c := range amount{
        if _, err := strconv.Atoi(string(c)); err == nil {
            amnt += string(c)
            index = i
        } else {
            break
        }           
    }
    if amount == "" {
        return 0, "UnknownDenom"
    }
    denom := amount[index+1:]
    floatAmnt, _ := strconv.ParseFloat(amnt, 64)
    return floatAmnt, denom 
}
// Checks if the message is allowed to send based on the whitelist/blacklist rules defined
// TODO: use regex to ensure the string is clean for query (remove **, urls, etc)
func isAllowedMessage (res MessageResponse) bool {
    switch res.Type.Filter {
    case "blacklist":
        for _, str := range res.Type.WhiteBlackList {
            if strings.Contains(res.Message, str) {
                logMsg := fmt.Sprintf("Filtered Message! Message of type %s contained blacklisted item: %s",res.TypeName,str)
                log.Println(color.YellowString(logMsg))
                return false
            }
        }
        return true
    case "whitelist":
        for _, str := range res.Type.WhiteBlackList {
            if strings.Contains(res.Message, str) {
                return true
            }
        }
        logMsg := fmt.Sprintf("Filtered Message! Message of type %s did not contain any whitelisted items",res.TypeName)
        log.Println(color.YellowString(logMsg))
        return false
    default:
        return true
    }
}
// TODO Allow this function to be used with other chains.
func isAllowedAmount(res MessageResponse, msg string) bool {
    amount, denom := splitAmountDenom(msg)
    switch res.Type.AmountFilter {
    case true:
        if denom == config.Chain.Denom {
            exp, _ := strconv.ParseFloat("1" + strings.Repeat("0",config.Chain.Exponent), 64)
            amt := math.Round((amount/exp)*100)/100
            currencyAmount := amt * *config.Chain.CoinGeckoData.Price
            if currencyAmount < res.Type.Threshold {
                logMsg := fmt.Sprintf("Filtered Message! Message of type %s did not meet the currency threshold of: %.0f %s",res.TypeName,res.Type.Threshold, config.Currency)
                log.Println(color.YellowString(logMsg))
                return false
            } else {
                return true
            }
        } else {
            logMsg := fmt.Sprintf("Filtered Message! Message of type %s is an unknown currency conversion, so could not meet the currency threshold",res.TypeName)
            log.Println(color.YellowString(logMsg))
            return false
        }
    case false:
        return true
    }
    return true
}
// TODO Have this function read asset data from the chains as well, instead of just the primary denoms'
func getIBC(amount float64, denom string) (float64,string, error) {
    var ibc IBCResponse
    url := config.Connections.Rest + "/ibc/apps/transfer/v1/denom_traces/" + denom
    getData(url, &ibc)
    for _, chain := range(config.OtherChains) {
        if chain.Denom == ibc.DenomTrace.BaseDenom {
            display := chain.DisplayName
            exp, _ := strconv.ParseFloat("1" + strings.Repeat("0",chain.Exponent), 64)
            amount = math.Round((amount/exp)*100)/100
            return amount, strings.ToUpper(display), nil
        }
    }
    return 0, "", errors.New("Data not available")
}
