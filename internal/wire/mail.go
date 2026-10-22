package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/service"
	"github.com/xcoder2088/PhoenixWire/internal/cards"
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
	return (service.MailerSend{APIKey: m.APIKey, From: m.From, BrandName: "Phoenix Wire", Purpose: "sender", Client: m.Client}).Send(ctx, to, code)
}
func (m MailerSend) Invite(ctx context.Context, p payment, claimURL string) error {
	// Recipient gets an opaque claim URL only. No address QR, seed or bearer Cashlink.
	subject, plain, body, err := cards.Invitation(p.Sender, cards.ProtectedCashlinkCard{ClaimID: p.ID, ClaimURL: claimURL, Amount: p.Amount, Network: "mainnet", Message: p.Message, Address: p.Address}, p.Language, p.Nimicon)
	if err != nil {
		return err
	}
	return m.send(ctx, p.Recipient, subject, plain, body)
}
func (m MailerSend) send(ctx context.Context, to, subject, plain, body string) error {
	if err := m.Validate(); err != nil {
		return err
	}
	a, err := mail.ParseAddress(to)
	if err != nil || a.Address != to {
		return errors.New("invalid_recipient")
	}
	payload, _ := json.Marshal(map[string]any{"from": map[string]string{"email": m.From, "name": "Phoenix Wire"}, "to": []map[string]string{{"email": to}}, "subject": subject, "text": plain, "html": body, "settings": map[string]bool{"track_opens": false, "track_clicks": false, "track_content": false}})
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mailersend.com/v1/email", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PhoenixWire/0.3.0")
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
