package main

import (
    "log"
    "os"
    "os/signal"
    "flag"

    telegram "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)
var (
    ChatID string
    BotKey string
    Url string
)
// Get flag values
func init(){
    flag.StringVar(&ChatID,"chid", "", "ChatID for your Channel\nExample: @MyAwesomeChannel")
    flag.StringVar(&BotKey,"api", "", "Bot API Key from the BotFather")
    flag.StringVar(&Url,"url", "wss://rpc1.unification.io/websocket", "URL for websocket connection")
    flag.Parse()
    if ChatID == "" || BotKey == "" {
        log.Fatal("ChatID and BotKey required, --help for how to pass them through")
        os.Exit(1)
    }
}

// Start the telegram bot and listen for messages from the resp channel
func main(){
    interrupt := make(chan os.Signal, 1) 
    signal.Notify(interrupt, os.Interrupt) 

    resp := make(chan string)
    go Connect(resp)
    bot, err := telegram.NewBotAPI(BotKey)
    if err != nil {
        log.Fatal("Cannot connect to bot, check your BotKey")
        os.Exit(2)
    }
    // bot.Debug = true

    go func(){
        for {
            select {
            case message := <- resp:
                msg := telegram.NewMessageToChannel(ChatID, message)
                msg.ParseMode = telegram.ModeHTML
                msg.DisableWebPagePreview = true
                _, err := bot.Send(msg)
                if err != nil {
                    log.Fatal("Could not sent message, check your ChatID")
                    os.Exit(2)
                }
                log.Println(message)
            }
        }
    }()
    select {
    case <- interrupt:
        log.Printf("Interrupted")
        return
    }
}
