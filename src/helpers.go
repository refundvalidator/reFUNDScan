package main

import (
	"strconv"
    "strings"
    "log"
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
// TODO: isAllowedMessage
