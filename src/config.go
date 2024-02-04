package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gorilla/websocket"
)

// Config struct to represent the structure of the TOML file
type ConfigFile struct {
	Telegram struct {
		ChatID string `toml:"chat-id"`
		API    string `toml:"api"`
	} `toml:"telegram"`
	Chain struct {
		Name string
	} `toml:"chain"`
	ChainInfo struct {
		Default      bool   `toml:"default"`
		PrettyName   string `toml:"pretty-name"`
		Coin         string `toml:"coin"`
		Denom        string `toml:"denom"`
		Exponent     int    `toml:"exponent"`
		CoinGeckoID  string `toml:"coin-gecko-id"`
		Bech32Prefix string `toml:"bech32-prefix"`
	} `toml:"chaininfo"`
	Connections struct {
		Default   bool
		Rest      string
		Websocket string
	} `toml:"connections"`
	ICNS struct {
		URL string
	} `toml:"icns"`
	Wallets  []WalletsConfig `toml:"wallet"`
	Explorer struct {
		Preset    string `toml:"explorer-preset"`
		Tx        string `toml:"explorer-custom-tx"`
		Account   string `toml:"explorer-custom-account"`
		Validator string `toml:"explorer-custom-validator"`
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
		RegisterAccount bool `toml:"register-account"`
		RegisterDomain  bool `toml:"register-domain"`
		TransferAccount bool `toml:"transfer-account"`
		TransferDomain  bool `toml:"transfer-domain"`
		DeleteAccount   bool `toml:"delete-account"`
	} `toml:"general"`
}
type WalletsConfig struct {
	Name    string `toml:"name"`
	Addr    string `toml:"addr"`
	ValAddr string `toml:"val-addr"`
}

type Config struct {
	API    string
	ChatID string

	Chain           string
	ChainPrettyName string
	Coin            string
	Denom           string
	CoinGeckoID     string
	Bech32Prefix    string
	Exponent        int

	RestURL      string
	WebsocketURL string
	ICNSUrl      string

	Transfers       bool
	IBCIn           bool
	IBCOut          bool
	Rewards         bool
	Commission      bool
	Delegations     bool
	Undelegations   bool
	Redelegations   bool
	Restake         bool
	RegisterAccount bool
	RegisterDomain  bool
	TransferAccount bool
	TransferDomain  bool
	DeleteAccount   bool

	Wallets []WalletsConfig

	RestTx string

	ExplorerTx        string
	ExplorerAccount   string
	ExplorerValidator string
}

var (
	configfile ConfigFile
	chain      ChainResponse
	assets     AssetsResponse
)

func (cfg *Config) parseConfig(filePath string) {

	if _, err := toml.DecodeFile(filePath+"/config.toml", &configfile); err != nil {
		log.Println("Error parsing TOML file:", err)
		return
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
	cfg.RegisterAccount = configfile.General.RegisterAccount
	cfg.RegisterDomain = configfile.General.RegisterDomain
	cfg.TransferAccount = configfile.General.TransferAccount
	cfg.TransferDomain = configfile.General.TransferDomain
	cfg.DeleteAccount = configfile.General.DeleteAccount
	cfg.Restake = configfile.General.Restake
	cfg.Wallets = configfile.Wallets

	if configfile.Connections.Default == true {
		err := getData(
			fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", config.Chain),
			&chain)
		if err != nil {
			log.Fatal("Failed to get the chain.json from the chain registry, check your configuration")
		}
		parsedRPC, err := url.Parse(chain.Apis.RPC[0].Address)
		if err != nil {
			log.Fatal("Error parsing Rest URL", err)
		}
		cfg.RestURL = chain.Apis.Rest[0].Address
		cfg.WebsocketURL = fmt.Sprintf("wss://%s/websocket", parsedRPC.Host)
	} else {
		cfg.RestURL = configfile.Connections.Rest
		cfg.WebsocketURL = configfile.Connections.Websocket
	}

	if configfile.ChainInfo.Default == true {
		err := getData(
			fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/assetlist.json", config.Chain),
			&assets)
		if err != nil {
			log.Fatal("Failed to get the assetslist.json from the chain registry, check your configuration")
		}
		err = getData(
			fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", config.Chain),
			&chain)
		if err != nil {
			log.Fatal("Failed to get the chain.json from the chain registry, check your configuration")
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
		base := "https://ping.pub/" + cfg.ChainPrettyName
		cfg.ExplorerTx = base + "/txs/"
		cfg.ExplorerAccount = base + "/account/"
		cfg.ExplorerValidator = base + "/staking/"
	case "atom":
		base := "https://atomscan.com/" + cfg.ChainPrettyName
		cfg.ExplorerTx = base + "/transactions/"
		cfg.ExplorerAccount = base + "/accounts/"
		cfg.ExplorerValidator = base + "/validators/"
	case "mint":
		base := "https://mintscan.io/" + cfg.ChainPrettyName
		cfg.ExplorerTx = base + "/tx/"
		cfg.ExplorerAccount = base + "/address/"
		cfg.ExplorerValidator = base + "/validators/"
	case "dipper":
		base := "https://bigdipper.live/" + cfg.ChainPrettyName
		cfg.ExplorerTx = base + "/transactions/"
		cfg.ExplorerAccount = base + "/accounts/"
		cfg.ExplorerValidator = base + "/validators/"
	default:
		base := "https://ping.pub/" + cfg.ChainPrettyName
		cfg.ExplorerTx = base + "/txs/"
		cfg.ExplorerAccount = base + "/account/"
		cfg.ExplorerValidator = base + "/staking/"
	}
	cfg.RestTx = cfg.RestURL + "/cosmos/tx/v1beta1/txs/"
	cfg.validateConfig()
}
func (cfg *Config) validateConfig() {
	log.Println("Validating Config...")
	log.Println("Testing ICNS URL...")
	client := &http.Client{Timeout: 10 * time.Second}

	if response, err := client.Head(cfg.ICNSUrl); err != nil || response.StatusCode != http.StatusNotImplemented {
		log.Fatal("Bad ICNS URL, Please verify your config")
	} else {
		log.Println("ICNS URL Valid\n")
	}
	log.Println("Testing Rest URL...")
	if response, err := client.Head(cfg.RestURL + "/cosmos/tx/v1beta1/txs"); err != nil || response.StatusCode != http.StatusNotImplemented {
		if configfile.Connections.Default != true {
			log.Fatal("Bad Rest URL, Please verify your config")
		}
		success := false
		for i, u := range chain.Apis.Rest {
			if i == 0 {
				continue
			}
			log.Println("Bad Rest URL, trying the next one in the registry...")
			cfg.RestURL = strings.TrimRight(u.Address, "/")
			if response, err := client.Head(cfg.RestURL + "/cosmos/tx/v1beta1/txs"); err == nil && response.StatusCode == http.StatusNotImplemented {
				log.Println("Using Rest URL: " + u.Address)
				log.Println("Rest URL Valid\n")
				success = true
				break
			}
		}
		if success != true {
			log.Fatal("Could not find valid Rest URL in the chain registry, please provide your own")
		}
	} else {
		log.Println("Rest URL Valid\n")
	}
	log.Println("Testing RPC/Websocket URL...")
	if _, _, err := websocket.DefaultDialer.Dial(cfg.WebsocketURL, nil); err != nil {
		if configfile.Connections.Default != true {
			log.Fatal("Bad RPC/Websocket URL, Please verify your config")
		}
		success := false
		for i, u := range chain.Apis.RPC {
			if i == 0 {
				continue
			}
			log.Println("Bad RPC/Websocket URL, trying the next one in the registry...")
			parsedRPC, err := url.Parse(u.Address)
			if err != nil {
				log.Println("Failed to Parse")
				continue
			}
			cfg.WebsocketURL = fmt.Sprintf("wss://%s/websocket", parsedRPC.Host)
			if _, _, err := websocket.DefaultDialer.Dial(cfg.WebsocketURL, nil); err == nil {
				log.Println("Using RPC/Websocket URL: " + u.Address)
				log.Println("RPC/Websocket URL Valid\n")
				success = true
				break
			}
		}
		if success != true {
			log.Fatal("Could not find valid RPC URL in the chain registry, please provide your own")
		}
	} else {
		log.Println("RPC/Websocket URL Valid\n")
	}
	//Format the information
	cfg.RestURL = strings.TrimRight(cfg.RestURL, "/")
	cfg.ICNSUrl = strings.TrimRight(cfg.ICNSUrl, "/")
	cfg.ChainPrettyName = strings.ReplaceAll(cfg.ChainPrettyName, " ", "-")
}

// Prints the parsed config to stdout, used for debugging
func (cfg *Config) showConfig() {
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
	fmt.Println("WebscoketURL: " + cfg.WebsocketURL)
	fmt.Println("ICSNUrl: " + cfg.ICNSUrl)

	fmt.Println("\n--[Explorer]--")
	fmt.Println("WebscoketURL: " + cfg.WebsocketURL)
	fmt.Println("ICSNUrl: " + cfg.ICNSUrl)

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
