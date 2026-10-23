package wire

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/service"
)

// A fake EVM node: Polygon, a card whose balance the test sets, no transaction ever.
type evmNode struct {
	mu      sync.Mutex
	balance string // balanceOf answer, hex without 0x, 64 digits
	native  string // the relayer's native coin, hex with 0x (1 coin when empty)
	block   int
}

func (n *evmNode) serve() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.mu.Lock()
		defer n.mu.Unlock()
		var in struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		json.NewDecoder(r.Body).Decode(&in)
		var result any
		switch in.Method {
		case "eth_chainId":
			result = "0x89"
		case "eth_blockNumber":
			result = "0x" + strings.TrimLeft(hex.EncodeToString([]byte{byte(n.block >> 8), byte(n.block)}), "0")
		case "eth_call":
			var call struct{ Data string }
			json.Unmarshal(in.Params[0], &call)
			result = "0x" + strings.Repeat("0", 64)
			if strings.HasPrefix(call.Data, "0x70a08231") { // balanceOf
				result = "0x" + n.balance
			}
		case "eth_getBalance":
			result = "0xde0b6b3a7640000"
			if n.native != "" {
				result = n.native
			}
		case "eth_getCode":
			result = "0x"
		case "eth_getBlockByNumber":
			result = map[string]string{"baseFeePerGas": "0x3b9aca00"}
		case "eth_maxPriorityFeePerGas", "eth_gasPrice":
			result = "0x3b9aca00"
		case "eth_getTransactionReceipt":
			result = nil
		default:
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": in.ID, "error": map[string]any{"code": -32601, "message": "no " + in.Method}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": in.ID, "result": result})
	}))
}

func word(n uint64) string {
	return strings.Repeat("0", 64-len(hex.EncodeToString(bigEndian(n)))) + hex.EncodeToString(bigEndian(n))
}
func bigEndian(n uint64) []byte {
	var b []byte
	for ; n > 0; n >>= 8 {
		b = append([]byte{byte(n)}, b...)
	}
	if len(b) == 0 {
		b = []byte{0}
	}
	return b
}

func TestUSDCCardFromTheSenderToTheInvitation(t *testing.T) {
	node := &evmNode{balance: word(0), block: 5000}
	rpc := node.serve()
	defer rpc.Close()
	t.Setenv("NIMIQ_CLAIM_USDTLINK_CHAINS", "polygon-usdc")
	t.Setenv("NIMIQ_CLAIM_EVM_RPC_POLYGON", rpc.URL)
	t.Setenv("NIMIQ_CLAIM_ALLOWED_ORIGINS", "https://wire.example.test")
	f := newFixture(t)
	if f.api.USDT() == nil {
		t.Fatal("the USDC engine did not start")
	}
	// nobody without a verified sender e-mail
	if w := f.call("POST", "/api/usdt/create", map[string]string{}); w.Code != 401 {
		t.Fatalf("an unverified visitor created a card: %d", w.Code)
	}
	f.verifySender("sender@example.test")
	id, key, owner, err := service.NewCardLink()
	if err != nil {
		t.Fatal(err)
	}
	cardID := hex.EncodeToString(id)
	input := map[string]string{"asset": "usdc_polygon", "card_id": cardID, "link_owner": owner, "amount": "2.5", "recipient": "friend@example.test", "message": "test <b>", "language": "fr"}
	// bad inputs
	for name, mutate := range map[string]func(map[string]string){
		"another asset": func(m map[string]string) { m["asset"] = "doge" },
		"short id":      func(m map[string]string) { m["card_id"] = "00" },
		"bad owner":     func(m map[string]string) { m["link_owner"] = "0x12" },
		"bad e-mail":    func(m map[string]string) { m["recipient"] = "nobody" },
		"line break":    func(m map[string]string) { m["message"] = "a\nb" },
	} {
		bad := map[string]string{}
		for k, v := range input {
			bad[k] = v
		}
		mutate(bad)
		if w := f.call("POST", "/api/usdt/create", bad); w.Code != 400 {
			t.Errorf("%s: %d %s", name, w.Code, w.Body)
		}
	}
	w := f.call("POST", "/api/usdt/create", input)
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var made struct {
		Card struct {
			ID                              string `json:"id"`
			Safe                            string `json:"safe"`
			Units, FeeUnits, RefundFeeUnits uint64
		} `json:"card"`
		Pay struct {
			Link   string `json:"link"`
			Units  uint64 `json:"units"`
			Amount string `json:"amount"`
		} `json:"pay"`
	}
	json.Unmarshal(w.Body.Bytes(), &made)
	if made.Card.ID != cardID || made.Card.Units != 2_500_000 || made.Card.FeeUnits != 150_000 || made.Card.RefundFeeUnits != 100_000 || made.Pay.Units != 2_650_000 || made.Pay.Amount != "2.65" ||
		!strings.HasPrefix(made.Pay.Link, "ethereum:0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359@137/transfer?address="+made.Card.Safe) {
		t.Fatalf("created card: %s", w.Body)
	}
	if strings.Contains(w.Body.String(), "sender_token") || strings.Contains(w.Body.String(), "senderToken") {
		t.Fatal("the sender's token must stay in Wire")
	}
	if again := f.call("POST", "/api/usdt/create", input); again.Code != 409 {
		t.Fatalf("the same card id twice: %d", again.Code)
	}
	claim := "https://wire.example.test/usdt.html?lang=fr" + service.ClaimFragment(id, key)
	card := "/api/usdt/card/" + cardID
	// not funded: the recipient is not told
	if w := f.call("POST", card+"/invite", map[string]string{"claim_url": claim}); w.Code != 409 || !strings.Contains(w.Body.String(), "card_not_funded") {
		t.Fatalf("invitation before the money: %d %s", w.Code, w.Body)
	}
	if f.mail.invites != 0 {
		t.Fatal("an e-mail left before the card was funded")
	}
	// the payment arrives
	node.mu.Lock()
	node.balance = word(2_650_000)
	node.mu.Unlock()
	if w := f.call("POST", card+"/status", map[string]string{}); w.Code != 200 || !strings.Contains(w.Body.String(), `"state":"ready"`) {
		t.Fatalf("status after the payment: %d %s", w.Code, w.Body)
	}
	// a link that is not this card's is refused
	other, otherKey, _, _ := service.NewCardLink()
	for name, bad := range map[string]string{
		"another card": "https://wire.example.test/usdt.html" + service.ClaimFragment(other, otherKey),
		"another site": strings.Replace(claim, "wire.example.test", "evil.example", 1),
		"no key":       "https://wire.example.test/usdt.html#c=" + strings.SplitN(strings.SplitN(claim, "#c=", 2)[1], ".", 2)[0],
	} {
		if w := f.call("POST", card+"/invite", map[string]string{"claim_url": bad}); w.Code != 400 {
			t.Errorf("%s: %d", name, w.Code)
		}
	}
	if f.mail.invites != 0 {
		t.Fatal("an e-mail left for a link that is not the card's")
	}
	if w := f.call("POST", card+"/invite", map[string]string{"claim_url": claim}); w.Code != 200 {
		t.Fatalf("invitation: %d %s", w.Code, w.Body)
	}
	if f.mail.invites != 1 || f.mail.lastLink != claim {
		t.Fatalf("the invitation: %d %q", f.mail.invites, f.mail.lastLink)
	}
	if w := f.call("POST", card+"/invite", map[string]string{"claim_url": claim}); w.Code != 429 {
		t.Fatalf("a second invitation at once: %d", w.Code)
	}
	// the claim link is never kept: not in the state file
	f.restart()
	raw, _ := json.Marshal(f.api.db.state.USDT)
	if strings.Contains(string(raw), claim) || strings.Contains(string(raw), hex.EncodeToString(key)) {
		t.Fatal("the claim link was written to the state")
	}
	// ownership: another verified sender cannot see or act on the card
	f.cookie = nil
	f.verifySender("intruder@example.test")
	if w := f.call("POST", card+"/status", map[string]string{}); w.Code != 404 {
		t.Fatalf("another sender reached the card: %d", w.Code)
	}
	if w := f.call("GET", "/api/usdt/cards", nil); w.Code != 200 || strings.Contains(w.Body.String(), cardID) {
		t.Fatalf("another sender sees the card: %s", w.Body)
	}
}
