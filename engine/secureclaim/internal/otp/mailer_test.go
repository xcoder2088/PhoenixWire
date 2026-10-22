package otp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMailerSendValidateRequiresRealSenderConfiguration(t *testing.T) {
	if err := (MailerSend{APIKey: "test-token", From: "otp@example.test"}).Validate(); err != nil {
		t.Fatalf("valid sender config rejected: %v", err)
	}
	for _, test := range []MailerSend{
		{APIKey: "", From: "otp@example.test"},
		{APIKey: "test-token", From: "not-an-email"},
		{APIKey: "test-token", From: "SecureClaim <otp@example.test>"},
	} {
		if err := test.Validate(); err == nil {
			t.Fatalf("invalid sender config accepted: %+v", test)
		}
	}
}

type captureTransport struct{ body []byte }

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.body, _ = io.ReadAll(r.Body)
	return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
}

func TestVerificationEmailCarriesTheLogoInTheHTMLAndDoesNotTrack(t *testing.T) {
	capture := &captureTransport{}
	m := MailerSend{APIKey: "key", From: "no-reply@phoenixmail.ca", Client: &http.Client{Transport: capture}}
	if err := m.Send(context.Background(), "someone@example.com", "123456"); err != nil {
		t.Fatal(err)
	}
	var sent struct {
		From        map[string]string `json:"from"`
		Subject     string            `json:"subject"`
		HTML        string            `json:"html"`
		Attachments []any             `json:"attachments"`
		Settings    map[string]bool   `json:"settings"`
	}
	if err := json.Unmarshal(capture.body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent.From["name"] != "PhoenixMail" || !strings.Contains(sent.Subject, "PhoenixMail") {
		t.Fatalf("sender/subject: %v %q", sent.From, sent.Subject)
	}
	if !strings.Contains(sent.HTML, `src="data:image/png;base64,`) || len(sent.Attachments) != 0 || strings.Contains(sent.HTML, "cid:") {
		t.Fatal("the logo must be in the HTML itself, with no attachment")
	}
	if sent.Settings["track_opens"] || sent.Settings["track_clicks"] || sent.Settings["track_content"] || len(sent.Settings) != 3 {
		t.Fatalf("tracking must be off: %v", sent.Settings)
	}
}

func sentMail(t *testing.T, m MailerSend) (from map[string]string, subject, html, text string) {
	t.Helper()
	capture := &captureTransport{}
	m.Client = &http.Client{Transport: capture}
	if err := m.Send(context.Background(), "someone@example.com", "123456"); err != nil {
		t.Fatal(err)
	}
	var sent struct {
		From    map[string]string `json:"from"`
		Subject string            `json:"subject"`
		HTML    string            `json:"html"`
		Text    string            `json:"text"`
	}
	if err := json.Unmarshal(capture.body, &sent); err != nil {
		t.Fatal(err)
	}
	return sent.From, sent.Subject, sent.HTML, sent.Text
}

func TestVerificationEmailBrandName(t *testing.T) {
	// default: PhoenixMail, "Phoenix" navy + "Mail" orange
	from, subject, html, text := sentMail(t, MailerSend{APIKey: "k", From: "no-reply@phoenixmail.ca"})
	if from["name"] != "PhoenixMail" || subject != "Your PhoenixMail verification code" || !strings.Contains(html, `Phoenix<span style="color:#f39043">Mail</span>`) || !strings.Contains(text, "Your PhoenixMail verification code is 123456") {
		t.Fatalf("default brand changed: %v %q", from, subject)
	}
	// PhoenixWire: same template and colours, its own name
	from, subject, html, text = sentMail(t, MailerSend{APIKey: "k", From: "no-reply@phoenixmail.ca", BrandName: "Phoenix Wire"})
	if from["name"] != "Phoenix Wire" || subject != "Your Phoenix Wire verification code" || !strings.Contains(html, `Phoenix<span style="color:#f39043">Wire</span>`) || strings.Contains(html, ">Mail<") || !strings.Contains(text, "Your Phoenix Wire verification code") {
		t.Fatalf("Phoenix Wire brand: %v %q", from, subject)
	}
	// any other name is escaped and shown plainly
	_, _, html, _ = sentMail(t, MailerSend{APIKey: "k", From: "no-reply@phoenixmail.ca", BrandName: "Acme <b>"})
	if strings.Contains(html, "<b>") || !strings.Contains(html, "Acme &lt;b&gt;") {
		t.Fatal("brand name must be escaped")
	}
}
