package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evmcard"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
)

// USDT cards (Ethereum / Polygon). No money passes through here: the recipient gives his address, the sender pays it
// directly from his wallet, and this service only watches the chain. The issuer (PhoenixMail) creates cards and reads
// their state; the recipient only ever talks to the public claim routes.

func (s *server) evmError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, evmcard.ErrNotFound):
		write(w, 404, map[string]string{"error": "claim_not_found"})
	case errors.Is(err, evmcard.ErrExpired):
		write(w, 410, map[string]string{"error": "claim_expired"})
	case errors.Is(err, evmcard.ErrCancelled):
		write(w, 410, map[string]string{"error": "card_cancelled"})
	case errors.Is(err, evmcard.ErrNotVerified):
		write(w, 403, map[string]string{"error": "email_not_verified"})
	case errors.Is(err, evmcard.ErrForbidden):
		write(w, 403, map[string]string{"error": "forbidden"})
	case errors.Is(err, evmcard.ErrAddressSet):
		write(w, 409, map[string]string{"error": "address_already_given"})
	case errors.Is(err, evmcard.ErrSameAddress):
		write(w, 400, map[string]string{"error": "same_address"})
	case errors.Is(err, evmcard.ErrCapacity):
		write(w, 503, map[string]string{"error": "capacity_reached"})
	case errors.Is(err, evm.ErrInvalidAddress):
		write(w, 400, map[string]string{"error": "invalid_address"})
	case errors.Is(err, evm.ErrInvalidAmount), errors.Is(err, evm.ErrUnknownAsset):
		write(w, 400, map[string]string{"error": err.Error()})
	case errors.Is(err, evm.ErrRPCUnavailable), errors.Is(err, evm.ErrRPCRejected):
		write(w, 502, map[string]string{"error": "chain_unavailable"})
	default:
		write(w, 400, map[string]string{"error": "invalid_request"})
	}
}

// evmIssuer serves /api/v1/evm/cards and /api/v1/evm/cards/{id}?action=status|cancel.
func (s *server) evmIssuer(w http.ResponseWriter, r *http.Request) {
	if s.evm == nil {
		write(w, 503, map[string]string{"error": "usdt_unavailable"})
		return
	}
	if r.Method != http.MethodPost {
		write(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !s.issuerAuthorized(r) {
		write(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/evm/cards"), "/")
	if rest == "" {
		var in struct {
			Asset         string `json:"asset"`
			Amount        string `json:"amount"`
			Email         string `json:"email"`
			Protected     bool   `json:"protected"`
			Message       string `json:"message"`
			SenderEmail   string `json:"sender_email"`
			SenderLang    string `json:"sender_lang"`
			SenderAddress string `json:"sender_address"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		if in.Protected && (s.devConsole || !s.otp.Durable()) {
			write(w, 503, map[string]string{"error": "durable_email_otp_required"})
			return
		}
		created, err := s.evm.Create(evmcard.Input{Asset: in.Asset, Amount: in.Amount, Email: in.Email, Protected: in.Protected, Message: in.Message,
			SenderEmail: in.SenderEmail, SenderLang: in.SenderLang, SenderAddress: in.SenderAddress})
		if err != nil {
			s.evmError(w, err)
			return
		}
		write(w, 201, created)
		return
	}
	id := rest
	switch r.URL.Query().Get("action") {
	case "status":
		st, err := s.evm.Status(ctx, id)
		if err != nil {
			s.evmError(w, err)
			return
		}
		write(w, 200, st)
	case "cancel":
		var in struct {
			SenderToken string `json:"sender_token"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		p, err := s.evm.Cancel(id, in.SenderToken)
		if err != nil {
			s.evmError(w, err)
			return
		}
		write(w, 200, p)
	default:
		write(w, 404, map[string]string{"error": "unknown_action"})
	}
}

// evmClaim serves the public claim routes of a USDT card.
func (s *server) evmClaim(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if r.Method == http.MethodGet {
		if r.URL.Query().Get("action") != "" {
			write(w, 400, map[string]string{"error": "invalid_action"})
			return
		}
		p, err := s.evm.Refresh(ctx, id)
		if err != nil {
			s.evmError(w, err)
			return
		}
		write(w, 200, p)
		return
	}
	if r.Method != http.MethodPost {
		write(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	switch r.URL.Query().Get("action") {
	case "request":
		if s.devConsole || !s.otp.Durable() {
			write(w, 503, map[string]string{"error": "durable_email_otp_required"})
			return
		}
		recipient, err := s.evm.Recipient(id)
		if err != nil {
			s.evmError(w, err)
			return
		}
		// The page sends its language so that the code email is written in it (presentation only).
		var language struct {
			Lang string `json:"lang"`
		}
		if decode(w, r, &language) == nil && len(language.Lang) <= 8 {
			ctx = otp.WithEmailLanguage(ctx, language.Lang)
		}
		if err := s.otp.IssueClaim(ctx, id, recipient); err != nil {
			if errors.Is(err, otp.ErrRateLimited) {
				write(w, 429, map[string]string{"error": "wait_before_resend"})
			} else {
				log.Print("OTP delivery unavailable")
				write(w, 502, map[string]string{"error": "otp_delivery_unavailable"})
			}
			return
		}
		write(w, 200, map[string]string{"status": "sent"})
	case "release":
		var in struct {
			Code string `json:"code"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		recipient, err := s.evm.Recipient(id)
		if err != nil {
			s.evmError(w, err)
			return
		}
		if err := s.otp.Verify(id, recipient, in.Code); err != nil {
			status := 403
			if errors.Is(err, otp.ErrUnavailable) {
				status = 503
			}
			write(w, status, map[string]string{"error": "invalid_or_expired_code"})
			return
		}
		if err := s.evm.MarkVerified(id); err != nil {
			s.evmError(w, err)
			return
		}
		write(w, 200, map[string]string{"status": "otp_verified"})
	case "address":
		var in struct {
			To string `json:"to"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		p, err := s.evm.SetRecipient(ctx, id, in.To)
		if err != nil {
			s.evmError(w, err)
			return
		}
		write(w, 200, p)
	default:
		write(w, 404, map[string]string{"error": "unknown_action"})
	}
}

// evmNotice tells the sender, by email, that the recipient has given his address and that it is time to pay.
func evmNotice(mailer otp.MailerSend, appURL string) func(evmcard.Record) {
	words := map[string][3]string{
		"en": {"Your recipient is ready to receive %s USDT", "The recipient of your %s USDT card gave an address. Open PhoenixMail, then Settings → My wallet (My USDT sends), and scan the QR code to pay: your wallet adds the network fee, as for any transfer.", "Open PhoenixMail"},
		"fr": {"Votre destinataire est prêt à recevoir %s USDT", "Le destinataire de votre carte de %s USDT a donné son adresse. Ouvrez PhoenixMail, puis Paramètres → Mon portefeuille (Mes envois USDT), et scannez le QR code pour payer : votre portefeuille ajoute les frais du réseau, comme pour tout transfert.", "Ouvrir PhoenixMail"},
		"es": {"Tu destinatario está listo para recibir %s USDT", "El destinatario de tu tarjeta de %s USDT ha dado su dirección. Abre PhoenixMail, luego Ajustes → Mi monedero (Mis envíos USDT), y escanea el QR para pagar: tu monedero añade la comisión de red, como en cualquier transferencia.", "Abrir PhoenixMail"},
		"de": {"Dein Empfänger ist bereit, %s USDT zu empfangen", "Der Empfänger deiner %s-USDT-Karte hat seine Adresse angegeben. Öffne PhoenixMail, dann Einstellungen → Meine Wallet (Meine USDT-Sendungen), und scanne den QR-Code zum Bezahlen: Deine Wallet fügt die Netzwerkgebühr hinzu, wie bei jeder Überweisung.", "PhoenixMail öffnen"},
		"zh": {"你的收款人已准备好接收 %s USDT", "你的 %s USDT 卡片的收款人已提供地址。打开 PhoenixMail，进入 设置 → 我的钱包（我的 USDT 发送），扫描二维码付款：你的钱包会像任何转账一样加上网络费用。", "打开 PhoenixMail"},
	}
	return func(r evmcard.Record) {
		if r.SenderEmail == "" {
			return
		}
		w, ok := words[r.SenderLang]
		if !ok {
			r.SenderLang, w = "en", words["en"]
		}
		if sym := evm.Symbol(r.Asset); sym != "USDT" { // the notice names the card's own stablecoin
			w = [3]string{strings.ReplaceAll(w[0], "USDT", sym), strings.ReplaceAll(w[1], "USDT", sym), w[2]}
		}
		heading := fmt.Sprintf(w[0], r.Amount)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := mailer.Notice(ctx, r.SenderEmail, r.SenderLang, heading, heading, fmt.Sprintf(w[1], r.Amount), w[2], appURL); err != nil {
			log.Printf("sender notice not delivered: %v", err)
		}
	}
}
