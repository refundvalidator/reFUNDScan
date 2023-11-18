package main

import (
    "log"
    "net/url"
    "os"
    "os/signal"
    "fmt"
    "encoding/json"
    "unicode"
    "math"
    "strconv"

    "github.com/gorilla/websocket"
)

const (
    fundExplorerTx string = "https://explorer.unification.io/transactions/"

    fundExplorerAccount string = "https://explorer.unification.io/accounts/"
    osmoExplorerAccount string = "https://www.mintscan.io/osmosis/address/"
    gravExplorerAccount string = "https://www.mintscan.io/gravity-bridge/address/"
)

type JsonResponse struct {
    Result struct {
        Events struct {
            MessageAction []string `json:"message.action"` 
            TransferSender []string `json:"transfer.sender"`
            TransferRecipient []string `json:"transfer.recipient"`
            IBCTransferSender []string `json:"ibc_transfer.sender"`
            IBCTransferReciever []string `json:"ibc_transfer.receiver"`
            IBCForeignSender []string `json:"fungible_token_packet.sender"`
            TransferAmount []string `json:"transfer.amount"`
            TxHash []string `json:"tx.hash"`
        } `json:"events"`
    } `json:"result"`
}

func denomToAmount(msg string) string {
    var amount string
    var denom string
    var breakpoint int
    // Extract the numerical amount from the msg
    for i, char := range msg {
        if unicode.IsDigit(char) {
            amount += string(char)
        } else {
            breakpoint = i
            break
        }
    } 
    // Extract denom like "nund" or "ibc/xxxx" from the msg
    for i, char := range msg {
        if i >= breakpoint {
            denom += string(char)
        } 
    }
    numericalAmount, _ := strconv.ParseFloat(amount, 64)

    if denom == "nund" {
        numericalAmount = math.Round((numericalAmount/1000000000)*100)/100
        formattedamount := strconv.FormatFloat(numericalAmount, 'f', 2, 64)
        return (formattedamount + " FUND")
     } else if denom == "ibc/ED07A3391A112B175915CD8FAF43A2DA8E4790EDE12566649D0C2F97716B8518" {
        //osmo
        numericalAmount = math.Round((numericalAmount/1000000)*100)/100
        formattedamount := strconv.FormatFloat(numericalAmount, 'f', 2, 64)
        return (formattedamount + " OSMO")
    }
    return "null"
}

func main() {
    interrupt := make(chan os.Signal, 1) 
    signal.Notify(interrupt, os.Interrupt) 
    u := url.URL {Scheme:"wss",Host:"rpc1.unification.io",Path:"/websocket"} 
    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)  
    if err != nil{
        log.Fatal("dail: ", err) 
        return
    } 
    defer c.Close()    

    subscribe := []byte(`{ "jsonrpc": "2.0", "method": "subscribe", "id": 0, "params": { "query": "tm.event='Tx'" } }`)
    err = c.WriteMessage(websocket.TextMessage, subscribe)
    if err != nil{
        log.Fatal("Couldn't Subscribe: " , err) 
        return
    } 

    done := make(chan string)  

    go func(){
        defer close(done)      
        for {
            _,m,err := c.ReadMessage()
            if err != nil{
                log.Fatal("read: ", err) 
                return
            } 
            var res JsonResponse // struct version of the json object
            if err := json.Unmarshal(m,&res); err != nil {
                fmt.Printf("Cannot Unmarshal")
            }
            if len(res.Result.Events.MessageAction) >= 1 {
                // TODO: Add restake transactions, rewards withdrawal, comission withdrawal, delegations, undelegations
                // governance votes, validator creations, validator edits, memos

                // On Chain Transfers
                if res.Result.Events.MessageAction[0] == "/cosmos.bank.v1beta1.MsgSend" {
                    fmt.Printf("Action:%s\nSender:%s\nReciever:%s\nAmount:%s\nHash:%s\n\n",
                        res.Result.Events.MessageAction[0],
                        res.Result.Events.TransferSender[0],
                        res.Result.Events.TransferRecipient[1],
                        denomToAmount(res.Result.Events.TransferAmount[1]),
                        res.Result.Events.TxHash[0]) 
                }
                // FUND > Other Chain Transfers
                if res.Result.Events.MessageAction[0] == "/ibc.applications.transfer.v1.MsgTransfer" {
                    fmt.Printf("Action:%s\nSender:%s\nReciever:%s\nAmount:%s\nHash:%s\n\n",
                        res.Result.Events.MessageAction[0],
                        res.Result.Events.IBCTransferSender[0],
                        res.Result.Events.IBCTransferReciever[0],
                        denomToAmount(res.Result.Events.TransferAmount[1]),
                        res.Result.Events.TxHash[0])
                }
                // Other Chain > FUND Transfers
                if len(res.Result.Events.MessageAction) >= 2 {
                    if res.Result.Events.MessageAction[1] == "/ibc.core.channel.v1.MsgRecvPacket" {
                        fmt.Printf("Action:%s\nSender:%s\nReciever:%s\nAmount:%s\nHash:%s\n\n",
                            res.Result.Events.MessageAction[1],
                            res.Result.Events.IBCForeignSender[0],
                            res.Result.Events.TransferRecipient[1],
                            denomToAmount(res.Result.Events.TransferAmount[1]),
                            res.Result.Events.TxHash[0])
                    }
                }
            }
        }  
    }()
    select {
    case <- done:
        fmt.Printf("Done")
        return
    case <- interrupt:
        fmt.Printf("Interrupt Detected, Done")
        return
    }
}
