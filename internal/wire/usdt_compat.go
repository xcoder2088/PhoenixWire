package wire

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/xcoder2088/PhoenixWire/internal/qrcode"
)

// PhoenixMail's payment window talks to /api/usdt/cashlink/*. Wire serves the same routes, in the same shape, so the window is used
// as it is: each one is answered by the gateway's own USDC / USDT card logic (sender session, card ownership, one engine).

// compatCall runs one of the gateway's handlers on a body and gives back what it answered.
func compatCall(run func(w http.ResponseWriter, r *http.Request), r *http.Request, body any) (map[string]json.RawMessage, int) {
	raw, _ := json.Marshal(body)
	inner := r.Clone(r.Context())
	inner.Body = nopCloser{bytes.NewReader(raw)}
	inner.ContentLength = int64(len(raw))
	inner.Header = http.Header{"Content-Type": {"application/json"}}
	rec := httptest.NewRecorder()
	run(rec, inner)
	out := map[string]json.RawMessage{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return out, rec.Code
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

// notReady is what the window shows (it prints a sentence it does not know) when the relayer cannot take a new card yet: its gas is below the
// reserve that keeps claims and refunds payable. Said in the sender's language, never as a code.
var notReady = map[string]string{
	"fr": "Wire ne peut pas prendre de nouvelle carte pour le moment : sa réserve de gaz est en cours de remplissage. Réessayez plus tard.",
	"en": "Wire cannot take a new card right now: its gas reserve is being refilled. Try again later.",
	"es": "Wire no puede aceptar una tarjeta nueva por ahora: su reserva de gas se está rellenando. Inténtalo más tarde.",
	"de": "Wire kann gerade keine neue Karte annehmen: Die Gas-Reserve wird aufgefüllt. Versuche es später erneut.",
	"zh": "Wire 目前无法接收新卡片：其燃料储备正在补充。请稍后再试。",
}

// compatFail answers the way PhoenixMail's routes do: {"error": code}.
func compatFail(w http.ResponseWriter, status int, out map[string]json.RawMessage, lang string) {
	code := "invalid_request"
	if c := str(out, "code"); c != "" {
		code = c
	}
	if code == "relayer_float_low" {
		if text, ok := notReady[lang]; ok {
			code = text
		} else {
			code = notReady["en"]
		}
	}
	respond(w, status, map[string]string{"error": code})
}

func (a *API) usdtCompat(w http.ResponseWriter, r *http.Request, s session, engine USDTEngine) {
	action := strings.TrimPrefix(r.URL.Path, "/api/usdt/cashlink/")
	var in struct {
		Asset         string `json:"asset"`
		CardID        string `json:"cardId"`
		LinkOwner     string `json:"linkOwner"`
		Amount        string `json:"amount"`
		Message       string `json:"message"`
		Recipient     string `json:"recipient"`
		SenderEmail   string `json:"senderEmail"`
		Lang          string `json:"lang"`
		ID            string `json:"id"`
		SenderToken   string `json:"senderToken"`
		To            string `json:"to"`
		Signatures    string `json:"signatures"`
		SenderAddress string `json:"senderAddress"`
	}
	if decode(w, r, &in) != nil {
		respond(w, 400, map[string]string{"error": "invalid_request"})
		return
	}
	switch action {
	case "create":
		out, status := compatCall(func(w http.ResponseWriter, r *http.Request) { a.usdtCreate(w, r, s, engine) }, r, map[string]string{"asset": in.Asset, "card_id": in.CardID,
			"link_owner": in.LinkOwner, "amount": in.Amount, "recipient": in.Recipient, "message": in.Message, "language": in.Lang})
		if status != 200 {
			compatFail(w, status, out, in.Lang)
			return
		}
		var pay map[string]any
		_ = json.Unmarshal(out["pay"], &pay)
		link, _ := pay["link"].(string)
		png, err := qrcode.Encode(link, qrcode.Medium, 256)
		if err != nil || !strings.HasPrefix(link, "ethereum:0x") {
			respond(w, 502, map[string]string{"error": "invalid_service_answer"})
			return
		}
		pay["qr"] = base64.StdEncoding.EncodeToString(png)
		// The browser keeps nothing secret here: Wire holds the card's sender token and checks the sender's session on every call.
		respond(w, 200, map[string]any{"card": out["card"], "pay": pay, "claimBase": a.origin + "/usdt.html", "senderToken": in.CardID})
	case "gas-status":
		respond(w, 200, map[string]any{"available": false, "reason": "off"})
	case "gas":
		respond(w, 503, map[string]string{"error": "gas_service_unavailable"})
	case "status", "cancel", "refund", "refund-claim":
		c, ok := a.db.state.USDT[in.ID]
		if !ok || c.Sender != s.Email || !cardID32.MatchString(in.ID) {
			respond(w, 404, map[string]string{"error": "not_found"})
			return
		}
		body := map[string]string{}
		if action == "refund" {
			body["to"] = in.To
		}
		if action == "refund-claim" {
			body["signatures"] = in.Signatures
		}
		out, status := compatCall(func(w http.ResponseWriter, r *http.Request) { a.usdtCardAction(w, r, engine, c, action) }, r, body)
		if status != 200 {
			compatFail(w, status, out, in.Lang)
			return
		}
		if v, ok := out["balance_units"]; ok { // the window reads camelCase
			out["balanceUnits"] = v
		}
		respond(w, 200, out)
	default:
		respond(w, 404, map[string]string{"error": "not_found"})
	}
}
