package main

import (
    "log"
    "fmt"
    "encoding/json"
    "reflect"

    "github.com/fatih/color"
    "github.com/gorilla/websocket"
)

// Connect to the websocket and serve the formatted responses to the given channel resp
func Connect(resp chan string, restart chan bool) {
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
                // Fix small amounts displaying as 0.00

                var msgType MessageConfig 
                msg := ""

                if ev == "/cosmos.bank.v1beta1.MsgSend" && config.Messages.Transfers.Enabled {
                    msgType = config.Messages.Transfers
                    // On Chain Transfers
                    msg +=
                        "\n**📬 Transfer 📬**" +
                        "\n\n**Sender:** " +
                        mkAccountLink(events.TransferSender[0]) +
                        "\n**Reciever:** " +
                        mkAccountLink(events.TransferRecipient[1]) +
                        "\n**Amount:** " +
                        mkTranscationLink(events.TxHash[0],events.TransferAmount[1]) 

                } else if ev == "/ibc.applications.transfer.v1.MsgTransfer" && config.Messages.IBCOut.Enabled {
                    // FUND > Other Chain IBC
                    msgType = config.Messages.IBCOut
                    msg += 
                        "\n**⚛️ IBC Transfer ⚛️**" + 
                        "\n\n**Sender:** " +
                        mkAccountLink(events.IBCTransferSender[0]) +
                        "\n**Reciever:** " +
                        mkAccountLink(events.IBCTransferReciever[0]) +
                        "\n**Amount:** " +
                        mkTranscationLink(events.TxHash[0],events.TransferAmount[1])

                } else if ev == "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward" && config.Messages.Rewards.Enabled {
                     // Withdraw rewards
                     msgType = config.Messages.Rewards
                     msg +=
                         "\n**💵 Withdraw Reward 💵**" +
                         "\n\n**Delegator:** \n" +
                         mkAccountLink(events.WithdrawRewardsDelegator[0]) +
                         "\n\n**Validators:** "
                     totaler := denomsToAmount()
                     var total string
                     for i, val := range events.WithdrawRewardsValidator{
                         msg += fmt.Sprintf("\n%s\n%s",mkAccountLink(val), denomToAmount(events.WithdrawRewardsAmount[i]))
                         total = totaler(events.WithdrawRewardsAmount[i])
                     }
                     msg += "\n\n**Total:** \n" + mkTranscationLink(events.TxHash[0],total)

                } else if ev == "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission" && config.Messages.Commission.Enabled {
                     // Withdraw commission
                     msgType = config.Messages.Commission
                     msg +=
                         "\n**💸 Withdraw Commission 💸**" +
                         "\n**Validator:** " +
                         mkAccountLink(events.WithdrawRewardsDelegator[0]) +
                         "\n**Amount:** " +
                         mkTranscationLink(events.TxHash[0],events.WithdrawCommissionAmount[0])

                } else if ev == "/cosmos.staking.v1beta1.MsgDelegate" && config.Messages.Delegations.Enabled {
                    // Delegations
                    msgType = config.Messages.Delegations
                    msg +=
                        "\n**❤️ Delegate ❤️**" + 
                        "\n\n**Validator:** " +
                        mkAccountLink(events.DelegateValidator[0]) +
                        "\n**Delegator:** " +
                        mkAccountLink(events.MessageSender[0]) +
                        "\n**Amount:** " +
                        mkTranscationLink(events.TxHash[0],events.DelegateAmount[0])

                } else if ev == "/cosmos.staking.v1beta1.MsgUndelegate" && config.Messages.Undelegations.Enabled {
                    // Undelegations
                    msgType = config.Messages.Undelegations
                    msg +=
                        "\n**💀 Undelegate 💀**" + 
                        "\n\n**Validator:** " +
                        mkAccountLink(events.UnbondValidator[0]) +
                        "\n**Delegator:** " +
                        mkAccountLink(events.MessageSender[0]) +
                        "\n**Amount:** " +
                        mkTranscationLink(events.TxHash[0],events.UnbondAmount[0])

                } else if ev == "/cosmos.staking.v1beta1.MsgBeginRedelegate" && config.Messages.Redelegations.Enabled {
                    // Redelegations
                    msgType = config.Messages.Redelegations
                    msg +=
                        "\n**💞 Redelegate 💞**" + 
                        "\n\n**Validators:** " +
                        mkAccountLink(events.RedelegateSourceValidator[0]) +
                        " **->** " +
                        mkAccountLink(events.RedelegateDestinationValidator[0]) +
                        "\n**Delegator:** " +
                        mkAccountLink(events.MessageSender[0]) +
                        "\n**Amount:** " +
                        mkTranscationLink(events.TxHash[0],events.RedelegateAmount[0])

                } else if ev == "/cosmos.authz.v1beta1.MsgExec" && config.Messages.Restake.Enabled {
                    // REStake Transactions
                    msgType = config.Messages.Restake
                    msg +=
                        "\n**♻️ REStake ♻️**" +
                        "\n\n**Validator:** \n" +
                        mkAccountLink(events.WithdrawRewardsValidator[0]) +
                        "\n\n**Delegators:** "
                    j := 0
                    var total string
                    totaler := denomsToAmount()
                    for i, delegator := range events.MessageSender {
                        if i >= 2 {
                            if i % 2 == 0 {
                                j += 1
                                msg += fmt.Sprintf("\n%s\n%s", mkAccountLink(delegator) ,denomToAmount(events.TransferAmount[j]))
                                total = totaler(events.TransferAmount[j])
                            }
                        }
                    }
                    msg += "\n\n**Total REStaked:** \n" + mkTranscationLink(events.TxHash[0],total) + "\n"

                } else if ev == "/ibc.core.channel.v1.MsgRecvPacket" && config.Messages.IBCIn.Enabled {
                    // Other Chain > FUND IBC
                    msgType = config.Messages.IBCIn
                    msg +=
                        "\n**⚛️ IBC Transfer ⚛️**" +
                        "\n\n**Sender:** " +
                        mkAccountLink(events.IBCForeignSender[0]) +
                        "\n**Reciever:** " +
                        mkAccountLink(events.TransferRecipient[1]) +
                        "\n**Amount:** " +
                        mkTranscationLink(events.TxHash[0],events.TransferAmount[1])

                } else if ev == "/starnamed.x.starname.v1beta1.MsgRegisterAccount" && config.Messages.RegisterAccount.Enabled {
                    // Starname specific
                    //⭐️

                    // Register new Starname -> Account
                    msgType = config.Messages.RegisterAccount
                    msg +=
                        "\n**⭐️️ Register Starname ⭐**️️" +
                        "\n\n"+events.AccountName[0]+"*"+events.DomainName[0]
                    //mkTranscationLink(events.TxHash[0], events.Registerer[0]) <--- Works only with amounts :(

                } else if ev == "/starnamed.x.starname.v1beta1.MsgRegisterDomain" && config.Messages.RegisterDomain.Enabled {
                    // Register new Starname -> Domain
                    msgType = config.Messages.RegisterDomain
                    msg +=
                        "\n**⭐️️ Register Starname ⭐**️️" +
                        "\n\n*"+events.DomainName[0]
                    //mkTranscationLink(events.TxHash[0], events.Registerer[0]) <--- Works only with amounts :(

                } else if ev == "/starnamed.x.starname.v1beta1.MsgTransferAccount" && config.Messages.TransferAccount.Enabled {
                    // Register new Starname -> Domain
                    msgType = config.Messages.TransferAccount
                    msg +=
                        "\n**⭐️️ Transfer Starname ⭐**️️" +
                        "\n\n"+events.AccountName[0]+"*"+events.DomainName[0] +
                        "\n\n**Sender:** " +
                        mkAccountLink(events.MessageSender[0]) +
                        "\n\n**Recipient:** " +
                        mkAccountLink(events.NewAccountOwner[0])

                } else if ev == "/starnamed.x.starname.v1beta1.MsgTransferDomain" && config.Messages.TransferDomain.Enabled {
                    // Register new Starname -> Domain
                    msgType = config.Messages.TransferDomain
                    msg +=
                        "\n**⭐️️ Transfer Starname ⭐**️️" +
                        "\n\n*"+ mkBold(events.DomainName[0]) +
                        "\n\n**Sender:** " +
                        mkAccountLink(events.MessageSender[0]) +
                        "\n\n**Recipient:** " +
                        mkAccountLink(events.NewDomainOwner[0])

                } else if ev == "/starnamed.x.starname.v1beta1.MsgDeleteAccount" && config.Messages.DeleteAccount.Enabled {
                    msgType = config.Messages.DeleteAccount
                    msg +=
                        "\n**⭐️️ Delete Starname ⭐**️️" +
                        "\n\n"+events.AccountName[0]+"*"+events.DomainName[0]
                }
                // Ensure the msg is not blank
                if msg == "" || reflect.DeepEqual(msgType, MessageConfig{}) {
                    break
                }
                // Add the memo if it exists
                if memo := getMemo(events.TxHash[0]); memo != "" {
                    msg += "\n**Memo: " + memo + "**"
                }
                // Top and bottom padding on the message using whitespace
                msg = "\n‎" + msg + "\n‎"
                // Check if the message adhears to the white/blacklist
                if !isAllowedMessage(msgType, msg) {
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
