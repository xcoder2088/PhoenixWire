package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evmcard"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evmtest"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
)

const (
	usdtRecipient = "0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb"
	usdtSender    = "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"
)

type evmRig struct {
	s     *server
	chain *evmtest.Chain
	mail  *testSender
	net   evm.Network
}

func newEVMRig(t *testing.T, durableOTP bool) *evmRig {
	t.Helper()
	mail := &testSender{}
	s, err := newServer(mail, "", false)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "state")
	os.MkdirAll(dir, 0700)
	if durableOTP {
		if s.otp, err = otp.NewPersistent(mail, dir, bytes.Repeat([]byte{0x41}, 32)); err != nil {
			t.Fatal(err)
		}
	}
	svc, err := evmcard.New(filepath.Join(dir, "evm"))
	if err != nil {
		t.Fatal(err)
	}
	n, _ := evm.Lookup(evm.AssetUSDTPolygon)
	chain := evmtest.NewChain(t, n.ChainID)
	srv := chain.Server()
	t.Cleanup(srv.Close)
	svc.UseClients(func(net evm.Network) *evm.Client {
		net.RPCs = []string{srv.URL, srv.URL}
		c := evm.NewClient(net)
		c.AllowHTTPForTests()
		return c
	})
	s.evm = svc
	s.creationKey = strings.Repeat("k", 64)
	return &evmRig{s: s, chain: chain, mail: mail, net: n}
}

func (r *evmRig) do(method, path, body string, issuer bool) (int, map[string]any) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if issuer {
		req.Header.Set("X-Claim-Service-Key", r.s.creationKey)
	}
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/claims/", r.s.handler)
	mux.HandleFunc("/api/v1/evm/cards", r.s.evmIssuer)
	mux.HandleFunc("/api/v1/evm/cards/", r.s.evmIssuer)
	mux.ServeHTTP(w, req)
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestUSDTIssuerRoutesRequireTheIssuerKey(t *testing.T) {
	r := newEVMRig(t, true)
	for _, path := range []string{"/api/v1/evm/cards", "/api/v1/evm/cards/" + strings.Repeat("a", 48) + "?action=status", "/api/v1/evm/cards/" + strings.Repeat("a", 48) + "?action=cancel"} {
		if code, _ := r.do("POST", path, `{}`, false); code != 401 {
			t.Fatalf("%s without key: %d", path, code)
		}
	}
	if code, _ := r.do("GET", "/api/v1/evm/cards", "", true); code != 405 {
		t.Fatalf("GET: %d", code)
	}
	if code, out := r.do("POST", "/api/v1/evm/cards", `{"asset":"dai_polygon","amount":"1"}`, true); code != 400 || out["error"] != "unknown_asset" {
		t.Fatalf("unknown asset: %d %v", code, out)
	}
	if code, _ := r.do("POST", "/api/v1/evm/cards", `{"asset":"usdt_polygon","amount":"1","protected":true,"email":"not-an-email"}`, true); code != 400 {
		t.Fatalf("bad email: %d", code)
	}
	noOTP := newEVMRig(t, false)
	if code, out := noOTP.do("POST", "/api/v1/evm/cards", `{"asset":"usdt_polygon","amount":"1","protected":true,"email":"a@b.co"}`, true); code != 503 {
		t.Fatalf("a protected card without durable OTP must be refused: %d %v", code, out)
	}
	if code, _ := noOTP.do("POST", "/api/v1/evm/cards", `{"asset":"usdt_polygon","amount":"1"}`, true); code != 201 {
		t.Fatalf("an open card does not need the email code: %d", code)
	}
}

func TestUSDTProtectedCardEndToEnd(t *testing.T) {
	r := newEVMRig(t, true)
	code, created := r.do("POST", "/api/v1/evm/cards", `{"asset":"usdt_polygon","amount":"7.5","protected":true,"email":"friend@example.com","message":"cadeau","sender_address":"`+usdtSender+`","sender_email":"me@example.com","sender_lang":"fr"}`, true)
	if code != 201 {
		t.Fatal(code, created)
	}
	id := created["id"].(string)
	if created["state"] != "awaiting_address" || created["sender_token"] == "" || created["recipient"] != nil {
		t.Fatalf("created: %v", created)
	}
	// The public view shows no email, no address, no sender token.
	_, pub := r.do("GET", "/api/v1/claims/"+id, "", false)
	raw, _ := json.Marshal(pub)
	for _, secret := range []string{"friend@example.com", "me@example.com", created["sender_token"].(string), usdtSender} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("public view leaks %q: %s", secret, raw)
		}
	}
	if pub["asset"] != "usdt_polygon" || pub["amount"] != "7.5" || pub["protected"] != true || pub["email_verified"] != false || pub["has_recipient"] != false {
		t.Fatalf("public: %v", pub)
	}
	// Without the code no address is accepted.
	if code, out := r.do("POST", "/api/v1/claims/"+id+"?action=address", `{"to":"`+usdtRecipient+`"}`, false); code != 403 || out["error"] != "email_not_verified" {
		t.Fatalf("address without code: %d %v", code, out)
	}
	if code, _ := r.do("POST", "/api/v1/claims/"+id+"?action=request", `{"lang":"fr"}`, false); code != 200 || r.mail.to != "friend@example.com" || len(r.mail.code) != 6 {
		t.Fatalf("code request: %d to=%s", code, r.mail.to)
	}
	if code, out := r.do("POST", "/api/v1/claims/"+id+"?action=release", `{"code":"000000"}`, false); code != 403 {
		t.Fatalf("wrong code: %d %v", code, out)
	}
	if code, out := r.do("POST", "/api/v1/claims/"+id+"?action=release", `{"code":"`+r.mail.code+`"}`, false); code != 200 || out["status"] != "otp_verified" {
		t.Fatalf("right code: %d %v", code, out)
	}
	if code, out := r.do("POST", "/api/v1/claims/"+id+"?action=address", `{"to":"not an address"}`, false); code != 400 || out["error"] != "invalid_address" {
		t.Fatalf("bad address: %d %v", code, out)
	}
	if code, out := r.do("POST", "/api/v1/claims/"+id+"?action=address", `{"to":"`+usdtSender+`"}`, false); code != 400 || out["error"] != "same_address" {
		t.Fatalf("the sender's own address: %d %v", code, out)
	}
	code, out := r.do("POST", "/api/v1/claims/"+id+"?action=address", `{"to":"`+usdtRecipient+`"}`, false)
	if code != 200 || out["state"] != "ready" || out["has_recipient"] != true || out["recipient"] != nil {
		t.Fatalf("address: %d %v", code, out)
	}
	if code, out := r.do("POST", "/api/v1/claims/"+id+"?action=address", `{"to":"`+usdtRecipient+`"}`, false); code != 409 || out["error"] != "address_already_given" {
		t.Fatalf("second address: %d %v", code, out)
	}
	// The sender's side now sees where to pay.
	_, st := r.do("POST", "/api/v1/evm/cards/"+id+"?action=status", `{}`, true)
	if st["state"] != "ready" || st["recipient"] != usdtRecipient || st["units"].(float64) != 7_500_000 || st["token"] != r.net.Token || st["chain_id"].(float64) != 137 {
		t.Fatalf("issuer status: %v", st)
	}
	// The payment is seen on the chain, on both sides.
	hash := r.chain.Transfer(r.chain.Head()+1, usdtSender, usdtRecipient, 7_500_000)
	r.chain.Mine(100)
	if _, p := r.do("GET", "/api/v1/claims/"+id, "", false); p["state"] != "paid" || p["tx_hash"] != hash || !strings.HasSuffix(p["tx_url"].(string), hash) {
		t.Fatalf("recipient's page: %v", p)
	}
	if _, st := r.do("POST", "/api/v1/evm/cards/"+id+"?action=status", `{}`, true); st["state"] != "paid" {
		t.Fatalf("sender's status: %v", st)
	}
	if code, _ := r.do("POST", "/api/v1/evm/cards/"+id+"?action=cancel", `{"sender_token":"`+created["sender_token"].(string)+`"}`, true); code != 403 {
		t.Fatalf("a paid card cannot be cancelled: %d", code)
	}
}

func TestUSDTCancelAndOpenCard(t *testing.T) {
	r := newEVMRig(t, false)
	_, created := r.do("POST", "/api/v1/evm/cards", `{"asset":"usdt_polygon","amount":"2"}`, true)
	id := created["id"].(string)
	if code, _ := r.do("POST", "/api/v1/evm/cards/"+id+"?action=cancel", `{"sender_token":"wrong"}`, true); code != 403 {
		t.Fatalf("wrong token: %d", code)
	}
	if code, out := r.do("POST", "/api/v1/evm/cards/"+id+"?action=cancel", `{"sender_token":"`+created["sender_token"].(string)+`"}`, true); code != 200 || out["state"] != "cancelled" {
		t.Fatalf("cancel: %d %v", code, out)
	}
	if code, out := r.do("POST", "/api/v1/claims/"+id+"?action=address", `{"to":"`+usdtRecipient+`"}`, false); code != 410 || out["error"] != "card_cancelled" {
		t.Fatalf("address on a cancelled card: %d %v", code, out)
	}
	// An open card takes the address of whoever holds the link, without any code.
	_, c2 := r.do("POST", "/api/v1/evm/cards", `{"asset":"usdt_polygon","amount":"1"}`, true)
	if code, out := r.do("POST", "/api/v1/claims/"+c2["id"].(string)+"?action=address", `{"to":"`+usdtRecipient+`"}`, false); code != 200 {
		t.Fatalf("open card: %d %v", code, out)
	}
}
