package otp

import (
	"context"
	"strings"
)

type emailLanguageKey struct{}

// WithEmailLanguage selects presentation only; it never changes code validation.
func WithEmailLanguage(ctx context.Context, language string) context.Context {
	language = strings.ToLower(strings.Split(strings.ReplaceAll(language, "_", "-"), "-")[0])
	return context.WithValue(ctx, emailLanguageKey{}, language)
}

func mailWords(ctx context.Context, sender bool) (string, [8]string) {
	rows := map[string][8]string{
		"en": {"Your %s verification code", "Your %s verification code is %s.", "Your protected payment", "Enter this verification code on the claim page to unlock your payment:", "Expires in 10 minutes. Do not share this code.", "If you did not request this code, ignore this email.", "Verify your sender email", "Enter this code in Phoenix Wire to verify your email before sending a payment:"},
		"fr": {"Votre code de vérification %s", "Votre code de vérification %s est %s.", "Votre paiement protégé", "Entrez ce code sur la page de réclamation pour déverrouiller votre paiement :", "Expire dans 10 minutes. Ne partagez pas ce code.", "Si vous n’avez pas demandé ce code, ignorez ce courriel.", "Vérifiez votre courriel d’expéditeur", "Entrez ce code dans Phoenix Wire pour vérifier votre courriel avant d’envoyer un paiement :"},
		"es": {"Tu código de verificación de %s", "Tu código de verificación de %s es %s.", "Tu pago protegido", "Introduce este código en la página de reclamación para desbloquear tu pago:", "Caduca en 10 minutos. No compartas este código.", "Si no solicitaste este código, ignora este correo.", "Verifica tu correo de remitente", "Introduce este código en Phoenix Wire para verificar tu correo antes de enviar un pago:"},
		"de": {"Dein Bestätigungscode für %s", "Dein Bestätigungscode für %s lautet %s.", "Deine geschützte Zahlung", "Gib diesen Code auf der Einlöseseite ein, um deine Zahlung freizugeben:", "Läuft in 10 Minuten ab. Teile diesen Code nicht.", "Ignoriere diese E-Mail, wenn du den Code nicht angefordert hast.", "Bestätige deine Absenderadresse", "Gib diesen Code in Phoenix Wire ein, um vor einer Zahlung deine E-Mail zu bestätigen:"},
		"zh": {"您的 %s 验证码", "您的 %s 验证码是 %s。", "您的受保护付款", "请在领取页面输入此验证码以解锁付款：", "10 分钟后过期，请勿分享此验证码。", "如果您未请求此验证码，请忽略此邮件。", "验证您的发件邮箱", "在 Phoenix Wire 中输入此验证码，验证邮箱后即可发送付款："},
	}
	lang, _ := ctx.Value(emailLanguageKey{}).(string)
	words, ok := rows[lang]
	if !ok {
		lang = "en"
		words = rows[lang]
	}
	if sender {
		words[2], words[3] = words[6], words[7]
	}
	return lang, words
}

type emailAccentKey struct{}

// WithEmailAccent colours the code e-mail like the card it belongs to (background and accent, #rrggbb). Presentation only.
func WithEmailAccent(ctx context.Context, background, accent string) context.Context {
	if !validHex(background) || !validHex(accent) {
		return ctx
	}
	return context.WithValue(ctx, emailAccentKey{}, [2]string{background, accent})
}

func emailAccent(ctx context.Context) ([2]string, bool) {
	v, ok := ctx.Value(emailAccentKey{}).([2]string)
	return v, ok
}

func validHex(c string) bool {
	if len(c) != 7 || c[0] != '#' {
		return false
	}
	for _, r := range c[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}
