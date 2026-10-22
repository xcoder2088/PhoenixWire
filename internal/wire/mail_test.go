package wire

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mailTransport func(*http.Request) (*http.Response, error)

func (f mailTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestInvitationContainsOnlyOpaqueClaimAndEscapedContent(t *testing.T) {
	called := false
	m := MailerSend{APIKey: "fixture-only", From: "wire@example.test", Client: &http.Client{Transport: mailTransport(func(r *http.Request) (*http.Response, error) {
		called = true
		if r.URL.String() != "https://api.mailersend.com/v1/email" || r.Header.Get("Authorization") != "Bearer fixture-only" {
			t.Fatal("invalid mail transport")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Fatal("invalid payload")
		}
		html := body["html"].(string)
		if strings.Contains(html, "<script>") || strings.Contains(html, "cashlink/#") || !strings.Contains(html, "&lt;script&gt;") || !strings.Contains(html, "/?claim=") {
			t.Fatal("unsafe invitation HTML")
		}
		if !strings.Contains(html, "padding:28px 24px") || !strings.Contains(html, "Powered by Phoenix Mail") {
			t.Fatal("missing spacing/branding")
		}
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}}
	err := m.Invite(context.Background(), payment{Sender: "sender@example.test", Recipient: "recipient@example.test", Amount: "2.5", Message: "<script>oops</script>"}, "https://wire.example.test/?claim="+strings.Repeat("a", 48))
	if err != nil || !called {
		t.Fatal(err)
	}
}
