// MIT. Copyright (c) 2026 xcoder2088.
// Wire adapter around the mirrored PhoenixMail renderer. No spending secrets.
package cards

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"image/png"
	"net"
	"net/url"
	"regexp"
	"strings"
)

//go:embed assets/email-wire.png
var wireLogo []byte

//go:embed assets/email-icon.png
var footerIcon []byte

//go:embed assets/email-wordmark.png
var footerWordmark []byte

const phoenixMailFooterIconCID = "wire-footer-icon"
const phoenixMailFooterCID = "wire-footer-wordmark"

type ProtectedCashlinkCard struct{ ClaimID, ClaimURL, Amount, Network, Message, Address string }

var claimID = regexp.MustCompile(`^[a-f0-9]{48}$`)
var amount = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,5})?$`)

func validateProtectedCard(c ProtectedCashlinkCard) error {
	u, err := url.Parse(c.ClaimURL)
	if err != nil {
		return errors.New("invalid_protected_card")
	}
	local := u.Hostname() == "localhost" || net.ParseIP(u.Hostname()) != nil && net.ParseIP(u.Hostname()).IsLoopback()
	if !claimID.MatchString(c.ClaimID) || !amount.MatchString(c.Amount) || c.Network != "mainnet" || u.Host == "" || u.User != nil || u.Fragment != "" || u.Path != "/" || u.RawQuery != "claim="+c.ClaimID || (u.Scheme != "https" && !(u.Scheme == "http" && local)) || strings.EqualFold(u.Hostname(), "hub.nimiq.com") {
		return errors.New("invalid_protected_card")
	}
	return nil
}
func ProtectedCashlinkIqonCID(c ProtectedCashlinkCard) string { return "wire-iqon-" + c.ClaimID }
func dataPNG(b []byte) string                                 { return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b) }

// CleanNimicon accepts only a bounded raster image and strips all metadata.
// This is decorative browser-supplied artwork, never proof of an address/payment.
func CleanNimicon(encoded string) ([]byte, error) {
	if len(encoded) > 90000 {
		return nil, errors.New("invalid_nimicon")
	}
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("invalid_nimicon")
	}
	config, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil || config.Width != 160 || config.Height != 160 {
		return nil, errors.New("invalid_nimicon")
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, errors.New("invalid_nimicon")
	}
	var out bytes.Buffer
	if err = png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func Footer(lang string) string {
	out := phoenixMailFooterHTML(Language(lang))
	out = strings.ReplaceAll(out, "cid:"+phoenixMailFooterIconCID, dataPNG(footerIcon))
	return strings.ReplaceAll(out, "cid:"+phoenixMailFooterCID, dataPNG(footerWordmark))
}
func Language(lang string) string {
	lang = strings.ToLower(strings.Split(strings.ReplaceAll(lang, "_", "-"), "-")[0])
	switch lang {
	case "fr", "en", "es", "de", "zh":
		return lang
	}
	return "en"
}

// Protected uses the same pastel family, copy and geometry as PhoenixMail.
// Wire changes only its full brand logo, data: image transport and the requested
// action placement outside the white box (18px gap). The mirror stays untouched.
func Protected(c ProtectedCashlinkCard, lang, nimicon string) (string, error) {
	image, err := CleanNimicon(nimicon)
	if err != nil {
		return "", err
	}
	out, err := ProtectedCashlinkHTML(c, Language(lang), true)
	if err != nil {
		return "", err
	}
	out = strings.ReplaceAll(out, "cid:"+ProtectedCashlinkIqonCID(c), dataPNG(image))
	return adaptWireCard(out)
}

// adaptWireCard gives a mirrored PhoenixMail card Wire's brand and the action placement (18px below the white box).
func adaptWireCard(out string) (string, error) {
	start := strings.Index(out, `<td valign="middle" style="text-align:left">`)
	if start < 0 {
		return "", errors.New("card_mirror_changed")
	}
	end := strings.Index(out[start:], `</td>`) + start
	if start < 0 || end < start {
		return "", errors.New("card_mirror_changed")
	}
	brand := fmt.Sprintf(`<td valign="middle" style="text-align:left"><img src="%s" width="215" alt="Phoenix Wire" style="display:block;width:100%%;max-width:215px;height:auto;border:0">`, dataPNG(wireLogo))
	out = out[:start] + brand + out[end:]
	actionStart := strings.Index(out, `<a href="`)
	if actionStart < 0 {
		return "", errors.New("card_mirror_changed")
	}
	actionEnd := strings.Index(out[actionStart:], `</a>`) + actionStart + 4
	action := strings.Replace(out[actionStart:actionEnd], "margin:12px auto 4px", "margin:18px auto 0", 1)
	action = strings.Replace(action, "background:#8170ed", "border:1.5px solid #8170ed;background:#eee9fc", 1)
	action = strings.Replace(action, "color:#ffffff", "color:#211c49", 1)
	out = out[:actionStart] + out[actionEnd:]
	marker := "</td></tr></table>\n<div style=\"height:8px"
	if !strings.Contains(out, marker) {
		return "", errors.New("card_mirror_changed")
	}
	out = strings.Replace(out, marker, "</td></tr></table>\n"+action+"\n<div style=\"height:8px", 1)
	return out, nil
}

type InvitationWords struct{ Subject, Intro, Instructions string }

func Words(lang string) InvitationWords {
	rows := map[string]InvitationWords{
		"fr": {"Phoenix Wire — un paiement protégé", "%s vous a envoyé %s NIM.", "Ouvrez votre carte. Un code envoyé à votre courriel vous permettra de recevoir les NIM."},
		"en": {"Phoenix Wire — a protected payment", "%s sent you %s NIM.", "Open your card. A code sent to your email will let you receive the NIM."},
		"es": {"Phoenix Wire — un pago protegido", "%s te ha enviado %s NIM.", "Abre tu tarjeta. Un código enviado a tu correo te permitirá recibir los NIM."},
		"de": {"Phoenix Wire — eine geschützte Zahlung", "%s hat dir %s NIM gesendet.", "Öffne deine Karte. Mit einem Code per E-Mail kannst du die NIM empfangen."},
		"zh": {"Phoenix Wire — 受保护的付款", "%s 向您发送了 %s NIM。", "打开卡片，使用发送至您邮箱的验证码领取 NIM。"},
	}
	return rows[Language(lang)]
}
func Invitation(sender string, c ProtectedCashlinkCard, lang, nimicon string) (string, string, string, error) {
	card, err := Protected(c, lang, nimicon)
	if err != nil {
		return "", "", "", err
	}
	w := Words(lang)
	intro := fmt.Sprintf(w.Intro, sender, c.Amount)
	plain := intro + "\n" + c.Message + "\n" + w.Instructions + "\n" + c.ClaimURL + "\nPowered by Phoenix Mail — https://phoenixmail.ca"
	body := `<!doctype html><html lang="` + Language(lang) + `"><head><meta name="viewport" content="width=device-width,initial-scale=1"></head><body style="margin:0;background:#fff;color:#20242b;font-family:Arial,Helvetica,sans-serif"><div style="padding:20px 24px;font-size:15px;line-height:1.6">` + html.EscapeString(intro) + `<br>` + html.EscapeString(w.Instructions) + `</div>` + card + Footer(lang) + `</body></html>`
	return w.Subject, plain, body, nil
}
