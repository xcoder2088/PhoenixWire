package otp

import (
	"bytes"
	"context"
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

// MailerSend sends claim verification emails without sharing PhoenixMail login
// sessions or account OTP endpoints. Credentials remain server-side.
type MailerSend struct {
	APIKey    string
	From      string
	BrandName string
	Client    *http.Client
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
	brand := m.BrandName
	if brand == "" {
		brand = "NimiqSecureClaim"
	}
	subject := brand + " — your verification code"
	plain := fmt.Sprintf("Your %s verification code is %s.\nIt expires in 10 minutes. Never share this code.\nIf you did not request it, ignore this email.\n", brand, code)
	body := `<!doctype html><html lang="en"><head><meta name="viewport" content="width=device-width,initial-scale=1"></head><body style="margin:0;background:#f4f2fa;color:#292044;font-family:Arial,sans-serif"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="background:#f4f2fa"><tr><td align="center" style="padding:28px 16px"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="max-width:520px;background:#fff;border:1px solid #ded8ec;border-radius:18px"><tr><td style="padding:28px 24px;font-size:16px;line-height:1.6"><h1 style="margin:0 0 20px;font-size:28px;color:#e87416">PhoenixMail</h1><h2 style="margin:0 0 20px;font-size:20px">Your protected Nimiq payment</h2><p style="margin:0 0 16px">Enter this verification code on the claim page to unlock your payment:</p><table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0"><tr><td align="center" style="padding:20px 12px;background:#eee7ff;border-radius:12px;font-size:32px;font-weight:bold;letter-spacing:6px;color:#583b92">` + html.EscapeString(code) + `</td></tr></table><p style="margin:20px 0 16px">Expires in 10 minutes. Do not share this code.</p><p style="margin:0;color:#626477">If you did not request this code, ignore this email.</p></td></tr></table></td></tr></table></body></html>`
	if m.BrandName != "" {
		body = strings.Replace(body, ">PhoenixMail</h1>", ">"+html.EscapeString(m.BrandName)+"</h1>", 1)
	}
	payload, err := json.Marshal(map[string]any{
		"from":    map[string]string{"email": m.From, "name": brand},
		"to":      []map[string]string{{"email": to}},
		"subject": subject,
		"text":    plain,
		"html":    body,
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
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("otp_mail_delivery_failed_status_%d", response.StatusCode)
	}
	return nil
}
