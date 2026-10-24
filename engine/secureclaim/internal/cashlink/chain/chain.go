// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

// Package chain is the narrow window the relayer needs onto an EVM chain. The HTTP implementation talks to several public
// nodes of one chain (the first that answers wins, each is checked to serve the right chain id); tests use an in-process EVM.
package chain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
)

// ErrUnavailable means no node of the chain answered usefully (all down, blocked, or serving another chain). It is a reason to
// retry later, never a verdict about a request.
var ErrUnavailable = errors.New("chain_unavailable")

// Log is one event of a mined transaction.
type Log struct {
	Address evm.Address
	Topics  [][]byte // 32 bytes each
	Data    []byte
}

// Receipt of a mined transaction.
type Receipt struct {
	Found   bool
	Success bool
	GasUsed uint64
	// GasPrice is the price per gas the transaction actually paid (wei), when the node says; nil otherwise.
	GasPrice *big.Int
	Block    uint64
	Logs     []Log
}

// Finality is what a caller needs to wait for DEPTH before believing a block: the latest block number, and a read of a past
// state. HTTP implements it; callers use it when the Chain they hold does.
type Finality interface {
	BlockNumber(ctx context.Context) (uint64, error)
	// CallAt is Call against the state at the given block.
	CallAt(ctx context.Context, from, to evm.Address, data []byte, block uint64) ([]byte, error)
}

// Chain is everything the relayer asks of a blockchain. Production code uses HTTP; tests use an in-process EVM that
// implements the same interface (see sim/relayer_test.go).
type Chain interface {
	ChainID() uint64
	// Call runs a read-only call as `from` against the latest state (from may be the zero address).
	Call(ctx context.Context, from, to evm.Address, data []byte) ([]byte, error)
	// EstimateGas fails when the call would revert.
	EstimateGas(ctx context.Context, from, to evm.Address, data []byte) (uint64, error)
	Code(ctx context.Context, addr evm.Address) ([]byte, error)
	PendingNonce(ctx context.Context, addr evm.Address) (uint64, error)
	NativeBalance(ctx context.Context, addr evm.Address) (*big.Int, error)
	// Fees returns the max priority fee and max fee per gas for a transaction that should be included within minutes.
	Fees(ctx context.Context) (tip, maxFee *big.Int, err error)
	Send(ctx context.Context, raw []byte) error
	Receipt(ctx context.Context, hash string) (Receipt, error)
}

// BlockTimer is what a caller needs to date an event: when a block was mined (the block's own timestamp, never the time of the
// question). HTTP implements it; callers use it when the Chain they hold does, and leave the date unknown when it does not.
type BlockTimer interface {
	BlockTime(ctx context.Context, block uint64) (time.Time, error)
}

// HTTP talks JSON-RPC to public nodes of one chain.
type HTTP struct {
	id        uint64
	endpoints []string
	minTip    *big.Int
	client    *http.Client
	allowHTTP bool
	mu        sync.Mutex
	checked   map[string]bool
}

// NewHTTP: minTip is the chain's minimum priority fee (Polygon refuses transactions below 25-30 gwei).
func NewHTTP(chainID uint64, endpoints []string, minTip *big.Int) *HTTP {
	return &HTTP{id: chainID, endpoints: endpoints, minTip: minTip, checked: map[string]bool{},
		client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// AllowHTTPForTests lets the client talk to plain-http loopback servers.
func (c *HTTP) AllowHTTPForTests() { c.allowHTTP = true }

// ChainID is the chain this client was built for (every node is checked to serve it).
func (c *HTTP) ChainID() uint64 { return c.id }

type rpcError struct{ msg string }

// Error makes a JSON-RPC error answer (for instance a revert) an error value distinct from ErrUnavailable.
func (e rpcError) Error() string { return "rpc: " + e.msg }

func (c *HTTP) post(ctx context.Context, endpoint, method string, params []any) (json.RawMessage, error) {
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "https" && !(c.allowHTTP && u.Scheme == "http")) {
		return nil, ErrUnavailable
	}
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PhoenixCashlink/1")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil || res.StatusCode != 200 {
		return nil, ErrUnavailable
	}
	var reply struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &reply) != nil {
		return nil, ErrUnavailable
	}
	if reply.Error != nil {
		return nil, rpcError{reply.Error.Message}
	}
	return reply.Result, nil
}

func (c *HTTP) verified(ctx context.Context, endpoint string) bool {
	c.mu.Lock()
	ok := c.checked[endpoint]
	c.mu.Unlock()
	if ok {
		return true
	}
	raw, err := c.post(ctx, endpoint, "eth_chainId", nil)
	var s string
	if err != nil || json.Unmarshal(raw, &s) != nil {
		return false
	}
	if n, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64); err != nil || n != c.id {
		return false
	}
	c.mu.Lock()
	c.checked[endpoint] = true
	c.mu.Unlock()
	return true
}

// call asks the nodes in turn. A JSON-RPC error (a revert, a bad request) is an answer, not an outage: it is returned.
func (c *HTTP) call(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	last := error(ErrUnavailable)
	for _, endpoint := range c.endpoints {
		if !c.verified(ctx, endpoint) {
			continue
		}
		raw, err := c.post(ctx, endpoint, method, params)
		if err == nil {
			return raw, nil
		}
		last = err
		if _, ok := err.(rpcError); ok {
			return nil, err
		}
	}
	return nil, last
}

func hexBig(raw json.RawMessage) (*big.Int, error) {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return nil, ErrUnavailable
	}
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return new(big.Int), nil
	}
	v, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, ErrUnavailable
	}
	return v, nil
}

func hexBytes(raw json.RawMessage) ([]byte, error) {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return nil, ErrUnavailable
	}
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, ErrUnavailable
	}
	return b, nil
}

func callObject(from, to evm.Address, data []byte) map[string]string {
	m := map[string]string{"to": to.Hex(), "data": "0x" + hex.EncodeToString(data)}
	if from != (evm.Address{}) {
		m["from"] = from.Hex()
	}
	return m
}

// Call is eth_call at the latest block.
func (c *HTTP) Call(ctx context.Context, from, to evm.Address, data []byte) ([]byte, error) {
	raw, err := c.call(ctx, "eth_call", callObject(from, to, data), "latest")
	if err != nil {
		return nil, err
	}
	return hexBytes(raw)
}

// EstimateGas is eth_estimateGas: it returns an error when the call would revert, which is how the relayer avoids
// sending a transaction that would fail.
func (c *HTTP) EstimateGas(ctx context.Context, from, to evm.Address, data []byte) (uint64, error) {
	raw, err := c.call(ctx, "eth_estimateGas", callObject(from, to, data))
	if err != nil {
		return 0, err
	}
	v, err := hexBig(raw)
	if err != nil || !v.IsUint64() {
		return 0, ErrUnavailable
	}
	return v.Uint64(), nil
}

// Code is eth_getCode: empty for an address where nothing is deployed (yet).
func (c *HTTP) Code(ctx context.Context, addr evm.Address) ([]byte, error) {
	raw, err := c.call(ctx, "eth_getCode", addr.Hex(), "latest")
	if err != nil {
		return nil, err
	}
	return hexBytes(raw)
}

// PendingNonce counts every transaction already sent from the address, mined or not.
func (c *HTTP) PendingNonce(ctx context.Context, addr evm.Address) (uint64, error) {
	raw, err := c.call(ctx, "eth_getTransactionCount", addr.Hex(), "pending")
	if err != nil {
		return 0, err
	}
	v, err := hexBig(raw)
	if err != nil || !v.IsUint64() {
		return 0, ErrUnavailable
	}
	return v.Uint64(), nil
}

// NativeBalance is the ETH/POL balance in wei.
func (c *HTTP) NativeBalance(ctx context.Context, addr evm.Address) (*big.Int, error) {
	raw, err := c.call(ctx, "eth_getBalance", addr.Hex(), "latest")
	if err != nil {
		return nil, err
	}
	return hexBig(raw)
}

// Fees: twice the current base fee plus the tip (never below the chain's minimum tip).
func (c *HTTP) Fees(ctx context.Context) (*big.Int, *big.Int, error) {
	raw, err := c.call(ctx, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return nil, nil, err
	}
	var block struct {
		BaseFee string `json:"baseFeePerGas"`
	}
	if json.Unmarshal(raw, &block) != nil || block.BaseFee == "" {
		return nil, nil, ErrUnavailable
	}
	base, err := hexBig(json.RawMessage(strconv.Quote(block.BaseFee)))
	if err != nil {
		return nil, nil, err
	}
	tip := new(big.Int)
	if c.minTip != nil {
		tip.Set(c.minTip)
	}
	if raw, err := c.call(ctx, "eth_maxPriorityFeePerGas"); err == nil {
		if v, err := hexBig(raw); err == nil && v.Cmp(tip) > 0 {
			tip = v
		}
	}
	return tip, new(big.Int).Add(new(big.Int).Mul(base, big.NewInt(2)), tip), nil
}

// Send broadcasts a signed transaction. "Already known" (a retry of the same bytes) counts as success.
func (c *HTTP) Send(ctx context.Context, raw []byte) error {
	_, err := c.call(ctx, "eth_sendRawTransaction", "0x"+hex.EncodeToString(raw))
	if err != nil {
		m := strings.ToLower(err.Error())
		if strings.Contains(m, "already known") || strings.Contains(m, "known transaction") || strings.Contains(m, "already imported") {
			return nil
		}
	}
	return err
}

// BlockTime is the timestamp of a mined block, in UTC. A block the node does not know, or a timestamp that makes no sense, is an error.
func (c *HTTP) BlockTime(ctx context.Context, block uint64) (time.Time, error) {
	raw, err := c.call(ctx, "eth_getBlockByNumber", "0x"+strconv.FormatUint(block, 16), false)
	if err != nil {
		return time.Time{}, err
	}
	var b struct {
		Number    string `json:"number"`
		Timestamp string `json:"timestamp"`
	}
	if string(raw) == "null" || json.Unmarshal(raw, &b) != nil {
		return time.Time{}, ErrUnavailable
	}
	number, e1 := hexBig(json.RawMessage(strconv.Quote(b.Number)))
	seconds, e2 := hexBig(json.RawMessage(strconv.Quote(b.Timestamp)))
	if e1 != nil || e2 != nil || !number.IsUint64() || number.Uint64() != block || !seconds.IsInt64() || seconds.Int64() < 1_400_000_000 || seconds.Int64() > 1<<40 {
		return time.Time{}, ErrUnavailable
	}
	return time.Unix(seconds.Int64(), 0).UTC(), nil
}

// Receipt reports whether a transaction is mined and whether it succeeded. A transaction not yet mined gives Found=false.
func (c *HTTP) Receipt(ctx context.Context, hash string) (Receipt, error) {
	raw, err := c.call(ctx, "eth_getTransactionReceipt", hash)
	if err != nil {
		return Receipt{}, err
	}
	if string(raw) == "null" {
		return Receipt{}, nil
	}
	var r struct {
		Status string `json:"status"`
		Gas    string `json:"gasUsed"`
		Price  string `json:"effectiveGasPrice"`
		Block  string `json:"blockNumber"`
		Logs   []struct {
			Address string   `json:"address"`
			Topics  []string `json:"topics"`
			Data    string   `json:"data"`
		} `json:"logs"`
	}
	if json.Unmarshal(raw, &r) != nil {
		return Receipt{}, ErrUnavailable
	}
	gas, e1 := hexBig(json.RawMessage(strconv.Quote(r.Gas)))
	block, e2 := hexBig(json.RawMessage(strconv.Quote(r.Block)))
	if e1 != nil || e2 != nil || !gas.IsUint64() || !block.IsUint64() {
		return Receipt{}, ErrUnavailable
	}
	out := Receipt{Found: true, Success: r.Status == "0x1", GasUsed: gas.Uint64(), Block: block.Uint64()}
	if r.Price != "" {
		if price, err := hexBig(json.RawMessage(strconv.Quote(r.Price))); err == nil {
			out.GasPrice = price
		}
	}
	for _, l := range r.Logs {
		addr, err := evm.ParseAddress(l.Address)
		data, derr := hex.DecodeString(strings.TrimPrefix(l.Data, "0x"))
		if err != nil || derr != nil || len(l.Topics) > 4 {
			return Receipt{}, ErrUnavailable
		}
		lg := Log{Address: addr, Data: data}
		for _, t := range l.Topics {
			tb, terr := hex.DecodeString(strings.TrimPrefix(t, "0x"))
			if terr != nil || len(tb) != 32 {
				return Receipt{}, ErrUnavailable
			}
			lg.Topics = append(lg.Topics, tb)
		}
		out.Logs = append(out.Logs, lg)
	}
	return out, nil
}

// BlockNumber is eth_blockNumber.
func (c *HTTP) BlockNumber(ctx context.Context) (uint64, error) {
	raw, err := c.call(ctx, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	n, err := hexBig(raw)
	if err != nil || !n.IsUint64() {
		return 0, ErrUnavailable
	}
	return n.Uint64(), nil
}

// CallAt is eth_call at a past block.
func (c *HTTP) CallAt(ctx context.Context, from, to evm.Address, data []byte, block uint64) ([]byte, error) {
	raw, err := c.call(ctx, "eth_call", callObject(from, to, data), "0x"+strconv.FormatUint(block, 16))
	if err != nil {
		return nil, err
	}
	return hexBytes(raw)
}

var _ Chain = (*HTTP)(nil)
var _ Finality = (*HTTP)(nil)

// Describe is a one-line summary for logs.
func (c *HTTP) Describe() string { return fmt.Sprintf("chain %d via %d nodes", c.id, len(c.endpoints)) }
