package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrRPCUnavailable = errors.New("rpc_unavailable")
	ErrRPCRejected    = errors.New("rpc_rejected")
)

// Client talks JSON-RPC to several public endpoints of ONE chain: the first answer wins for ordinary
// reads, while balance reads that decide whether money is there need two endpoints to agree.
type Client struct {
	net       Network
	http      *http.Client
	mu        sync.Mutex
	verified  map[string]bool
	allowHTTP bool
}

// AllowHTTPForTests lets a client talk to a plain-http loopback server. Production code never calls it.
func (c *Client) AllowHTTPForTests() { c.allowHTTP = true }

func NewClient(n Network) *Client {
	return &Client{net: n, verified: map[string]bool{}, http: &http.Client{Timeout: 12 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type rpcReply struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) post(ctx context.Context, endpoint, method string, params []any) (json.RawMessage, error) {
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "https" && !(c.allowHTTP && u.Scheme == "http")) {
		return nil, ErrRPCUnavailable
	}
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, ErrRPCUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PhoenixMail-SecureClaim/1")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, ErrRPCUnavailable
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil || res.StatusCode != 200 {
		return nil, ErrRPCUnavailable
	}
	var reply rpcReply
	if json.Unmarshal(raw, &reply) != nil {
		return nil, ErrRPCUnavailable
	}
	if reply.Error != nil {
		return nil, fmt.Errorf("%w: %s", ErrRPCRejected, reply.Error.Message)
	}
	return reply.Result, nil
}

// checked makes sure an endpoint really serves this network before any answer from it is trusted.
func (c *Client) checked(ctx context.Context, endpoint string) bool {
	c.mu.Lock()
	ok := c.verified[endpoint]
	c.mu.Unlock()
	if ok {
		return true
	}
	raw, err := c.post(ctx, endpoint, "eth_chainId", nil)
	if err != nil {
		return false
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return false
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	if err != nil || id != c.net.ChainID {
		return false
	}
	c.mu.Lock()
	c.verified[endpoint] = true
	c.mu.Unlock()
	return true
}

// call returns the first successful answer. A node that answers with a JSON-RPC error is treated as an answer
// (the request itself is wrong), not as an outage.
func (c *Client) call(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	var last error = ErrRPCUnavailable
	for _, endpoint := range c.net.RPCs {
		if !c.checked(ctx, endpoint) {
			continue
		}
		raw, err := c.post(ctx, endpoint, method, params)
		if err == nil {
			return raw, nil
		}
		last = err
		if errors.Is(err, ErrRPCRejected) {
			return nil, err
		}
	}
	return nil, last
}

// quorum asks up to two endpoints and returns every answer; at least min(2, endpoints) must succeed.
func (c *Client) quorum(ctx context.Context, method string, params ...any) ([]json.RawMessage, error) {
	want := 2
	if len(c.net.RPCs) < want {
		want = len(c.net.RPCs)
	}
	var out []json.RawMessage
	for _, endpoint := range c.net.RPCs {
		if !c.checked(ctx, endpoint) {
			continue
		}
		if raw, err := c.post(ctx, endpoint, method, params); err == nil {
			out = append(out, raw)
			if len(out) == 2 {
				break
			}
		}
	}
	if want == 0 || len(out) < want {
		return nil, ErrRPCUnavailable
	}
	return out, nil
}

func hexBig(raw json.RawMessage) (*big.Int, error) {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return nil, ErrRPCUnavailable
	}
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return big.NewInt(0), nil
	}
	v, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, ErrRPCUnavailable
	}
	return v, nil
}

func blockTag(block uint64) string {
	if block == 0 {
		return "latest"
	}
	return "0x" + strconv.FormatUint(block, 16)
}

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	raw, err := c.call(ctx, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	v, err := hexBig(raw)
	if err != nil || !v.IsUint64() {
		return 0, ErrRPCUnavailable
	}
	return v.Uint64(), nil
}

// Settled is the highest block old enough to count (latest minus the network's confirmation depth).
func (c *Client) Settled(ctx context.Context) (uint64, error) {
	head, err := c.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	if head <= c.net.Confirmations {
		return 0, ErrRPCUnavailable
	}
	return head - c.net.Confirmations, nil
}

const transferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" // Transfer(address,address,uint256)

// Transfer is one USDT transfer seen on the chain.
type Transfer struct {
	Hash  string
	Block uint64
	Index uint64
	From  string // EIP-55
	To    string
	Value *big.Int
}

func topicAddress(a string) string {
	return "0x" + strings.Repeat("0", 24) + strings.ToLower(strings.TrimPrefix(a, "0x"))
}

func addressFromTopic(t string) string {
	if len(t) != 66 {
		return ""
	}
	return Checksum("0x" + t[26:])
}

// callAny is call() for methods some public nodes refuse or limit (eth_getLogs): it moves on to the next node on any failure.
func (c *Client) callAny(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	for _, endpoint := range c.net.RPCs {
		if !c.checked(ctx, endpoint) {
			continue
		}
		if raw, err := c.post(ctx, endpoint, method, params); err == nil {
			return raw, nil
		}
	}
	return nil, ErrRPCUnavailable
}

type logEntry struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber string   `json:"blockNumber"`
	TxHash      string   `json:"transactionHash"`
	LogIndex    string   `json:"logIndex"`
	Removed     bool     `json:"removed"`
}

// parseTransfers keeps the well-formed USDT Transfer logs addressed to `recipient`.
func (c *Client) parseTransfers(logs []logEntry, recipient string) []Transfer {
	var out []Transfer
	for _, l := range logs {
		if l.Removed || !strings.EqualFold(l.Address, c.net.Token) || len(l.Topics) != 3 || l.Topics[0] != transferTopic || !strings.EqualFold(l.Topics[2], topicAddress(recipient)) {
			continue
		}
		value, ok := new(big.Int).SetString(strings.TrimPrefix(l.Data, "0x"), 16)
		block, err1 := hexBig(json.RawMessage(strconv.Quote(l.BlockNumber)))
		index, err2 := hexBig(json.RawMessage(strconv.Quote(l.LogIndex)))
		if !ok || err1 != nil || err2 != nil || !block.IsUint64() || !index.IsUint64() || len(l.TxHash) != 66 {
			continue
		}
		out = append(out, Transfer{Hash: l.TxHash, Block: block.Uint64(), Index: index.Uint64(), From: addressFromTopic(l.Topics[1]), To: Checksum(recipient), Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Block != out[j].Block {
			return out[i].Block < out[j].Block
		}
		return out[i].Index < out[j].Index
	})
	return out
}

// Transfers lists candidate USDT transfers to `recipient` (from `sender` too when given) between two blocks, both
// included. One node answers (few public nodes serve log queries); every candidate must then be confirmed with Confirm.
func (c *Client) Transfers(ctx context.Context, from, to uint64, recipient, sender string) ([]Transfer, error) {
	topics := []any{transferTopic, nil, topicAddress(recipient)}
	if sender != "" {
		topics[1] = topicAddress(sender)
	}
	raw, err := c.callAny(ctx, "eth_getLogs", map[string]any{"address": c.net.Token, "fromBlock": blockTag(from), "toBlock": blockTag(to), "topics": topics})
	if err != nil {
		return nil, err
	}
	var logs []logEntry
	if json.Unmarshal(raw, &logs) != nil {
		return nil, ErrRPCUnavailable
	}
	return c.parseTransfers(logs, recipient), nil
}

// Confirm checks one transaction on two independent nodes: it succeeded and really carries a USDT transfer to
// `recipient` of at least `min`. It returns that transfer (the largest such). One dishonest node cannot invent a payment.
func (c *Client) Confirm(ctx context.Context, hash, recipient string, min *big.Int) (Transfer, bool, error) {
	answers, err := c.quorum(ctx, "eth_getTransactionReceipt", hash)
	if err != nil {
		return Transfer{}, false, err
	}
	var best *Transfer
	for i, a := range answers {
		var receipt struct {
			Status string     `json:"status"`
			Logs   []logEntry `json:"logs"`
		}
		if string(a) == "null" || json.Unmarshal(a, &receipt) != nil || receipt.Status != "0x1" {
			return Transfer{}, false, nil
		}
		var found *Transfer
		for _, t := range c.parseTransfers(receipt.Logs, recipient) {
			if t.Hash == hash && t.Value.Cmp(min) >= 0 && (found == nil || t.Value.Cmp(found.Value) > 0) {
				t := t
				found = &t
			}
		}
		if found == nil {
			return Transfer{}, false, nil
		}
		if i == 0 {
			best = found
		} else if best.Block != found.Block || best.Value.Cmp(found.Value) != 0 {
			return Transfer{}, false, nil
		}
	}
	return *best, true, nil
}
