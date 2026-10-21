package nimiq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const cashlinkFundingMarker = "0082809287"

var ErrRPC = errors.New("nimiq_mainnet_rpc_unavailable")

// MainnetVerifier queries a configured Nimiq Albatross RPC endpoint.
// RPC answers are trusted third-party data, NOT independently verified
// consensus. Production should use a trusted own node or independent quorum.
type MainnetVerifier struct {
	endpoint string
	client   *http.Client
}

func NewMainnetVerifier(rawURL string) (*MainnetVerifier, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u == nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return nil, ErrRPC
	}
	host := u.Hostname()
	local := host == "localhost" || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return nil, ErrRPC
	}
	return &MainnetVerifier{endpoint: rawURL, client: &http.Client{
		Timeout:       8 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

type rpcEnvelope struct {
	Result *struct {
		Data json.RawMessage `json:"data"`
	} `json:"result"`
	Error *json.RawMessage `json:"error"`
}

func (v *MainnetVerifier) call(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		return nil, ErrRPC
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, ErrRPC
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, ErrRPC
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrRPC
	}
	var answer rpcEnvelope
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 65537))
	if err := decoder.Decode(&answer); err != nil || answer.Error != nil || answer.Result == nil || len(answer.Result.Data) == 0 || len(answer.Result.Data) > 65536 {
		return nil, ErrRPC
	}
	return answer.Result.Data, nil
}
func uintFromJSON(data json.RawMessage) (uint64, error) {
	var value json.Number
	if err := json.Unmarshal(data, &value); err == nil {
		return strconv.ParseUint(value.String(), 10, 64)
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		return strconv.ParseUint(s, 10, 64)
	}
	return 0, ErrRPC
}
func normalized(s string) string { return strings.ToUpper(strings.ReplaceAll(s, " ", "")) }

// NetworkReady checks the configured node before PhoenixMail opens a wallet.
func (v *MainnetVerifier) NetworkReady(ctx context.Context) error {
	if v == nil || ctx == nil {
		return ErrRPC
	}
	raw, err := v.call(ctx, "getBlockNumber", []any{})
	if err != nil {
		return err
	}
	height, err := uintFromJSON(raw)
	if err != nil || height < 3 {
		return ErrRPC
	}
	raw, err = v.call(ctx, "getBlockByNumber", []any{height - 1, false})
	if err != nil {
		return err
	}
	var block struct {
		Network string `json:"network"`
	}
	if json.Unmarshal(raw, &block) != nil || block.Network != "MainAlbatross" {
		return ErrRPC
	}
	return nil
}

// ConfirmedSpendable checks a node's Mainnet identity, current account
// balance and at least one confirmed Cashlink-funding transaction. This
// intentionally fails closed if a node omits any required field. It does not
// reserve funds or authorize settlement, which require an atomic state machine.
func (v *MainnetVerifier) ConfirmedSpendable(ctx context.Context, network, address string, exactLuna uint64) (bool, error) {
	if v == nil || network != "mainnet" || exactLuna == 0 || len(address) != 36 || !strings.HasPrefix(address, "NQ") || ctx == nil {
		return false, ErrRPC
	}
	heightRaw, err := v.call(ctx, "getBlockNumber", []any{})
	if err != nil {
		return false, err
	}
	height, err := uintFromJSON(heightRaw)
	if err != nil || height < 3 {
		return false, ErrRPC
	}
	blockRaw, err := v.call(ctx, "getBlockByNumber", []any{height - 1, false})
	if err != nil {
		return false, err
	}
	var block struct {
		Network string `json:"network"`
	}
	if err := json.Unmarshal(blockRaw, &block); err != nil || block.Network != "MainAlbatross" {
		return false, ErrRPC
	}
	accountRaw, err := v.call(ctx, "getAccountByAddress", []any{address})
	if err != nil {
		return false, err
	}
	var account struct {
		Balance json.RawMessage `json:"balance"`
	}
	if err := json.Unmarshal(accountRaw, &account); err != nil {
		return false, ErrRPC
	}
	balance, err := uintFromJSON(account.Balance)
	if err != nil {
		return false, ErrRPC
	}
	if balance < exactLuna {
		return false, nil
	}
	transactionsRaw, err := v.call(ctx, "getTransactionsByAddress", []any{address, 50, nil})
	if err != nil {
		return false, err
	}
	var transactions []struct {
		To              string          `json:"to"`
		Value           json.RawMessage `json:"value"`
		RecipientData   string          `json:"recipientData"`
		Confirmations   int64           `json:"confirmations"`
		ExecutionResult bool            `json:"executionResult"`
	}
	if err := json.Unmarshal(transactionsRaw, &transactions); err != nil {
		return false, ErrRPC
	}
	for _, tx := range transactions {
		amount, err := uintFromJSON(tx.Value)
		if err != nil {
			continue
		}
		if normalized(tx.To) == address && amount >= exactLuna && tx.Confirmations >= 2 && tx.ExecutionResult && strings.EqualFold(tx.RecipientData, cashlinkFundingMarker) {
			return true, nil
		}
	}
	return false, nil
}
