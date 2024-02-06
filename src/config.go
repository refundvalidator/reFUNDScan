package main

import (
    "fmt"
    "log"
    "time"
    "strings"
    "net/url"
    "net/http"
    "os"

    "github.com/BurntSushi/toml"
    "github.com/fatih/color"
    "github.com/gorilla/websocket"
)

// Config struct to represent the structure of the TOML file
type ConfigFile struct {
    Clients struct{
        Clients    []string `toml:"clients"`
        TgAPI      string   `toml:"telegram-api"`
        TgChatIDs  []string `toml:"telegram-chat-ids"`
        DscAPI     string   `toml:"discord-api"`
        DscChatIDs []string `toml:"discord-chat-ids"`
    }`toml:"clients"`
    Chain struct {
        Name string
    }`toml:"chain"`
    ChainInfo struct {
        Default      bool   `toml:"default"`
        PrettyName   string `toml:"pretty-name"`
        Coin         string `toml:"coin"`
        Denom        string `toml:"denom"`
        Exponent     int    `toml:"exponent"`
        CoinGeckoID  string `toml:"coin-gecko-id"`
        Bech32Prefix string `toml:"bech32-prefix"`
    }`toml:"chaininfo"`
    Connections struct {
        Default    bool `toml:"default"`
        Rest       string `toml:"rest"`
        Websocket  string `toml:"websocket"`
    }`toml:"connections"`
    ICNS struct{
        Default bool `toml:"default"`
        Rest string `toml:"rest"`
    }`toml:"icns"`
    Address struct {
        Addresses []AddressConfig `toml:"named"` 
    } `toml:"address"`
    Messages MessagesConfig `toml:"messages"`
    Explorer struct {
        Preset    string `toml:"explorer-preset"`
        Tx        string `toml:"explorer-custom-tx"`
        Account   string `toml:"explorer-custom-account"`
        Validator string `toml:"explorer-custom-validator"`
        AutoPath  bool   `toml:"auto-path"`
        Path      string `toml:"path"`
    } `toml:"explorer"`
}
type AddressConfig struct {
    Name string `toml:"name"` 
    Addr string `toml:"addr"` 
}
type MessageConfig struct {
    Enabled        bool     `toml:"enable"`
    Filter         string   `toml:"filter"`
    WhiteBlackList []string `toml:"list"`
}
type MessagesConfig struct {
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


type Config struct {
    Clients         []string
    TgAPI           string
    DscAPI          string
    TgChatIDs       []string
    DscChatIDs      []string

    Chain           string
    ChainPrettyName string
    Coin            string
    Denom           string
    CoinGeckoID     string
    Bech32Prefix    string
    Exponent        int

    RestURL         string
    WebsocketURL    string
    ICNSUrl         string

    Messages        MessagesConfig 

    Named           []AddressConfig

    RestTx          string
    RestValidators  string
    RestCoinGecko   string

    ExplorerTx         string
    ExplorerAccount    string
    ExplorerValidator  string

    ICNSAccount     string
}

var (
    configfile ConfigFile
    chain      ChainResponse
    icns       ChainResponse
    assets     AssetsResponse
)

func (cfg *Config) parseConfig(filePath string) {
    if strings.HasSuffix(filePath, "config.toml") {
        filePath = strings.TrimSuffix(filePath, "config.toml")
    }
    filePath = strings.TrimRight(filePath,"/")
    if _, err := toml.DecodeFile(filePath + "/config.toml", &configfile); err != nil {
        log.Fatal(color.RedString("Error parsing config.toml file, verify your configuation:", err))
    }
    cfg.Clients = configfile.Clients.Clients
    cfg.TgAPI = configfile.Clients.TgAPI
    cfg.TgChatIDs = configfile.Clients.TgChatIDs
    cfg.DscAPI = configfile.Clients.DscAPI
    cfg.DscChatIDs = configfile.Clients.DscChatIDs
    cfg.Chain = configfile.Chain.Name
    cfg.Messages = configfile.Messages
    cfg.Named = configfile.Address.Addresses

    // Grab the first available Rest URL for ICNS from the chain registry, if default = true
    if configfile.ICNS.Default == true {
        err := getData(
            "https://raw.githubusercontent.com/cosmos/chain-registry/master/osmosis/chain.json",
            &icns)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the chain.json from the osmosis chain registry, Please enter an ICNS URL manually"))
        }
        if len(icns.Apis.Rest) == 0 {
            log.Fatal(color.RedString("Failed to get any ICNS Urls from the osmosis chain registry, Please enter an ICNS URL manually"))
        } 
        cfg.ICNSUrl = icns.Apis.Rest[0].Address
    } else {
        cfg.ICNSUrl = configfile.ICNS.Rest
    }

    // Grab the first available Rest and RPC/Websocket URL from the chain registry, if default = true
    if configfile.Connections.Default == true {
        err := getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", config.Chain),
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
        cfg.RestURL = chain.Apis.Rest[0].Address
        cfg.WebsocketURL = fmt.Sprintf("wss://%s/websocket",parsedRPC.Host)
    } else {
        cfg.RestURL = configfile.Connections.Rest
        cfg.WebsocketURL = configfile.Connections.Websocket
    }

    // Grab the chain info from the registry, if default = true
    if configfile.ChainInfo.Default == true {
        err := getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/assetlist.json", config.Chain),
            &assets)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the assetslist.json from the chain registry, verify your chains' name matches the entry from the chain registry"))
        }
        err = getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", config.Chain),
            &chain)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the chain.json from the chain registry, verify your chains' name matches the entry from the chain registry"))
        }
        cfg.ChainPrettyName = chain.PrettyName
        cfg.Bech32Prefix = chain.Bech32Prefix
        cfg.Denom = assets.Assets[0].DenomUnits[0].Denom
        cfg.Coin = assets.Assets[0].Coin
        cfg.Exponent = assets.Assets[0].DenomUnits[1].Exponent
        cfg.CoinGeckoID = assets.Assets[0].CoingeckoID
    } else {
        cfg.ChainPrettyName = configfile.ChainInfo.PrettyName
        cfg.Bech32Prefix = configfile.ChainInfo.Bech32Prefix
        cfg.Denom = configfile.ChainInfo.Denom
        cfg.Coin = configfile.ChainInfo.Coin
        cfg.Exponent = configfile.ChainInfo.Exponent
        cfg.CoinGeckoID = configfile.ChainInfo.CoinGeckoID
    }
    if len(cfg.Clients) == 0 {
        log.Fatal(color.RedString("No client selected, check your config."))
    }
    switch configfile.Explorer.Preset {
    case "custom":
        cfg.ExplorerTx = configfile.Explorer.Tx
        cfg.ExplorerAccount = configfile.Explorer.Account
        cfg.ExplorerValidator = configfile.Explorer.Validator
    case "ping":
        var base string
        if configfile.Explorer.AutoPath {
            base = "https://ping.pub/" + cfg.ChainPrettyName
        } else {
            base = "https://ping.pub/" + configfile.Explorer.Path
        }
        cfg.ExplorerTx = base + "/tx/"
        cfg.ExplorerAccount = base + "/account/"
        cfg.ExplorerValidator = base + "/staking/"
    case "atom":
        var base string
        if configfile.Explorer.AutoPath {
            base = "https://atomscan.com/" + cfg.ChainPrettyName
        } else {
            base = "https://atomscan.com/" + configfile.Explorer.Path
        }
        cfg.ExplorerTx = base + "/transactions/"
        cfg.ExplorerAccount = base + "/accounts/"
        cfg.ExplorerValidator = base + "/validators/"
    case "mint":
        var base string
        if configfile.Explorer.AutoPath {
            base = "https://mintscan.io/" + cfg.ChainPrettyName
        } else {
            base = "https://mintscan.io/" + configfile.Explorer.Path
        }
        cfg.ExplorerTx = base + "/tx/"
        cfg.ExplorerAccount = base + "/address/"
        cfg.ExplorerValidator = base + "/validators/"
    case "dipper":
        var base string
        if configfile.Explorer.AutoPath {
            base = "https://bigdipper.live/" + cfg.ChainPrettyName
        } else {
            base = "https://bigdipper.live/" + configfile.Explorer.Path
        }
        cfg.ExplorerTx = base + "/transactions/"
        cfg.ExplorerAccount = base + "/accounts/"
        cfg.ExplorerValidator = base + "/validators/"
    default:
        var base string
        if configfile.Explorer.AutoPath {
            base = "https://ping.pub/" + cfg.ChainPrettyName
        } else {
            base = "https://ping.pub/" + configfile.Explorer.Path
        }
        cfg.ExplorerTx = base + "/tx/"
        cfg.ExplorerAccount = base + "/account/"
        cfg.ExplorerValidator = base + "/staking/"
    }
    cfg.validateConfig()
}
func (cfg *Config) validateConfig(){
    log.Println(color.BlueString("Validating Config..."))
    log.Println(color.BlueString("Testing ICNS URL..."))
    client := &http.Client{Timeout: 10 * time.Second}

    if response, err := client.Head(cfg.ICNSUrl + "/cosmwasm/wasm/v1/contract/osmo1xk0s8xgktn9x5vwcgtjdxqzadg88fgn33p8u9cnpdxwemvxscvast52cdd/smart/");
    err != nil || response.StatusCode != http.StatusNotImplemented {
        if configfile.ICNS.Default != true {
            log.Fatal(color.RedString("Bad ICNS URL, Please verify your config"))
        }
        success := false
        for i, u := range(icns.Apis.Rest){
            if i == 0 {
                continue
            }
            cfg.RestURL = strings.TrimRight(u.Address, "/")
            log.Println(color.YellowString("Bad ICNS URL, trying the next one in the registry..."))
            log.Println(color.BlueString("Testing ICNS URL: " + cfg.ICNSUrl))
            if response, err := client.Head(cfg.ICNSUrl + "/cosmwasm/wasm/v1/contract/osmo1xk0s8xgktn9x5vwcgtjdxqzadg88fgn33p8u9cnpdxwemvxscvast52cdd/smart/"); err == nil && response.StatusCode == http.StatusNotImplemented {
                success = true
                break
            }
        }
        if success != true {
            log.Fatal(color.RedString("Could not find valid ICNS URL in the chain registry, please provide your own"))
        }
        log.Println(color.GreenString("Using ICNS URL: " + cfg.ICNSUrl))
        log.Println(color.GreenString("Rest ICNS Valid\n"))

    } else {
        log.Println(color.GreenString("Using ICNS URL: " + cfg.ICNSUrl))
        log.Println(color.GreenString("ICNS URL Valid\n"))
    }
    log.Println(color.BlueString("Testing Rest URL: " + cfg.RestURL))
    if response, err := client.Head(cfg.RestURL + "/cosmos/tx/v1beta1/txs"); err != nil || response.StatusCode != http.StatusNotImplemented {
        if configfile.Connections.Default != true {
            log.Fatal(color.RedString("Bad Rest URL, Please verify your config"))
        }
        success := false
        for i, u := range(chain.Apis.Rest){
            if i == 0 {
                continue
            }
            cfg.RestURL = strings.TrimRight(u.Address, "/")
            log.Println(color.YellowString("Bad Rest URL, trying the next one in the registry..."))
            log.Println(color.BlueString("Testing Rest URL: " + cfg.RestURL))
            if response, err := client.Head(cfg.RestURL + "/cosmos/tx/v1beta1/txs"); err == nil && response.StatusCode == http.StatusNotImplemented {
                success = true
                break
            }
        }
        if success != true {
            log.Fatal(color.RedString("Could not find valid Rest URL in the chain registry, please provide your own"))
        }
        log.Println(color.GreenString("Using Rest URL: " + cfg.RestURL))
        log.Println(color.GreenString("Rest URL Valid\n"))
    } else {
        log.Println(color.GreenString("Using Rest URL: " + cfg.RestURL))
        log.Println(color.GreenString("Rest URL Valid\n"))
    }
    log.Println(color.BlueString("Testing RPC/Websocket URL: " + cfg.WebsocketURL))
    if _, _, err := websocket.DefaultDialer.Dial(cfg.WebsocketURL, nil); err != nil {
        if configfile.Connections.Default != true {
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
            cfg.WebsocketURL = fmt.Sprintf("wss://%s/websocket",parsedRPC.Host)
            log.Println(color.YellowString("Bad RPC/Websocket URL, trying the next one in the registry..."))
            log.Println(color.BlueString("Testing RPC/Websocket URL: " + cfg.WebsocketURL))
            if _, _, err := websocket.DefaultDialer.Dial(cfg.WebsocketURL, nil); err == nil {
                success = true
                break
            }
        }
        if success != true {
            log.Fatal(color.RedString("Could not find valid RPC URL in the chain registry, please provide your own"))
        }
        log.Println(color.GreenString("Using RPC/Websocket URL: " + cfg.WebsocketURL))
        log.Println(color.GreenString("RPC/Websocket URL Valid\n"))
    } else {
        log.Println(color.GreenString("Using RPC/Websocket URL: " + cfg.WebsocketURL))
        log.Println(color.GreenString("RPC/Websocket URL Valid\n"))
    }
    //Format the information
    cfg.RestURL = strings.TrimRight(cfg.RestURL, "/")
    cfg.ICNSUrl = strings.TrimRight(cfg.ICNSUrl, "/")
    cfg.ChainPrettyName = strings.ReplaceAll(cfg.ChainPrettyName," ","-")
    cfg.RestTx = cfg.RestURL + "/cosmos/tx/v1beta1/txs/"
    cfg.RestCoinGecko = "https://api.coingecko.com/api/v3/coins/" + cfg.CoinGeckoID
    cfg.RestValidators = cfg.RestURL + "/cosmos/staking/v1beta1/validators?pagination.limit=100000"
    cfg.ICNSAccount = cfg.ICNSUrl + "/cosmwasm/wasm/v1/contract/osmo1xk0s8xgktn9x5vwcgtjdxqzadg88fgn33p8u9cnpdxwemvxscvast52cdd/smart/"
    log.Println(color.GreenString("Using configuation for: " + cfg.Chain))
}

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


[explorer]
# Which explorer to use for the hyperlinks
# Current options:
# "ping" for ping.pub
# "atom" for atomscan.com
# "mint" for mintscan.io
# "dipper" for bigdipper.live
# "custom" to set custom URLs
explorer-preset = "ping"

# Ignored if explorer-preset = "custom"
# If auto-path = true, the explorers' chain path will use the chains' pretty name from the registry or chaininfo
# If auto-path = false, the explorers' chain path will use the value set by path
auto-path = true 
path = "Unification"

# The following only take affect if explorer-preset = "custom"
# Overrides the path option above
explorer-custom-tx = "https://ping.pub/Unification/tx"
explorer-custom-account = "https://ping.pub/Unification/accounts"
explorer-custom-validator = "https://ping.pub/Unification/staking"

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

[messages.ibc-transfers-in]
enable = false
filter = "default"
list = []
[messages.ibc-transfers-out]
enable = false
filter = "default"
list = []
[messages.withdraw-rewards]
enable = false
filter = "default"
list = []
[messages.withdraw-commission]
enable = false
filter = "default"
list = []
[messages.delegations]
enable = true
filter = "default"
list = []
[messages.unlegations]
enable = true
filter = "default"
list = []
[messages.redelegations]
enable = true
filter = "default"
list = []
[messages.restake]
enable = false
filter = "default"
list = []
# Starname specific
[messages.register-account]
enable = true
filter = "default"
list = []
[messages.register-domain]
enable = true
filter = "default"
list = []
[messages.transfer-account]
enable = true
filter = "default"
list = []
[messages.transfer-domain]
enable = true
filter = "default"
list = []
[messages.delete-account]
enable = true
filter = "default"
list = []

[address]
# Optionally define a list of wallets to be named when their account/val addresses
# are recognized.
[[address.named]]
name = "Burn Address 🔥"
addr = "und18mcmhkq6fmhu9hpy3sx5cugqwv6z0wrz7nn5d7"

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
