package otp

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"html"
)

// The code e-mail of PhoenixWire is drawn like the card it belongs to: the same frame, gradient, rounded corners, logo, "Protected" label
// and white inner panel as the card Wire sends (internal/cards/renderer_mirror.go). Everything that comes from Wire looks like one thing.
//
//go:embed assets/wire-email-logo.png
var wireLogoPNG []byte

const (
	cardMid = "#e2eeff" // the gradient stops of every Wire card
	cardEnd = "#ffe8f4"
)

// lavender is the card colour of an e-mail that belongs to no card (the sender's sign-in).
var lavender = [2]string{"#ece3ff", "#c7b1ee"}

var protectedLabel = map[string]string{"en": "Protected", "fr": "Protégé", "es": "Protegido", "de": "Geschützt", "zh": "受保护"}

// cardLayout returns the HTML of a code e-mail in English; the words are translated afterwards, the same way as for the other layout.
// background and border are the card's colours; withLabel adds the "Protected" label next to the logo.
func cardLayout(brand, code, background, border string, withLabel bool) string {
	label := ""
	if withLabel {
		label = `<span style="display:inline-block;padding:7px 9px;border:1px solid ` + border + `;border-radius:22px;background:#ffffff;color:#6752a2;font-size:10px;font-weight:bold">@@PROTECTED@@</span>`
	}
	return fmt.Sprintf(`<!doctype html><html lang="en"><head><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light only"><meta name="supported-color-schemes" content="light only"></head><body style="margin:0;padding:0;background:#ffffff"><table role="presentation" align="center" cellpadding="0" cellspacing="0" width="100%%" style="max-width:440px;width:100%%;margin:20px auto;border:1px solid %[2]s;border-radius:26px;background:%[1]s;background-image:linear-gradient(125deg,%[1]s 0%%,%[5]s 56%%,%[6]s 100%%);color:#211e49;font-family:Arial,Helvetica,sans-serif"><tr><td style="padding:22px 22px 20px;text-align:center">
<table role="presentation" cellpadding="0" cellspacing="0" width="100%%" style="border-collapse:collapse;margin-bottom:14px"><tr><td valign="middle" style="text-align:left"><img src="data:image/png;base64,%[3]s" width="215" alt="%[4]s" style="display:block;width:100%%;max-width:215px;height:auto;border:0"></td><td valign="middle" align="right">%[7]s</td></tr></table>
<p style="font-size:12px;letter-spacing:1.7px;text-transform:uppercase;color:#7461aa;font-weight:bold;margin:18px 0 12px">Your protected payment</p>
<table role="presentation" cellpadding="0" cellspacing="0" width="100%%" style="background:#ffffff;border:1px solid #ffffff;border-radius:20px"><tr><td style="padding:19px 14px 21px;text-align:center">
<p style="margin:0 0 14px;font-size:14px;line-height:1.55;color:#49415f">Enter this verification code on the claim page to unlock your payment:</p>
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0"><tr><td align="center" style="padding:16px 12px;background:%[1]s;border:1px solid %[2]s;border-radius:14px;font-size:32px;font-weight:bold;letter-spacing:6px;color:#583b92;-webkit-user-select:all;user-select:all">%[8]s</td></tr></table>
<p style="margin:16px 0 8px;font-size:13px;line-height:1.5;color:#49415f">Expires in 10 minutes. Do not share this code.</p>
<p style="margin:0;font-size:12px;line-height:1.5;color:#73759c">If you did not request this code, ignore this email.</p>
</td></tr></table><div style="height:8px;line-height:8px;font-size:8px">&nbsp;</div></td></tr></table></body></html>`,
		background, border, base64.StdEncoding.EncodeToString(wireLogoPNG), html.EscapeString(brand), cardMid, cardEnd, label, html.EscapeString(code))
}
