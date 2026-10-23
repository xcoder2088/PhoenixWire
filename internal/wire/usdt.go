package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xcoder2088/NimiqSecureClaim/service"
	"github.com/xcoder2088/PhoenixWire/internal/cards"

	"net/http"
)

// USDC / USDT cards. The sender's browser makes the card id and the link key and keeps them (they are what lets the sender take the
// money back); Wire's engine makes the card's two-owner Safe address and pays the gas of its claim from its own relayer. Wire never
// stores the link key: the browser hands the claim link to Wire once, only to be e-mailed to the recipient, after the card is funded.

type usdtCard struct {
	ID, Asset, Chain, Symbol     string
	Sender, Recipient, Amount    string
	Message, Language            string
	LinkOwner, Safe, SenderToken string
	Created, LastInvite          time.Time
	Invites                      int
	Invited                      bool
}

type usdtInvite struct {
	Sender, Recipient string
	Card              cards.USDTCard
}

var cardID32 = regexp.MustCompile(`^[a-f0-9]{32}$`)
var evmAddress = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
var assetChain = map[string][2]string{"usdt_ethereum": {"ethereum", "USDT"}, "usdc_polygon": {"polygon", "USDC"}, "usdt_polygon": {"polygon", "USDT"}}

const (
	usdtCardsPerDay  = 20
	usdtInviteSpace  = 30 * time.Second
	usdtInviteMax    = 6
	usdtCardLifetime = 14 * 24 * time.Hour
)

// engineAnswer parses the engine's JSON answer; a status that is not a success becomes a failure with the engine's own error code.
func engineAnswer(status int, body []byte, err error) (map[string]json.RawMessage, int, string) {
	if err != nil {
		return nil, 502, "usdt_unavailable"
	}
	var out map[string]json.RawMessage
	if json.Unmarshal(body, &out) != nil {
		return nil, 502, "invalid_service_answer"
	}
	if status < 200 || status >= 300 {
		code := "invalid_request"
		var s string
		if json.Unmarshal(out["error"], &s) == nil && regexp.MustCompile(`^[a-z0-9_]{1,63}$`).MatchString(s) {
			code = s
		}
		if status < 400 || status > 599 {
			status = 502
		}
		return nil, status, code
	}
	return out, 0, ""
}

func str(m map[string]json.RawMessage, key string) string {
	var s string
	_ = json.Unmarshal(m[key], &s)
	return s
}
func num(m map[string]json.RawMessage, key string) uint64 {
	var n uint64
	_ = json.Unmarshal(m[key], &n)
	return n
}

func formatUnits(u uint64) string {
	s := fmt.Sprintf("%d.%06d", u/1_000_000, u%1_000_000)
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}

func (a *API) usdtServe(w http.ResponseWriter, r *http.Request, s session) {
	engine := a.USDT()
	if engine == nil {
		failure(w, 503, "usdt_unavailable")
		return
	}
	path := r.URL.Path
	switch {
	case path == "/api/usdt/create" && r.Method == "POST":
		a.usdtCreate(w, r, s, engine)
	case path == "/api/usdt/cards" && r.Method == "GET":
		list := []map[string]any{}
		for _, c := range a.db.state.USDT {
			if c.Sender == s.Email {
				list = append(list, map[string]any{"id": c.ID, "asset": c.Asset, "chain": c.Chain, "symbol": c.Symbol, "amount": c.Amount, "recipient": c.Recipient, "message": c.Message, "created": c.Created, "invited": c.Invited, "safe": c.Safe})
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i]["created"].(time.Time).After(list[j]["created"].(time.Time)) })
		if len(list) > 50 {
			list = list[:50]
		}
		respond(w, 200, map[string]any{"cards": list})
	case strings.HasPrefix(path, "/api/usdt/card/") && r.Method == "POST":
		parts := strings.Split(strings.TrimPrefix(path, "/api/usdt/card/"), "/")
		if len(parts) != 2 || !cardID32.MatchString(parts[0]) {
			failure(w, 404, "not_found")
			return
		}
		c, ok := a.db.state.USDT[parts[0]]
		if !ok || c.Sender != s.Email {
			failure(w, 404, "not_found")
			return
		}
		a.usdtCardAction(w, r, engine, c, parts[1])
	default:
		failure(w, 404, "not_found")
	}
}

func (a *API) usdtCreate(w http.ResponseWriter, r *http.Request, s session, engine USDTEngine) {
	var in struct {
		Asset     string `json:"asset"`
		CardID    string `json:"card_id"`
		LinkOwner string `json:"link_owner"`
		Amount    string `json:"amount"`
		Recipient string `json:"recipient"`
		Message   string `json:"message"`
		Language  string `json:"language"`
	}
	if decode(w, r, &in) != nil {
		failure(w, 400, "invalid_request")
		return
	}
	ac, known := assetChain[in.Asset]
	addr, err := mail.ParseAddress(strings.TrimSpace(in.Recipient))
	if !known || !service.USDTAssetKnown(in.Asset) || !cardID32.MatchString(in.CardID) || !evmAddress.MatchString(in.LinkOwner) || err != nil || addr.Address != strings.TrimSpace(in.Recipient) || len(in.Recipient) > 254 ||
		utf8.RuneCountInString(in.Message) > 140 || strings.ContainsAny(in.Message, "\r\n") || len(in.Amount) > 12 {
		failure(w, 400, "invalid_request")
		return
	}
	lang := cards.Language(in.Language)
	if _, exists := a.db.state.USDT[in.CardID]; exists {
		failure(w, 409, "card_exists")
		return
	}
	recent := 0
	for _, c := range a.db.state.USDT {
		if c.Sender == s.Email && a.now().Sub(c.Created) < 24*time.Hour {
			recent++
		}
	}
	if recent >= usdtCardsPerDay {
		failure(w, 429, "rate_limited")
		return
	}
	body, _ := json.Marshal(map[string]string{"asset": in.Asset, "card_id": in.CardID, "link_owner": in.LinkOwner, "amount": in.Amount, "email": in.Recipient,
		"message": in.Message, "sender_email": s.Email, "sender_lang": lang})
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	created, status, code := engineAnswer(engine.IssuerCall(ctx, "POST", "/api/v1/usdt/issuer/cards", body))
	if code != "" {
		failure(w, status, code)
		return
	}
	units, fee, refundFee := num(created, "units"), num(created, "fee_units"), num(created, "refund_fee_units")
	var protected bool
	_ = json.Unmarshal(created["protected"], &protected)
	var owners []string
	_ = json.Unmarshal(created["owners"], &owners)
	safe, token := str(created, "safe"), str(created, "sender_token")
	if str(created, "id") != in.CardID || str(created, "asset") != in.Asset || !protected || str(created, "state") != "awaiting_funds" || len(owners) != 2 || units == 0 || fee == 0 || refundFee == 0 || refundFee > fee || len(token) < 20 || len(token) > 64 {
		failure(w, 502, "invalid_service_answer")
		return
	}
	link, err := service.USDTPayLink(in.Asset, safe, units+fee)
	if err != nil {
		failure(w, 502, "invalid_service_answer")
		return
	}
	card := usdtCard{ID: in.CardID, Asset: in.Asset, Chain: ac[0], Symbol: ac[1], Sender: s.Email, Recipient: in.Recipient, Amount: str(created, "amount"), Message: in.Message, Language: lang,
		LinkOwner: in.LinkOwner, Safe: safe, SenderToken: token, Created: a.now()}
	a.db.state.USDT[card.ID] = card
	if a.db.save() != nil {
		delete(a.db.state.USDT, card.ID)
		failure(w, 503, "state_unavailable")
		return
	}
	respond(w, 200, map[string]any{"card": map[string]any{"id": card.ID, "asset": card.Asset, "chain": card.Chain, "symbol": card.Symbol, "amount": card.Amount, "units": units, "feeUnits": fee,
		"refundFeeUnits": refundFee, "chainId": num(created, "chain_id"), "safe": safe, "owners": owners, "token": str(created, "token"), "relayer": str(created, "relayer")},
		"pay": map[string]any{"recipient": safe, "units": units + fee, "amount": formatUnits(units + fee), "link": link}})
}

func (a *API) usdtCardAction(w http.ResponseWriter, r *http.Request, engine USDTEngine, c usdtCard, action string) {
	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()
	call := func(path string, payload map[string]any, keep ...string) {
		body, _ := json.Marshal(payload)
		out, status, code := engineAnswer(engine.IssuerCall(ctx, "POST", "/api/v1/usdt/issuer/cards/"+c.ID+path, body))
		if code != "" {
			failure(w, status, code)
			return
		}
		pick := map[string]json.RawMessage{}
		for _, k := range keep {
			if v, ok := out[k]; ok {
				pick[k] = v
			}
		}
		respond(w, 200, pick)
	}
	switch action {
	case "status":
		call("/status", map[string]any{}, "state", "balance_units", "tx_hash", "tx_kind", "recovering", "stale")
	case "cancel":
		call("/cancel", map[string]any{"sender_token": c.SenderToken}, "state")
	case "refund":
		var in struct {
			To string `json:"to"`
		}
		if decode(w, r, &in) != nil || !evmAddress.MatchString(in.To) {
			failure(w, 400, "invalid_request")
			return
		}
		call("/refund", map[string]any{"sender_token": c.SenderToken, "to": in.To}, "owner", "signature", "units", "nonce", "fee_units")
	case "refund-claim":
		var in struct {
			Signatures string `json:"signatures"`
		}
		if decode(w, r, &in) != nil || !regexp.MustCompile(`^0x[0-9a-fA-F]{130,390}$`).MatchString(in.Signatures) {
			failure(w, 400, "invalid_request")
			return
		}
		call("/refund-claim", map[string]any{"sender_token": c.SenderToken, "signatures": in.Signatures}, "status", "tx", "state")
	case "invite":
		a.usdtInvite(w, r, engine, c)
	default:
		failure(w, 404, "not_found")
	}
}

// usdtInvite e-mails the card to its recipient, once the money is on it. The claim link comes from the sender's browser and is checked
// against the card (its id, and a key that belongs to the card's link owner); it is sent in the e-mail and not kept.
func (a *API) usdtInvite(w http.ResponseWriter, r *http.Request, engine USDTEngine, c usdtCard) {
	var in struct {
		ClaimURL string `json:"claim_url"`
	}
	if decode(w, r, &in) != nil || service.CheckClaimLink(in.ClaimURL, a.origin, c.ID, c.LinkOwner) != nil {
		failure(w, 400, "invalid_claim_link")
		return
	}
	if !a.now().Before(c.Created.Add(usdtCardLifetime)) {
		failure(w, 410, "card_expired")
		return
	}
	if c.Invites >= usdtInviteMax {
		failure(w, 429, "rate_limited")
		return
	}
	if !c.LastInvite.IsZero() && a.now().Sub(c.LastInvite) < usdtInviteSpace {
		failure(w, 429, "rate_limited")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	out, status, code := engineAnswer(engine.IssuerCall(ctx, "POST", "/api/v1/usdt/issuer/cards/"+c.ID+"/status", []byte("{}")))
	if code != "" {
		failure(w, status, code)
		return
	}
	if state := str(out, "state"); state != "ready" {
		failure(w, 409, "card_not_funded") // the recipient is told only when the money is there, deep enough
		return
	}
	// the attempt is written down BEFORE the e-mail leaves: a crash cannot give back a free one
	c.Invites++
	c.LastInvite = a.now()
	a.db.state.USDT[c.ID] = c
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	err := a.mail.InviteUSDT(ctx, usdtInvite{Sender: c.Sender, Recipient: c.Recipient, Card: cards.USDTCard{ID: c.ID, Chain: c.Chain, Symbol: c.Symbol, Amount: c.Amount, Message: c.Message, ClaimURL: in.ClaimURL}}, c.Language)
	if err != nil {
		failure(w, 502, "email_delivery_unavailable")
		return
	}
	c.Invited = true
	a.db.state.USDT[c.ID] = c
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	respond(w, 200, map[string]bool{"invited": true})
}

var errUSDTMail = errors.New("email_delivery_unavailable")
