package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
)

type AssetsResponse struct {
    Assets []struct {
        DenomUnits []struct {
            Denom    string `json:"denom"`
            Exponent int    `json:"exponent"`
        } `json:"denom_units"`
        CoingeckoID string `json:"coingecko_id"`
        Coin        string `json:"symbol"`
        Display     string `json:"display"`
        Denom       string `json:"base"`
    } `json:"assets"`
}
type ChainResponse struct {
    PrettyName   string `json:"pretty_name"`
    Bech32Prefix string `json:"bech32_prefix"`
    Apis         struct {
        RPC []struct {
            Address string `json:"address"`
        } `json:"rpc"`
        Rest []struct {
            Address string `json:"address"`
        } `json:"rest"`
    } `json:"apis"`
}
type GitHubResponse struct {
    Chains []string `json:"mainnets"`
}
type IBCResponse struct {
    DenomTrace struct {
        BaseDenom string `json:"base_denom"` 
    } `json:"denom_trace"`
}
type ICNSResponse struct {
    Data struct {
        PrimaryName string `json:"primary_name"`
    } `json:"data"`
}
type TxResponse struct {
    Tx struct {
        Body struct {
            Memo string `json:"memo"`
        } `json:"body"`
    }
}
type CoinGeckoResponse struct {
    MarketData struct {
        CurrentPrice struct {
			Aed  float64 `json:"aed"`
			Ars  float64 `json:"ars"`
			Aud  float64 `json:"aud"`
			Bch  float64 `json:"bch"`
			Bdt  float64 `json:"bdt"`
			Bhd  float64 `json:"bhd"`
			Bmd  float64 `json:"bmd"`
			Bnb  float64 `json:"bnb"`
			Brl  float64 `json:"brl"`
			Btc  float64 `json:"btc"`
			Cad  float64 `json:"cad"`
			Chf  float64 `json:"chf"`
			Clp  float64 `json:"clp"`
			Cny  float64 `json:"cny"`
			Czk  float64 `json:"czk"`
			Dkk  float64 `json:"dkk"`
			Dot  float64 `json:"dot"`
			Eos  float64 `json:"eos"`
			Eth  float64 `json:"eth"`
			Eur  float64 `json:"eur"`
			Gbp  float64 `json:"gbp"`
			Gel  float64 `json:"gel"`
			Hkd  float64 `json:"hkd"`
			Huf  float64 `json:"huf"`
			Idr  float64 `json:"idr"`
			Ils  float64 `json:"ils"`
			Inr  float64 `json:"inr"`
			Jpy  float64 `json:"jpy"`
			Krw  float64 `json:"krw"`
			Kwd  float64 `json:"kwd"`
			Lkr  float64 `json:"lkr"`
			Ltc  float64 `json:"ltc"`
			Mmk  float64 `json:"mmk"`
			Mxn  float64 `json:"mxn"`
			Myr  float64 `json:"myr"`
			Ngn  float64 `json:"ngn"`
			Nok  float64 `json:"nok"`
			Nzd  float64 `json:"nzd"`
			Php  float64 `json:"php"`
			Pkr  float64 `json:"pkr"`
			Pln  float64 `json:"pln"`
			Rub  float64 `json:"rub"`
			Sar  float64 `json:"sar"`
			Sek  float64 `json:"sek"`
			Sgd  float64 `json:"sgd"`
			Thb  float64 `json:"thb"`
			Try  float64 `json:"try"`
			Twd  float64 `json:"twd"`
			Uah  float64 `json:"uah"`
			Usd  float64 `json:"usd"`
			Vef  float64 `json:"vef"`
			Vnd  float64 `json:"vnd"`
			Xag  float64 `json:"xag"`
			Xau  float64 `json:"xau"`
			Xdr  float64 `json:"xdr"`
			Xlm  float64 `json:"xlm"`
			Xrp  float64 `json:"xrp"`
			Yfi  float64 `json:"yfi"`
			Zar  float64 `json:"zar"`
			Bits float64 `json:"bits"`
			Link float64 `json:"link"`
			Sats float64 `json:"sats"`
		} `json:"current_price"`
    } `json:"market_data"`
}

type Coin struct {
    Denom  string `json:"denom"`
    Amount string `json:"amount"`
}

//{\"@type\":\"/starnamed.x.starname.v1beta1.Account\",\"domain\":\"me\",\"name\":\"observer-test\",\"owner\":\"star15k7tssu0wyrfq57zj7ye297n50ew3sffy25me8\",\"broker\":\"\",\"valid_until\":\"1738468896\",\"resources\":[],\"certificates\":[],\"metadata_uri\":\"\"}

type EscrowObject struct {
    Type       string `json:"@type"`
    Domain     string `json:"domain"`
    Name       string `json:"name"`
    Owner      string `json:"owner"`
    Broker     string `json:"broker"`
    ValidUntil string `json:"valid_until"`
}

type WebsocketResponse struct {
    Result struct {
        Events struct {
            MessageAction                  []string `json:"message.action"`
            TransferSender                 []string `json:"transfer.sender"`
            TransferRecipient              []string `json:"transfer.recipient"`
            IBCTransferSender              []string `json:"ibc_transfer.sender"`
            IBCTransferRecipient           []string `json:"ibc_transfer.receiver"`
            IBCForeignSender               []string `json:"fungible_token_packet.sender"`
            TransferAmount                 []string `json:"transfer.amount"`
            TxHash                         []string `json:"tx.hash"`
            WithdrawRewardsValidator       []string `json:"withdraw_rewards.validator"`
            WithdrawRewardsDelegator       []string `json:"withdraw_rewards.delegator"`
            WithdrawRewardsAmount          []string `json:"withdraw_rewards.amount"`
            WithdrawCommissionAmount       []string `json:"withdraw_commission.amount"`
            MessageSender                  []string `json:"message.sender"`
            DelegateAmount                 []string `json:"delegate.amount"`
            DelegateValidator              []string `json:"delegate.validator"`
            UnbondValidator                []string `json:"unbond.validator"`
            UnbondAmount                   []string `json:"unbond.amount"`
            RedelegateSourceValidator      []string `json:"redelegate.source_validator"`
            RedelegateDestinationValidator []string `json:"redelegate.destination_validator"`
            RedelegateAmount               []string `json:"redelegate.amount"`
            // Starname
            AccountName        []string `json:"message.account_name"`
            DomainName         []string `json:"message.domain_name"`
            Registerer         []string `json:"message.registerer"`
            NewAccountOwner    []string `json:"message.new_account_owner"`
            NewDomainOwner     []string `json:"message.new_domain_owner"`
            CreateEscrowPrice  []string `json:"starnamed.x.escrow.v1beta1.EventCreatedEscrow.price"`
            CreateEscrowObject []string `json:"starnamed.x.escrow.v1beta1.EventCreatedEscrow.object"`
        } `json:"events"`
    } `json:"result"`
}
type ValidatorResponse struct {
    Validators []struct {
        OperatorAddress string `json:"operator_address"`
        ConsensusPubkey struct {
            Type string `json:"@type"`
            Key  string `json:"key"`
        } `json:"consensus_pubkey"`
        Jailed          bool   `json:"jailed"`
        Status          string `json:"status"`
        Tokens          string `json:"tokens"`
        DelegatorShares string `json:"delegator_shares"`
        Description     struct {
            Moniker         string `json:"moniker"`
            Identity        string `json:"identity"`
            Website         string `json:"website"`
            SecurityContact string `json:"security_contact"`
            Details         string `json:"details"`
        } `json:"description"`
        UnbondingHeight string    `json:"unbonding_height"`
        UnbondingTime   time.Time `json:"unbonding_time"`
        Commission      struct {
            CommissionRates struct {
                Rate          string `json:"rate"`
                MaxRate       string `json:"max_rate"`
                MaxChangeRate string `json:"max_change_rate"`
            } `json:"commission_rates"`
            UpdateTime time.Time `json:"update_time"`
        } `json:"commission"`
        MinSelfDelegation string `json:"min_self_delegation"`
    } `json:"validators"`
    Pagination struct {
        NextKey string `json:"next_key"`
        Total   string `json:"total"`
    } `json:"pagination"`
}

func getData(url string, container interface{}) error {
    resp, err := http.Get(url)
    if err != nil {
        return errors.Join(err, errors.New("Failed to get Reponse Information from: "+url))
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return errors.Join(err, errors.New("Failed to read Response Information from: "+url))
    }
    if err := json.Unmarshal(body, container); err != nil {
        return errors.Join(err, errors.New("Failed to unmarshall Response Information from: "+url))
    }
    return nil
}
// TODO create an autoRefresh function specifically for coin gecko, which handles delays/rate limits
// and discards responses if the data is 0. Coin Gecko's rate limit doesnt block the URL, instead it
// returns garbage data (0's)
func autoRefresh(url string, container interface{}) {
    delay := time.Duration(300) * time.Second
    // SLUG this is a cheesy fix
    // Stagger the coin gecko responses
    if strings.Contains(url, "coingecko") {
        delay = time.Duration(rand.Intn(600-300+1)+300) * time.Second
    }
    ticker := time.NewTicker(delay)
    if err := getData(url, container); err != nil {
        log.Println(color.YellowString("Failed to get AutoRefresh Data: ", err))
    }
    for {
        select {
        case <-ticker.C:
            if err := getData(url, container); err != nil {
                log.Println(color.YellowString("Failed to get AutoRefresh Data: ", err))
            }
        }
    }
}
