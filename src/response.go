package main

import (
    "encoding/json"
    "errors"
    "io"
    "log"
    "net/http"
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
            USD float64 `json:"usd"`
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
            IBCTransferReciever            []string `json:"ibc_transfer.receiver"`
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
func autoRefresh(url string, container interface{}) {
    ticker := time.NewTicker(time.Second * 60)
    if err := getData(url, container); err != nil {
        log.Println(color.RedString("Failed to get AutoRefresh Data: ", err))
    }
    for {
        select {
        case <-ticker.C:
            if err := getData(url, container); err != nil {
                log.Println(color.RedString("Failed to get AutoRefresh Data: ", err))
            }
        }
    }
}
