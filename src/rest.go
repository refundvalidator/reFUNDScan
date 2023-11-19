package main

import(
    "net/http"
    "log"
    "encoding/json"
    "io"
    "time"
)

type TxResponse struct {
    
} 
type CoinGeckoResponse struct {
    
}
type ValidatorResponse struct {
    
}

var Validators ValidatorResponse
var Price CoinGeckoResponse

func autoRefresh() {
    refreshPrice()
    refreshValidators()
    time.Sleep(time.Second * 60)
    autoRefresh()
}
func refreshPrice() CoinGeckoResponse {
    response, err := http.Get("") 
    if err != nil {
        log.Fatal("Failed to grab response from CoinGecko: ", err)
    }
    defer response.Body.Close()
    body, err := io.ReadAll(response.Body)
    if err != nil {
        log.Fatal("Failed to read response from CoinGecko: ", err)
    }
    var cg CoinGeckoResponse
    err = json.Unmarshal(body, &cg)
    if err != nil {
        log.Fatal("Failed to unmarshal JSON from CoinGecko: ", err)
    }
    return cg
}
func refreshValidators() ValidatorResponse {
     response, err := http.Get("") 
    if err != nil {
        log.Fatal("Failed to grab response from REST: ", err)
    }
    defer response.Body.Close()
    body, err := io.ReadAll(response.Body)
    if err != nil {
        log.Fatal("Failed to read response from REST: ", err)
    }
    var vals ValidatorResponse
    err = json.Unmarshal(body, &vals)
    if err != nil {
        log.Fatal("Failed to unmarshal JSON from REST: ", err)
    }
    return vals 
}
func getTx(hash string) TxResponse {
    response, err := http.Get("") 
    if err != nil {
        log.Fatal("Failed to grab response from REST: ", err)
    }
    defer response.Body.Close()
    body, err := io.ReadAll(response.Body)
    if err != nil {
        log.Fatal("Failed to read response from REST: ", err)
    }
    var tx TxResponse
    err = json.Unmarshal(body, &tx)
    if err != nil {
        log.Fatal("Failed to unmarshal JSON from REST: ", err)
    }
    return tx
}
