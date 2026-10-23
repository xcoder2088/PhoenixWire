package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BlockTime reads the timestamp of the block itself, and believes nothing it cannot check.
func TestBlockTimeIsTheTimestampOfThatBlock(t *testing.T) {
	number, timestamp := "0x64", "0x68f9fc59" // block 100, 2025-10-23 (UTC)
	var null bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		switch req.Method {
		case "eth_chainId":
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x89"}`)
		case "eth_getBlockByNumber":
			if null {
				fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":null}`)
				return
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":{"number":%q,"timestamp":%q}}`, number, timestamp)
		default:
			t.Errorf("unexpected method %s", req.Method)
		}
	}))
	defer srv.Close()
	c := NewHTTP(137, []string{srv.URL}, nil)
	c.AllowHTTPForTests()
	ctx := context.Background()
	at, err := c.BlockTime(ctx, 100)
	if err != nil || !at.Equal(time.Unix(0x68f9fc59, 0)) || at.Location() != time.UTC {
		t.Fatalf("block time: %v %v", at, err)
	}
	// another block than the one asked for, an absurd timestamp, an unknown block: errors, never a made-up date
	number = "0x65"
	if _, err := c.BlockTime(ctx, 100); err == nil {
		t.Fatal("a node answering for another block must not be believed")
	}
	number, timestamp = "0x64", "0x1"
	if _, err := c.BlockTime(ctx, 100); err == nil {
		t.Fatal("a timestamp of 1970 is not a block time")
	}
	timestamp = "0xffffffffffffffff"
	if _, err := c.BlockTime(ctx, 100); err == nil {
		t.Fatal("a timestamp far in the future is not a block time")
	}
	null = true
	if _, err := c.BlockTime(ctx, 100); err == nil {
		t.Fatal("an unknown block has no time")
	}
}
