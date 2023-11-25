package main

import (
    "log"
    "fmt"
    "encoding/json"

    "github.com/gorilla/websocket"
)

// Connect to the websocket and serve the formatted responses to the given channel resp
func Connect(resp chan string, restart chan bool) {
    c, _, err := websocket.DefaultDialer.Dial(WebsocketUrl, nil)  
    if err != nil{
        log.Println("Failed to dial websocket: ", err) 
        restart <- true
        return
    } 
    defer c.Close()    
    log.Println("Connected to websocket")

    subscribe := []byte(`{ "jsonrpc": "2.0", "method": "subscribe", "id": 0, "params": { "query": "tm.event='Tx'" } }`)
    err = c.WriteMessage(websocket.TextMessage, subscribe)
    if err != nil{
        log.Println("Couldn't subscribe to websocket: " , err) 
        restart <- true
        return
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
            var res WebsocketResponse // struct version of the json object
            if err := json.Unmarshal(m,&res); err != nil {
                log.Println("Couldn't unmarshal json response: ", err)
                restart <- true
                break
            }
            events := res.Result.Events
            if len(events.MessageAction) >= 1 {
                // TODO: Add rewards withdrawal, commission withdrawal,
                // governance votes, validator creations, validator edits 

                // Fix small amounts displaying as 0.00

                if events.MessageAction[0] == "/cosmos.bank.v1beta1.MsgSend" {
                    // On Chain Transfers
                    msg := "‎" +
                        mkBold("\n📬 Transfer 📬") +
                        mkBold("\n\nSender: ") +
                        mkAccountLink(events.TransferSender[0]) +
                        mkBold("\nReciever: ") +
                        mkAccountLink(events.TransferRecipient[1]) +
                        mkBold("\nAmount: ") +
                        mkTranscationLink(events.TxHash[0],events.TransferAmount[1]) 
                    if memo := getMemo(events.TxHash[0]); memo != "" {
                        msg += mkBold("\nMemo: " + memo)
                    }
                    msg += "\n‎"
                    resp <- msg 
                } else if res.Result.Events.MessageAction[0] == "/ibc.applications.transfer.v1.MsgTransfer" {
                    // FUND > Other Chain IBC
                    msg := "‎" +
                        mkBold("\n⚛️ IBC Transfer ⚛️") + 
                        mkBold("\n\nSender: ") +
                        mkAccountLink(events.IBCTransferSender[0]) +
                        mkBold("\nReciever: ") +
                        mkAccountLink(events.IBCTransferReciever[0]) +
                        mkBold("\nAmount: ") +
                        mkTranscationLink(events.TxHash[0],events.TransferAmount[1])
                    if memo := getMemo(events.TxHash[0]); memo != "" {
                        msg += mkBold("\nMemo: " + memo)
                    }
                    msg += "\n‎"
                    resp <- msg 
                } else if res.Result.Events.MessageAction[0] == "/cosmos.staking.v1beta1.MsgDelegate" {
                    // Delegations
                    msg := "‎" +
                        mkBold("\n❤️ Delegate ❤️") + 
                        mkBold("\n\nValidator: ") +
                        mkAccountLink(events.DelegateValidator[0]) +
                        mkBold("\nDelegator: ") +
                        mkAccountLink(events.MessageSender[0]) +
                        mkBold("\nAmount: ") +
                        mkTranscationLink(events.TxHash[0],events.DelegateAmount[0])
                    if memo := getMemo(events.TxHash[0]); memo != "" {
                        msg += mkBold("\nMemo: " + memo)
                    }
                    msg += "\n‎"
                    resp <- msg 
                } else if res.Result.Events.MessageAction[0] == "/cosmos.staking.v1beta1.MsgUndelegate" {
                    // Undelegations
                    msg := "‎" +
                        mkBold("\n💀 Undelegate 💀") + 
                        mkBold("\n\nValidator: ") +
                        mkAccountLink(events.UnbondValidator[0]) +
                        mkBold("\nDelegator: ") +
                        mkAccountLink(events.MessageSender[0]) +
                        mkBold("\nAmount: ") +
                        mkTranscationLink(events.TxHash[0],events.UnbondAmount[0])
                    if memo := getMemo(events.TxHash[0]); memo != "" {
                        msg += mkBold("\nMemo: " + memo)
                    }
                    msg += "\n‎"
                    resp <- msg 
                } else if res.Result.Events.MessageAction[0] == "/cosmos.staking.v1beta1.MsgBeginRedelegate" {
                    // Redelegations
                    msg := "‎" +
                        mkBold("\n💞 Redelegate 💞") + 
                        mkBold("\n\nValidators: ") +
                        mkAccountLink(events.RedelegateSourceValidator[0]) +
                        mkBold(" 👉 ") +
                        mkAccountLink(events.RedelegateDestinationValidator[0]) +
                        mkBold("\nDelegator: ") +
                        mkAccountLink(events.MessageSender[0]) +
                        mkBold("\nAmount: ") +
                        mkTranscationLink(events.TxHash[0],events.RedelegateAmount[0])
                    if memo := getMemo(events.TxHash[0]); memo != "" {
                        msg += mkBold("\nMemo: " + memo)
                    }
                    msg += "\n‎"
                    resp <- msg 
                } else if events.MessageAction[0] == "/cosmos.authz.v1beta1.MsgExec" {
                    // REStake Transactions
                    msg := "‎" +
                        mkBold("\n♻️ REStake ♻️") +
                        mkBold("\n\nValidator: ") +
                        mkAccountLink(events.WithdrawRewardsValidator[0]) +
                        mkBold("\nDelegators: \n")
                    j := 0
                    for i, delegator := range events.MessageSender {
                        if i >= 2 {
                            if i % 2 != 0 {
                                j += 1
                                msg += fmt.Sprintf("\n%s\n%s\n", mkAccountLink(delegator) ,mkTranscationLink(events.TxHash[0],events.TransferAmount[j]))
                            }
                        }
                    }
                    msg += mkBold("\nFees paid by ")
                    msg += mkAccountLink(events.WithdrawRewardsValidator[0]) + ": " + mkTranscationLink(events.TxHash[0], events.TransferAmount[0])
                    if memo := getMemo(events.TxHash[0]); memo != "" {
                        msg += mkBold("\nMemo: " + memo)
                    }
                    msg += "\n‎"
                    resp <- msg 
                } else if len(events.MessageAction) >= 2 {
                    // Other Chain > FUND IBC
                    if events.MessageAction[1] == "/ibc.core.channel.v1.MsgRecvPacket" {
                        msg := "‎" +
                            mkBold("\n⚛️ IBC Transfer ⚛️") +
                            mkBold("\n\nSender: ") +
                            mkAccountLink(events.IBCForeignSender[0]) +
                            mkBold("\nReciever: ") +
                            mkAccountLink(events.TransferRecipient[1]) +
                            mkBold("\nAmount: ") +
                            mkTranscationLink(events.TxHash[0],events.TransferAmount[1])
                        if memo := getMemo(events.TxHash[0]); memo != "" {
                            msg += mkBold("\nMemo: " + memo)
                        }
                        msg += "\n‎"
                        resp <- msg 
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
