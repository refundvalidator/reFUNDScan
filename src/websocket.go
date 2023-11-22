package main

import (
    "log"
    "fmt"
    "encoding/json"
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

// The JSON Response received by the websocket
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

// Checks if the wallet is a names wallet and returns the name if it is, otherwise returns a
// Truncated version of the address
func getAccountName(msg string) string {
    named := map[string]string{
        "BitForex": "und18mcmhkq6fmhu9hpy3sx5cugqwv6z0wrz7nn5d7",
        "Poloniex" : "und186slma7kkxlghwc3hzjr9gkqwhefhln5pw5k26",
        "ProBit" : "und1jkhkllr3ws3uxclawn4kpuuglffg327wvfg8r9",
        "DigiFinex" : "und1xnrruk9qlgnmh8qxcz9ypfezj45qk96v2rgnzk",
    }
    for key, val := range named {
        if val == msg{
            return key
        }
    }
    // Return truncated addr if the addr isnt in the named map
    return fmt.Sprintf("%s...%s",msg[:7],msg[len(msg)-7:])
}

// Returns the correct ExplorerAccount url depending on the address type
// Defaults to fundExplorerAccount if the addr type is unknown
func getExplorerAccount(msg string) string {
    switch msg[:3] {
    case "osm":
        return osmoExplorerAccount
    case "gra":
        return gravExplorerAccount
    default:
        return fundExplorerAccount
    }
}

// Converts the denom to the formatted amount
// E.G. 1000000000nund becomes 1.00 FUND
// If the denom is unknown, returns "null"
func denomToAmount(msg string) string {
    var amount string
    var denom string

    switch msg[len(msg)-4:] {
    case "nund":
        denom = "nund"
        amount = msg[:len(msg)-4]
    default:
        // Other IBC denoms such as ibc/xxxx
        // IBC denom hash is always 64 chars + 4 chars for the ibc/
        denom = msg[len(msg)-68:]
        amount = msg[:len(msg)-68]
    }

    numericalAmount, _ := strconv.ParseFloat(amount, 64)

    switch denom {
    case "nund":
        // Fund
        numericalAmount = math.Round((numericalAmount/1000000000)*100)/100
        formattedamount := strconv.FormatFloat(numericalAmount, 'f', 2, 64)
        return (formattedamount + " FUND")
    case "ibc/ED07A3391A112B175915CD8FAF43A2DA8E4790EDE12566649D0C2F97716B8518":
        // Osmo
        numericalAmount = math.Round((numericalAmount/1000000)*100)/100
        formattedamount := strconv.FormatFloat(numericalAmount, 'f', 2, 64)
        return (formattedamount + " OSMO")
    default:
        // Unknown IBC Types
        return "null"
    }
}

// Connect to the websocket and serve the formatted responses to the given channel resp
func Connect(resp chan string, restart chan bool) {
    c, _, err := websocket.DefaultDialer.Dial(Url, nil)  
    if err != nil{
        log.Println("Failed to dial websocket: ", err) 
        restart <- true
    } 
    defer c.Close()    
    log.Println("Connected to websocket")

    subscribe := []byte(`{ "jsonrpc": "2.0", "method": "subscribe", "id": 0, "params": { "query": "tm.event='Tx'" } }`)
    err = c.WriteMessage(websocket.TextMessage, subscribe)
    if err != nil{
        log.Println("Couldn't subscribe to websocket: " , err) 
        restart <- true
    } 
    log.Println("Subscribed to websocket")

    done := make(chan string)  

    go func(){
        log.Println("Listening for messages")
        defer close(done)      
        for {
            _,m,err := c.ReadMessage()
            if err != nil{
                log.Println("Failed to read json response: ", err) 
                restart <- true
                break
            } 
            var res JsonResponse // struct version of the json object
            if err := json.Unmarshal(m,&res); err != nil {
                log.Println("Couldn't unmarshal json response: ", err)
                restart <- true
                break
            }
            events := res.Result.Events
            if len(events.MessageAction) >= 1 {
                // TODO: Add restake transactions, rewards withdrawal, comission withdrawal, delegations, undelegations
                // governance votes, validator creations, validator edits, memos

                if events.MessageAction[0] == "/cosmos.bank.v1beta1.MsgSend" {
                    // On Chain Transfers
                    resp <- fmt.Sprintf("‎\n<b>📬%s📬</b>\n\n<b>Sender:</b> <a href=\"%s%s\">%s</a>\n<b>Reciever:</b> <a href=\"%s%s\">%s</a>\n<b>Amount:</b> <a href=\"%s%s\">%s</a>\n\n",
                        "Transfer",
                        fundExplorerAccount,
                        events.TransferSender[0],
                        getAccountName(events.TransferSender[0]),
                        fundExplorerAccount,
                        events.TransferRecipient[1],
                        getAccountName(events.TransferRecipient[1]),
                        fundExplorerTx,
                        events.TxHash[0],
                        denomToAmount(events.TransferAmount[1]))
                } else if res.Result.Events.MessageAction[0] == "/ibc.applications.transfer.v1.MsgTransfer" {
                    // FUND > Other Chain IBC
                    resp <- fmt.Sprintf("‎\n<b>⚛️%s⚛️</b>\n\n<b>Sender:</b> <a href=\"%s%s\">%s</a>\n<b>Reciever:</b> <a href=\"%s%s\">%s</a>\n<b>Amount:</b> <a href=\"%s%s\">%s</a>\n\n",
                        "IBC Transfer",
                        getExplorerAccount(events.IBCTransferSender[0]),
                        events.IBCTransferSender[0],
                        getAccountName(events.IBCTransferSender[0]),
                        getExplorerAccount(events.IBCTransferReciever[0]),
                        events.IBCTransferReciever[0],
                        getAccountName(events.IBCTransferReciever[0]),
                        fundExplorerTx,
                        events.TxHash[0],
                        denomToAmount(events.TransferAmount[1]))

                } else if len(events.MessageAction) >= 2 {
                    // Other Chain > FUND IBC
                    if events.MessageAction[1] == "/ibc.core.channel.v1.MsgRecvPacket" {
                        resp <- fmt.Sprintf("‎\n<b>⚛️%s⚛️</b>\n\n<b>Sender:</b> <a href=\"%s%s\">%s</a>\n<b>Reciever:</b> <a href=\"%s%s\">%s</a>\n<b>Amount:</b> <a href=\"%s%s\">%s</a>\n\n",
                            "IBC Transfer",
                            getExplorerAccount(events.IBCForeignSender[0]),
                            events.IBCForeignSender[0],
                            getAccountName(events.IBCForeignSender[0]),
                            getExplorerAccount(events.TransferRecipient[1]),
                            events.TransferRecipient[1],
                            getAccountName(events.TransferRecipient[1]),
                            fundExplorerTx,
                            events.TxHash[0],
                            denomToAmount(events.TransferAmount[1]))
                    }
                }
            }
        }  
    }()
    select {
    case <- done:
        log.Println("Listener terminating")
        return
    }
}
