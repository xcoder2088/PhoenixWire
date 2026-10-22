package wire

import (
	"bytes"
	"context"
	"encoding/base64"
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

type fixture struct {
	api    *API
	engine *service.Engine
	mail   *testDelivery
	rpc    *httptest.Server
	root   string
	funded bool
	rpcMu  sync.Mutex
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
