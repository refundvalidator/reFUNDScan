package main

import (
	"encoding/json"
    "errors"
	"io"
	"log"
	"net/http"
	"time"
)

const (
    fundCoinGecko = "https://api.coingecko.com/api/v3/coins/unification"
    fundExplorerValidators = "https://rest.unification.io/cosmos/staking/v1beta1/validators?pagination.limit=100000"
)

// The JSON Response received by the websocket
type WebsocketResponse struct {
    Result struct {
        Events struct {
            MessageAction []string `json:"message.action"` 
            TransferSender []string `json:"transfer.sender"`
            TransferRecipient []string `json:"transfer.recipient"`
            IBCTransferSender []string `json:"ibc_transfer.sender"`
            IBCTransferReciever []string `json:"ibc_transfer.receiver"`
            IBCForeignSender []string `json:"fungible_token_packet.sender"`
            TransferAmount []string `json:"transfer.amount"`
            TxHash []string `json:"tx.hash"`
            WithdrawRewardsValidator []string `json:"withdraw_rewards.validator"`
            MessageSender []string `json:"message.sender"`
        } `json:"events"`
    } `json:"result"`
}
// The JSON response from a REST TX Query
type TxResponse struct {
    Tx struct {
        Body struct {
            Memo string `json:"memo"`
        } `json:"body"`
    }
} 
// The JSON response from a REST ValidatorSet Query
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
func (vals *ValidatorResponse) getData() error {
    resp, err := http.Get(fundExplorerValidators); 
    if err != nil {
        return errors.New("Failed to get Validator Information")
    } 
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return errors.New("Failed to read Validator Information")
    }
    if err := json.Unmarshal(body, vals); err != nil {
        return errors.New("Failed to unmarshall Validator Information")
    }
    return nil
}
func (vals *ValidatorResponse) autoRefresh(){
    ticker := time.NewTicker(time.Second * 60)
    if err := vals.getData(); err != nil {
        log.Println(err)
    }
    for {
        select {
        case <- ticker.C:
            if err := cg.getData(); err != nil {
                log.Println(err)
            }
        }
    }
}
type CoinGeckoResponse struct {
    MarketData struct{
        CurrentPrice struct {
            USD float64 `json:"usd"`
        } `json:"current_price"`
    } `json:"market_data"`
}
func (cg *CoinGeckoResponse) getData() error {
    resp, err := http.Get(fundCoinGecko); 
    if err != nil {
        return errors.New("Failed to get CoinGecko Information")
    } 
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return errors.New("Failed to read CoinGecko Information")
    }
    if err := json.Unmarshal(body, cg); err != nil {
        return errors.New("Failed to unmarshall CoinGecko Information")
    }
    return nil
}

func (cg *CoinGeckoResponse) autoRefresh() {
    ticker := time.NewTicker(time.Second * 60)
    if err := cg.getData(); err != nil {
        log.Println(err)
    }
    for {
        select {
        case <- ticker.C:
            if err := cg.getData(); err != nil {
                log.Println(err)
            }
        }
    }
}
