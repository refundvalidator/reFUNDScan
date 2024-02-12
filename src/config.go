package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

// Config struct to represent the structure of the TOML file
type ConfigFile struct {
    ClientsConfig ClientsConfig `toml:"clients"`
    ChainConfig ChainConfig `toml:"chain"`
    ChainInfoConfig ChainInfoConfig `toml:"chaininfo"`
    ConnectionsConfig ConnectionsConfig `toml:"connections"`
    ICNSConfig ICNSConfig `toml:"icns"` 
    AddressesConfig AddressesConfig `toml:"address"`
    MessagesConfig  MessagesConfig `toml:"messages"`
}
type ClientsConfig struct{
    Clients    []string `toml:"clients"`
    TgAPI      string   `toml:"telegram-api"`
    TgChatIDs  []string `toml:"telegram-chat-ids"`
    DscAPI     string   `toml:"discord-api"`
    DscChatIDs []string `toml:"discord-chat-ids"`
}
type ChainConfig struct {
    Name string
}
type ChainInfoConfig struct {
    Default      bool   `toml:"default"`
    PrettyName   string `toml:"pretty-name"`
    Coin         string `toml:"coin"`
    Denom        string `toml:"denom"`
    Exponent     int    `toml:"exponent"`
    CoinGeckoID  string `toml:"coin-gecko-id"`
    Bech32Prefix string `toml:"bech32-prefix"`
}
type ConnectionsConfig struct {
    Default    bool `toml:"default"`
    Rest       string `toml:"rest"`
    Websocket  string `toml:"websocket"`
}
type ICNSConfig struct{
    Default bool `toml:"default"`
    Rest string `toml:"rest"`
}
type AddressesConfig struct {
    Addresses []AddressConfig `toml:"named"`
}
type AddressConfig struct {
    Name string `toml:"name"` 
    Addr string `toml:"addr"` 
}
type MessageConfig struct {
    Enabled        bool     `toml:"enable"`
    Filter         string   `toml:"filter"`
    WhiteBlackList []string `toml:"list"`
    AmountFilter   bool     `toml:"amount-filter"`
    Threshold      float64  `toml:"threshold"`
}
type MessagesConfig struct {
    Currency        string        `toml:"currency"`
    Transfers       MessageConfig `toml:"transfers"`
    IBCIn           MessageConfig `toml:"ibc-transfers-in"`
    IBCOut          MessageConfig `toml:"ibc-transfers-out"`
    Rewards         MessageConfig `toml:"withdraw-rewards"`
    Commission      MessageConfig `toml:"withdraw-commission"`
    Delegations     MessageConfig `toml:"delegations"`
    Undelegations   MessageConfig `toml:"undelegations"`
    Redelegations   MessageConfig `toml:"redelegations"`
    Restake         MessageConfig `toml:"restake"`
    RegisterAccount MessageConfig `toml:"register-account"`
    RegisterDomain  MessageConfig `toml:"register-domain"`
    TransferAccount MessageConfig `toml:"transfer-account"`
    TransferDomain  MessageConfig `toml:"transfer-domain"`
    DeleteAccount   MessageConfig `toml:"delete-account"`
}

type ChainData struct {
   DisplayName       string 
   Denom             string
   Exponent          int 
   Prefix            string
   ExplorerPath      string
   CoinGeckoID       string
}
type ConnectionData struct {
    Rest            string
    Websocket       string
    ICNS            string
}
type ExplorerData struct {
    Base            string
    Account         string
    Validator       string
    TX              string
}

// Runtime Config
type Config struct {
    Config          ConfigFile

    Chain           ChainData
    Connections     ConnectionData
    Explorer        ExplorerData
    OtherChains     []ChainData
    Currency        string
    CurrencyAmount  *float64
}

var (
    configfile ConfigFile
    chain      ChainResponse
    icns       ChainResponse
    assets     AssetsResponse
    git        GitHubResponse
)

func (cfg *Config) parseConfig(filePath string) {
    if strings.HasSuffix(filePath, "config.toml") {
        filePath = strings.TrimSuffix(filePath, "config.toml")
    }
    filePath = strings.TrimRight(filePath,"/")
    if _, err := toml.DecodeFile(filePath + "/config.toml", &configfile); err != nil {
        log.Fatal(color.RedString("Error parsing config.toml file, verify your configuation:", err))
    }

    cfg.Config = configfile


    // Grab the first available Rest URL for ICNS from the chain registry, if default = true
    if cfg.Config.ICNSConfig.Default == true {
        err := getData(
            "https://raw.githubusercontent.com/cosmos/chain-registry/master/osmosis/chain.json",
            &icns)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the chain.json from the osmosis chain registry, Please enter an ICNS URL manually"))
        }
        if len(icns.Apis.Rest) == 0 {
            log.Fatal(color.RedString("Failed to get any ICNS Urls from the osmosis chain registry, Please enter an ICNS URL manually"))
        } 
        cfg.Connections.ICNS = icns.Apis.Rest[0].Address
    } else {
        cfg.Connections.ICNS = configfile.ICNSConfig.Rest
    } 

    // Grab the first available Rest and RPC/Websocket URL from the chain registry, if default = true
    if cfg.Config.ConnectionsConfig.Default == true {
        err := getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", configfile.ChainConfig.Name),
            &chain)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the chain.json from the chain registry, verify your chains' name matches the entry from the chain registry"))
        }
        if len(chain.Apis.RPC) == 0 {
            log.Fatal(color.RedString("Failed to retrieve any RPC/Websocket urls from the chain registry, please enter a RPC/Websocket URL manually"))
        }
        parsedRPC, err := url.Parse(chain.Apis.RPC[0].Address)
        if err != nil{
            log.Fatal(color.RedString("Error parsing RPC URL", err))
        }
        if len(chain.Apis.Rest) == 0 {
            log.Fatal(color.RedString("Failed to retrieve any Rest urls from the chain registry, please enter a Rest URL manually"))
        }
        cfg.Connections.Rest = chain.Apis.Rest[0].Address
        cfg.Connections.Websocket = fmt.Sprintf("wss://%s/websocket",parsedRPC.Host)
    } else {
        cfg.Connections.Rest = configfile.ConnectionsConfig.Rest
        cfg.Connections.Websocket = configfile.ConnectionsConfig.Websocket
    }

    // Grab the chain info from the registry, if default = true
    if cfg.Config.ChainInfoConfig.Default == true {
        err := getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/assetlist.json", configfile.ChainConfig.Name),
            &assets)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the assetslist.json from the chain registry, verify your chains' name matches the entry from the chain registry"))
        }
        err = getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", configfile.ChainConfig.Name),
            &chain)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the chain.json from the chain registry, verify your chains' name matches the entry from the chain registry"))
        }
        cfg.Chain = ChainData {
            DisplayName: assets.Assets[0].Display, 
            Denom: assets.Assets[0].DenomUnits[0].Denom,
            Exponent: assets.Assets[0].DenomUnits[1].Exponent,
            Prefix: chain.Bech32Prefix,
            ExplorerPath: chain.PrettyName,
            CoinGeckoID: assets.Assets[0].CoingeckoID,
        }
    } else {
        cfg.Chain = ChainData {
            DisplayName: configfile.ChainInfoConfig.Coin, 
            Denom: configfile.ChainInfoConfig.Denom,
            Exponent: configfile.ChainInfoConfig.Exponent,
            Prefix: configfile.ChainInfoConfig.Bech32Prefix,
            ExplorerPath: configfile.ChainInfoConfig.PrettyName,
            CoinGeckoID: configfile.ChainInfoConfig.CoinGeckoID,
        }
    }
    // Set the currency type
    r := reflect.ValueOf(&cg.MarketData.CurrentPrice).Elem()
    for i := 0; i < r.NumField(); i++ {
        if strings.ToLower(configfile.MessagesConfig.Currency) == strings.ToLower(r.Type().Field(i).Name) {
            cfg.Currency = strings.ToUpper(r.Type().Field(i).Name)
            cfg.CurrencyAmount = r.Field(i).Addr().Interface().(*float64)
        }
    }
    // Grab OtherChains Configurations
    log.Println(color.BlueString("Querying Asset and Chain data for other available chains..."))
    err := getData("https://raw.githubusercontent.com/refundvalidator/chain-registry/master/mainnets.json", &git)
    if err != nil {
        logMsg := fmt.Sprintf("Failed to get other chain data from github, other chains' currency will appear as Unknown IBC: " + err.Error())
        log.Println(color.YellowString(logMsg))
    }
    log.Println(color.GreenString(fmt.Sprintf("%d Chains Available, Querying their configurations...", len(git.Chains))))
    available := 0
    for _, c := range git.Chains {
        var ass AssetsResponse
        var chain ChainResponse
        var data ChainData
        err := getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", c),
            &chain)
        if err != nil {
            log.Println(color.YellowString("Failed to get Chain Data for: " + c))
            continue
        } else {
            // Fixes for specific chains, that don't adhear their paths to the chain registry
            if chain.PrettyName == "Cosmos Hub" {
               data.ExplorerPath = "Cosmos" 
            } else {
                data.ExplorerPath = strings.ReplaceAll(chain.PrettyName," ","-")
            }
            data.DisplayName = chain.PrettyName
            data.Prefix = chain.Bech32Prefix
        }
        err = getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/assetlist.json", c),
            &ass)
        if err != nil {
            log.Println(color.YellowString("Failed to get Asset Data for: " + c))
            continue
        } else {
            // Verify we can grab the correct DisplayName, Exponent, and Denom from the list
            for _, denom := range(ass.Assets[0].DenomUnits) {
                switch strings.ToUpper(denom.Denom){
                case strings.ToUpper(ass.Assets[0].Display):
                    data.DisplayName = strings.ToUpper(denom.Denom)
                    data.Exponent = denom.Exponent
                case strings.ToUpper(ass.Assets[0].Denom):
                    data.Denom = strings.ToLower(ass.Assets[0].Denom)
                }
            }
            if data.DisplayName == "" || data.Exponent == 0 || data.Denom == "" {
               continue 
            }
        }
        cfg.OtherChains = append(cfg.OtherChains, data)
        available += 1
    }
    if available > 0 {
        log.Println(color.GreenString(fmt.Sprintf("%d Chains Succesfully Queried!", available)))
    } else {
        log.Println(color.YellowString(fmt.Sprintf("No chains could be queried")))
    }
    cfg.validateConfig()
}
func (cfg *Config) validateConfig(){
    log.Println(color.BlueString("Validating Config..."))
    // Confirm there is no empty data for these fields
    if len(cfg.Config.ClientsConfig.Clients) == 0 {
        log.Fatal(color.RedString("No client selected, check your config."))
    }
    if cfg.Currency == "" {
        log.Fatal(color.RedString("Invalid Currency Type, Check your config"))
    }
    // Format information
    cfg.Chain.DisplayName = strings.ToUpper(cfg.Chain.DisplayName)
    cfg.Chain.Denom = strings.ToLower(cfg.Chain.Denom)
    cfg.Chain.Prefix = strings.ToLower(cfg.Chain.Prefix)
    ensureTrailingSlash(&cfg.Connections.Rest)
    ensureTrailingSlash(&cfg.Connections.Websocket)
    ensureTrailingSlash(&cfg.Connections.ICNS)
    ensureNoSpaces(&cfg.Chain.ExplorerPath)
    // Set URL Pathings
    cfg.Explorer.Base = "https://ping.pub/"
    cfg.Explorer.Account = cfg.Explorer.Base + cfg.Chain.ExplorerPath + "/account/"
    cfg.Explorer.Validator = cfg.Explorer.Base + cfg.Chain.ExplorerPath + "/staking/"
    cfg.Explorer.TX = cfg.Explorer.Base + cfg.Chain.ExplorerPath + "/tx/"

    // Begin Testing URL connections
    log.Println(color.BlueString("Testing ICNS URL..."))
    client := &http.Client{Timeout: 10 * time.Second}

    // Verify ICNS connection can be made, otherwise try the next URL in the config if default = true
    if response, err := client.Head(cfg.Connections.ICNS + "/cosmwasm/wasm/v1/contract/osmo1xk0s8xgktn9x5vwcgtjdxqzadg88fgn33p8u9cnpdxwemvxscvast52cdd/smart/");
    err != nil || response.StatusCode != http.StatusNotImplemented {
        if configfile.ICNSConfig.Default != true {
            log.Fatal(color.RedString("Bad ICNS URL, Please verify your config"))
        }
        success := false
        for i, u := range(icns.Apis.Rest){
            if i == 0 {
                continue
            }
            cfg.Connections.ICNS = strings.TrimRight(u.Address, "/")
            log.Println(color.YellowString("Bad ICNS URL, trying the next one in the registry..."))
            log.Println(color.BlueString("Testing ICNS URL: " + cfg.Connections.ICNS))
            if response, err := client.Head(cfg.Connections.ICNS + "/cosmwasm/wasm/v1/contract/osmo1xk0s8xgktn9x5vwcgtjdxqzadg88fgn33p8u9cnpdxwemvxscvast52cdd/smart/"); err == nil && response.StatusCode == http.StatusNotImplemented {
                success = true
                break
            }
        }
        if success != true {
            log.Fatal(color.RedString("Could not find valid ICNS URL in the chain registry, please provide your own"))
        }
        log.Println(color.GreenString("Using ICNS URL: " + cfg.Connections.ICNS))
        log.Println(color.GreenString("ICNS URL Valid\n"))
    } else {
        log.Println(color.GreenString("Using ICNS URL: " + cfg.Connections.ICNS))
        log.Println(color.GreenString("ICNS URL Valid\n"))
    }

    // Verify REST connection can be made, otherwise try the next URL in the config if default = true
    log.Println(color.BlueString("Testing Rest URL: " + cfg.Connections.Rest))
    if response, err := client.Head(cfg.Connections.Rest + "/cosmos/tx/v1beta1/txs"); err != nil || response.StatusCode != http.StatusNotImplemented {
        if configfile.ConnectionsConfig.Default != true {
            log.Fatal(color.RedString("Bad Rest URL, Please verify your config"))
        }
        success := false
        for i, u := range(chain.Apis.Rest){
            if i == 0 {
                continue
            }
            cfg.Connections.Rest = strings.TrimRight(u.Address, "/")
            log.Println(color.YellowString("Bad Rest URL, trying the next one in the registry..."))
            log.Println(color.BlueString("Testing Rest URL: " + cfg.Connections.Rest))
            if response, err := client.Head(cfg.Connections.Rest + "/cosmos/tx/v1beta1/txs"); err == nil && response.StatusCode == http.StatusNotImplemented {
                success = true
                break
            }
        }
        if success != true {
            log.Fatal(color.RedString("Could not find valid Rest URL in the chain registry, please provide your own"))
        }
        log.Println(color.GreenString("Using Rest URL: " + cfg.Connections.Rest))
        log.Println(color.GreenString("Rest URL Valid\n"))
    } else {
        log.Println(color.GreenString("Using Rest URL: " + cfg.Connections.Rest))
        log.Println(color.GreenString("Rest URL Valid\n"))
    }

    // Verify Websocket connection can be made, otherwise try the next URL in the config if default = true
    log.Println(color.BlueString("Testing RPC/Websocket URL: " + cfg.Connections.Websocket))
    if _, _, err := websocket.DefaultDialer.Dial(cfg.Connections.Websocket, nil); err != nil {
        if configfile.ConnectionsConfig.Default != true {
            log.Fatal(color.RedString("Bad RPC/Websocket URL, Please verify your config"))
        }
        success := false
        for i, u := range(chain.Apis.RPC){
            if i == 0 {
                continue
            }
            parsedRPC, err := url.Parse(u.Address)
            if err != nil {
                log.Println(color.YellowString("Failed to Parse"))
                continue
            }
            cfg.Connections.Websocket = fmt.Sprintf("wss://%s/websocket",parsedRPC.Host)
            log.Println(color.YellowString("Bad RPC/Websocket URL, trying the next one in the registry..."))
            log.Println(color.BlueString("Testing RPC/Websocket URL: " + cfg.Connections.Websocket))
            if _, _, err := websocket.DefaultDialer.Dial(cfg.Connections.Websocket, nil); err == nil {
                success = true
                break
            }
        }
        if success != true {
            log.Fatal(color.RedString("Could not find valid RPC URL in the chain registry, please provide your own"))
        }
        log.Println(color.GreenString("Using RPC/Websocket URL: " + cfg.Connections.Websocket))
        log.Println(color.GreenString("RPC/Websocket URL Valid\n"))
    } else {
        log.Println(color.GreenString("Using RPC/Websocket URL: " + cfg.Connections.Websocket))
        log.Println(color.GreenString("RPC/Websocket URL Valid\n"))
    }

    log.Println(color.GreenString("Using configuation for: " + cfg.Chain.DisplayName))
}

// Generate a configfile
func initConfig(filePath string){
    file := `
[clients]
# Multiple messaging clients can be used at once
# example: clients = [ "telegram", "discord" ]
clients = [ "telegram" ]

# Discord API Key
# example: discord-api = "MTA0NzY2OTU1OTU5Mjb1OTtzNg.G3ZDLz.xaONq-mqDaX5Zv5K-Fx5dDQxnooOdP7gOWOc4Q"
discord-api = ""

# Chat ID of the channels to send the message to, mulitple can be used at once
# example: discord-chat-ids = [ "1125944525457975326", "1125944523659096234"]
discord-chat-ids = [ "" ]

# Telegram bot token given by the botfather
# example: telegram-api = "5750057848:AAGb4KvbF6FP6-1GmV5Kaun8WSukLpePLXF"
telegram-api = ""

# ChannelIDs for the channels to send messages to, multiple can be used at once
# example: telegram-chat-ids = [ "@MyAwesomeChannel", "@MyAwesomeChannel2"]
telegram-chat-ids = [ "" ]

[chain]
# The name of the chain as it appears in the cosmos chain registry
# example: name = "osmosis"
name = "unification"

[chaininfo]
# If default = true, reFUNDScan will use the information given by the cosmos chain registry for the chain name
default = true

# The following will be ignored if default=true
pretty-name = "Unification"
coin = "FUND"
denom = "nund"
exponent = 9
coin-gecko-id = "unification"
bech32-prefix = "und"

[icns]
# Rest URL to query for ICNS naming
default = true

# Ignored if default = true
rest = "https://lcd.osmosis.zone/"

[connections]
# If default = true, reFUNDScan will automatically attempt each of the RPCs and REST urls
# given by the cosmos chain registry for the chain name until it find a valid one
default = true

# The following will be ignored if default=true
rest = "https://rest.unification.io/"
websocket = "wss://rpc1.unification.io/websocket" 

[messages]

# Currency to display for the messages.
# example: "usd" or "cad" or "eur"
currency = "usd"

[messages.transfers]
# Enable or disable this message type entirely
enable = true

# "default" means no filtering will take place, and will ignore the list
# "whitelist" will enable whitelist mode for this message type
# "blacklist" will enable blacklist mode for this message type
filter = "default"

# When filter is set to blacklist, Define a blacklist condition(s),
# Conditions can be an address, memo, name, etc. If the string is recoginzed in the
# message, it will prevent the message from sending.

# Whitelist is the inverse of blacklist, this will ONLY allow messages that
# contain an item defined in the list.
list = [ "Delegate(rewards)", "Cosmostation" , "100100" ,"und1hdn830wndtquqxzaz3rds7r7hqgpsg5q9ggxpk" ]

# Enable or disable the amount filter, if this is set to true, will filter messages if the total
# of the transaction falls below the threshold. Threshold should be set to units of the currency amount.
amount-filter = false

# Only takes affect if amount-filter = true.
# example: if currency = "usd" and threshold = 100, then any messages totaling below $100
# will be filtered, and will not send.
threshold = 1000

[messages.ibc-transfers-in]
enable = false
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.ibc-transfers-out]
enable = false
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.withdraw-rewards]
enable = false
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.withdraw-commission]
enable = false
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.delegations]
enable = true
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.undelegations]
enable = true
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.redelegations]
enable = true
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.restake]
enable = false
filter = "default"
list = []
amount-filter = false
threshold = 1000
# Starname specific
[messages.register-account]
enable = true
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.register-domain]
enable = true
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.transfer-account]
enable = true
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.transfer-domain]
enable = true
filter = "default"
list = []
amount-filter = false
threshold = 1000
[messages.delete-account]
enable = true
filter = "default"
list = []
amount-filter = false
threshold = 1000

[address]
# Optionally define a list of wallets to be named when their account/val addresses
# are recognized.
[[address.named]]
name = "Burn Address 🔥"
addr = "und1qqqqqqqqqqqqqqqqqqqqqqqqqqqqph4djz5txt"

[[address.named]]
name = "reFUND"
addr = "undvaloper1k03uvkkzmtkvfedufaxft75yqdfkfgvgsgjfwa"
`
    // Write the content to the file
    err := os.WriteFile(filePath + "/config.toml", []byte(file), 0644)
    if err != nil {
        log.Fatal(color.RedString("Failed to write file: " , err))
    }
}
