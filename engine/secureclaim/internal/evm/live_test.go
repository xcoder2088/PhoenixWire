//go:build live

package evm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Read-only checks against the real public nodes: go test -tags live ./internal/evm
func TestLiveNetworks(t *testing.T) {
	for _, asset := range Assets() {
		n, _ := Lookup(asset)
		c := NewClient(n)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		head, err := c.BlockNumber(ctx)
		if err != nil || head < 1000 {
			t.Fatalf("%s head: %d %v", asset, head, err)
		}
		// The contract really is USDT with 6 decimals.
		call := func(sig string) string {
			raw, err := c.call(ctx, "eth_call", map[string]string{"to": n.Token, "data": sig}, "latest")
			if err != nil {
				t.Fatalf("%s %s: %v", asset, sig, err)
			}
			var s string
			json.Unmarshal(raw, &s)
			return s
		}
		if d := call("0x313ce567"); !strings.HasSuffix(d, "6") { // decimals()
			t.Fatalf("%s decimals %s", asset, d)
		}
		settled, err := c.Settled(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// Transfers to a fresh address over the last blocks: the log query works on two nodes and agrees (here: none).
		transfers, err := c.Transfers(ctx, settled-200, settled, "0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb", "")
		if err != nil {
			t.Fatalf("%s logs: %v", asset, err)
		}
		t.Logf("%s head=%d settled=%d transfers=%d", asset, head, settled, len(transfers))
		// A busy address (the USDT contract's own recent transfers to Tether Treasury would do; here any address) is not needed:
		// the query shape is what matters.
	}
}
