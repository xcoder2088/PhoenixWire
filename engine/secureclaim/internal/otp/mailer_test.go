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

func TestCodeEmailWearsTheColourOfItsCard(t *testing.T) {
	capture := &captureTransport{}
	m := MailerSend{APIKey: "key", From: "no-reply@phoenixmail.ca", Client: &http.Client{Transport: capture}}
	ctx := WithEmailAccent(context.Background(), "#dcf5e9", "#9bd8bc")
	if err := m.Send(ctx, "someone@example.com", "123456"); err != nil {
		t.Fatal(err)
	}
	var sent struct {
		HTML string `json:"html"`
	}
	if err := json.Unmarshal(capture.body, &sent); err != nil {
		t.Fatal(err)
	}
	for _, must := range []string{"max-width:520px;background:#dcf5e9;border:1px solid #9bd8bc", "background:#ffffff;border:1px solid #9bd8bc;", "user-select:all", "123456"} {
		if !strings.Contains(sent.HTML, must) {
			t.Fatalf("missing %q", must)
		}
	}
	if strings.Contains(sent.HTML, "#eee7ff") || strings.Contains(sent.HTML, "#ded8ec") || strings.Contains(sent.HTML, "#f4f2fa") {
		t.Fatal("the default colours must be replaced: the page is white, the card wears the colour")
	}
	if !strings.Contains(sent.HTML, `content="light only"`) {
		t.Fatal("the e-mail must ask mail apps not to darken the card's colours")
	}
	// a malformed colour is ignored: the default e-mail is sent
	capture2 := &captureTransport{}
	m.Client = &http.Client{Transport: capture2}
	if err := m.Send(WithEmailAccent(context.Background(), "red;x", "#9bd8bc"), "someone@example.com", "123456"); err != nil || !strings.Contains(string(capture2.body), "#eee7ff") {
		t.Fatal("a malformed colour must leave the default e-mail untouched", err)
	}
}

// Everything Wire sends looks like one thing: the code e-mail is drawn like the card it belongs to.
func TestWireCodeEmailIsDrawnLikeItsCard(t *testing.T) {
	send := func(m MailerSend, ctx context.Context) string {
		t.Helper()
		capture := &captureTransport{}
		m.Client = &http.Client{Transport: capture}
		if err := m.Send(ctx, "someone@example.com", "382526"); err != nil {
			t.Fatal(err)
		}
		var sent struct {
			HTML string `json:"html"`
		}
		if err := json.Unmarshal(capture.body, &sent); err != nil {
			t.Fatal(err)
		}
		return sent.HTML
	}
	wire := MailerSend{APIKey: "k", From: "no-reply@phoenixmail.ca", BrandName: "Phoenix Wire", CardLayout: true}
	for lang, chip := range map[string]string{"en": "Protected", "fr": "Protégé", "es": "Protegido", "de": "Geschützt", "zh": "受保护"} {
		ctx := WithEmailLanguage(WithEmailAccent(context.Background(), "#ffe2ee", "#efaac7"), lang)
		page := send(wire, ctx)
		for _, must := range []string{
			"border-radius:26px", "border:1px solid #efaac7", "background:#ffe2ee;background-image:linear-gradient(125deg,#ffe2ee 0%,#e2eeff 56%,#ffe8f4 100%)",
			"border-radius:20px", "data:image/png;base64,", `alt="Phoenix Wire"`, ">" + chip + "<", "382526", "user-select:all", `<html lang="` + lang + `">`, `content="light only"`,
		} {
			if !strings.Contains(page, must) {
				t.Fatalf("%s: missing %q", lang, must)
			}
		}
		if strings.Contains(page, "@@") || strings.Contains(page, "#f4f2fa") || strings.Contains(page, "#eee7ff") {
			t.Fatalf("%s: a placeholder or the old template is left", lang)
		}
	}
	// the badge under the card is the one Wire puts under its cards, in the language of the e-mail
	badged := wire
	badged.Footer = func(lang string) string { return `<div data-footer="` + lang + `">powered</div>` }
	if page := send(badged, WithEmailLanguage(WithEmailAccent(context.Background(), "#ffe2ee", "#efaac7"), "de")); !strings.Contains(page, `<div data-footer="de">powered</div></body></html>`) {
		t.Fatal("the powered-by badge must come under the card, in the e-mail's language")
	}
	// a language that is not supported falls back to English, without a placeholder
	if page := send(wire, WithEmailLanguage(WithEmailAccent(context.Background(), "#dcf5e9", "#9bd8bc"), "xx")); !strings.Contains(page, ">Protected<") || strings.Contains(page, "@@") {
		t.Fatal("unsupported language")
	}
	// the sender's sign-in belongs to no card: same look, in the lavender family, without the "Protected" label
	sender := wire
	sender.Purpose = "sender"
	page := send(sender, WithEmailLanguage(context.Background(), "en"))
	if !strings.Contains(page, "background:#ece3ff;") || !strings.Contains(page, "border:1px solid #c7b1ee") || strings.Contains(page, "Protected") || strings.Contains(page, "@@") {
		t.Fatal("the sender e-mail must wear the lavender card without a label")
	}
	// PhoenixMail's own e-mail is untouched
	if old := send(MailerSend{APIKey: "k", From: "no-reply@phoenixmail.ca"}, context.Background()); !strings.Contains(old, "#f4f2fa") || strings.Contains(old, "border-radius:26px") {
		t.Fatal("the default layout must not change")
	}
}
