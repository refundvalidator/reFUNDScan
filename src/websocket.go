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
    c, _, err := websocket.DefaultDialer.Dial(config.WebsocketURL, nil)  
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
            events := res.Result.Events
            for _, ev := range events.MessageAction {
                // TODO: governance votes, validator creations, validator edits 
                // Fix small amounts displaying as 0.00: maybe not <?
                // Split this file, maybe into messages.go?

                var msg MessageResponse

                if ev == "/cosmos.bank.v1beta1.MsgSend" && config.Messages.Transfers.Enabled {
                    msg.Type = config.Messages.Transfers
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
                        break
                    }

                } else if ev == "/ibc.applications.transfer.v1.MsgTransfer" && config.Messages.IBCOut.Enabled {
                    // FUND > Other Chain IBC
                    msg.Type = config.Messages.IBCOut
                    msg.TypeName = "IBCOut"
                    msg.Message += 
                        "\n** ⚛️ IBC Transfer ⚛️ **" + 
                        "\n\n**Sender:** " +
                        mkAccountLink(events.IBCTransferSender[0]) +
                        "\n**Recipient:** " +
                        mkAccountLink(events.IBCTransferRecipient[0]) +
                        "\n**Amount:** " +
                        mkTranscationLink(events.TxHash[0],events.TransferAmount[1])
                    if !isAllowedAmount(msg, events.TransferAmount[1]) {
                        break
                    }

                // FIXME: throws out of index errors
                } else if ev == "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward" && config.Messages.Rewards.Enabled {
                     // Withdraw rewards
                     msg.Type = config.Messages.Rewards
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
                        break
                    }

                // FIXME Never fires, because Rewards withdrawl will always trigger first 
                } else if ev == "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission" && config.Messages.Commission.Enabled {
                     // Withdraw commission
                     msg.Type = config.Messages.Commission
                     msg.TypeName = "Commission"
                     msg.Message +=
                         "\n** 💸 Withdraw Commission 💸 **" +
                         "\n**Validator:** " +
                         mkAccountLink(events.WithdrawRewardsDelegator[0]) +
                         "\n**Amount:** " +
                         mkTranscationLink(events.TxHash[0],events.WithdrawCommissionAmount[0])
                    if !isAllowedAmount(msg, events.WithdrawCommissionAmount[0]) {
                        break
                    }               

                } else if ev == "/cosmos.staking.v1beta1.MsgDelegate" && config.Messages.Delegations.Enabled {
                    // Delegations
                    msg.Type = config.Messages.Delegations
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
                        break
                    }

                } else if ev == "/cosmos.staking.v1beta1.MsgUndelegate" && config.Messages.Undelegations.Enabled {
                    // Undelegations
                    msg.Type = config.Messages.Undelegations
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
                        break
                    }

                } else if ev == "/cosmos.staking.v1beta1.MsgBeginRedelegate" && config.Messages.Redelegations.Enabled {
                    // Redelegations
                    msg.Type = config.Messages.Redelegations
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
                        break
                    }
                // TODO: This breaks on most chains
                } else if ev == "/cosmos.authz.v1beta1.MsgExec" && config.Messages.Restake.Enabled {
                    // REStake Transactions
                    msg.Type = config.Messages.Restake
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
                        break
                    }

                } else if ev == "/ibc.core.channel.v1.MsgRecvPacket" && config.Messages.IBCIn.Enabled {
                    // Other Chain > FUND IBC
                    msg.Type = config.Messages.IBCIn
                    msg.TypeName = "IBCIn"
                    msg.Message +=
                        "\n** ⚛️ IBC Transfer ⚛️ **" +
                        "\n\n**Sender:** " +
                        mkAccountLink(events.IBCForeignSender[0]) +
                        "\n**Recipient:** " +
                        mkAccountLink(events.TransferRecipient[1]) +
                        "\n**Amount:** " +
                        mkTranscationLink(events.TxHash[0],events.TransferAmount[1])
                    if !isAllowedAmount(msg, events.TransferAmount[1]) {
                        break
                    }

                } else if ev == "/starnamed.x.starname.v1beta1.MsgRegisterAccount" && config.Messages.RegisterAccount.Enabled {
                    // Starname specific
                    //⭐️

                    // Register new Starname -> Account
                    msg.Type = config.Messages.RegisterAccount
                    msg.TypeName = "RegisterAccount"
                    msg.Message +=
                        "\n** ⭐️️ Register Starname ⭐ **" +
                        "\n\n"+events.AccountName[0]+"*"+events.DomainName[0]

                    //mkTranscationLink(events.TxHash[0], events.Registerer[0]) <--- Works only with amounts :(

                } else if ev == "/starnamed.x.starname.v1beta1.MsgRegisterDomain" && config.Messages.RegisterDomain.Enabled {
                    // Register new Starname -> Domain
                    msg.Type = config.Messages.RegisterDomain
                    msg.TypeName = "RegisterDomain"
                    msg.Message +=
                        "\n** ⭐️️ Register Starname ⭐ **" +
                        "\n\n*"+events.DomainName[0]
                    //mkTranscationLink(events.TxHash[0], events.Registerer[0]) <--- Works only with amounts :(

                } else if ev == "/starnamed.x.starname.v1beta1.MsgTransferAccount" && config.Messages.TransferAccount.Enabled {
                    // Register new Starname -> Domain
                    msg.Type = config.Messages.TransferAccount
                    msg.TypeName = "TransferAccount"
                    msg.Message +=
                        "\n** ⭐️️ Transfer Starname ⭐ **" +
                        "\n\n"+events.AccountName[0]+"*"+events.DomainName[0] +
                        "\n\n**Sender:** " +
                        mkAccountLink(events.MessageSender[0]) +
                        "\n\n**Recipient:** " +
                        mkAccountLink(events.NewAccountOwner[0])

                } else if ev == "/starnamed.x.starname.v1beta1.MsgTransferDomain" && config.Messages.TransferDomain.Enabled {
                    // Register new Starname -> Domain
                    msg.Type = config.Messages.TransferDomain
                    msg.TypeName = "TransferDomain"
                    msg.Message +=
                        "\n** ⭐️️ Transfer Starname ⭐ **" +
                        "\n\n*"+ events.DomainName[0] +
                        "\n\n**Sender:** " +
                        mkAccountLink(events.MessageSender[0]) +
                        "\n\n**Recipient:** " +
                        mkAccountLink(events.NewDomainOwner[0])

                } else if ev == "/starnamed.x.starname.v1beta1.MsgDeleteAccount" && config.Messages.DeleteAccount.Enabled {
                    msg.Type = config.Messages.DeleteAccount
                    msg.TypeName = "DeleteAccount"
                    msg.Message +=
                        "\n** ⭐️️ Delete Starname ⭐ **" +
                        "\n\n"+events.AccountName[0]+"*"+events.DomainName[0]
                }
                // Ensure the msg is not blank
                if msg.Message == "" || reflect.DeepEqual(msg.Type, MessageConfig{}) {
                    break
                }
                // Add the memo if it exists
                if memo := getMemo(events.TxHash[0]); memo != "" {
                    msg.Message += "\n**Memo: " + memo + "**"
                }
                // Top and bottom padding on the message using whitespace
                msg.Message = "\n‎" + msg.Message + "\n‎"
                // Check if the message adhears to the white/blacklist
                if !isAllowedMessage(msg) {
                    break 
                }
                resp <- msg
                break
            }
        }
    }()
    select {
    case <- done:
        log.Println(color.BlueString("Listener terminating"))
        return
    }
}
