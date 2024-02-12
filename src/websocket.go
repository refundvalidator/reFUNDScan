package main

import (
    "log"
    "fmt"
    "encoding/json"
    "reflect"

    "github.com/fatih/color"
    "github.com/gorilla/websocket"
)
type MessageResponse struct {
    Type     MessageConfig 
    TypeName string
    Amount   string 
    Message  string
}
// Connect to the websocket and serve the formatted responses to the given channel resp
func Connect(resp chan MessageResponse, restart chan bool) {
    c, _, err := websocket.DefaultDialer.Dial(config.Connections.Websocket, nil)  
    if err != nil{
        log.Println(color.YellowString("Failed to dial websocket: ", err))
        restart <- true
        return
    }
    defer c.Close()
    log.Println(color.BlueString("Connected to websocket"))

    subscribe := []byte(`{ "jsonrpc": "2.0", "method": "subscribe", "id": 0, "params": { "query": "tm.event='Tx'" } }`)
    err = c.WriteMessage(websocket.TextMessage, subscribe)
    if err != nil{
        log.Println(color.YellowString("Couldn't subscribe to websocket: " , err))
        restart <- true
        return
    }
    log.Println(color.BlueString("Subscribed to websocket"))

    done := make(chan string)  

    go func(){
        log.Println(color.GreenString("Listening for messages"))
        defer close(done)
        for {
            _,m,err := c.ReadMessage()
            if err != nil{
                log.Println(color.YellowString("Failed to read json response: ", err))
                restart <- true
                break
            }
            var res WebsocketResponse // struct version of the json object
            if err := json.Unmarshal(m,&res); err != nil {
                log.Println(color.YellowString("Couldn't unmarshal json response: ", err))
                restart <- true
                break
            }
            // Execute the parsing in its own thread, since some functions can delay the message
            // causing blockage
            go func(){
                events := res.Result.Events
                sent := 0
                for _, ev := range events.MessageAction {
                    // TODO: governance votes, validator creations, validator edits 
                    // Fix small amounts displaying as 0.00: maybe not <?
                    // Split this file, maybe into messages.go?

                    var msg MessageResponse
                    // Probably not possible, but just in case
                    if len(events.TxHash) < 1 {
                        continue 
                    }
                    if ev == "/cosmos.bank.v1beta1.MsgSend" && config.Config.MessagesConfig.Transfers.Enabled {
                        if len(events.TransferSender) < 1 ||
                        len(events.TransferRecipient) < 2 ||
                        len(events.TransferAmount) < 2 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.Transfers
                        msg.TypeName = "Transfer"
                        // On Chain Transfers
                        msg.Message +=
                            "\n** 📬 Transfer 📬 **" +
                            "\n\n**Sender:** " +
                            mkAccountLink(events.TransferSender[0]) +
                            "\n**Recipient:** " +
                            mkAccountLink(events.TransferRecipient[1]) +
                            "\n**Amount:** " +
                            mkTranscationLink(events.TxHash[0],events.TransferAmount[1]) 
                        if !isAllowedAmount(msg, events.TransferAmount[1]) {
                            continue
                        }

                    } else if ev == "/ibc.applications.transfer.v1.MsgTransfer" && config.Config.MessagesConfig.IBCOut.Enabled {
                        // FUND > Other Chain IBC
                        if len(events.IBCTransferSender) < 1 ||
                        len(events.IBCTransferRecipient) < 1 ||
                        len(events.TransferAmount) < 2 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.IBCOut
                        msg.TypeName = "IBCOut"
                        msg.Message += 
                            "\n** ⚛️ IBC Out ⚛️ **" + 
                            "\n\n**Sender:** " +
                            mkAccountLink(events.IBCTransferSender[0]) +
                            "\n**Recipient:** " +
                            mkAccountLink(events.IBCTransferRecipient[0]) +
                            "\n**Amount:** " +
                            mkTranscationLink(events.TxHash[0],events.TransferAmount[1])
                        if !isAllowedAmount(msg, events.TransferAmount[1]) {
                            continue
                        }

                    } else if ev == "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward" && config.Config.MessagesConfig.Rewards.Enabled {
                        // Withdraw rewards
                        if len(events.WithdrawRewardsDelegator) < 1 ||
                        len (events.WithdrawRewardsValidator) < 1 ||
                        len (events.WithdrawRewardsAmount) < 1 {
                            continue 
                        }
                        msg.Type = config.Config.MessagesConfig.Rewards
                        msg.TypeName = "Rewards"
                        msg.Message +=
                             "\n** 💵 Withdraw Reward 💵 **" +
                             "\n\n**Delegator:** \n" +
                             mkAccountLink(events.WithdrawRewardsDelegator[0]) +
                             "\n\n**Validators:** "
                        var total string
                        totaler := denomTotaler()
                        for i, val := range events.WithdrawRewardsValidator{
                            msg.Message += fmt.Sprintf("\n%s\n%s",mkAccountLink(val), denomToAmount(events.WithdrawRewardsAmount[i]))
                            total = totaler(events.WithdrawRewardsAmount[i])
                        }
                        msg.Message += "\n\n**Total:** \n" + mkTranscationLink(events.TxHash[0],total)
                        if !isAllowedAmount(msg, total) {
                            continue
                        }

                    } else if ev == "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission" && config.Config.MessagesConfig.Commission.Enabled {
                        // Withdraw commission
                        if len(events.WithdrawCommissionAmount) < 1 ||
                        len(events.WithdrawRewardsDelegator) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.Commission
                        msg.TypeName = "Commission"
                        msg.Message +=
                             "\n** 💸 Withdraw Commission 💸 **" +
                             "\n\n**Validator:** " +
                             mkAccountLink(events.WithdrawRewardsDelegator[0]) +
                             "\n**Amount:** " +
                             mkTranscationLink(events.TxHash[0],events.WithdrawCommissionAmount[0])
                        if !isAllowedAmount(msg, events.WithdrawCommissionAmount[0]) {
                            continue
                        }               

                    } else if ev == "/cosmos.staking.v1beta1.MsgDelegate" && config.Config.MessagesConfig.Delegations.Enabled {
                        // Delegations
                        if len(events.DelegateValidator) < 1 ||
                        len(events.MessageSender) < 1 ||
                        len(events.DelegateAmount) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.Delegations
                        msg.TypeName = "Delegations"
                        msg.Message +=
                            "\n** ❤️ Delegate ❤️ **"+ 
                            "\n\n**Validator:** " +
                            mkAccountLink(events.DelegateValidator[0]) +
                            "\n**Delegator:** " +
                            mkAccountLink(events.MessageSender[0]) +
                            "\n**Amount:** " +
                            mkTranscationLink(events.TxHash[0],events.DelegateAmount[0])
                        if !isAllowedAmount(msg, events.DelegateAmount[0]) {
                            continue
                        }

                    } else if ev == "/cosmos.staking.v1beta1.MsgUndelegate" && config.Config.MessagesConfig.Undelegations.Enabled {
                        // Undelegations
                        if len(events.UnbondAmount) < 1 ||
                        len(events.MessageSender) < 1 ||
                        len(events.UnbondValidator) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.Undelegations
                        msg.TypeName = "Undelegations"
                        msg.Message +=
                            "\n** 💀 Undelegate 💀 **" + 
                            "\n\n**Validator:** " +
                            mkAccountLink(events.UnbondValidator[0]) +
                            "\n**Delegator:** " +
                            mkAccountLink(events.MessageSender[0]) +
                            "\n**Amount:** " +
                            mkTranscationLink(events.TxHash[0],events.UnbondAmount[0])
                        if !isAllowedAmount(msg, events.UnbondAmount[0]) {
                            continue
                        }

                    } else if ev == "/cosmos.staking.v1beta1.MsgBeginRedelegate" && config.Config.MessagesConfig.Redelegations.Enabled {
                        // Redelegations
                        if len(events.RedelegateSourceValidator) < 1 ||
                        len(events.RedelegateDestinationValidator) < 1 ||
                        len(events.RedelegateAmount) < 1 ||
                        len(events.MessageSender) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.Redelegations
                        msg.TypeName = "Redelegations"
                        msg.Message +=
                            "\n** 💞 Redelegate 💞 **" + 
                            "\n\n**Validators:** " +
                            mkAccountLink(events.RedelegateSourceValidator[0]) +
                            " **->** " +
                            mkAccountLink(events.RedelegateDestinationValidator[0]) +
                            "\n**Delegator:** " +
                            mkAccountLink(events.MessageSender[0]) +
                            "\n**Amount:** " +
                            mkTranscationLink(events.TxHash[0],events.RedelegateAmount[0])
                        if !isAllowedAmount(msg, events.RedelegateAmount[0]) {
                            continue
                        }
                    } else if ev == "/cosmos.authz.v1beta1.MsgExec" && config.Config.MessagesConfig.Restake.Enabled {
                        // REStake Transactions
                        if len(events.WithdrawRewardsValidator) < 1 ||
                        len(events.MessageSender) < 1 ||
                        len(events.TransferAmount) < 1 {
                            continue 
                        }
                        msg.Type = config.Config.MessagesConfig.Restake
                        msg.TypeName = "Restake"
                        msg.Message +=
                            "\n** ♻️ REStake ♻️ **" +
                            "\n\n**Validator:** \n" +
                            mkAccountLink(events.WithdrawRewardsValidator[0]) +
                            "\n\n**Delegators:** "
                        j := 0
                        var total string
                        totaler := denomTotaler()
                        for i, delegator := range events.MessageSender {
                            if i >= 2 {
                                if i % 2 == 0 {
                                    j += 1
                                    msg.Message += fmt.Sprintf("\n%s\n%s", mkAccountLink(delegator) ,denomToAmount(events.TransferAmount[j]))
                                    total = totaler(events.TransferAmount[j])
                                }
                            }
                        }
                        msg.Message += "\n\n**Total REStaked:** \n" + mkTranscationLink(events.TxHash[0],total) + "\n"
                        if !isAllowedAmount(msg, total) {
                            continue
                        }

                    } else if ev == "/ibc.core.channel.v1.MsgRecvPacket" && config.Config.MessagesConfig.IBCIn.Enabled {
                        // Other Chain > FUND IBC
                        if len(events.IBCForeignSender) < 1 ||
                        len(events.TransferAmount) < 2 ||
                        len(events.TransferRecipient) < 2 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.IBCIn
                        msg.TypeName = "IBCIn"
                        msg.Message +=
                            "\n** ⚛️ IBC In ⚛️ **" +
                            "\n\n**Sender:** " +
                            mkAccountLink(events.IBCForeignSender[0]) +
                            "\n**Recipient:** " +
                            mkAccountLink(events.TransferRecipient[1]) +
                            "\n**Amount:** " +
                            mkTranscationLink(events.TxHash[0],events.TransferAmount[1])
                        if !isAllowedAmount(msg, events.TransferAmount[1]) {
                            continue
                        }

                    } else if ev == "/starnamed.x.starname.v1beta1.MsgRegisterAccount" && config.Config.MessagesConfig.RegisterAccount.Enabled {
                        // Starname specific
                        //⭐️

                        // Register new Starname -> Account
                        if len(events.AccountName) < 1 ||
                        len(events.DomainName) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.RegisterAccount
                        msg.TypeName = "RegisterAccount"
                        msg.Message +=
                            "\n** ⭐️️ Register Starname ⭐ **" +
                            "\n\n"+events.AccountName[0]+"*"+events.DomainName[0]

                        //mkTranscationLink(events.TxHash[0], events.Registerer[0]) <--- Works only with amounts :(

                    } else if ev == "/starnamed.x.starname.v1beta1.MsgRegisterDomain" && config.Config.MessagesConfig.RegisterDomain.Enabled {
                        // Register new Starname -> Domain
                        if len(events.DomainName) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.RegisterDomain
                        msg.TypeName = "RegisterDomain"
                        msg.Message +=
                            "\n** ⭐️️ Register Starname ⭐ **" +
                            "\n\n*"+events.DomainName[0]
                        //mkTranscationLink(events.TxHash[0], events.Registerer[0]) <--- Works only with amounts :(

                    } else if ev == "/starnamed.x.starname.v1beta1.MsgTransferAccount" && config.Config.MessagesConfig.TransferAccount.Enabled {
                        // Register new Starname -> Domain
                        if len(events.AccountName) < 1 ||
                        len(events.DomainName) < 1 ||
                        len(events.MessageSender) < 1 ||
                        len(events.NewAccountOwner) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.TransferAccount
                        msg.TypeName = "TransferAccount"
                        msg.Message +=
                            "\n** ⭐️️ Transfer Starname ⭐ **" +
                            "\n\n"+events.AccountName[0]+"*"+events.DomainName[0] +
                            "\n\n**Sender:** " +
                            mkAccountLink(events.MessageSender[0]) +
                            "\n\n**Recipient:** " +
                            mkAccountLink(events.NewAccountOwner[0])

                    } else if ev == "/starnamed.x.starname.v1beta1.MsgTransferDomain" && config.Config.MessagesConfig.TransferDomain.Enabled {
                        // Register new Starname -> Domain
                        if len(events.DomainName) < 1 ||
                        len(events.MessageSender) < 1 ||
                        len(events.NewDomainOwner) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.TransferDomain
                        msg.TypeName = "TransferDomain"
                        msg.Message +=
                            "\n** ⭐️️ Transfer Starname ⭐ **" +
                            "\n\n*"+ events.DomainName[0] +
                            "\n\n**Sender:** " +
                            mkAccountLink(events.MessageSender[0]) +
                            "\n\n**Recipient:** " +
                            mkAccountLink(events.NewDomainOwner[0])

                    } else if ev == "/starnamed.x.starname.v1beta1.MsgDeleteAccount" && config.Config.MessagesConfig.DeleteAccount.Enabled {
                        if len(events.AccountName) < 1 ||
                        len(events.DomainName) < 1 {
                            continue
                        }
                        msg.Type = config.Config.MessagesConfig.DeleteAccount
                        msg.TypeName = "DeleteAccount"
                        msg.Message +=
                            "\n** ⭐️️ Delete Starname ⭐ **" +
                            "\n\n"+events.AccountName[0]+"*"+events.DomainName[0]
                    }
                    // Ensure the msg is not blank, continue through the events if no messages are set to be sent
                    if msg.Message == "" || reflect.DeepEqual(msg.Type, MessageConfig{}) {
                        continue
                    }
                    // Add the memo if it exists
                    if memo := getMemo(events.TxHash[0]); memo != "" {
                        msg.Message += "\n**Memo: " + memo + "**"
                    }
                    // Top and bottom padding on the message using whitespace
                    msg.Message = "\n‎" + msg.Message + "\n‎"
                    // Check if the message adhears to the white/blacklist
                    if isAllowedMessage(msg) && sent == 0 {
                        resp <- msg
                    }
                    // Sent is needed to keep track of the amount of sent messages if it has sent a
                    // rewards message, since when withdrawing comission, it always withdraws rewards as well.
                    if msg.TypeName == "Rewards" && sent == 0 {
                        sent += 1
                        continue 
                    }
                    sent = 0
                    break
                }
            }()
        }
    }()
    select {
    case <- done:
        log.Println(color.BlueString("Listener terminating"))
        return
    }
}
