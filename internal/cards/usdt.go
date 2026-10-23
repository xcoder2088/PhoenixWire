// MIT. Copyright (c) 2026 xcoder2088.
// The e-mail card of a USDC / USDT payment, on the mirrored PhoenixMail renderer with Wire's brand. It carries a claim link whose
// key is in the fragment (the recipient's mail provider sees it, as with any link) and no QR code: the card is protected.
package cards

import (
	_ "embed"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
)

//go:embed assets/usdc-coin.png
var usdcCoin []byte

//go:embed assets/usdt-coin.png
var usdtCoin []byte

// USDTCard is what the card of an invitation needs.
type USDTCard struct {
	ID, Chain, Symbol, Amount, Message, ClaimURL string // Chain: ethereum | polygon; Symbol: USDC | USDT
}

var cardID32 = regexp.MustCompile(`^[a-f0-9]{32}$`)

func validateUSDTCard(c USDTCard) error {
	u, err := url.Parse(c.ClaimURL)
	if err != nil || !cardID32.MatchString(c.ID) || !amount.MatchString(c.Amount) || (c.Chain != "ethereum" && c.Chain != "polygon") || (c.Symbol != "USDC" && c.Symbol != "USDT") ||
		u.Host == "" || u.User != nil || u.Path != "/usdt.html" || !strings.HasPrefix(u.Fragment, "c=") || len([]rune(c.Message)) > 140 || strings.ContainsAny(c.Message, "\r\n") {
		return errors.New("invalid_usdt_card")
	}
	return nil
}

// usdtWords: the texts of the protected USDC / USDT card, in the five languages (the same as PhoenixMail's), with the symbol put in
// place of "USDT" afterwards.
var usdtWords = map[string][4]string{ // protected heading, chip, hint, button
	"fr": {"Paiement protégé", "Protégé", "Un code à six chiffres vous est envoyé par courriel pour confirmer votre adresse, puis vous choisissez où recevoir.", "Recevoir mes USDT"},
	"en": {"Protected payment", "Protected", "We email you a six-digit code to confirm your email, then you choose where to receive.", "Receive my USDT"},
	"es": {"Pago protegido", "Protegido", "Te enviamos por correo un código de seis dígitos para confirmar tu correo y después eliges dónde recibir.", "Recibir mis USDT"},
	"de": {"Geschützte Zahlung", "Geschützt", "Du erhältst einen sechsstelligen Code per E-Mail zur Bestätigung und wählst danach, wo du empfangen willst.", "Meine USDT empfangen"},
	"zh": {"受保护的付款", "受保护", "我们会通过电子邮件发送六位验证码以确认你的邮箱，然后你再选择接收地址。", "领取我的 USDT"},
}

var usdtIntro = map[string]string{
	"fr": "%s vous a envoyé %s %s.", "en": "%s sent you %s %s.", "es": "%s te ha enviado %s %s.", "de": "%s hat dir %s %s gesendet.", "zh": "%s 向您发送了 %s %s。",
}

func chainLabel(chain string) string { return strings.ToUpper(chain[:1]) + chain[1:] }

// USDTCardHTML is the card of the invitation: the colour comes from the card's id (one of the twelve families), like the Nimiq cards.
func USDTCardHTML(c USDTCard, lang string) (string, error) {
	if err := validateUSDTCard(c); err != nil {
		return "", err
	}
	lang = Language(lang)
	w := usdtWords[lang]
	heading, chip, hint, button := w[0], w[1], w[2], w[3]
	coin, alt := usdtCoin, "Tether USDT"
	if c.Symbol == "USDC" {
		coin, alt = usdcCoin, "USD Coin USDC"
		hint, button = strings.ReplaceAll(hint, "USDT", "USDC"), strings.ReplaceAll(button, "USDT", "USDC")
	}
	body := fmt.Sprintf(`<p style="margin:12px 0 6px"><img src="%s" width="64" height="64" alt="%s" style="border-radius:32px"></p>`, dataPNG(coin), alt) +
		`<p style="font-size:14px">` + html.EscapeString(hint) + `</p>`
	if c.Message != "" {
		body += `<p style="font-size:13px">` + html.EscapeString(c.Message) + `</p>`
	}
	colors := PaymentCardColorsFor(c.ID, true)
	out := renderPaymentCard(paymentCardDesign{ID: c.ID, Heading: heading, Network: chip, Amount: c.Amount, BodyHTML: body, ActionURL: c.ClaimURL, ActionLabel: button,
		Brand: c.Symbol + " · " + chainLabel(c.Chain), Unit: c.Symbol, Colors: colors}) // the same gradient stops as the Nimiq card: every card of Wire has one look
	return adaptWireCard(out)
}

// USDTInvitation returns the subject, the plain text and the HTML of the invitation e-mail.
func USDTInvitation(sender string, c USDTCard, lang string) (string, string, string, error) {
	card, err := USDTCardHTML(c, lang)
	if err != nil {
		return "", "", "", err
	}
	lang = Language(lang)
	w := Words(lang)
	intro := fmt.Sprintf(usdtIntro[lang], sender, c.Amount, c.Symbol)
	plain := intro + "\n" + c.Message + "\n" + strings.ReplaceAll(usdtWords[lang][2], "USDT", c.Symbol) + "\n" + c.ClaimURL + "\nPowered by Phoenix Mail — https://phoenixmail.ca"
	instructions := strings.ReplaceAll(usdtWords[lang][2], "USDT", c.Symbol)
	body := `<!doctype html><html lang="` + lang + `"><head><meta name="viewport" content="width=device-width,initial-scale=1"></head><body style="margin:0;background:#fff;color:#20242b;font-family:Arial,Helvetica,sans-serif"><div style="padding:20px 24px;font-size:15px;line-height:1.6">` + html.EscapeString(intro) + `<br>` + html.EscapeString(instructions) + `</div>` + card + Footer(lang) + `</body></html>`
	return w.Subject, plain, body, nil
}
