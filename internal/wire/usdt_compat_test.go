package wire

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/service"
)

// PhoenixMail's payment window is served as it is: it talks to /api/usdt/cashlink/* and /api/usdt/status in PhoenixMail's shape.
func TestPaymentWindowRoutes(t *testing.T) {
	node := &evmNode{balance: word(0), block: 5000}
	rpc := node.serve()
	defer rpc.Close()
	t.Setenv("NIMIQ_CLAIM_USDTLINK_CHAINS", "polygon-usdc")
	t.Setenv("NIMIQ_CLAIM_USDTLINK_COINGECKO", "off") // tests never ask the internet for a price
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
	if json.Unmarshal(w.Body.Bytes(), &made) != nil || made.Card.ID != cardID || len(made.Card.Owners) != 2 || made.Card.ChainID != 137 || made.Card.Units != 2_500_000 || made.Card.FeeUnits <= 150_000 || made.Card.FeeUnits > 400_000 || made.Card.RefundFeeUnits != made.Card.FeeUnits ||
		made.Pay.Recipient != made.Card.Safe || made.Pay.Units != 2_500_000+made.Card.FeeUnits || made.Pay.Amount != formatUnits(made.Pay.Units) || made.ClaimBase != "https://wire.example.test/usdt.html" || !strings.HasPrefix(made.Pay.Link, "ethereum:0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359@137/") {
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
	t.Setenv("NIMIQ_CLAIM_USDTLINK_COINGECKO", "off") // tests never ask the internet for a price
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

// The sender's gas: Wire's relayer gives a few cents of native coin for the one payment, once per card, when that is switched on.
func TestPaymentWindowGasForTheSender(t *testing.T) {
	node := &evmNode{balance: word(0), block: 5000}
	rpc := node.serve()
	defer rpc.Close()
	t.Setenv("NIMIQ_CLAIM_USDTLINK_CHAINS", "polygon-usdc")
	t.Setenv("NIMIQ_CLAIM_USDTLINK_COINGECKO", "off") // tests never ask the internet for a price
	t.Setenv("NIMIQ_CLAIM_EVM_RPC_POLYGON", rpc.URL)
	t.Setenv("NIMIQ_CLAIM_ALLOWED_ORIGINS", "https://wire.example.test")
	t.Setenv("NIMIQ_CLAIM_USDTLINK_GAS", "on")
	f := newFixture(t)
	f.verifySender("sender@example.test")
	var capability struct{ Gas bool }
	if w := f.call("GET", "/api/usdt/status", nil); json.Unmarshal(w.Body.Bytes(), &capability) != nil || !capability.Gas {
		t.Fatalf("the window shows its gas light only when Wire gives gas: %s", f.call("GET", "/api/usdt/status", nil).Body)
	}
	if w := f.call("POST", "/api/usdt/cashlink/gas-status", map[string]string{"asset": "usdc_polygon"}); !strings.Contains(w.Body.String(), `"available":true`) || !strings.Contains(w.Body.String(), `"reason":"ok"`) {
		t.Fatalf("green when the relayer can pay: %s", w.Body)
	}
	if w := f.call("POST", "/api/usdt/cashlink/gas-status", map[string]string{"asset": "doge"}); w.Code != 400 {
		t.Fatalf("an unknown asset: %d", w.Code)
	}
	id, _, owner, err := service.NewCardLink()
	if err != nil {
		t.Fatal(err)
	}
	cardID := hex.EncodeToString(id)
	if w := f.call("POST", "/api/usdt/cashlink/create", map[string]string{"asset": "usdc_polygon", "cardId": cardID, "linkOwner": owner, "amount": "2.5", "recipient": "friend@example.test", "lang": "en"}); w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	sender := "0x" + strings.Repeat("12", 20)
	node.mu.Lock()
	node.poor = sender
	node.mu.Unlock()
	// not another sender's card, not a bad address
	if w := f.call("POST", "/api/usdt/cashlink/gas", map[string]string{"id": hex.EncodeToString([]byte("0123456789abcdef")), "senderAddress": sender}); w.Code != 404 {
		t.Fatalf("someone else's card: %d", w.Code)
	}
	if w := f.call("POST", "/api/usdt/cashlink/gas", map[string]string{"id": cardID, "senderAddress": "nope"}); w.Code != 400 {
		t.Fatalf("a bad address: %d", w.Code)
	}
	w := f.call("POST", "/api/usdt/cashlink/gas", map[string]string{"id": cardID, "senderAddress": sender})
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"tx"`) || node.sent != 1 {
		t.Fatalf("the gas of the sender's payment: %d %s (sent %d)", w.Code, w.Body, node.sent)
	}
	// once per card: asking again gives the same answer and sends nothing more
	if again := f.call("POST", "/api/usdt/cashlink/gas", map[string]string{"id": cardID, "senderAddress": sender}); again.Code != 200 || !strings.Contains(again.Body.String(), `"already":true`) || node.sent != 1 {
		t.Fatalf("once per card: %d %s (sent %d)", again.Code, again.Body, node.sent)
	}
	// a relayer below its floor: red, with the reason
	node.mu.Lock()
	node.native = "0x0"
	node.mu.Unlock()
	if w := f.call("POST", "/api/usdt/cashlink/gas-status", map[string]string{"asset": "usdc_polygon"}); !strings.Contains(w.Body.String(), `"available":false`) || !strings.Contains(w.Body.String(), `"reason":"float"`) {
		t.Fatalf("red when the relayer is low: %s", w.Body)
	}
}

// The light is public (shown on the sign-in screen), cheap (remembered for a moment) and says nothing private.
func TestGasLightIsPublic(t *testing.T) {
	node := &evmNode{balance: word(0), block: 5000}
	rpc := node.serve()
	defer rpc.Close()
	t.Setenv("NIMIQ_CLAIM_USDTLINK_CHAINS", "polygon-usdc")
	t.Setenv("NIMIQ_CLAIM_USDTLINK_COINGECKO", "off") // tests never ask the internet for a price
	t.Setenv("NIMIQ_CLAIM_EVM_RPC_POLYGON", rpc.URL)
	t.Setenv("NIMIQ_CLAIM_ALLOWED_ORIGINS", "https://wire.example.test")
	t.Setenv("NIMIQ_CLAIM_USDTLINK_GAS", "on")
	f := newFixture(t) // nobody is signed in
	read := func() (polygon, ethereum struct {
		Available bool
		Reason    string
	}) {
		t.Helper()
		w := f.call("GET", "/api/gas", nil)
		var out struct {
			Polygon, Ethereum struct {
				Available bool
				Reason    string
			}
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatalf("/api/gas: %d %s", w.Code, w.Body)
		}
		return out.Polygon, out.Ethereum
	}
	if polygon, ethereum := read(); !polygon.Available || polygon.Reason != "ok" || ethereum.Available || ethereum.Reason != "off" {
		t.Fatalf("green for Polygon, nothing for a chain that is not running: %+v %+v", polygon, ethereum)
	}
	// remembered for a moment: the chain is not asked again, then it is
	node.mu.Lock()
	node.native = "0x0"
	node.mu.Unlock()
	if polygon, _ := read(); !polygon.Available {
		t.Fatal("the light is remembered for a short while")
	}
	later := f.api.now().Add(gasLightTTL + time.Second)
	f.api.now = func() time.Time { return later }
	if polygon, _ := read(); polygon.Available || polygon.Reason != "float" {
		t.Fatalf("then it follows the relayer: %+v", polygon)
	}
	if strings.Contains(f.call("GET", "/api/gas", nil).Body.String(), "0x") {
		t.Fatal("nothing private in the public light")
	}
}
