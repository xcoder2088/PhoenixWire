package wire

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/service"
	"github.com/xcoder2088/PhoenixWire/internal/cards"
)

// Explicit fixtures: no real email, chain transaction or production credentials.
type testDelivery struct {
	mu       sync.Mutex
	codes    map[string]string
	invites  int
	notices  []string // "<sender>|<kind>" of every notice sent
	fail     bool
	lastLink string
	lastArt  string
}

func (m *testDelivery) Send(_ context.Context, to, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[to] = code
	return nil
}
func (m *testDelivery) Notice(_ context.Context, to string, n cards.Notice, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notices = append(m.notices, to+"|"+n.Kind)
	return nil
}
func (m *testDelivery) Invite(_ context.Context, p payment, link string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invites++
	m.lastLink = link
	m.lastArt = p.Nimicon
	if m.fail {
		return errors.New("fixture_delivery_failure")
	}
	return nil
}

func (m *testDelivery) InviteUSDT(_ context.Context, i usdtInvite, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invites++
	m.lastLink = i.Card.ClaimURL
	if m.fail {
		return errors.New("fixture_delivery_failure")
	}
	return nil
}

type fixture struct {
	api    *API
	engine *service.Engine
	mail   *testDelivery
	rpc    *httptest.Server
	root   string
	funded bool
	rpcMu  sync.Mutex
	sent   []string // raw transactions the node was handed
	mined  int      // confirmations the node reports for them
	cookie *http.Cookie
	t      *testing.T
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	t.Setenv("PHOENIXWIRE_SENDERS", "open") // the tests sign in new senders; the invitation limit has its own test
	f := &fixture{t: t, root: t.TempDir(), mail: &testDelivery{codes: map[string]string{}}}
	os.Chmod(f.root, 0700)
	f.rpc = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.rpcMu.Lock()
		defer f.rpcMu.Unlock()
		var in struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		json.NewDecoder(r.Body).Decode(&in)
		var data any
		switch in.Method {
		case "getBlockNumber":
			data = 100
		case "getBlockByNumber":
			data = map[string]string{"network": "MainAlbatross"}
		case "getAccountByAddress":
			balance := 0
			if f.funded {
				balance = 250000
			}
			data = map[string]int{"balance": balance}
		case "getTransactionsByAddress":
			var address string
			json.Unmarshal(in.Params[0], &address)
			data = []map[string]any{{"from": "NQ28 CQ7G QEBT 10UK R9LG 83C7 YAMY JD8Q K8S4", "to": address, "value": 250000, "recipientData": "0082809287", "confirmations": 2, "executionResult": true}}
		case "getRawTransactionInfo", "sendRawTransaction":
			var raw string
			json.Unmarshal(in.Params[0], &raw)
			sum := sha256.Sum256([]byte(raw))
			hash := hex.EncodeToString(sum[:])
			if in.Method == "getRawTransactionInfo" {
				data = map[string]string{"hash": hash}
			} else {
				f.sent = append(f.sent, raw)
				data = hash
			}
		case "getTransactionByHash":
			var hash string
			json.Unmarshal(in.Params[0], &hash)
			for _, raw := range f.sent {
				if sum := sha256.Sum256([]byte(raw)); hex.EncodeToString(sum[:]) == hash {
					json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"data": map[string]any{"confirmations": f.mined, "executionResult": true}}})
					return
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": -32603, "message": "not found"}})
			return
		default:
			t.Errorf("unexpected RPC method %s", in.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"data": data}})
	}))
	f.open()
	t.Cleanup(func() {
		if f.api != nil {
			f.api.Close()
		}
		if f.engine != nil {
			f.engine.Close()
		}
		f.rpc.Close()
	})
	return f
}
func (f *fixture) open() {
	f.t.Helper()
	var err error
	f.engine, err = service.New(service.Config{Directory: filepath.Join(f.root, "engine"), RPCURL: f.rpc.URL, Master: bytes.Repeat([]byte{53}, 32), Sender: f.mail})
	if err != nil {
		f.t.Fatal(err)
	}
	f.api, err = New(Config{Directory: filepath.Join(f.root, "gateway"), Origin: "https://wire.example.test", Master: bytes.Repeat([]byte{42}, 32), Engine: f.engine, Mail: f.mail})
	if err != nil {
		f.t.Fatal(err)
	}
}
func (f *fixture) restart() { f.api.Close(); f.engine.Close(); f.api = nil; f.engine = nil; f.open() }
func (f *fixture) call(method, path string, body any) *httptest.ResponseRecorder {
	f.t.Helper()
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://wire.example.test")
	r.RemoteAddr = "127.0.0.1:1234"
	if f.cookie != nil {
		r.AddCookie(f.cookie)
	}
	w := httptest.NewRecorder()
	f.api.Handler().ServeHTTP(w, r)
	for _, c := range w.Result().Cookies() {
		f.cookie = c
	}
	return w
}
func (f *fixture) verifySender(to string) {
	f.t.Helper()
	if w := f.call("POST", "/api/sender/request", map[string]string{"email": to}); w.Code != 200 {
		f.t.Fatal(w.Code, w.Body)
	}
	code := f.mail.codes[to]
	if w := f.call("POST", "/api/sender/verify", map[string]string{"code": code}); w.Code != 200 {
		f.t.Fatal(w.Code, w.Body)
	}
}
func (f *fixture) create() payment {
	f.t.Helper()
	w := f.call("POST", "/api/payments", paymentInput())
	if w.Code != 201 && w.Code != 200 {
		f.t.Fatal(w.Code, w.Body)
	}
	var data struct {
		Payment payment `json:"payment"`
	}
	json.Unmarshal(w.Body.Bytes(), &data)
	return data.Payment
}
func paymentInput() map[string]string {
	return map[string]string{"recipient": "recipient@example.test", "amount": "2.5", "currency": "NIM", "message": "Hello <world>", "request_id": strings.Repeat("a", 32), "language": "fr"}
}
func good(w *httptest.ResponseRecorder, t *testing.T) {
	t.Helper()
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
}

func TestCompleteWireFlowWithRealEngineAndRestart(t *testing.T) {
	f := newFixture(t)
	if w := f.call("POST", "/api/payments", paymentInput()); w.Code != 401 {
		t.Fatal("anonymous payment allowed")
	}
	good(f.call("POST", "/api/sender/request", map[string]string{"email": "sender@example.test"}), t)
	pending := f.cookie
	code := f.mail.codes["sender@example.test"]
	good(f.call("POST", "/api/sender/verify", map[string]string{"code": code}), t)
	if f.cookie.Value == pending.Value || !f.cookie.HttpOnly || !f.cookie.Secure || f.cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("session did not rotate with secure cookie")
	}
	owner := f.cookie
	f.cookie = pending
	if w := f.call("POST", "/api/sender/verify", map[string]string{"code": code}); w.Code != 403 {
		t.Fatal("sender OTP replay accepted")
	}
	f.cookie = owner
	p := f.create()
	art := testNimicon(t)
	if w := f.call("POST", "/api/payments/"+p.ID+"/art", map[string]string{"png": base64.StdEncoding.EncodeToString([]byte(`<svg onload="bad()"/>`))}); w.Code != 400 {
		t.Fatal("non-raster artwork accepted")
	}
	good(f.call("POST", "/api/payments/"+p.ID+"/art", map[string]string{"png": art}), t)
	ownerCookie := f.cookie
	f.cookie = nil
	if w := f.call("POST", "/api/payments/"+p.ID+"/art", map[string]string{"png": art}); w.Code != 401 {
		t.Fatal("anonymous artwork mutation")
	}
	f.cookie = ownerCookie
	same := f.create()
	if p.ID != same.ID || p.Address != same.Address {
		t.Fatal("idempotent create changed funding")
	}
	altered := paymentInput()
	altered["recipient"] = "other@example.test"
	if w := f.call("POST", "/api/payments", altered); w.Code != 409 {
		t.Fatal("immutable recipient changed")
	}
	if w := f.call("POST", "/api/claims/"+p.ID+"/request", map[string]string{}); w.Code != 404 {
		t.Fatal("provisioned but unfunded claim accessible")
	}
	started := f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{})
	good(started, t)
	for _, secret := range []string{"cashlink/#", "seed", "master"} {
		if strings.Contains(started.Body.String(), secret) {
			t.Fatal("checkout exposes spending material")
		}
	}
	if w := f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}); w.Code != 409 {
		t.Fatal("second wallet checkout allowed")
	}
	if w := f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}); w.Code != 409 {
		t.Fatal("unfunded payment confirmed")
	}
	if f.mail.invites != 0 {
		t.Fatal("card sent before chain confirmation")
	}
	f.restart()
	good(f.call("GET", "/api/session", nil), t)
	if f.api.db.state.Payments[p.ID].Nimicon == "" {
		t.Fatal("artwork lost on restart")
	}
	if w := f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}); w.Code != 409 {
		t.Fatal("restart permitted a second funding")
	}
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	good(f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}), t)
	good(f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}), t)
	if f.mail.invites != 1 || strings.Contains(f.mail.lastLink, "cashlink") || !strings.HasSuffix(f.mail.lastLink, "?claim="+p.ID) {
		t.Fatal("invalid card delivery")
	}
	if f.mail.lastArt == "" {
		t.Fatal("invitation has no persisted Nimicon")
	}
	public := f.call("GET", "/api/claims/"+p.ID, nil)
	good(public, t)
	if strings.Contains(public.Body.String(), "hub.nimiq") || strings.Contains(public.Body.String(), "recipient@example.test") {
		t.Fatal("public claim leaks private details")
	}
	good(f.call("POST", "/api/claims/"+p.ID+"/request", map[string]string{}), t)
	recipientCode := f.mail.codes["recipient@example.test"]
	f.restart()
	bad := "000000"
	if recipientCode == bad {
		bad = "111111"
	}
	denied := f.call("POST", "/api/claims/"+p.ID+"/release", map[string]string{"code": bad})
	if denied.Code != 403 || strings.Contains(denied.Body.String(), "cashlink/#") {
		t.Fatal("wrong OTP exposed link")
	}
	released := f.call("POST", "/api/claims/"+p.ID+"/release", map[string]string{"code": recipientCode})
	good(released, t)
	var answer map[string]string
	json.Unmarshal(released.Body.Bytes(), &answer)
	payload, err := base64.URLEncoding.DecodeString(strings.TrimPrefix(answer["cashlink"], "https://hub.nimiq.com/cashlink/#"))
	if err != nil || len(payload) != 40 {
		t.Fatal("invalid original Hub Cashlink")
	}
	clear(payload)
	if w := f.call("POST", "/api/claims/"+p.ID+"/release", map[string]string{"code": recipientCode}); w.Code != 409 {
		t.Fatal("recipient OTP replay accepted")
	}
	recovery := f.call("POST", "/api/payments/"+p.ID+"/recovery", map[string]string{})
	good(recovery, t)
	var recovered map[string]string
	json.Unmarshal(recovery.Body.Bytes(), &recovered)
	if recovered["cashlink"] != answer["cashlink"] {
		t.Fatal("recovery created a different payment")
	}
	f.cookie = nil
	if w := f.call("POST", "/api/payments/"+p.ID+"/recovery", map[string]string{}); w.Code != 401 {
		t.Fatal("anonymous recovery allowed")
	}
	raw, err := os.ReadFile(filepath.Join(f.root, "gateway", "wire-state.sealed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, plain := range []string{"sender@example.test", "recipient@example.test", code, recipientCode} {
		if bytes.Contains(raw, []byte(plain)) {
			t.Fatal("private state is plaintext")
		}
	}
}

func TestSenderAttemptLimitsSurviveRestartAndNewCookie(t *testing.T) {
	f := newFixture(t)
	good(f.call("POST", "/api/sender/request", map[string]string{"email": "sender@example.test"}), t)
	code := f.mail.codes["sender@example.test"]
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 5; i++ {
		if w := f.call("POST", "/api/sender/verify", map[string]string{"code": wrong}); w.Code != 403 {
			t.Fatal("wrong sender code accepted")
		}
	}
	f.restart()
	if w := f.call("POST", "/api/sender/verify", map[string]string{"code": code}); w.Code != 403 {
		t.Fatal("attempt limits lost")
	}
	f.cookie = nil
	if w := f.call("POST", "/api/sender/request", map[string]string{"email": "sender@example.test"}); w.Code != 429 {
		t.Fatal("new cookie bypassed resend limit")
	}
}
func TestOriginsAssetsAndOwnership(t *testing.T) {
	f := newFixture(t)
	r := httptest.NewRequest("POST", "/api/sender/request", strings.NewReader(`{"email":"sender@example.test"}`))
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.api.Handler().ServeHTTP(w, r)
	if w.Code != 403 || len(f.mail.codes) > 0 {
		t.Fatal("cross-site send allowed")
	}
	f.verifySender("sender@example.test")
	bad := paymentInput()
	bad["currency"] = "USDT_ETHEREUM"
	if w := f.call("POST", "/api/payments", bad); w.Code != 400 {
		t.Fatal("unimplemented asset enabled")
	}
	bad = paymentInput()
	bad["sender"] = "victim@example.test"
	if w := f.call("POST", "/api/payments", bad); w.Code != 400 {
		t.Fatal("forged sender accepted")
	}
	p := f.create()
	f.cookie = nil
	f.verifySender("other@example.test")
	if w := f.call("POST", "/api/payments/"+p.ID+"/recovery", map[string]string{}); w.Code != 404 {
		t.Fatal("another sender read recovery")
	}
}
func TestDeliveryFailureRetainsSameFundedPayment(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t)
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	f.mail.fail = true
	if w := f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}); w.Code != 502 {
		t.Fatal("delivery failure hidden")
	}
	f.restart()
	good(f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}), t)
	if f.mail.invites != 1 {
		t.Fatal("lost response automatically resent email")
	}
	f.api.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	f.mail.fail = false
	good(f.call("POST", "/api/payments/"+p.ID+"/resend", map[string]string{}), t)
	if f.mail.invites != 2 {
		t.Fatal("explicit resend failed")
	}
	if w := f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}); w.Code != 409 {
		t.Fatal("email retry caused second funding")
	}
}
func TestWrongMasterAndSecondProcessFailClosed(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	_, err := New(Config{Directory: filepath.Join(f.root, "gateway"), Origin: "https://wire.example.test", Master: bytes.Repeat([]byte{42}, 32), Engine: f.engine, Mail: f.mail})
	if err == nil {
		t.Fatal("second process accepted")
	}
	f.api.Close()
	f.api = nil
	_, err = New(Config{Directory: filepath.Join(f.root, "gateway"), Origin: "https://wire.example.test", Master: bytes.Repeat([]byte{43}, 32), Engine: f.engine, Mail: f.mail})
	if err == nil {
		t.Fatal("wrong master accepted")
	}
}

func TestExpiredPreparedClaimCannotStartButOwnerCanRecover(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	saved := f.api.db.state.Payments[p.ID]
	saved.Payment.Created = time.Now().Add(-25 * time.Hour)
	f.api.db.state.Payments[p.ID] = saved
	if w := f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}); w.Code != 410 {
		t.Fatal("expired claim offered new funding")
	}
	good(f.call("POST", "/api/payments/"+p.ID+"/recovery", map[string]string{}), t)
}

func TestRecipientCanHaveTheCardPaidToAnAddress(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t)
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	good(f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}), t)
	f.cookie = nil // the recipient is another person: no sender session
	to := "NQ07 0000 0000 0000 0000 0000 0000 0000 0000"
	if w := f.call("POST", "/api/claims/"+p.ID+"/payout", map[string]string{"to": "NQ28 CQ7G QEBT 10UK R9LG 83C7 YAMY JD8Q K8S4"}); w.Code != 403 {
		t.Fatal("a payout started without a code", w.Code)
	}
	if w := f.call("POST", "/api/claims/"+p.ID+"/payout_status", map[string]string{}); w.Code != 404 {
		t.Fatal("status of a payout nobody asked for", w.Code)
	}
	if w := f.call("POST", "/api/claims/"+p.ID+"/payout", map[string]string{"code": "123456", "to": strings.Repeat("N", 65)}); w.Code != 403 {
		t.Fatal("oversized destination reached the engine", w.Code)
	}
	good(f.call("POST", "/api/claims/"+p.ID+"/request", map[string]string{}), t)
	code := f.mail.codes["recipient@example.test"]
	if w := f.call("POST", "/api/claims/"+p.ID+"/payout", map[string]string{"code": code, "to": to}); w.Code != 400 {
		t.Fatal("the burn address was accepted", w.Code)
	}
	destination := "NQ28 CQ7G QEBT 10UK R9LG 83C7 YAMY JD8Q K8S4"
	w := f.call("POST", "/api/claims/"+p.ID+"/payout", map[string]string{"code": code, "to": destination, "extra": "ignored"})
	good(w, t)
	var started map[string]any
	json.Unmarshal(w.Body.Bytes(), &started)
	if started["status"] != "payout_pending" || started["value"] != float64(250000) || len(f.sent) != 1 {
		t.Fatal("payout did not start", started, len(f.sent))
	}
	if w := f.call("POST", "/api/claims/"+p.ID+"/release", map[string]string{"code": code}); w.Code != 409 {
		t.Fatal("the Hub link stayed open after a payout began", w.Code)
	}
	f.restart() // the journal survives a restart
	f.cookie = nil
	f.rpcMu.Lock()
	f.mined = 2
	f.rpcMu.Unlock()
	w = f.call("POST", "/api/claims/"+p.ID+"/payout_status", map[string]string{})
	good(w, t)
	var followed map[string]any
	json.Unmarshal(w.Body.Bytes(), &followed)
	if followed["status"] != "payout_confirmed" || followed["hash"] != started["hash"] || len(f.sent) != 1 {
		t.Fatal("payout not followed", followed)
	}
	if w := f.call("POST", "/api/claims/"+p.ID+"/payout", map[string]string{"to": "NQ24 0358 HL85 UQ05 SLSS C72R KEX3 NU2K RN0N"}); w.Code != 409 {
		t.Fatal("a second destination was accepted", w.Code)
	}
}

func TestRecipientCodeCrossesToTheAppOnceAndOnlyForItsCard(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t)
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	good(f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}), t)
	other := payment{ID: strings.Repeat("a", 48)} // a card that was never sent
	f.cookie = nil
	base := "/api/claims/" + p.ID + "/handoff"
	if w := f.call("POST", base, map[string]string{"code": "12ab56"}); w.Code != 400 {
		t.Fatal("a malformed code was taken", w.Code)
	}
	if w := f.call("POST", "/api/claims/"+other.ID+"/handoff", map[string]string{"code": "123456"}); w.Code != 404 {
		t.Fatal("a card that was never sent offered a hand-over", w.Code)
	}
	w := f.call("POST", base, map[string]string{"code": "123456"})
	good(w, t)
	var made struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &made)
	if len(made.Token) != 43 || strings.Contains(w.Body.String(), "123456") {
		t.Fatal("the answer must carry the token and never the code")
	}
	if w := f.call("POST", "/api/claims/"+other.ID+"/handoff/redeem", map[string]string{"token": made.Token}); w.Code != 404 {
		t.Fatal("a token opened another card", w.Code)
	}
	w = f.call("POST", base, map[string]string{"code": "654321"})
	json.Unmarshal(w.Body.Bytes(), &made)
	w = f.call("POST", base+"/redeem", map[string]string{"token": made.Token})
	good(w, t)
	if !strings.Contains(w.Body.String(), "654321") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("the app did not get the code", w.Body.String())
	}
	if w := f.call("POST", base+"/redeem", map[string]string{"token": made.Token}); w.Code != 404 {
		t.Fatal("token reused", w.Code)
	}
	// Five minutes, then nothing.
	json.Unmarshal(f.call("POST", base, map[string]string{"code": "111111"}).Body.Bytes(), &made)
	f.api.now = func() time.Time { return time.Now().Add(6 * time.Minute) }
	if w := f.call("POST", base+"/redeem", map[string]string{"token": made.Token}); w.Code != 404 {
		t.Fatal("an expired token worked", w.Code)
	}
	// Nothing about it is ever written to disk.
	raw, _ := os.ReadFile(filepath.Join(f.root, "gateway", "wire-state.sealed"))
	if bytes.Contains(raw, []byte(made.Token)) {
		t.Fatal("token on disk")
	}
}

func TestListSaysWhereACardsCoinsAre(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t)
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	good(f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}), t)
	state := func() string {
		w := f.call("GET", "/api/payments", nil)
		good(w, t)
		var out struct {
			Payments []payment `json:"payments"`
		}
		json.Unmarshal(w.Body.Bytes(), &out)
		return out.Payments[0].ClaimState
	}
	if state() != "" { // the first list only starts the look: it never waits for the chain
		t.Fatal("a state appeared before the chain was asked")
	}
	f.api.lookCardState(p.ID, time.Now())
	f.api.stateMu.Lock()
	for id, e := range f.api.cardStates { // let the wait pass without sleeping
		e.Looking = false
		f.api.cardStates[id] = e
	}
	f.api.stateMu.Unlock()
	if got := state(); got != "unclaimed" {
		t.Fatal("a funded card still holding its coins reads", got)
	}
	// The coins leave the card (claimed, or recovered): the next look says so, and it is final.
	f.rpcMu.Lock()
	f.funded = false
	f.rpcMu.Unlock()
	f.api.lookCardState(p.ID, time.Now())
	if got := state(); got != "claimed" {
		t.Fatal("an emptied card reads", got)
	}
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	f.api.lookCardState(p.ID, time.Now())
	if got := state(); got != "claimed" {
		t.Fatal("coins cannot come back to a card, but it reads", got)
	}
}

func TestSenderCanGiveUpACardThatWasNeverPaid(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t)
	// A payment that has already reached the card is not cancelled.
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	if w := f.call("POST", "/api/payments/"+p.ID+"/cancel", map[string]string{}); w.Code != 409 {
		t.Fatal("a funded card was cancelled", w.Code)
	}
	f.rpcMu.Lock()
	f.funded = false
	f.rpcMu.Unlock()
	good(f.call("POST", "/api/payments/"+p.ID+"/cancel", map[string]string{}), t)
	good(f.call("POST", "/api/payments/"+p.ID+"/cancel", map[string]string{}), t) // again: the same answer
	if f.api.db.state.Payments[p.ID].Payment.Stage != "cancelled" {
		t.Fatal("not cancelled")
	}
	// It can no longer be funded, followed to delivery or opened by a recipient.
	for _, action := range []string{"start", "checkout", "confirm", "resend"} {
		if w := f.call("POST", "/api/payments/"+p.ID+"/"+action, map[string]string{}); w.Code < 400 {
			t.Fatal(action, "worked on a cancelled card", w.Code)
		}
	}
	if w := f.call("GET", "/api/claims/"+p.ID, nil); w.Code != 404 {
		t.Fatal("a cancelled card can be opened", w.Code)
	}
	if f.mail.invites != 0 {
		t.Fatal("an invitation went out for a cancelled card")
	}
	// A payment that was still on its way shows up afterwards: the list says the card holds money again, so it can be taken back.
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	state := func() string {
		w := f.call("GET", "/api/payments", nil)
		var out struct {
			Payments []payment `json:"payments"`
		}
		json.Unmarshal(w.Body.Bytes(), &out)
		return out.Payments[0].ClaimState
	}
	f.api.lookCardState(p.ID, time.Now())
	state()
	f.api.stateMu.Lock()
	e := f.api.cardStates[p.ID]
	e.Cancelled, e.Looking = true, false
	f.api.cardStates[p.ID] = e
	f.api.stateMu.Unlock()
	f.api.lookCardState(p.ID, time.Now())
	if got := state(); got != "unclaimed" {
		t.Fatal("late money on a cancelled card reads", got)
	}
	// And an empty cancelled card is never "received".
	f.rpcMu.Lock()
	f.funded = false
	f.rpcMu.Unlock()
	f.api.lookCardState(p.ID, time.Now())
	if got := state(); got == "claimed" {
		t.Fatal("an unpaid cancelled card reads as received")
	}
	if w := f.call("POST", "/api/payments/"+p.ID+"/recovery", map[string]string{}); w.Code != 200 {
		t.Fatal("a cancelled card lost its recovery", w.Code)
	}
}

func TestOnlyRegisteredSendersCanSignIn(t *testing.T) {
	f := newFixture(t)
	f.verifySender("owner@example.test") // registered while the app was open: signs in, creates a card
	p := f.create()
	t.Setenv("PHOENIXWIRE_SENDERS", "") // now by invitation
	f.cookie = nil
	if !f.api.InviteOnly() {
		t.Fatal("the page must be told that sending is by invitation")
	}
	// a stranger gets no code and nothing is sent
	before := len(f.mail.codes)
	w := f.call("POST", "/api/sender/request", map[string]string{"email": "stranger@example.test"})
	if w.Code != 403 || !strings.Contains(w.Body.String(), "invite_only") || len(f.mail.codes) != before {
		t.Fatal("a stranger was let in", w.Code, w.Body)
	}
	// the registered sender still signs in (the rule is checked directly: the e-mail code of this sender was just sent, its own resend limit applies)
	if !f.api.senderAllowed("owner@example.test") || f.api.senderAllowed("stranger@example.test") {
		t.Fatal("registered senders are the ones who created a card")
	}
	// an address named by the operator is let in too
	t.Setenv("PHOENIXWIRE_SENDERS", "friend@example.test, other@example.test")
	if w := f.call("POST", "/api/sender/request", map[string]string{"email": "friend@example.test"}); w.Code != 200 {
		t.Fatal("a named sender was refused", w.Code, w.Body)
	}
	// recipients are not senders: a card that was sent opens for anyone, signed in or not
	f.cookie = nil
	if w := f.call("GET", "/api/claims/"+p.ID, nil); w.Code != 404 { // not funded in this test, so not claimable; the route itself must not ask for a sender
		t.Fatal("the claim route must stay public and answer by the card's own state", w.Code)
	}
	t.Setenv("PHOENIXWIRE_SENDERS", "open")
	if f.api.InviteOnly() {
		t.Fatal("open mode must say so")
	}
	if w := f.call("POST", "/api/sender/request", map[string]string{"email": "stranger@example.test"}); w.Code != 200 {
		t.Fatal("open mode refused a sender", w.Code)
	}
}

func TestAnExpiredNIMCardGoesBackToItsSenderByItself(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t)
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	good(f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}), t)
	ctx := context.Background()
	at := func(h time.Duration) { f.api.now = func() time.Time { return p.Created.Add(h) } }
	stage := func() string { return f.api.db.state.Payments[p.ID].Payment.Stage }
	// early on, nothing happens
	at(time.Hour)
	f.api.sweepExpiry(ctx)
	if len(f.mail.notices) != 0 || len(f.sent) != 0 {
		t.Fatal("a young card was touched", f.mail.notices, len(f.sent))
	}
	// an hour before the end: one reminder, once
	at(23*time.Hour + 5*time.Minute)
	f.api.sweepExpiry(ctx)
	f.api.sweepExpiry(ctx)
	if len(f.mail.notices) != 1 || f.mail.notices[0] != "sender@example.test|reminder" {
		t.Fatal("one reminder to the sender expected", f.mail.notices)
	}
	// the time is over: the money is paid back to the address that funded the card, once
	at(24*time.Hour + time.Minute)
	f.api.sweepExpiry(ctx)
	if stage() != "refunding" || len(f.sent) != 1 {
		t.Fatal("refund not started", stage(), len(f.sent))
	}
	f.rpcMu.Lock()
	f.mined = 2
	f.rpcMu.Unlock()
	f.api.sweepExpiry(ctx)
	f.api.sweepExpiry(ctx)
	if stage() != "refunded" || len(f.sent) < 1 {
		t.Fatal("refund not completed", stage(), len(f.sent))
	}
	if len(f.mail.notices) != 2 || f.mail.notices[1] != "sender@example.test|refunded" {
		t.Fatal("the sender must be told once", f.mail.notices)
	}
	// a refunded card cannot be opened any more
	f.cookie = nil
	if w := f.call("GET", "/api/claims/"+p.ID, nil); w.Code != 404 {
		t.Fatal("a refunded card can be opened", w.Code)
	}
}

func TestTheAdministratorOpensAndClosesAccess(t *testing.T) {
	t.Setenv("PHOENIXWIRE_ADMINS", "owner@example.test, boss@example.test")
	f := newFixture(t)
	f.verifySender("owner@example.test")
	f.verifySender("regular@example.test") // a registered sender who is not an administrator
	regular := f.cookie
	// the administrator's session says so; an ordinary sender's does not, and the panel's routes are closed to him
	if w := f.call("GET", "/api/session", nil); !strings.Contains(w.Body.String(), `"admin":true`) && !strings.Contains(w.Body.String(), `"admin": true`) {
		// the cookie is now the ordinary sender's: that session must not say admin
		if strings.Contains(w.Body.String(), "admin") {
			t.Fatal("an ordinary sender is told he is an administrator", w.Body)
		}
	}
	if w := f.call("GET", "/api/admin/access", nil); w.Code != 403 {
		t.Fatal("an ordinary sender reached the panel", w.Code)
	}
	if w := f.call("POST", "/api/admin/access", map[string]bool{"open": true}); w.Code != 403 {
		t.Fatal("an ordinary sender opened access", w.Code)
	}
	f.cookie = nil
	if w := f.call("GET", "/api/admin/access", nil); w.Code != 401 {
		t.Fatal("a visitor reached the panel", w.Code)
	}
	f.verifySender("boss@example.test") // an administrator signs in even though he has never sent anything
	if w := f.call("GET", "/api/session", nil); !strings.Contains(w.Body.String(), "admin") {
		t.Fatal("the administrator's session does not say so", w.Body)
	}
	t.Setenv("PHOENIXWIRE_SENDERS", "") // by invitation
	view := func(w *httptest.ResponseRecorder) map[string]any {
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	if v := view(f.call("GET", "/api/admin/access", nil)); v["open"] != false || v["registered_count"] == nil || len(v["registered"].([]any)) != 0 {
		t.Fatal("the panel", v)
	}
	enter := func(address string) int {
		saved := f.cookie
		f.cookie = nil
		defer func() { f.cookie = saved }()
		return f.call("POST", "/api/sender/request", map[string]string{"email": address}).Code
	}
	if enter("visitor1@example.test") != 403 {
		t.Fatal("a stranger entered while access was closed")
	}
	// open: a stranger enters; close again: another stranger does not; the administrator's own cookie is untouched
	if w := f.call("POST", "/api/admin/access", map[string]bool{"open": true}); w.Code != 200 || view(w)["open"] != true || f.api.InviteOnly() {
		t.Fatal("access was not opened", w.Code, w.Body)
	}
	if enter("visitor2@example.test") != 200 {
		t.Fatal("a stranger was refused while access was open")
	}
	if w := f.call("POST", "/api/admin/access", map[string]bool{"open": false}); w.Code != 200 || !f.api.InviteOnly() {
		t.Fatal("access was not closed", w.Code)
	}
	if enter("visitor3@example.test") != 403 {
		t.Fatal("a stranger entered after the close")
	}
	// allow one address
	if w := f.call("POST", "/api/admin/allow", map[string]string{"email": "Friend@Example.test"}); w.Code != 200 {
		t.Fatal("allow", w.Code, w.Body)
	}
	if enter("friend@example.test") != 200 {
		t.Fatal("an allowed address was refused")
	}
	// block a registered sender: refused at once, and his session ends
	if w := f.call("POST", "/api/admin/block", map[string]string{"email": "regular@example.test"}); w.Code != 200 {
		t.Fatal("block", w.Code, w.Body)
	}
	if enter("regular@example.test") != 403 {
		t.Fatal("a blocked sender signed in")
	}
	saved := f.cookie
	f.cookie = regular
	if w := f.call("GET", "/api/session", nil); strings.Contains(w.Body.String(), `"verified":true`) {
		t.Fatal("a blocked sender kept his session", w.Body)
	}
	f.cookie = saved
	// an administrator cannot be blocked; allowing lifts a block
	if w := f.call("POST", "/api/admin/block", map[string]string{"email": "owner@example.test"}); w.Code != 409 {
		t.Fatal("an administrator was blocked", w.Code)
	}
	if w := f.call("POST", "/api/admin/allow", map[string]string{"email": "regular@example.test"}); w.Code != 200 || enter("regular@example.test") == 403 {
		t.Fatal("a block was not lifted")
	}
	if w := f.call("POST", "/api/admin/allow", map[string]string{"email": "not an address"}); w.Code != 400 {
		t.Fatal("a bad address was accepted", w.Code)
	}
	// the access survives a restart
	f.restart()
	if v := f.api.db.state.Access; len(v.Allowed) == 0 {
		t.Fatal("the list was not kept", v)
	}
}

func TestOneSenderCannotFillTheStateAndAFullStateDoesNotStopEveryone(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	// a sender may create a limited number of NIM cards a day
	for i := 0; i < nimCardsPerDay; i++ {
		in := paymentInput()
		in["request_id"] = fmt.Sprintf("%032x", i+1)
		w := f.call("POST", "/api/payments", in)
		if w.Code != 200 && w.Code != 201 {
			t.Fatalf("card %d: %d %s", i, w.Code, w.Body)
		}
	}
	over := paymentInput()
	over["request_id"] = fmt.Sprintf("%032x", 999)
	if w := f.call("POST", "/api/payments", over); w.Code != 429 {
		t.Fatal("a sender created more than the daily limit", w.Code)
	}
	// the state nearly full: new cards and new artwork are refused where they come in, and the others keep working
	f.api.mu.Lock()
	big := strings.Repeat("A", 1024*1024)
	for i := 0; i < 7; i++ {
		id := fmt.Sprintf("%048x", 5000+i)
		f.api.db.state.Payments[id] = savedPayment{Payment: payment{ID: id, Sender: "ballast@example.test", Stage: "ready", Created: f.api.now()}, Nimicon: big}
	}
	f.api.mu.Unlock()
	f.cookie = nil
	f.verifySender("other@example.test")
	again := paymentInput()
	again["request_id"] = fmt.Sprintf("%032x", 4242)
	if w := f.call("POST", "/api/payments", again); w.Code != 429 || !strings.Contains(w.Body.String(), "payment_capacity") {
		t.Fatal("a card was taken in a full state", w.Code, w.Body)
	}
	if f.api.db.failed {
		t.Fatal("a full state locked the whole application")
	}
	if w := f.call("GET", "/api/session", nil); w.Code != 200 {
		t.Fatal("the application stopped answering", w.Code)
	}
	// old drafts nobody paid are forgotten, which gives the room back
	f.api.mu.Lock()
	for id, saved := range f.api.db.state.Payments {
		saved.Payment.Created = f.api.now().Add(-3 * 24 * time.Hour)
		f.api.db.state.Payments[id] = saved
	}
	f.api.pruneDrafts(f.api.now())
	left := len(f.api.db.state.Payments)
	f.api.mu.Unlock()
	if left != 0 {
		t.Fatal("unpaid drafts older than two days were kept", left)
	}
}

func TestAPaymentThatLandedWhileNobodyLookedStillEndsLikeAnyCard(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t) // the wallet is asked: the window is closed, nobody confirms
	ctx := context.Background()
	stage := func() string { return f.api.db.state.Payments[p.ID].Payment.Stage }
	f.api.now = func() time.Time { return p.Created.Add(time.Hour) }
	f.api.sweepExpiry(ctx)
	if stage() != "funding_started" {
		t.Fatal("nothing was paid yet", stage())
	}
	f.rpcMu.Lock()
	f.funded = true // the payment lands later
	f.rpcMu.Unlock()
	f.api.sweepExpiry(ctx)
	if stage() != "funded" || f.mail.invites != 0 {
		t.Fatal("a late payment must be known as funded, without any invitation", stage(), f.mail.invites)
	}
	// and a card cancelled and then paid is refunded at the end like the others
	q := f.create() // the same payment: create is idempotent per request
	_ = q
	f.api.now = func() time.Time { return p.Created.Add(25 * time.Hour) }
	f.api.sweepExpiry(ctx)
	if stage() != "refunding" || len(f.sent) != 1 {
		t.Fatal("the late-paid card was not refunded at the end", stage(), len(f.sent))
	}
}

func TestTheVisitorsAddressIsTheTunnelsOnlyFromThisMachine(t *testing.T) {
	req := func(remote, header string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		if header != "" {
			r.Header.Set("CF-Connecting-IP", header)
		}
		return r
	}
	if got := clientIP(req("127.0.0.1:5000", "203.0.113.9")); got != "203.0.113.9" {
		t.Fatal("behind the tunnel the edge's address counts", got)
	}
	if got := clientIP(req("198.51.100.4:5000", "203.0.113.9")); got != "198.51.100.4" {
		t.Fatal("a direct visitor cannot choose his address", got)
	}
	if got := clientIP(req("127.0.0.1:5000", "not an address")); got != "127.0.0.1" {
		t.Fatal("a bad header is ignored", got)
	}
}

func TestThePanelSaysWhenTheConfigurationForcesAccessOpen(t *testing.T) {
	t.Setenv("PHOENIXWIRE_ADMINS", "owner@example.test")
	f := newFixture(t) // the fixture opens access through the environment
	f.verifySender("owner@example.test")
	var v map[string]any
	w := f.call("GET", "/api/admin/access", nil)
	json.Unmarshal(w.Body.Bytes(), &v)
	if v["locked_open"] != true || v["open"] != true {
		t.Fatal("the panel must say the configuration keeps access open", v)
	}
	t.Setenv("PHOENIXWIRE_SENDERS", "")
	w = f.call("GET", "/api/admin/access", nil)
	json.Unmarshal(w.Body.Bytes(), &v)
	if v["locked_open"] != false {
		t.Fatal("not locked when the configuration does not force it", v)
	}
}

func TestPersonalDataGoesSixtyDaysAfterACardEndsAndOnlyWhenItHoldsNothing(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	other := paymentInput()
	other["request_id"] = fmt.Sprintf("%032x", 777)
	var made struct {
		Payment payment `json:"payment"`
	}
	w := f.call("POST", "/api/payments", other)
	if w.Code != 200 && w.Code != 201 {
		t.Fatal(w.Code, w.Body)
	}
	json.Unmarshal(w.Body.Bytes(), &made)
	q := made.Payment
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t)
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	good(f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}), t)
	ctx := context.Background()
	has := func() bool { _, ok := f.api.db.state.Payments[p.ID]; return ok }
	// sixty days after the end: still kept while the card holds money
	f.api.now = func() time.Time { return p.Created.Add(24*time.Hour + 61*24*time.Hour) }
	young := f.api.db.state.Payments[q.ID] // another card, made ten days ago from that moment: never touched
	young.Payment.Created = f.api.now().Add(-10 * 24 * time.Hour)
	f.api.db.state.Payments[q.ID] = young
	f.api.forgetOld(ctx, f.api.now())
	if !has() {
		t.Fatal("a card that still holds money was forgotten")
	}
	// the money is gone (claimed): the personal data goes
	f.rpcMu.Lock()
	f.funded = false
	f.rpcMu.Unlock()
	f.api.forgetOld(ctx, f.api.now())
	if has() {
		t.Fatal("an empty card past its retention kept its data")
	}
	if _, ok := f.api.db.state.Notices[p.ID]; ok {
		t.Fatal("notices of a forgotten card were kept")
	}
	// a young card is never touched, empty or not
	if _, ok := f.api.db.state.Payments[q.ID]; !ok {
		t.Fatal("a young card was forgotten")
	}
}

func TestTheAdministratorCanRevokeAndForgetAddressesAndOnlySeesHisOwnList(t *testing.T) {
	t.Setenv("PHOENIXWIRE_ADMINS", "boss@example.test")
	f := newFixture(t)
	f.verifySender("boss@example.test")
	boss := f.cookie
	f.cookie = nil
	f.verifySender("regular@example.test") // an ordinary sender, signed in while access was still open
	regular := f.cookie
	f.cookie = boss
	t.Setenv("PHOENIXWIRE_SENDERS", "") // by invitation
	view := func(w *httptest.ResponseRecorder) map[string]any {
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	list := func(v map[string]any, key string) []string {
		var out []string
		for _, e := range v[key].([]any) {
			out = append(out, e.(string))
		}
		return out
	}
	// the panel lists what the administrator entered, never the senders who signed in by themselves
	f.api.mu.Lock()
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("%048x", 9000+i)
		f.api.db.state.Payments[id] = savedPayment{Payment: payment{ID: id, Sender: fmt.Sprintf("user%d@example.test", i), Stage: "sent", Created: f.api.now()}}
	}
	f.api.mu.Unlock()
	v := view(f.call("GET", "/api/admin/access", nil))
	if len(v["registered"].([]any)) != 0 || v["registered_count"] != float64(5) || len(list(v, "allowed")) != 0 {
		t.Fatal("the panel must count the registered senders and list only invitations", v)
	}
	enter := func(address string) int {
		saved := f.cookie
		f.cookie = nil
		defer func() { f.cookie = saved }()
		return f.call("POST", "/api/sender/request", map[string]string{"email": address}).Code
	}
	// invite, then revoke: the invitation is gone and so is the way in
	good(f.call("POST", "/api/admin/allow", map[string]string{"email": "guest@example.test"}), t)
	if got := list(view(f.call("GET", "/api/admin/access", nil)), "allowed"); len(got) != 1 || got[0] != "guest@example.test" {
		t.Fatal("the invitation is not listed", got)
	}
	w := f.call("POST", "/api/admin/revoke", map[string]string{"email": "Guest@Example.test"})
	if w.Code != 200 || len(list(view(w), "allowed")) != 0 {
		t.Fatal("revoke", w.Code, w.Body)
	}
	if enter("guest@example.test") != 403 {
		t.Fatal("a revoked guest still signs in")
	}
	// a sender who has cards is not locked out of his money by a revoke; a block does cut him off
	good(f.call("POST", "/api/admin/allow", map[string]string{"email": "user1@example.test"}), t)
	good(f.call("POST", "/api/admin/revoke", map[string]string{"email": "user1@example.test"}), t)
	f.api.mu.Lock()
	still := f.api.senderAllowed("user1@example.test")
	f.api.mu.Unlock()
	if !still {
		t.Fatal("revoking an invitation must not lock a sender with cards out of his refunds")
	}
	good(f.call("POST", "/api/admin/block", map[string]string{"email": "user1@example.test"}), t)
	if enter("user1@example.test") != 403 {
		t.Fatal("a blocked sender signed in")
	}
	// remove forgets an address altogether, blocked ones included
	w = f.call("POST", "/api/admin/remove", map[string]string{"email": "user1@example.test"})
	v = view(w)
	if w.Code != 200 || len(list(v, "blocked")) != 0 || len(list(v, "allowed")) != 0 {
		t.Fatal("remove must empty both lists for that address", w.Code, v)
	}
	// an address that is not on any list can be removed without harm, and a bad one is refused
	good(f.call("POST", "/api/admin/remove", map[string]string{"email": "nobody@example.test"}), t)
	if w := f.call("POST", "/api/admin/revoke", map[string]string{"email": "not an address"}); w.Code != 400 {
		t.Fatal("a bad address was accepted", w.Code)
	}
	// only administrators
	f.cookie = regular
	if w := f.call("POST", "/api/admin/remove", map[string]string{"email": "guest@example.test"}); w.Code != 403 {
		t.Fatal("an ordinary sender used the panel", w.Code)
	}
}

func TestAskingForCodesTooOftenSaysToWaitAndIsNotAFailure(t *testing.T) {
	f := newFixture(t)
	first := f.call("POST", "/api/sender/request", map[string]string{"email": "quick@example.test"})
	good(first, t)
	second := f.call("POST", "/api/sender/request", map[string]string{"email": "quick@example.test"})
	if second.Code != 429 || !strings.Contains(second.Body.String(), "wait_before_resend") {
		t.Fatal("a second code asked at once must say to wait", second.Code, second.Body)
	}
	var answer struct {
		RetryAfter int `json:"retry_after"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &answer); err != nil || answer.RetryAfter < 1 || answer.RetryAfter > 20 || second.Header().Get("Retry-After") == "" {
		t.Fatal("the wait must be said in seconds, for the countdown", answer.RetryAfter, second.Body)
	}
}
