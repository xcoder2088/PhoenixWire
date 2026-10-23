// Package evmtest is a tiny fake EVM node for tests: a head block, the USDT Transfer logs it serves and their receipts. Test code only.
package evmtest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type log struct {
	block   uint64
	index   uint64
	hash    string
	from    string
	to      string
	value   *big.Int
	phantom bool // listed by eth_getLogs but unknown to eth_getTransactionReceipt: a lying node
}

// Chain serves eth_chainId, eth_blockNumber and eth_getLogs.
type Chain struct {
	mu      sync.Mutex
	t       *testing.T
	chainID uint64
	head    uint64
	logs    []log
	Calls   map[string]int
}

// NewChain returns a chain at block 1000.
func NewChain(t *testing.T, chainID uint64) *Chain {
	return &Chain{t: t, chainID: chainID, head: 1000, Calls: map[string]int{}}
}

func (f *Chain) token() string {
	if f.chainID == 1 {
		return "0xdAC17F958D2ee523a2206206994597C13D831ec7"
	}
	return "0xc2132D05D31c914a87C6611C10748AEb04B58e8F"
}

func lower(a string) string { return strings.ToLower(a) }

// Mine moves the head forward.
func (f *Chain) Mine(blocks uint64) { f.mu.Lock(); f.head += blocks; f.mu.Unlock() }

// Head is the current block.
func (f *Chain) Head() uint64 { f.mu.Lock(); defer f.mu.Unlock(); return f.head }

// PhantomTransfer is a transfer that only the log query reports: a dishonest node inventing a payment.
func (f *Chain) PhantomTransfer(block uint64, from, to string, value int64) {
	f.Transfer(block, from, to, value)
	f.mu.Lock()
	f.logs[len(f.logs)-1].phantom = true
	f.mu.Unlock()
}

// Transfer records a USDT transfer included in the given block; the head moves up to it if needed.
func (f *Chain) Transfer(block uint64, from, to string, value int64) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if block > f.head {
		f.head = block
	}
	hash := fmt.Sprintf("0x%064x", len(f.logs)+1)
	f.logs = append(f.logs, log{block: block, index: uint64(len(f.logs)), hash: hash, from: lower(from), to: lower(to), value: big.NewInt(value)})
	return hash
}

func pad(a string) string { return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(lower(a), "0x") }

func (f *Chain) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var req struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	f.Calls[req.Method]++
	reply := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":%s}`, b)
	}
	switch req.Method {
	case "eth_chainId":
		reply(fmt.Sprintf("0x%x", f.chainID))
	case "eth_blockNumber":
		reply(fmt.Sprintf("0x%x", f.head))
	case "eth_getLogs":
		var q struct {
			FromBlock string `json:"fromBlock"`
			ToBlock   string `json:"toBlock"`
			Topics    []any  `json:"topics"`
			Address   string `json:"address"`
		}
		json.Unmarshal(req.Params[0], &q)
		num := func(tag string) uint64 {
			if tag == "latest" {
				return f.head
			}
			n, _ := strconv.ParseUint(strings.TrimPrefix(tag, "0x"), 16, 64)
			return n
		}
		from, to := num(q.FromBlock), num(q.ToBlock)
		out := []map[string]any{}
		for _, l := range f.logs {
			if l.block < from || l.block > to {
				continue
			}
			if len(q.Topics) == 3 {
				if s, ok := q.Topics[1].(string); ok && s != pad(l.from) {
					continue
				}
				if s, ok := q.Topics[2].(string); ok && s != pad(l.to) {
					continue
				}
			}
			out = append(out, map[string]any{"address": q.Address, "blockNumber": fmt.Sprintf("0x%x", l.block), "transactionHash": l.hash, "logIndex": fmt.Sprintf("0x%x", l.index),
				"topics": []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", pad(l.from), pad(l.to)}, "data": fmt.Sprintf("0x%064x", l.value), "removed": false})
		}
		reply(out)
	case "eth_getTransactionReceipt":
		var hash string
		json.Unmarshal(req.Params[0], &hash)
		var entries []map[string]any
		var block uint64
		for _, l := range f.logs {
			if l.hash == hash && !l.phantom {
				block = l.block
				entries = append(entries, map[string]any{"address": f.token(), "blockNumber": fmt.Sprintf("0x%x", l.block), "transactionHash": l.hash, "logIndex": fmt.Sprintf("0x%x", l.index),
					"topics": []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", pad(l.from), pad(l.to)}, "data": fmt.Sprintf("0x%064x", l.value), "removed": false})
			}
		}
		if len(entries) == 0 {
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":null}`)
			return
		}
		reply(map[string]any{"status": "0x1", "blockNumber": fmt.Sprintf("0x%x", block), "logs": entries})
	default:
		f.t.Fatal("unexpected method", req.Method)
	}
}

// Server exposes the node over HTTP (plain http: clients must call AllowHTTPForTests).
func (f *Chain) Server() *httptest.Server { return httptest.NewServer(http.HandlerFunc(f.handle)) }
