package otp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

// Notice sends a short, brand-styled information email (a heading, a line, one button) with the same header, sender and
// tracking-off rules as the verification codes. It carries no code and decides nothing.
func (m MailerSend) Notice(ctx context.Context, to, lang, subject, heading, line, button, buttonURL string) error {
	if err := m.Validate(); err != nil {
		return err
	}
	recipient, err := mail.ParseAddress(to)
	if err != nil || recipient.Address != to {
		return errors.New("notice_recipient_invalid")
	}
	u, err := url.Parse(buttonURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return errors.New("notice_link_invalid")
	}
	brand, first, second := m.brand()
	name := html.EscapeString(first)
	if second != "" {
		name += `<span style="color:#f39043">` + html.EscapeString(second) + `</span>`
	}
	body := `<!doctype html><html lang="` + html.EscapeString(lang) + `"><head><meta name="viewport" content="width=device-width,initial-scale=1"></head><body style="margin:0;background:#f4f2fa;color:#292044;font-family:Arial,sans-serif"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="background:#f4f2fa"><tr><td align="center" style="padding:28px 16px"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="max-width:520px;background:#fff;border:1px solid #ded8ec;border-radius:18px"><tr><td style="padding:28px 24px;font-size:16px;line-height:1.6"><table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin:0 0 22px"><tr><td style="padding-right:12px"><a href="https://phoenixmail.ca" target="_blank" rel="noopener" style="text-decoration:none;border:0"><img src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(logoPNG) + `" width="60" height="58" alt="" style="display:block;border:0"></a></td><td style="font-size:32px;line-height:34px;font-weight:bold;letter-spacing:-1px;color:#1a164d"><a href="https://phoenixmail.ca" target="_blank" rel="noopener" style="text-decoration:none;color:#1a164d">` + name + `</a></td></tr></table><h2 style="margin:0 0 20px;font-size:20px">` + html.EscapeString(heading) + `</h2><p style="margin:0 0 24px">` + html.EscapeString(line) + `</p><p style="margin:0"><a href="` + html.EscapeString(buttonURL) + `" style="display:inline-block;text-decoration:none;padding:13px 20px;border-radius:12px;background:#8170ed;color:#ffffff;font-weight:bold;font-size:15px">` + html.EscapeString(button) + `</a></p></td></tr></table></td></tr></table></body></html>`
	payload, err := json.Marshal(map[string]any{
		"from":     map[string]string{"email": m.From, "name": brand},
		"to":       []map[string]string{{"email": to}},
		"subject":  subject,
		"text":     heading + "\n\n" + line + "\n\n" + buttonURL + "\n",
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
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("notice_delivery_failed_status_%d", response.StatusCode)
	}
	return nil
}
