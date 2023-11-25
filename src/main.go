package main

import (
    "log"
    "os"
    "os/signal"
    "flag"

    telegram "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)
var (
    // Flags
    ChatID string
    BotKey string
    WebsocketUrl string
    RestUrl string
    ICNSUrl string

    // Persistent json responses
    cg CoinGeckoResponse
    vals ValidatorResponse
)
// Get flag values
func init(){
    flag.StringVar(&ChatID,"chid", "", "ChatID for your Channel\nExample: @MyAwesomeChannel")
    flag.StringVar(&BotKey,"api", "", "Bot API Key from the BotFather")
    flag.StringVar(&WebsocketUrl,"ws", "wss://rpc1.unification.io/websocket", "URL for blockchain websocket connection")
    flag.StringVar(&RestUrl,"rest", "https://rest.unification.io", "URL for blockchain REST connection")
    flag.StringVar(&ICNSUrl,"icns", "https://lcd.osmosis.zone", "URL for ICNS REST connection")
    flag.Parse()
    if ChatID == "" || BotKey == "" {
        log.Fatal("ChatID and BotKey required, --help for how to pass them through")
    }
}

// Start the telegram bot and listen for messages from the resp channel
func main(){
    interrupt := make(chan os.Signal, 1) 
    signal.Notify(interrupt, os.Interrupt) 

    resp := make(chan string)
    restart := make(chan bool)
    go Connect(resp, restart)
    bot, err := telegram.NewBotAPI(BotKey)
    if err != nil {
        log.Fatal("Cannot connect to bot, check your BotKey or internet connection")
    }
    // bot.Debug = true

    // AutoRefresh coin gecko data
    go cg.autoRefresh()
    go vals.autoRefresh()

    go func(){
        for {
            select {
            case message := <- resp:
                msg := telegram.NewMessageToChannel(ChatID, message)
                msg.ParseMode = telegram.ModeHTML
                msg.DisableWebPagePreview = true
                _, err := bot.Send(msg)
                if err != nil {
                    log.Println("Could not sent message, check your internet connection or ChatID")
                }
                log.Println(message)
            case <- restart:
                log.Println("Restarting websocket connection")
                go Connect(resp, restart)
            }
        }
    }()
    select {
    case <- interrupt:
        log.Println("Interrupted")
        return
    }
}
