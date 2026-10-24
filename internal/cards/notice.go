// MIT. Copyright (c) 2026 Xcoder.
package cards

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"strings"
)

// A notice is a short e-mail to the SENDER about a card whose time is running out or over: a reminder an hour before the end, the refund that
// went back to him, or the end of a USDC / USDT card (which he takes back himself). No amount is moved by it and nothing in it opens a payment:
// the one link is Phoenix Wire's own address.
type Notice struct {
	Kind      string // "reminder" | "refunded" | "expired"
	Amount    string
	Symbol    string
	Recipient string
	URL       string // https://… Phoenix Wire's own page
	Auto      bool   // the card is refunded by itself (NIM); otherwise the sender takes the money back himself (USDC / USDT)
}

// the words of each kind and language: subject, then the paragraph. %[1]s the amount, %[2]s the symbol, %[3]s the recipient.
var noticeWords = map[string]map[string][3]string{ // kind -> language -> {subject, paragraph, how it ends}
	"reminder": {
		"fr": {"Ta carte de %[1]s %[2]s expire dans environ une heure", "La carte de %[1]s %[2]s envoyée à %[3]s n’a pas encore été ouverte. Elle expire dans environ une heure.", ""},
		"en": {"Your %[1]s %[2]s card expires in about an hour", "The %[1]s %[2]s card you sent to %[3]s has not been opened yet. It expires in about an hour.", ""},
		"es": {"Tu tarjeta de %[1]s %[2]s caduca en aproximadamente una hora", "La tarjeta de %[1]s %[2]s enviada a %[3]s aún no se ha abierto. Caduca en aproximadamente una hora.", ""},
		"de": {"Deine Karte über %[1]s %[2]s läuft in etwa einer Stunde ab", "Die Karte über %[1]s %[2]s an %[3]s wurde noch nicht geöffnet. Sie läuft in etwa einer Stunde ab.", ""},
		"zh": {"你的 %[1]s %[2]s 卡片约一小时后到期", "发送给 %[3]s 的 %[1]s %[2]s 卡片尚未被打开，约一小时后到期。", ""},
	},
	"refunded": {
		"fr": {"Ta carte de %[1]s %[2]s a été remboursée", "La carte de %[1]s %[2]s envoyée à %[3]s n’a pas été ouverte à temps. L’argent t’a été retourné automatiquement, à l’adresse d’où il venait.", ""},
		"en": {"Your %[1]s %[2]s card was refunded", "The %[1]s %[2]s card you sent to %[3]s was not opened in time. The money was returned to you automatically, to the address it came from.", ""},
		"es": {"Tu tarjeta de %[1]s %[2]s ha sido reembolsada", "La tarjeta de %[1]s %[2]s enviada a %[3]s no se abrió a tiempo. El dinero se te devolvió automáticamente a la dirección de la que venía.", ""},
		"de": {"Deine Karte über %[1]s %[2]s wurde erstattet", "Die Karte über %[1]s %[2]s an %[3]s wurde nicht rechtzeitig geöffnet. Das Geld wurde dir automatisch an die Adresse zurückgeschickt, von der es kam.", ""},
		"zh": {"你的 %[1]s %[2]s 卡片已退款", "发送给 %[3]s 的 %[1]s %[2]s 卡片未在期限内被打开。款项已自动退回到原付款地址。", ""},
	},
	"expired": {
		"fr": {"Ta carte de %[1]s %[2]s a expiré", "La carte de %[1]s %[2]s envoyée à %[3]s a expiré sans être ouverte. Ton argent est toujours sur la carte : ouvre Phoenix Wire et reprends-le.", ""},
		"en": {"Your %[1]s %[2]s card has expired", "The %[1]s %[2]s card you sent to %[3]s expired without being opened. Your money is still on the card: open Phoenix Wire and take it back.", ""},
		"es": {"Tu tarjeta de %[1]s %[2]s ha caducado", "La tarjeta de %[1]s %[2]s enviada a %[3]s caducó sin abrirse. Tu dinero sigue en la tarjeta: abre Phoenix Wire y recupéralo.", ""},
		"de": {"Deine Karte über %[1]s %[2]s ist abgelaufen", "Die Karte über %[1]s %[2]s an %[3]s ist abgelaufen, ohne geöffnet zu werden. Dein Geld ist noch auf der Karte: Öffne Phoenix Wire und hole es zurück.", ""},
		"zh": {"你的 %[1]s %[2]s 卡片已到期", "发送给 %[3]s 的 %[1]s %[2]s 卡片到期时未被打开。你的资金仍在卡片上：请打开 Phoenix Wire 取回。", ""},
	},
}

// how a card ends, said after a reminder: by itself (NIM) or by the sender's own hand (USDC / USDT); the relayer keeps its fee in both.
var noticeEnding = map[bool]map[string]string{
	true: {
		"fr": "S’il n’est pas ouvert d’ici là, l’argent te sera retourné automatiquement, moins les frais du relais s’il y en a.",
		"en": "If it is not opened by then, the money will be returned to you automatically, less the relayer’s fee where there is one.",
		"es": "Si no se abre antes, el dinero se te devolverá automáticamente, menos la comisión del relé si la hay.",
		"de": "Wird sie bis dahin nicht geöffnet, wird dir das Geld automatisch zurückgeschickt, abzüglich der Relay-Gebühr, falls es eine gibt.",
		"zh": "如果届时仍未打开，款项将自动退回给你（如有中继费用则会扣除）。",
	},
	false: {
		"fr": "S’il n’est pas ouvert d’ici là, ouvre Phoenix Wire pour reprendre ton argent. Les frais du relais sont conservés.",
		"en": "If it is not opened by then, open Phoenix Wire to take your money back. The relayer’s fee is kept.",
		"es": "Si no se abre antes, abre Phoenix Wire para recuperar tu dinero. La comisión del relé se conserva.",
		"de": "Wird sie bis dahin nicht geöffnet, öffne Phoenix Wire, um dein Geld zurückzuholen. Die Relay-Gebühr bleibt einbehalten.",
		"zh": "如果届时仍未打开，请打开 Phoenix Wire 取回款项。中继费用将被保留。",
	},
}

var noticeOpen = map[string]string{"fr": "Ouvrir Phoenix Wire", "en": "Open Phoenix Wire", "es": "Abrir Phoenix Wire", "de": "Phoenix Wire öffnen", "zh": "打开 Phoenix Wire"}

// NoticeEmail builds the subject, the plain text and the HTML of a notice.
func NoticeEmail(n Notice, lang string) (string, string, string, error) {
	lang = Language(lang)
	words, ok := noticeWords[n.Kind]
	if !ok || n.Amount == "" || n.Symbol == "" || n.Recipient == "" {
		return "", "", "", errors.New("invalid_notice")
	}
	u, err := url.Parse(n.URL)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", "", "", errors.New("invalid_notice")
	}
	w := words[lang]
	subject := fmt.Sprintf(w[0], n.Amount, n.Symbol, n.Recipient)
	paragraph := fmt.Sprintf(w[1], n.Amount, n.Symbol, n.Recipient)
	if n.Kind == "reminder" {
		paragraph += " " + noticeEnding[n.Auto][lang]
	}
	plain := paragraph + "\n" + noticeOpen[lang] + ": " + n.URL + "\nPowered by Phoenix Mail — https://phoenixmail.ca"
	body := `<!doctype html><html lang="` + lang + `"><head><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light only"><meta name="supported-color-schemes" content="light only"><style>:root{color-scheme:light only;supported-color-schemes:light only}</style></head><body style="margin:0;background:#fff;color:#20242b;font-family:Arial,Helvetica,sans-serif"><div style="padding:24px;font-size:15px;line-height:1.6">` +
		html.EscapeString(paragraph) + `<p style="margin:22px 0 0"><a href="` + html.EscapeString(n.URL) + `" style="display:inline-block;background:#5b46d6;color:#fff;text-decoration:none;font-weight:bold;padding:12px 22px;border-radius:12px">` + html.EscapeString(noticeOpen[lang]) + `</a></p></div>` + Footer(lang) + `</body></html>`
	return strings.TrimSpace(subject), plain, body, nil
}
