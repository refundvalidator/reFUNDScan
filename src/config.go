package main

import (
	"fmt"
	"log"
    "net/url"

	"github.com/BurntSushi/toml"
)

// Config struct to represent the structure of the TOML file
type ConfigFile struct {
	Telegram     TelegramConfig
	Chain        ChainConfig
    ChainInfo    ChainInfoConfig
	Connections  ConnectionsConfig
	ICNS         ICNSConfig
}
type TelegramConfig struct {
	ChatID string `toml:"chat-id"`
	API    string `toml:"api"`
}
type ChainConfig struct {
	Name string
}
type ChainInfoConfig struct {
    Default      bool   `toml:"default"`
    PrettyName   string `toml:"pretty-name"`
    Coin         string
    Denom        string
    Exponent     int 
    CoinGeckoID  string `toml:"coin-gecko-id"`
    Bech32Prefix string `toml:"bech32-prefix"`
}
type ConnectionsConfig struct {
	Default    bool
	Rest       string
	Websocket  string
}
type ICNSConfig struct {
	URL string
}

type Config struct {
    API             string
    ChatID          string

    Chain           string
    ChainPrettyName string
    Coin            string
    Denom           string
    Exponent        int
    CoinGeckoID     string
    Bech32Prefix    string

    RestURL         string
    WebsocketURL    string
    ICNSUrl         string

    //Used for formatting
    restTx             string
    explorerBase       string
    explorerTx         string
    explorerValidators string
    explorerAccount    string
}
func (cfg *Config) parseConfig(filePath string) {
    var configfile ConfigFile
    var chain ChainResponse
    var assets AssetsResponse

	if _, err := toml.DecodeFile(filePath, &configfile); err != nil {
		log.Println("Error parsing TOML file:", err)
		return
	}
    cfg.API = configfile.Telegram.API
    cfg.ChatID = configfile.Telegram.ChatID
    cfg.Chain = configfile.Chain.Name
    cfg.ICNSUrl = configfile.ICNS.URL

    if configfile.Connections.Default == true {
        getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", config.Chain),
            &chain)
        parsedRPC, err := url.Parse(chain.Apis.RPC[0].Address)
        if err != nil{
            log.Fatal("Error parsing Rest URL", err)
        }
        cfg.RestURL = chain.Apis.Rest[0].Address
        cfg.WebsocketURL = fmt.Sprintf("wss://%s/websocket",parsedRPC.Host)
    } else {
        cfg.RestURL = configfile.Connections.Rest
        cfg.WebsocketURL = configfile.Connections.Websocket
    }

    if configfile.ChainInfo.Default == true {
        getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/assetlist.json", config.Chain),
            &assets)
        getData(
            fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", config.Chain),
            &chain)
        cfg.ChainPrettyName = chain.PrettyName
        cfg.Bech32Prefix = chain.Bech32Prefix
        cfg.Denom = assets.Assets[0].DenomUnits[0].Denom
        cfg.Coin = assets.Assets[0].DenomUnits[1].Denom
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
    cfg.restTx = cfg.RestURL + "/cosmos/tx/v1beta1/txs/"
    cfg.explorerBase = fmt.Sprintf("https://ping.pub/%s/", cfg.ChainPrettyName)
    cfg.explorerTx = cfg.explorerBase + "tx/"
    cfg.explorerValidators = cfg.explorerBase + "staking/"
    cfg.explorerAccount = cfg.explorerBase + "account/"
}
