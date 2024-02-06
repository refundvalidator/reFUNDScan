package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/fatih/color"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)
var (
    // Persistent json responses
    cg CoinGeckoResponse
    vals ValidatorResponse

    tgbot *telegram.BotAPI
    dscbot *discord.Session


    config Config

    // Flags
    configpath string
    initconfig bool
)

func init(){
    flag.StringVar(&configpath,"config", ".", "Directory containing your config.toml")
    flag.BoolVar(&initconfig,"init", false, "Creates a predefined config.toml file, if the config path is not set, defaults to the CWD")
    flag.Parse()
    configpath = strings.TrimRight(configpath,"/")
    if initconfig {
        initConfig(configpath) 
        log.Println(color.GreenString("Config file generated at: " + configpath + "/config.toml"))
        os.Exit(1)
    }
    config.parseConfig(configpath)
    // config.showConfig()
}
type MessageResponse struct {
    TgResponse chan string
    DscResponse chan discord.MessageEmbed
}

// Start the telegram bot and listen for messages from the resp channel
func main(){
    var err error
    interrupt := make(chan os.Signal, 1) 
    signal.Notify(interrupt, os.Interrupt) 
    resp := make(chan string)
    restart := make(chan bool)

    for _, client := range config.Clients{
        switch client {
        case "discord":
            dscbot, err = discord.New("Bot " + config.DscAPI)
            if err != nil {
                log.Fatal(color.RedString("Cannot connect to discord bot, check your BotKey or internet connection"))
            }    
            dscbot.Identify.Intents = discord.IntentsGuildMessages
            err = dscbot.Open()
            if err != nil {
                log.Fatal(color.RedString("Cannot connect to discord bot, check your BotKey or internet connection"))
            }
            log.Println(color.GreenString("Connected to Discord"))
        case "telegram":
            tgbot, err = telegram.NewBotAPI(config.TgAPI)
            if err != nil {
                log.Fatal(color.RedString("Cannot connect to telegram bot, check your BotKey or internet connection"))
            }
            log.Println(color.GreenString("Connected to Telegram"))
        }
    }
    // Connect to the websocket
    go Connect(resp, restart)
    // AutoRefresh coin gecko and validator set data
    go autoRefresh(config.RestCoinGecko,&cg)
    go autoRefresh(config.RestValidators,&vals)
    // Listen and serve
    go func(){
        for {
            select {
            case message := <- resp:
                for _, client := range config.Clients {
                    switch client {
                    case "telegram":
                        for _, chat := range config.TgChatIDs {
                            msg := telegram.NewMessageToChannel(chat, message)
                            msg.ParseMode = telegram.ModeHTML
                            msg.DisableWebPagePreview = true
                            _, err := tgbot.Send(msg)
                            if err != nil {
                                log.Println(color.YellowString("Could not sent telegram message, check your internet connection or ChatID"))
                            }
                            log.Println(color.BlueString(message))
                        }
                    case "discord":
                        for _, chat := range config.DscChatIDs {
                            _, err := dscbot.ChannelMessageSend(chat, message)
                            if err != nil {
                                log.Println(color.YellowString("Could not sent discord message, check your internet connection or ChatID"))
                            }
                        }
                    }
            }
            case <- restart:
                log.Println(color.BlueString("Restarting websocket connection in 10 seconds"))
                time.Sleep(time.Second * 10)
                go Connect(resp, restart)
        }
    }
}()
    select {
    case <- interrupt:
        log.Println(color.RedString("Interrupted"))
        return
    }
}
