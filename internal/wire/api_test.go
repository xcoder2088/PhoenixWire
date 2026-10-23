package wire

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/service"
)

// Explicit fixtures: no real email, chain transaction or production credentials.
type testDelivery struct {
	mu       sync.Mutex
	codes    map[string]string
	invites  int
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
			data = []map[string]any{{"to": address, "value": 250000, "recipientData": "0082809287", "confirmations": 2, "executionResult": true}}
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
