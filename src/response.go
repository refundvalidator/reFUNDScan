package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)
type AssetsResponse struct {
	Assets    []struct {
		DenomUnits  []struct {
			Denom    string `json:"denom"`
			Exponent int    `json:"exponent"`
		} `json:"denom_units"`
		CoingeckoID string `json:"coingecko_id"`
        Coin        string `json:"symbol"`
	} `json:"assets"`
}
type ChainResponse struct {
	PrettyName   string   `json:"pretty_name"`
    Bech32Prefix string `json:"bech32_prefix"`
	Apis struct {
		RPC []struct {
			Address  string `json:"address"`
		} `json:"rpc"`
		Rest []struct {
			Address  string `json:"address"`
		} `json:"rest"`
	} `json:"apis"`
}
type ICNSResponse struct {
	Data struct {
		Name string `json:"name"`
	} `json:"data"`
}
type TxResponse struct {
    Tx struct {
        Body struct {
            Memo string `json:"memo"`
        } `json:"body"`
    }
}
// Only used for verifying a valid rest url response at startup
type RestResponse struct {
    Supply []struct {
        Denom  string `json:"denom"`
        Amount string `json:"amount"`
    } `json:"supply"`
}
type CoinGeckoResponse struct {
    MarketData struct{
        CurrentPrice struct {
            USD float64 `json:"usd"`
        } `json:"current_price"`
    } `json:"market_data"`
}
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
            WithdrawRewardsDelegator []string `json:"withdraw_rewards.delegator"`
            WithdrawRewardsAmount []string `json:"withdraw_rewards.amount"`
            WithdrawCommissionAmount []string `json:"withdraw_commission.amount"`
            MessageSender []string `json:"message.sender"`
            DelegateAmount []string `json:"delegate.amount"`
            DelegateValidator []string `json:"delegate.validator"`
            UnbondValidator []string `json:"unbond.validator"`
            UnbondAmount []string `json:"unbond.amount"`
            RedelegateSourceValidator []string `json:"redelegate.source_validator"`
            RedelegateDestinationValidator []string `json:"redelegate.destination_validator"`
            RedelegateAmount []string `json:"redelegate.amount"`
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
    resp, err := http.Get(url); 
    if err != nil {
        return errors.Join(err,errors.New("Failed to get Reponse Information from: " + url))
    } 
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return errors.Join(err,errors.New("Failed to read Response Information from: " + url))
    }
    if err := json.Unmarshal(body, container); err != nil {
        return errors.Join(err,errors.New("Failed to unmarshall Response Information from: " + url))
    }
    return nil
}
func autoRefresh(url string, container interface{}){
    ticker := time.NewTicker(time.Second * 60)
    if err := getData(url, container); err != nil {
        log.Println(err)
    }
    for {
        select {
        case <- ticker.C:
            if err := getData(url, container); err != nil {
                log.Println(err)
            }
        }
    }
}
