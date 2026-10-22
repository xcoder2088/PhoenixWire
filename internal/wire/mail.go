package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

type Delivery interface {
	Send(context.Context, string, string) error
	Invite(context.Context, payment, string) error
}
type MailerSend struct {
	APIKey, From string
	Client       *http.Client
}

func (m MailerSend) Validate() error {
	a, err := mail.ParseAddress(m.From)
	if err != nil || a.Address != m.From || strings.TrimSpace(m.APIKey) == "" {
		return errors.New("mail_configuration_invalid")
	}
	return nil
}
func (m MailerSend) Send(ctx context.Context, to, code string) error {
	return m.send(ctx, to, "Phoenix Wire — verify your sender email", "Your Phoenix Wire sender verification code: "+code+"\nExpires in 10 minutes. Never share it.", "<h2>Verify your sender email</h2><p>Your Phoenix Wire code:</p><p style=\"font-size:32px;letter-spacing:6px\">"+html.EscapeString(code)+"</p><p>Expires in 10 minutes. Never share it. Ignore this message if you did not request it.</p>")
}
func (m MailerSend) Invite(ctx context.Context, p payment, claimURL string) error {
	// Recipient gets an opaque claim URL only. No address QR, seed or bearer Cashlink.
	plain := fmt.Sprintf("%s sent you %s NIM with Phoenix Wire.\n%s\nOpen your protected payment: %s\nYou will receive a verification code at this email address.\nPowered by Phoenix Mail", p.Sender, p.Amount, p.Message, claimURL)
	body := `<h1 style="color:#d66720;margin:0 0 20px">Phoenix Wire</h1><p>` + html.EscapeString(p.Sender) + `</p><table role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:30px 20px;background:#eee7ff;border-radius:18px;color:#392e55"><strong style="font-size:32px">` + html.EscapeString(p.Amount) + ` NIM</strong><p>` + html.EscapeString(p.Message) + `</p><a style="display:inline-block;padding:12px 20px;background:#d66720;color:#fff;border-radius:9px;text-decoration:none" href="` + html.EscapeString(claimURL) + `">Open protected payment</a><p>A code will be sent to your email to unlock this payment.</p></td></tr></table><p style="font-size:12px">Powered by Phoenix Mail</p>`
	return m.send(ctx, p.Recipient, "Phoenix Wire — a protected Nimiq payment", plain, body)
}
func (m MailerSend) send(ctx context.Context, to, subject, plain, body string) error {
	if err := m.Validate(); err != nil {
		return err
	}
	a, err := mail.ParseAddress(to)
	if err != nil || a.Address != to {
		return errors.New("invalid_recipient")
	}
	wrapped := `<!doctype html><html><body style="margin:0;background:#f4f2fa;font:16px Arial,sans-serif;color:#292044"><table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:28px 16px"><table role="presentation" width="100%" style="max-width:520px;background:white;border-radius:18px"><tr><td style="padding:28px 24px;line-height:1.6">` + body + `</td></tr></table></td></tr></table></body></html>`
	payload, _ := json.Marshal(map[string]any{"from": map[string]string{"email": m.From, "name": "Phoenix Wire"}, "to": []map[string]string{{"email": to}}, "subject": subject, "text": plain, "html": wrapped})
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mailersend.com/v1/email", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("email_delivery_unavailable")
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 202 {
		return errors.New("email_delivery_unavailable")
	}
	return nil
}
