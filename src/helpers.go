package main

import (
	"log"
	"strconv"
	"strings"
    "fmt"
    "math"

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
func isAllowedAmount(res MessageResponse, msg string) bool {
    amount, denom := splitAmountDenom(msg)
    switch res.Type.AmountFilter {
    case true:
        if denom == config.Denom {
            exp, _ := strconv.ParseFloat("1" + strings.Repeat("0",config.Exponent), 64)
            amt := math.Round((amount/exp)*100)/100
            currencyAmount := amt * *config.CurrencyAmount     
            if currencyAmount < res.Type.Threshold {
                logMsg := fmt.Sprintf("Filtered Message! Message of type %s did not meet the currency threshold of: %.0f %s",res.TypeName,res.Type.Threshold, config.Currency)
                log.Println(color.YellowString(logMsg))
                return false
            } else {
                return true
            }
        } else {
            logMsg := fmt.Sprintf("Filtered Message! Message of type %s is an unknown conversion, so could not meet the currency threshold",res.TypeName)
            log.Println(color.YellowString(logMsg))
            return false
        }
    case false:
        return true
    }
    return true
}
