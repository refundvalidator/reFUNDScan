package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

// Config struct to represent the structure of the TOML file
type ConfigFile struct {
	Telegram struct{
     	ChatID string `toml:"chat-id"`
        API    string `toml:"api"`
    }`toml:"telegram"`
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
    	URL string `toml:"url"`
    }`toml:"icns"`
    Wallets []WalletsConfig `toml:"wallet"`
    Explorer struct {
       Preset    string `toml:"explorer-preset"`
       Tx        string `toml:"explorer-custom-tx"`
       Account   string `toml:"explorer-custom-account"`
       Validator string `toml:"explorer-custom-validator"`
       AutoPath  bool   `toml:"auto-path"`
       Path      string `toml:"path"`
    } `toml:"explorer"`
    General struct {
        Transfers       bool `toml:"transfers"`
        IBCIn           bool `toml:"ibc-transfers-in"`
        IBCOut          bool `toml:"ibc-transfers-out"`
        Rewards         bool `toml:"withdraw-rewards"`
        Commission      bool `toml:"withdraw-commission"`
        Delegations     bool `toml:"delegations"`
        Undelegations   bool `toml:"undelegations"`
        Redelegations   bool `toml:"redelegations"`
        Restake         bool `toml:"restake"`
    }`toml:"general"`
}
type WalletsConfig struct {
    Name    string `toml:"name"` 
    Addr    string `toml:"addr"` 
    ValAddr string `toml:"val-addr"` 
}

type Config struct {
    API             string
    ChatID          string

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

    Transfers       bool
    IBCIn           bool
    IBCOut          bool
    Rewards         bool
    Commission      bool
    Delegations     bool
    Undelegations   bool
    Redelegations   bool
    Restake         bool

    Wallets         []WalletsConfig

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
    assets     AssetsResponse
    icns       ChainResponse
)

func (cfg *Config) parseConfig(filePath string) {

	if _, err := toml.DecodeFile(filePath + "/config.toml", &configfile); err != nil {
		log.Fatal(color.RedString("Error parsing config.toml file, verify your configuation:", err))
	}
    cfg.API = configfile.Telegram.API
    cfg.ChatID = configfile.Telegram.ChatID
    cfg.Chain = configfile.Chain.Name
    cfg.ICNSUrl = configfile.ICNS.URL
    cfg.Transfers = configfile.General.Transfers
    cfg.IBCIn = configfile.General.IBCIn
    cfg.IBCOut = configfile.General.IBCOut
    cfg.Rewards = configfile.General.Rewards
    cfg.Commission = configfile.General.Commission
    cfg.Delegations = configfile.General.Delegations
    cfg.Undelegations = configfile.General.Undelegations
    cfg.Redelegations = configfile.General.Redelegations
    cfg.Restake = configfile.General.Restake
    cfg.Wallets = configfile.Wallets

    // Grab the first available Rest URL for ICNS from the chain registry, if default = true
    if configfile.ICNS.Default == true {
        err := getData(
            "https://raw.githubusercontent.com/cosmos/chain-registry/master/osmosis/chain.json",
            &icns)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the chain.json from the osmosis chain registry, Please enter an ICNS URL manually"))
        }
        cfg.ICNSUrl = icns.Apis.Rest[0].Address
    } else {
        cfg.ICNSUrl = configfile.ICNS.URL
    }

    // Grab the first available Rest and RPC/Websocket URL from the chain registry, if default = true
    if configfile.Connections.Default == true {
        err := getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", config.Chain),
            &chain)
        if err != nil {
            log.Fatal(color.RedString("Failed to get the chain.json from the chain registry, verify your chains' name matches the entry from the chain registry"))
        }
        parsedRPC, err := url.Parse(chain.Apis.RPC[0].Address)
        if err != nil{
            log.Fatal(color.RedString("Error parsing RPC URL", err))
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

// Prints the parsed config to stdout, used for debugging
func (cfg *Config) showConfig(){
    fmt.Println("--[Telegram]--")
    fmt.Println("API: " + cfg.API) 
    fmt.Println("ChadID: " + cfg.ChatID)

    fmt.Println("\n--[Chain]--")
    fmt.Println("Chain: " + cfg.Chain)
    fmt.Println("ChainPrettyName: " + cfg.ChainPrettyName)
    fmt.Println("Coin: " + cfg.Coin)
    fmt.Println("Denom: " + cfg.Denom)
    fmt.Printf("Exponent: %d\n", cfg.Exponent)
    fmt.Println("CoinGecko ID: " + cfg.CoinGeckoID)
    fmt.Println("Bech32Prefix: " + cfg.Bech32Prefix)

    fmt.Println("\n--[URLS]--")
    fmt.Println("RestURL: " + cfg.RestURL)
    fmt.Println("WebsocketURL: " + cfg.WebsocketURL)
    fmt.Println("ICSNUrl: " + cfg.ICNSUrl)
    fmt.Println("RestTxURL: " + cfg.RestTx)
    fmt.Println("RestValidatorsURL: " + cfg.RestValidators)
    fmt.Println("RestCoinGecko: " + cfg.RestCoinGecko)
    fmt.Println("ExplorerTxURL: " + cfg.ExplorerTx)
    fmt.Println("ExplorerAccountURL: " + cfg.ExplorerAccount)
    fmt.Println("ExplorerValidatorURL: " + cfg.ExplorerValidator)


    fmt.Println("\n--[Preferences]--")
    fmt.Printf("Transfers Enabled: %t\n", cfg.Transfers)
    fmt.Printf("IBC In Transfers Enabled: %t\n", cfg.IBCIn)
    fmt.Printf("IBC Out Transfers Enabled: %t\n", cfg.IBCOut)
    fmt.Printf("Rewards Withdrawal Enabled: %t\n", cfg.Rewards)
    fmt.Printf("Comission Withdrawal Enabled: %t\n", cfg.Commission)
    fmt.Printf("Delegations Enabled: %t\n", cfg.Delegations)
    fmt.Printf("Undelegations Enabled: %t\n", cfg.Undelegations)
    fmt.Printf("Redelegations Enabled: %t\n", cfg.Redelegations)
    fmt.Printf("Restake Enabled: %t\n", cfg.Restake)


    fmt.Println("\n--[Wallets]--")
	for _, wallet := range cfg.Wallets {
		fmt.Printf("Name: %s\nAddr: %s\nValidator Addr: %s\n\n", wallet.Name, wallet.Addr, wallet.ValAddr)
	}
    fmt.Println()
}
