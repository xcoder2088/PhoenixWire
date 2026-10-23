package wire

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func draftBody() map[string]string {
	return map[string]string{"recipient": "friend@example.test", "amount": "2.5", "message": "Test nimpay", "language": "fr"}
}

func newHandoff(t *testing.T, f *fixture) string {
	t.Helper()
	w := f.call("POST", "/api/handoff", draftBody())
	var out struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || len(out.Token) != 43 || out.ExpiresIn != 120 {
		t.Fatalf("handoff: %d %s", w.Code, w.Body)
	}
	return out.Token
}

// The Nimiq Pay app is another browser: it gets the sender and the draft through a token, not through the address.
func TestHandoffCarriesTheVerifiedSenderToAnotherBrowser(t *testing.T) {
	f := newFixture(t)
	if w := f.call("POST", "/api/handoff", draftBody()); w.Code != 401 {
		t.Fatalf("nobody gets a token without being verified: %d", w.Code)
	}
	f.verifySender("sender@example.test")
	for name, bad := range map[string]map[string]string{
		"bad recipient": {"recipient": "nobody", "amount": "2.5"},
		"bad amount":    {"recipient": "friend@example.test", "amount": "lots"},
		"long message":  {"recipient": "friend@example.test", "amount": "2.5", "message": strings.Repeat("x", 161)},
	} {
		if w := f.call("POST", "/api/handoff", bad); w.Code != 400 {
			t.Errorf("%s: %d", name, w.Code)
		}
	}
	original := f.cookie
	token := newHandoff(t, f)
	if _, ok := f.api.handoffs[token]; ok || len(f.api.handoffs) != 1 {
		t.Fatal("only the hash of the token is kept")
	}
	// the other browser: no cookie, only the token
	f.cookie = nil
	w := f.call("POST", "/api/handoff/redeem", map[string]string{"token": token})
	var out struct {
		Email string
		Draft handoffDraft
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Email != "sender@example.test" || out.Draft != (handoffDraft{Recipient: "friend@example.test", Amount: "2.5", Message: "Test nimpay", Language: "fr"}) {
		t.Fatalf("redeem: %d %s", w.Code, w.Body)
	}
	if f.cookie == nil || f.cookie.Value == original.Value || f.cookie.Value == token || strings.Contains(w.Body.String(), f.cookie.Value) {
		t.Fatal("the other browser gets its own session cookie, never the token or the cookie of the first one")
	}
	var session struct {
		Verified bool
		Email    string
	}
	if s := f.call("GET", "/api/session", nil); json.Unmarshal(s.Body.Bytes(), &session) != nil || !session.Verified || session.Email != "sender@example.test" {
		t.Fatalf("signed in without a new code: %s", s.Body)
	}
	// one use
	f.cookie = nil
	again := f.call("POST", "/api/handoff/redeem", map[string]string{"token": token})
	if again.Code != 404 || !strings.Contains(again.Body.String(), "invalid_or_expired") {
		t.Fatalf("a token is used once: %d %s", again.Code, again.Body)
	}
	// the first browser is still signed in
	f.cookie = original
	if s := f.call("GET", "/api/session", nil); !strings.Contains(s.Body.String(), `"verified":true`) {
		t.Fatal("the first browser stays signed in")
	}
}

func TestHandoffExpiresIsCancelledBySigningOutAndSaysNothingElse(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	original := f.cookie
	redeem := func(token string) (int, string) {
		f.cookie = nil
		w := f.call("POST", "/api/handoff/redeem", map[string]string{"token": token})
		return w.Code, w.Body.String()
	}
	// two minutes
	token := newHandoff(t, f)
	f.cookie = original
	base := f.api.now()
	f.api.now = func() time.Time { return base.Add(handoffTTL + time.Second) }
	if code, body := redeem(token); code != 404 || !strings.Contains(body, "invalid_or_expired") {
		t.Fatalf("expired: %d %s", code, body)
	}
	f.api.now = func() time.Time { return base }
	// signing out cancels what was handed over
	f.cookie = original
	token = newHandoff(t, f)
	f.cookie = original
	if w := f.call("POST", "/api/logout", map[string]string{}); w.Code != 200 {
		t.Fatalf("logout: %d", w.Code)
	}
	if code, _ := redeem(token); code != 404 {
		t.Fatalf("a token of a signed-out session: %d", code)
	}
	// unknown, malformed, wrong length: the same answer, in the same words
	var first string
	for _, bad := range []string{"", "short", strings.Repeat("A", 43), strings.Repeat("A", 44), strings.Repeat("é", 43), strings.Repeat("A", 42) + "+"} {
		code, body := redeem(bad)
		if code != 404 || !strings.Contains(body, "invalid_or_expired") {
			t.Fatalf("%q: %d %s", bad, code, body)
		}
		if first == "" {
			first = body
		} else if body != first {
			t.Fatalf("every failure says the same thing: %q vs %q", body, first)
		}
	}
	if w := f.call("GET", "/api/handoff/redeem", nil); w.Code == 200 {
		t.Fatal("redeem is a POST")
	}
}

// A hand-over never makes a sign-in last longer than it already would, and one sender cannot ask for tokens without end.
func TestHandoffDoesNotExtendTheSessionAndIsLimited(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	original := f.cookie
	// the first session has one hour left
	for k, s := range f.api.db.state.Sessions {
		s.Expires = f.api.now().Add(time.Hour)
		f.api.db.state.Sessions[k] = s
	}
	token := newHandoff(t, f)
	f.cookie = nil
	if w := f.call("POST", "/api/handoff/redeem", map[string]string{"token": token}); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	latest := f.api.now().Add(time.Hour)
	for _, s := range f.api.db.state.Sessions {
		if s.Expires.After(latest) {
			t.Fatal("the session of the other browser lasts longer than the one it came from")
		}
	}
	f.cookie = original
	for i := 0; i < handoffPerSpell-1; i++ { // one was already asked for
		newHandoff(t, f)
	}
	if w := f.call("POST", "/api/handoff", draftBody()); w.Code != http.StatusTooManyRequests {
		t.Fatalf("a sender cannot ask for tokens without end: %d", w.Code)
	}
	f.api.now = func() time.Time { return time.Now().Add(handoffSpell + time.Minute) }
	for k, s := range f.api.db.state.Sessions { // the session must still be valid at that later time
		s.Expires = f.api.now().Add(time.Hour)
		f.api.db.state.Sessions[k] = s
	}
	newHandoff(t, f)
}
