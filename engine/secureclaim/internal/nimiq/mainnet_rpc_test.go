package nimiq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCashlinkAddressDeterministicVector(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}
	addr, err := (AddressDeriver{}).CashlinkAddress(seed)
	if err != nil || addr != "NQ46KLJE5TMF4Y1A1255CJHJYG1SH0NUT604" {
		t.Fatalf("Nimiq derivation mismatch: %q %v", addr, err)
	}
	if _, err := (AddressDeriver{}).CashlinkAddress(seed[:31]); err == nil {
		t.Fatal("short seed accepted")
	}
}
func TestMainnetVerifierRejectsBadRPCAndFunding(t *testing.T) {
	addr := "NQ46KLJE5TMF4Y1A1255CJHJYG1SH0NUT604"
	network := "MainAlbatross"
	balance := uint64(250001)
	confirmed := int64(3)
	marker := cashlinkFundingMarker
	markerField := "recipientData"
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		var req struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		var data any
		switch req.Method {
		case "getBlockNumber":
			data = 100
		case "getBlockByNumber":
			data = map[string]any{"network": network}
		case "getAccountByAddress":
			data = map[string]any{"balance": balance}
		case "getTransactionsByAddress":
			data = []any{map[string]any{"to": addr, "value": 250000, markerField: marker, "confirmations": confirmed, "executionResult": true}}
		default:
			t.Errorf("unexpected method %q", req.Method)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"data": data}})
	}))
	defer srv.Close()
	v, err := NewMainnetVerifier(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ok, err := v.ConfirmedSpendable(ctx, "mainnet", addr, 250000)
	if err != nil || !ok || count != 4 {
		t.Fatalf("valid mainnet funding not recognized: %v %v calls=%d", ok, err, count)
	}
	network = "TestAlbatross"
	if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); ok || err == nil {
		t.Fatal("mainnet node accepted")
	}
	network = "MainAlbatross"
	balance = 1
	if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); ok || err != nil {
		t.Fatal("insufficient balance accepted", err)
	}
	balance = 250001
	confirmed = 0
	if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); ok || err != nil {
		t.Fatal("unconfirmed payment accepted", err)
	}
	confirmed = 3
	markerField = "senderData"
	if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); ok || err != nil {
		t.Fatal("marker in sender data alone accepted", err)
	}
	markerField = "recipientData"
	marker = "wrong"
	if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); ok || err != nil {
		t.Fatal("wrong funding marker accepted", err)
	}
	// Nimiq Pay sends the marker as UTF-8 text: that exact form, and only that, is the same funding mark.
	marker = "00C282C280C292C287"
	if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); !ok || err != nil {
		t.Fatal("the funding mark as Nimiq Pay sends it was not recognized", err)
	}
	for _, other := range []string{"", "00", "0082809287c2", "00c282c280c292c288", "c282c280c292c287", "00c282c280c292c28700", "00c2", "0082809288", "00 82 80 92 87", "0x0082809287"} {
		marker = other
		if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); ok || err != nil {
			t.Fatalf("funding mark %q accepted", other)
		}
	}
	// the amount, the address and the confirmations are still checked with the Nimiq Pay form
	marker = cashlinkFundingMarkerNimiqPay
	balance = 1
	if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); ok || err != nil {
		t.Fatal("insufficient balance accepted with the Nimiq Pay mark", err)
	}
	balance = 250001
	confirmed = 1
	if ok, err = v.ConfirmedSpendable(ctx, "mainnet", addr, 250000); ok || err != nil {
		t.Fatal("unconfirmed payment accepted with the Nimiq Pay mark", err)
	}
	confirmed = 3
	if ok, err = v.ConfirmedSpendable(ctx, "testnet", addr, 250000); ok || err == nil {
		t.Fatal("mainnet request accepted")
	}
}
func TestRPCRejectsRedirectAndPublicHTTP(t *testing.T) {
	if _, err := NewMainnetVerifier("http://example.com"); err == nil {
		t.Fatal("public HTTP allowed")
	}
	if _, err := NewMainnetVerifier("https://example.com/path"); err == nil {
		t.Fatal("unreviewed RPC path allowed")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://example.org", 302) }))
	defer srv.Close()
	v, err := NewMainnetVerifier(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := v.ConfirmedSpendable(context.Background(), "mainnet", "NQ46KLJE5TMF4Y1A1255CJHJYG1SH0NUT604", 100)
	if ok || err == nil {
		t.Fatal("redirect accepted")
	}
}
