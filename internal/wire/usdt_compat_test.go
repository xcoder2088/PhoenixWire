package wire

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/service"
)

// PhoenixMail's payment window is served as it is: it talks to /api/usdt/cashlink/* and /api/usdt/status in PhoenixMail's shape.
func TestPaymentWindowRoutes(t *testing.T) {
	node := &evmNode{balance: word(0), block: 5000}
	rpc := node.serve()
	defer rpc.Close()
	t.Setenv("NIMIQ_CLAIM_USDTLINK_CHAINS", "polygon-usdc")
	t.Setenv("NIMIQ_CLAIM_EVM_RPC_POLYGON", rpc.URL)
	t.Setenv("NIMIQ_CLAIM_ALLOWED_ORIGINS", "https://wire.example.test")
	f := newFixture(t)
	for _, path := range []string{"/api/usdt/status", "/api/usdt/cashlink/create"} {
		method := "POST"
		if path == "/api/usdt/status" {
			method = "GET"
		}
		if w := f.call(method, path, map[string]string{}); w.Code != 401 {
			t.Fatalf("%s without a verified sender: %d", path, w.Code)
		}
	}
	f.verifySender("sender@example.test")
	var capability struct {
		Cashlink, Protected, Send, Gas, Requests bool
		Assets                                   []string
	}
	w := f.call("GET", "/api/usdt/status", nil)
	if json.Unmarshal(w.Body.Bytes(), &capability) != nil || !capability.Cashlink || !capability.Protected || !capability.Send || capability.Gas || capability.Requests || len(capability.Assets) != 3 {
		t.Fatalf("what the window asks first: %d %s", w.Code, w.Body)
	}
	if w := f.call("POST", "/api/usdt/cashlink/gas-status", map[string]string{"asset": "usdc_polygon"}); !strings.Contains(w.Body.String(), `"available":false`) {
		t.Fatalf("free gas is off: %s", w.Body)
	}
	id, _, owner, err := service.NewCardLink()
	if err != nil {
		t.Fatal(err)
	}
	cardID := hex.EncodeToString(id)
	// exactly what the window sends
	w = f.call("POST", "/api/usdt/cashlink/create", map[string]string{"asset": "usdc_polygon", "cardId": cardID, "linkOwner": owner, "amount": "2.5", "message": "test", "recipient": "friend@example.test", "senderEmail": "sender@example.test", "lang": "en"})
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var made struct {
		Card struct {
			ID, Safe, Token, Relayer string
			Owners                   []string
			Units, FeeUnits          uint64
			RefundFeeUnits           uint64
			ChainID                  int
		}
		Pay struct {
			Recipient, Link, Amount, QR string
			Units                       uint64
		}
		ClaimBase   string
		SenderToken string
	}
	if json.Unmarshal(w.Body.Bytes(), &made) != nil || made.Card.ID != cardID || len(made.Card.Owners) != 2 || made.Card.ChainID != 137 || made.Card.Units != 2_500_000 || made.Card.FeeUnits != 150_000 || made.Card.RefundFeeUnits != 100_000 ||
		made.Pay.Recipient != made.Card.Safe || made.Pay.Units != 2_650_000 || made.Pay.Amount != "2.65" || made.ClaimBase != "https://wire.example.test/usdt.html" || !strings.HasPrefix(made.Pay.Link, "ethereum:0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359@137/") {
		t.Fatalf("the card the window shows: %s", w.Body)
	}
	if png, err := base64.StdEncoding.DecodeString(made.Pay.QR); err != nil || len(png) < 100 || string(png[1:4]) != "PNG" {
		t.Fatal("the QR the window draws is a PNG of the payment link")
	}
	if len(made.SenderToken) < 20 || strings.Contains(w.Body.String(), f.api.db.state.USDT[cardID].SenderToken) {
		t.Fatal("Wire keeps the card's real sender token")
	}
	// the window asks for the state in camelCase
	w = f.call("POST", "/api/usdt/cashlink/status", map[string]string{"id": cardID})
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"state":"awaiting_funds"`) || !strings.Contains(w.Body.String(), `"balanceUnits"`) {
		t.Fatalf("status: %d %s", w.Code, w.Body)
	}
	// a card that is not this sender's, a bad request, an unknown route: PhoenixMail's {"error": …}
	other := hex.EncodeToString([]byte("0123456789abcdef"))
	if w := f.call("POST", "/api/usdt/cashlink/status", map[string]string{"id": other}); w.Code != 404 || !strings.Contains(w.Body.String(), `"error":"not_found"`) {
		t.Fatalf("someone else's card: %d %s", w.Code, w.Body)
	}
	if w := f.call("POST", "/api/usdt/cashlink/create", map[string]string{"asset": "doge", "cardId": "00", "linkOwner": "x", "amount": "1", "recipient": "x"}); w.Code != 400 || !strings.Contains(w.Body.String(), `"error":`) {
		t.Fatalf("a bad card: %d %s", w.Code, w.Body)
	}
	if w := f.call("POST", "/api/usdt/cashlink/gas", map[string]string{"id": cardID, "senderToken": made.SenderToken, "senderAddress": owner}); w.Code != 503 {
		t.Fatalf("free gas is not offered: %d", w.Code)
	}
	if w := f.call("POST", "/api/usdt/cashlink/nothing", map[string]string{}); w.Code != 404 {
		t.Fatalf("an unknown route: %d", w.Code)
	}
}

// A relayer below its gas reserve takes no new card: the window gets a sentence in the sender's language, not a code it cannot read.
func TestPaymentWindowSaysWhenTheRelayerIsNotReady(t *testing.T) {
	node := &evmNode{balance: word(0), block: 5000, native: "0x0"}
	rpc := node.serve()
	defer rpc.Close()
	t.Setenv("NIMIQ_CLAIM_USDTLINK_CHAINS", "polygon-usdc")
	t.Setenv("NIMIQ_CLAIM_EVM_RPC_POLYGON", rpc.URL)
	t.Setenv("NIMIQ_CLAIM_ALLOWED_ORIGINS", "https://wire.example.test")
	f := newFixture(t)
	f.verifySender("sender@example.test")
	id, _, owner, err := service.NewCardLink()
	if err != nil {
		t.Fatal(err)
	}
	for lang, want := range map[string]string{"fr": "réserve de gaz", "en": "gas reserve", "zh": "燃料储备", "xx": "gas reserve"} {
		w := f.call("POST", "/api/usdt/cashlink/create", map[string]string{"asset": "usdc_polygon", "cardId": hex.EncodeToString(id), "linkOwner": owner, "amount": "2.5", "recipient": "friend@example.test", "lang": lang})
		var out struct{ Error string }
		if w.Code != 503 || json.Unmarshal(w.Body.Bytes(), &out) != nil || !strings.Contains(out.Error, want) || strings.Contains(out.Error, "relayer_float_low") {
			t.Fatalf("%s: %d %s", lang, w.Code, w.Body)
		}
	}
}
