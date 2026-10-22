package wire

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/xcoder2088/NimiqSecureClaim/service"
	"image"
	"image/png"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

type mailTransport func(*http.Request) (*http.Response, error)

func testNimicon(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 160, 160))); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

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
		if !strings.Contains(html, "padding:20px 24px") || !strings.Contains(html, "POWERED BY") || !strings.Contains(html, `href="https://phoenixmail.ca"`) || strings.Contains(html, "cid:") || !strings.Contains(html, "data:image/png;base64,") {
			t.Fatal("missing spacing/branding")
		}
		settings, ok := body["settings"].(map[string]any)
		if !ok || len(settings) != 3 || settings["track_opens"] != false || settings["track_clicks"] != false || settings["track_content"] != false || r.Header.Get("User-Agent") == "" {
			t.Fatal("tracking or user agent")
		}
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}}
	err := m.Invite(context.Background(), payment{ID: strings.Repeat("a", 48), Sender: "sender@example.test", Recipient: "recipient@example.test", Amount: "2.5", Language: "en", Nimicon: testNimicon(t), Message: "<script>oops</script>"}, "https://wire.example.test/?claim="+strings.Repeat("a", 48))
	if err != nil || !called {
		t.Fatal(err)
	}
}

func TestSenderCodeUsesSharedLocalizedTemplate(t *testing.T) {
	for _, lang := range []string{"fr", "en", "es", "de", "zh"} {
		m := MailerSend{APIKey: "fixture", From: "wire@example.test", Client: &http.Client{Transport: mailTransport(func(r *http.Request) (*http.Response, error) {
			var body map[string]any
			if json.NewDecoder(r.Body).Decode(&body) != nil {
				t.Fatal("payload")
			}
			h := body["html"].(string)
			if !strings.Contains(h, `lang="`+lang+`"`) || strings.Contains(h, "claim page") || !strings.Contains(h, "123456") || !strings.Contains(h, "data:image/png;base64,") || r.Header.Get("User-Agent") == "" {
				t.Fatal("sender email template")
			}
			visible := regexp.MustCompile(`data:image/png;base64,[^"]+`).ReplaceAllString(h, "")
			if strings.Contains(visible, "OTP") || strings.Contains(visible, "Mainnet") {
				t.Fatal("technical wording")
			}
			return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
		})}}
		if err := m.Send(service.WithEmailLanguage(context.Background(), lang), "person@example.test", "123456"); err != nil {
			t.Fatal(err)
		}
	}
}
