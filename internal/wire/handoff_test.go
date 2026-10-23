package wire

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Two browsers on one phone: Safari and the Nimiq Pay app. Each has its own cookie jar.
type browsers struct {
	f           *fixture
	safari, app *http.Cookie
}

func (b *browsers) inSafari(fn func()) { b.f.cookie = b.safari; fn(); b.safari = b.f.cookie }
func (b *browsers) inApp(fn func())    { b.f.cookie = b.app; fn(); b.app = b.f.cookie }

func newBrowsers(t *testing.T) *browsers {
	t.Helper()
	f := newFixture(t)
	f.verifySender("sender@example.test")
	return &browsers{f: f, safari: f.cookie}
}

type handed struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

func (b *browsers) handoff(t *testing.T, id string) handed {
	t.Helper()
	var out handed
	b.inSafari(func() {
		w := b.f.call("POST", "/api/handoff", map[string]string{"payment_id": id})
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || len(out.Token) != 43 || out.ExpiresIn != 300 {
			t.Fatalf("handoff: %d %s", w.Code, w.Body)
		}
	})
	return out
}

func (b *browsers) redeem(token string) *httptest.ResponseRecorder {
	var w *httptest.ResponseRecorder
	b.inApp(func() { w = b.f.call("POST", "/api/handoff/redeem", map[string]string{"token": token}) })
	return w
}

// signInApp makes the app's browser a signed-in sender (a session as a verified e-mail code would make it).
func (b *browsers) signInApp(email string) {
	raw := token()
	b.f.api.db.state.Sessions[sessionKey(raw)] = session{Email: email, Verified: true, Expires: b.f.api.now().Add(time.Hour)}
	b.app = &http.Cookie{Name: b.f.api.cookieName(), Value: raw}
}

func (b *browsers) appCall(method, path string, body any) *httptest.ResponseRecorder {
	var w *httptest.ResponseRecorder
	b.inApp(func() { w = b.f.call(method, path, body) })
	return w
}

func (b *browsers) safariCall(method, path string, body any) *httptest.ResponseRecorder {
	var w *httptest.ResponseRecorder
	b.inSafari(func() { w = b.f.call(method, path, body) })
	return w
}

func code(w *httptest.ResponseRecorder, body string) string {
	var out struct{ Code string }
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	_ = body
	return out.Code
}

// From Safari (signed in) to the app (no cookie): the card, its amount and its recipient, funded and followed there, with no second code and
// no second payment.
func TestHandoffFundsOnePaymentInTheOtherBrowser(t *testing.T) {
	b := newBrowsers(t)
	var p payment
	b.inSafari(func() { p = b.f.create() })
	if w := b.appCall("GET", "/api/payments", nil); w.Code != 401 {
		t.Fatalf("the app has nothing before the hand-over: %d", w.Code)
	}
	h := b.handoff(t, p.ID)
	if _, ok := b.f.api.handoffs[h.Token]; ok || len(b.f.api.handoffs) != 1 {
		t.Fatal("only the hash of the right is kept")
	}
	w := b.redeem(h.Token)
	var redeemed struct {
		Payment payment
		Scoped  bool
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &redeemed) != nil || !redeemed.Scoped || redeemed.Payment.ID != p.ID || redeemed.Payment.Amount != "2.5" || redeemed.Payment.Recipient != "recipient@example.test" {
		t.Fatalf("redeem: %d %s", w.Code, w.Body)
	}
	if b.app == nil || b.app.Value == b.safari.Value || b.app.Value == h.Token || strings.Contains(w.Body.String(), b.app.Value) {
		t.Fatal("the app gets its own cookie: never the token, never Safari's")
	}
	// what the app is: not a signed-in sender, but allowed to fund this payment
	var session struct {
		Verified bool
		Email    string
		Handoff  struct {
			PaymentID string `json:"payment_id"`
		}
	}
	if s := b.appCall("GET", "/api/session", nil); json.Unmarshal(s.Body.Bytes(), &session) != nil || session.Verified || session.Email != "" || session.Handoff.PaymentID != p.ID {
		t.Fatalf("the app's session: %s", s.Body)
	}
	var list struct{ Payments []payment }
	if l := b.appCall("GET", "/api/payments", nil); json.Unmarshal(l.Body.Bytes(), &list) != nil || len(list.Payments) != 1 || list.Payments[0].ID != p.ID {
		t.Fatalf("the app sees this payment only: %s", l.Body)
	}
	// funding starts once: from the app, with the checkout the server fixed; no second start from either browser
	start := b.appCall("POST", "/api/payments/"+p.ID+"/start", map[string]string{})
	var started struct {
		Checkout struct {
			Recipient     string
			Value         uint64
			RecipientData []int
		}
	}
	if start.Code != 200 || json.Unmarshal(start.Body.Bytes(), &started) != nil || started.Checkout.Recipient != p.Address || started.Checkout.Value != 250000 || len(started.Checkout.RecipientData) != 5 {
		t.Fatalf("start in the app: %d %s", start.Code, start.Body)
	}
	if w := b.appCall("POST", "/api/payments/"+p.ID+"/start", map[string]string{}); w.Code != 409 {
		t.Fatalf("a second start from the app: %d", w.Code)
	}
	if w := b.safariCall("POST", "/api/payments/"+p.ID+"/start", map[string]string{}); w.Code != 409 {
		t.Fatalf("a second start from Safari: %d", w.Code)
	}
	if w := b.appCall("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}); w.Code != 409 {
		t.Fatalf("not funded yet: %d", w.Code)
	}
	b.f.rpcMu.Lock()
	b.f.funded = true
	b.f.rpcMu.Unlock()
	if w := b.appCall("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}); w.Code != 200 || b.f.mail.invites != 1 {
		t.Fatalf("funded in the app: %d %s (e-mails %d)", w.Code, w.Body, b.f.mail.invites)
	}
	// Safari follows the same card
	var sent struct{ Payments []payment }
	if l := b.safariCall("GET", "/api/payments", nil); json.Unmarshal(l.Body.Bytes(), &sent) != nil || sent.Payments[0].Stage != "sent" || sent.Payments[0].SentAt == nil {
		t.Fatalf("Safari sees the card sent: %s", l.Body)
	}
	// the app survives a restart of the service, still scoped
	b.f.restart()
	if s := b.appCall("GET", "/api/session", nil); !strings.Contains(s.Body.String(), p.ID) || strings.Contains(s.Body.String(), `"verified":true`) {
		t.Fatalf("after a restart: %s", s.Body)
	}
}

// The app may fund and follow its payment. Nothing else.
func TestHandoffSessionReachesNothingBeyondItsPayment(t *testing.T) {
	b := newBrowsers(t)
	var p, other payment
	b.inSafari(func() {
		p = b.f.create()
		in := paymentInput()
		in["request_id"] = strings.Repeat("b", 32)
		w := b.f.call("POST", "/api/payments", in)
		var o struct{ Payment payment }
		json.Unmarshal(w.Body.Bytes(), &o)
		other = o.Payment
	})
	if other.ID == "" || other.ID == p.ID {
		t.Fatal("a second payment")
	}
	b.redeem(b.handoff(t, p.ID).Token)
	for name, probe := range map[string]struct{ method, path string }{
		"create a payment":         {"POST", "/api/payments"},
		"another payment's start":  {"POST", "/api/payments/" + other.ID + "/start"},
		"another payment's resend": {"POST", "/api/payments/" + other.ID + "/resend"},
		"the recovery key":         {"POST", "/api/payments/" + p.ID + "/recovery"},
		"prepare":                  {"POST", "/api/payments/" + p.ID + "/prepare"},
		"USDC / USDT cards":        {"GET", "/api/usdt/cards"},
		"a USDC card":              {"POST", "/api/usdt/cashlink/create"},
		"the window's status":      {"GET", "/api/usdt/status"},
		"a hand-over of its own":   {"POST", "/api/handoff"},
	} {
		if w := b.appCall(probe.method, probe.path, map[string]string{"payment_id": p.ID}); w.Code != 403 || code(w, "") != "handoff_scope" {
			t.Errorf("%s: %d %s", name, w.Code, w.Body)
		}
	}
	// the sender's other payment is not in its list, and signing in as the sender is another thing (a code)
	var list struct{ Payments []payment }
	if l := b.appCall("GET", "/api/payments", nil); json.Unmarshal(l.Body.Bytes(), &list) != nil || len(list.Payments) != 1 {
		t.Fatalf("the list of the app: %s", l.Body)
	}
	// signing out of the app ends it
	if w := b.appCall("POST", "/api/logout", map[string]string{}); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := b.appCall("GET", "/api/payments", nil); w.Code != 401 {
		t.Fatalf("after signing out: %d", w.Code)
	}
}

// The right is used once, lasts five minutes, belongs to one fundable payment of the sender and dies with the session it came from.
func TestHandoffRightIsOneUseShortAndTiedToItsPayment(t *testing.T) {
	b := newBrowsers(t)
	var p payment
	b.inSafari(func() { p = b.f.create() })
	// not an unknown payment, not a malformed id, not another sender's payment
	for _, id := range []string{strings.Repeat("0", 48), "x", ""} {
		if w := b.safariCall("POST", "/api/handoff", map[string]string{"payment_id": id}); w.Code != 404 && w.Code != 400 {
			t.Errorf("payment %q: %d", id, w.Code)
		}
	}
	raw := token()
	b.f.api.db.state.Sessions[sessionKey(raw)] = session{Email: "other@example.test", Verified: true, Expires: b.f.api.now().Add(time.Hour)}
	b.safari = &http.Cookie{Name: b.f.api.cookieName(), Value: raw}
	if w := b.safariCall("POST", "/api/handoff", map[string]string{"payment_id": p.ID}); w.Code != 404 {
		t.Errorf("another sender's payment: %d", w.Code)
	}
	b = newBrowsers(t)
	b.inSafari(func() { p = b.f.create() })
	h := b.handoff(t, p.ID)
	if w := b.redeem(h.Token); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	b.app = nil
	if w := b.redeem(h.Token); w.Code != 404 || code(w, "") != "invalid_or_expired" {
		t.Fatalf("a right is used once: %d %s", w.Code, w.Body)
	}
	// five minutes
	other := func() payment {
		var q payment
		b.inSafari(func() {
			in := paymentInput()
			in["request_id"] = strings.Repeat("c", 32)
			var o struct{ Payment payment }
			json.Unmarshal(b.f.call("POST", "/api/payments", in).Body.Bytes(), &o)
			q = o.Payment
		})
		return q
	}()
	h = b.handoff(t, other.ID)
	base := b.f.api.now()
	b.f.api.now = func() time.Time { return base.Add(handoffTTL + time.Second) }
	b.app = nil
	if w := b.redeem(h.Token); w.Code != 404 || code(w, "") != "invalid_or_expired" {
		t.Fatalf("expired: %d %s", w.Code, w.Body)
	}
	b.f.api.now = func() time.Time { return base }
	// signing out of Safari cancels it
	h = b.handoff(t, other.ID)
	b.safariCall("POST", "/api/logout", map[string]string{})
	b.app = nil
	if w := b.redeem(h.Token); w.Code != 404 {
		t.Fatalf("a right of a signed-out session: %d", w.Code)
	}
	// unknown or malformed: the same words
	var first string
	for _, bad := range []string{"", "short", strings.Repeat("A", 43), strings.Repeat("A", 44), strings.Repeat("é", 43), strings.Repeat("A", 42) + "+"} {
		b.app = nil
		w := b.redeem(bad)
		if w.Code != 404 || code(w, "") != "invalid_or_expired" {
			t.Fatalf("%q: %d %s", bad, w.Code, w.Body)
		}
		if first == "" {
			first = w.Body.String()
		} else if w.Body.String() != first {
			t.Fatalf("every failure says the same thing")
		}
	}
}

// Funding started elsewhere or delivered: no right, no second payment.
func TestHandoffRefusesAPaymentThatCannotBeFundedAnymore(t *testing.T) {
	b := newBrowsers(t)
	var p payment
	b.inSafari(func() { p = b.f.create() })
	h := b.handoff(t, p.ID) // asked for, then the funding starts in Safari (the Hub) before the app opens
	if w := b.safariCall("POST", "/api/payments/"+p.ID+"/start", map[string]string{}); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if w := b.redeem(h.Token); w.Code != 409 || code(w, "") != "payment_not_fundable" || b.app != nil {
		t.Fatalf("started elsewhere: %d %s (cookie %v)", w.Code, w.Body, b.app)
	}
	if w := b.safariCall("POST", "/api/handoff", map[string]string{"payment_id": p.ID}); w.Code != 409 {
		t.Fatalf("no right for a payment that has started: %d", w.Code)
	}
	// an expired day
	b = newBrowsers(t)
	b.inSafari(func() { p = b.f.create() })
	base := b.f.api.now()
	b.f.api.now = func() time.Time { return base.Add(25 * time.Hour) }
	for k, s := range b.f.api.db.state.Sessions {
		s.Expires = b.f.api.now().Add(time.Hour)
		b.f.api.db.state.Sessions[k] = s
	}
	if w := b.safariCall("POST", "/api/handoff", map[string]string{"payment_id": p.ID}); w.Code != 410 {
		t.Fatalf("a payment past its day: %d", w.Code)
	}
}

// A browser that is signed in as someone else is never replaced; the same sender is not downgraded.
func TestHandoffNeverReplacesAnotherSessionSilently(t *testing.T) {
	b := newBrowsers(t)
	var p payment
	b.inSafari(func() { p = b.f.create() })
	h := b.handoff(t, p.ID)
	// the app is signed in as another sender
	b.signInApp("someone.else@example.test")
	before := b.app.Value
	if w := b.redeem(h.Token); w.Code != 409 || code(w, "") != "session_conflict" || b.app.Value != before {
		t.Fatalf("another sender is replaced: %d %s", w.Code, w.Body)
	}
	if s := b.appCall("GET", "/api/session", nil); !strings.Contains(s.Body.String(), "someone.else@example.test") {
		t.Fatal("the other session must be intact")
	}
	// the right was not used up: from a clean browser it still works
	b.app = nil
	if w := b.redeem(h.Token); w.Code != 200 {
		t.Fatalf("the right survives a refused hand-over: %d %s", w.Code, w.Body)
	}
	// the same sender already signed in the app: nothing is handed over, nothing is downgraded
	h = b.handoff(t, p.ID)
	b.signInApp("sender@example.test")
	before = b.app.Value
	w := b.redeem(h.Token)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "already_signed_in") || b.app.Value != before {
		t.Fatalf("the same sender: %d %s", w.Code, w.Body)
	}
	if s := b.appCall("GET", "/api/session", nil); !strings.Contains(s.Body.String(), `"verified":true`) {
		t.Fatal("the sender stays fully signed in")
	}
}

// The scoped session lasts two hours at most, never beyond the payment's day or the session it came from; rights are limited and race-safe.
func TestHandoffSessionBoundsLimitsAndConcurrency(t *testing.T) {
	b := newBrowsers(t)
	var p payment
	b.inSafari(func() { p = b.f.create() })
	now := b.f.api.now()
	for k, s := range b.f.api.db.state.Sessions { // Safari's session has 30 minutes left
		s.Expires = now.Add(30 * time.Minute)
		b.f.api.db.state.Sessions[k] = s
	}
	b.redeem(b.handoff(t, p.ID).Token)
	scoped := 0
	for _, s := range b.f.api.db.state.Sessions {
		if s.Scope != "" {
			scoped++
			if s.Verified || s.Expires.After(now.Add(30*time.Minute)) {
				t.Fatalf("a hand-over must not outlive the session it came from, nor become a sender: %+v", s)
			}
		}
	}
	if scoped != 1 {
		t.Fatalf("one scoped session, got %d", scoped)
	}
	// two browsers racing for the same right: one wins
	for k, s := range b.f.api.db.state.Sessions {
		s.Expires = now.Add(time.Hour)
		b.f.api.db.state.Sessions[k] = s
	}
	b.app = nil
	var q payment
	b.inSafari(func() {
		in := paymentInput()
		in["request_id"] = strings.Repeat("d", 32)
		var o struct{ Payment payment }
		json.Unmarshal(b.f.call("POST", "/api/payments", in).Body.Bytes(), &o)
		q = o.Payment
	})
	h := b.handoff(t, q.ID)
	var wg sync.WaitGroup
	results := make([]int, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/handoff/redeem", bytes.NewReader([]byte(`{"token":"`+h.Token+`"}`)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "https://wire.example.test")
			req.RemoteAddr = "127.0.0.1:1234"
			w := httptest.NewRecorder()
			b.f.api.Handler().ServeHTTP(w, req)
			results[i] = w.Code
		}(i)
	}
	wg.Wait()
	wins := 0
	for _, c := range results {
		if c == 200 {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("one winner out of a race: %v", results)
	}
	// a sender cannot ask for rights without end
	for i := 0; i < handoffPerSpell; i++ {
		b.safariCall("POST", "/api/handoff", map[string]string{"payment_id": p.ID})
	}
	if w := b.safariCall("POST", "/api/handoff", map[string]string{"payment_id": q.ID}); w.Code != http.StatusTooManyRequests && w.Code != 409 {
		t.Fatalf("too many rights: %d", w.Code)
	}
}
