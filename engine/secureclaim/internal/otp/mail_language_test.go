package otp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestSenderAndRecipientEmailUseSharedTemplateInFiveLanguages(t *testing.T) {
	for _, lang := range []string{"en", "fr", "es", "de", "zh"} {
		for _, purpose := range []string{"", "sender"} {
			capture := &captureTransport{}
			m := MailerSend{APIKey: "fixture", From: "wire@example.test", BrandName: "Phoenix Wire", Purpose: purpose, Client: &http.Client{Transport: capture}}
			if err := m.Send(WithEmailLanguage(context.Background(), lang), "person@example.test", "123456"); err != nil {
				t.Fatal(err)
			}
			var body struct{ HTML, Subject, Text string }
			if err := json.Unmarshal(capture.body, &body); err != nil {
				t.Fatal(err)
			}
			_, words := mailWords(WithEmailLanguage(context.Background(), lang), purpose == "sender")
			if !strings.Contains(body.HTML, words[2]) || !strings.Contains(body.HTML, `lang="`+lang+`"`) || !strings.Contains(body.HTML, "data:image/png;base64,") || !strings.Contains(body.Text, "123456") {
				t.Fatalf("wrong template for %s/%s", lang, purpose)
			}
			if purpose == "sender" && strings.Contains(body.HTML, "claim page") {
				t.Fatal("sign-in email must not tell sender to claim")
			}
		}
	}
}
