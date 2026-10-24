package otp

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

// ConsoleSender is allowed ONLY by the explicit development-only server mode.
type ConsoleSender struct{}

func (ConsoleSender) Send(_ context.Context, to, code string) error {
	log.Printf("DEV-ONLY claim OTP for %s: %s (never enable in production)", to, code)
	return nil
}

// The PhoenixMail emblem is written into the HTML itself (data: image). MailerSend does not honour inline attachments (the picture
// ended up as a separate attachment and an empty square), and a hosted image is blocked by many mail apps. The name stays next to
// the emblem, so clients that drop data: images still show a complete header.
//
//go:embed assets/phoenixmail-email-icon.png
var logoPNG []byte

// MailerSend sends claim verification emails without sharing PhoenixMail login
// sessions or account OTP endpoints. Credentials remain server-side.
type MailerSend struct {
	APIKey string
	From   string
	// BrandName is the product shown as sender, in the subject and in the header ("PhoenixMail" when empty, "Phoenix Wire" for
	// PhoenixWire). A name starting with "Phoenix" is written like the cards: "Phoenix" in navy, the rest in orange.
	BrandName string
	// Purpose "sender" uses the same email template for Wire's sender sign-in.
	Purpose string
	// CardLayout draws the e-mail like a PhoenixWire card (frame, gradient, logo, white panel), in the colour of the card it belongs to.
	CardLayout bool
	// Footer, with CardLayout, gives the "powered by" badge under the card in the language of the e-mail (Wire passes the badge it puts under its cards).
	Footer func(lang string) string
	Client *http.Client
}

func (m MailerSend) brand() (name, first, second string) {
	name = strings.TrimSpace(m.BrandName)
	if name == "" {
		name = "PhoenixMail"
	}
	if strings.HasPrefix(name, "Phoenix") {
		return name, "Phoenix", strings.TrimSpace(strings.TrimPrefix(name, "Phoenix"))
	}
	return name, name, ""
}

func (m MailerSend) Validate() error {
	from, err := mail.ParseAddress(m.From)
	if err != nil || from.Address != m.From || strings.TrimSpace(m.APIKey) == "" {
		return errors.New("otp_mail_configuration_invalid")
	}
	return nil
}

func (m MailerSend) Send(ctx context.Context, to, code string) error {
	if err := m.Validate(); err != nil {
		return err
	}
	recipient, e2 := mail.ParseAddress(to)
	if e2 != nil || recipient.Address != to {
		return errors.New("otp_mail_configuration_invalid")
	}
	brand, brandFirst, brandSecond := m.brand()
	subject := "Your " + brand + " verification code"
	plain := fmt.Sprintf("Your %s verification code is %s.\nIt expires in 10 minutes. Never share this code.\nIf you did not request it, ignore this email.\n", brand, code)
	body := `<!doctype html><html lang="en"><head><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light only"><meta name="supported-color-schemes" content="light only"></head><body style="margin:0;background:#f4f2fa;color:#292044;font-family:Arial,sans-serif"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="background:#f4f2fa"><tr><td align="center" style="padding:28px 16px"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="max-width:520px;background:#fff;border:1px solid #ded8ec;border-radius:18px"><tr><td style="padding:28px 24px;font-size:16px;line-height:1.6"><table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin:0 0 22px"><tr><td style="padding-right:12px"><a href="https://phoenixmail.ca" target="_blank" rel="noopener" style="text-decoration:none;border:0"><img src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(logoPNG) + `" width="60" height="58" alt="" style="display:block;border:0"></a></td><td style="font-size:32px;line-height:34px;font-weight:bold;letter-spacing:-1px;color:#1a164d"><a href="https://phoenixmail.ca" target="_blank" rel="noopener" style="text-decoration:none;color:#1a164d">` + html.EscapeString(brandFirst) + `` + func() string {
		if brandSecond == "" {
			return ""
		}
		return `<span style="color:#f39043">` + html.EscapeString(brandSecond) + `</span>`
	}() + `</a></td></tr></table><h2 style="margin:0 0 20px;font-size:20px">Your protected payment</h2><p style="margin:0 0 16px">Enter this verification code on the claim page to unlock your payment:</p><table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0"><tr><td align="center" style="padding:20px 12px;background:#eee7ff;border-radius:12px;font-size:32px;font-weight:bold;letter-spacing:6px;color:#583b92">` + html.EscapeString(code) + `</td></tr></table><p style="margin:20px 0 16px">Expires in 10 minutes. Do not share this code.</p><p style="margin:0;color:#626477">If you did not request this code, ignore this email.</p></td></tr></table></td></tr></table></body></html>`
	if m.CardLayout {
		colours, withLabel := lavender, m.Purpose != "sender"
		if accent, ok := emailAccent(ctx); ok {
			colours = accent
		}
		body = cardLayout(brand, code, colours[0], colours[1], withLabel)
	}
	// the e-mail wears the colour of its card (code box and frame); the code itself is selectable in one tap
	if accent, ok := emailAccent(ctx); ok && !m.CardLayout {
		// The page stays white. The CARD wears the colour of the payment card (as the card sent by Wire does), and the code sits in a white box
		// with the card's border colour so that it stays easy to read.
		body = strings.NewReplacer("max-width:520px;background:#fff;border:1px solid #ded8ec", "max-width:520px;background:"+accent[0]+";border:1px solid "+accent[1],
			"background:#eee7ff;", "background:#ffffff;border:1px solid "+accent[1]+";", "#f4f2fa", "#ffffff").Replace(body)
	}
	if !m.CardLayout {
		body = strings.Replace(body, "letter-spacing:6px;color:#583b92", "letter-spacing:6px;color:#583b92;-webkit-user-select:all;user-select:all", 1)
	}
	lang, words := mailWords(ctx, m.Purpose == "sender")
	if lang != "en" || m.Purpose == "sender" {
		body = strings.NewReplacer(`<html lang="en">`, `<html lang="`+lang+`">`,
			"Your protected payment", html.EscapeString(words[2]),
			"Enter this verification code on the claim page to unlock your payment:", html.EscapeString(words[3]),
			"Expires in 10 minutes. Do not share this code.", html.EscapeString(words[4]),
			"If you did not request this code, ignore this email.", html.EscapeString(words[5])).Replace(body)
		subject = fmt.Sprintf(words[0], brand)
		plain = fmt.Sprintf(words[1], brand, code) + "\n" + words[4] + "\n" + words[5] + "\n"
	}
	body = strings.ReplaceAll(body, "@@PROTECTED@@", html.EscapeString(protectedLabel[lang]))
	if m.CardLayout && m.Footer != nil {
		body = strings.Replace(body, "</body></html>", m.Footer(lang)+"</body></html>", 1)
	}
	payload, err := json.Marshal(map[string]any{
		"from":     map[string]string{"email": m.From, "name": brand},
		"to":       []map[string]string{{"email": to}},
		"subject":  subject,
		"text":     plain,
		"html":     body,
		"settings": map[string]bool{"track_clicks": false, "track_opens": false, "track_content": false},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mailersend.com/v1/email", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(m.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PhoenixMail-claim/1.0")
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	answer, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusAccepted {
		// The provider says why (an account that may only send to verified recipients, a bad sender, a limit): its own message is logged, never the
		// key, the code or the address.
		var why struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(answer, &why)
		note := strings.Map(func(r rune) rune {
			if r < 32 || r == 127 {
				return ' '
			}
			return r
		}, why.Message)
		if len(note) > 200 {
			note = note[:200]
		}
		log.Printf("mail: the provider answered %d: %s", response.StatusCode, note)
		return fmt.Errorf("otp_mail_delivery_failed_status_%d", response.StatusCode)
	}
	return nil
}
