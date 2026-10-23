package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/chain"
	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/relayer"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/usdtlink"
)

// USDT cashlinks (Ethereum / Polygon): HTTP layer of internal/usdtlink. Routes (all JSON):
//
//	browser (recipient), session cookie + allowed Origin on every write:
//	  GET  /api/v1/usdt/cards/{id}                         public view of a card
//	  POST /api/v1/usdt/sessions                           starts a session (sets the cookie)
//	  POST /api/v1/usdt/cards/{id}/email-checks            {idempotency_key}                         -> challenge (the code comes FIRST)
//	  POST /api/v1/usdt/cards/{id}/email-checks/{cid}/code {signature, lang}                         -> code e-mailed (the check is stored only now)
//	  POST /api/v1/usdt/email-checks/{cid}/verify          {code}                                    -> {valid_until}
//	  POST /api/v1/usdt/cards/{id}/authorizations          {destination, idempotency_key, check_id}  -> snapshot + challenge
//	  POST /api/v1/usdt/authorizations/{aid}/grant         {signature}                               -> {grant, expires_at}
//	  POST /api/v1/usdt/authorizations/{aid}/signature     {grant}                                   -> {owner, signature}
//	  POST /api/v1/usdt/authorizations/{aid}/claims        {grant, signatures}                       -> {tx}
//	issuer (PhoenixMail, X-Claim-Service-Key, never from a browser):
//	  POST /api/v1/usdt/issuer/cards                       create       POST .../{id}/status   POST .../{id}/cancel
//	  POST /api/v1/usdt/issuer/cards/{id}/gas              {sender_token, sender_address} -> the relayer gives a token-only sender the native coin of one transfer (once per card)
//	  POST /api/v1/usdt/issuer/cards/{id}/refund           {sender_token, to} -> co-signature of a full-sweep refund
//	  POST /api/v1/usdt/issuer/cards/{id}/refund-claim     {sender_token, signatures} -> relays that refund
//	  GET|POST /api/v1/usdt/issuer/relayers                     what every relayer holds (native coin, fees earned, cards waiting)
//	  POST /api/v1/usdt/issuer/relayers/{chain}/withdraw   {what: "native"|asset id, to} -> sends ALL of it to `to` (the owner's administrator only)
//
// The recipient's link key never reaches this service: the browser proves it holds it by signing (internal/cashlink/proof).

const usdtCookie = "__Host-usdtlink"

func usdtlinkError(w http.ResponseWriter, err error) {
	code, status := "invalid_request", 400
	switch {
	case errors.Is(err, usdtlink.ErrNotFound):
		code, status = "card_not_found", 404
	case errors.Is(err, usdtlink.ErrSession):
		code, status = "session_required", 401
	case errors.Is(err, usdtlink.ErrForbidden), errors.Is(err, usdtlink.ErrBadGrant):
		code, status = "forbidden", 403
	case errors.Is(err, usdtlink.ErrLinkProof):
		code, status = "invalid_link_proof", 403
	case errors.Is(err, usdtlink.ErrBadCode):
		code, status = "invalid_or_expired_code", 403
	case errors.Is(err, relayer.ErrNotDeployedYet):
		code, status = "deployment_not_visible", 503
	case errors.Is(err, usdtlink.ErrBalanceUnknown):
		code, status = "balance_unreadable", 503
	case errors.Is(err, usdtlink.ErrReserveRequired):
		code, status = "reserve_required", 409
	case errors.Is(err, usdtlink.ErrWithdrawDisabled):
		code, status = "withdraw_not_configured", 403
	case errors.Is(err, usdtlink.ErrWithdrawDestination):
		code, status = "withdraw_destination_not_allowed", 403
	case errors.Is(err, usdtlink.ErrNothingToWithdraw):
		code, status = "nothing_to_withdraw", 409
	case errors.Is(err, usdtlink.ErrEmailCheck):
		code, status = "email_check_required", 403
	case errors.Is(err, usdtlink.ErrExpired), errors.Is(err, usdtlink.ErrAuthExpired):
		code, status = "expired", 410
	case errors.Is(err, usdtlink.ErrCancelled):
		code, status = "card_cancelled", 409
	case errors.Is(err, usdtlink.ErrPaid):
		code, status = "already_paid", 409
	case errors.Is(err, usdtlink.ErrPending):
		code, status = "payment_in_progress", 409
	case errors.Is(err, usdtlink.ErrAddressSet):
		code, status = "destination_already_bound", 409
	case errors.Is(err, usdtlink.ErrQuoteChanged):
		code, status = "quote_changed", 409
	case errors.Is(err, usdtlink.ErrIdempotency), errors.Is(err, usdtlink.ErrAuthState), errors.Is(err, usdtlink.ErrAuthNotSigned), errors.Is(err, usdtlink.ErrDuplicate):
		code, status = "conflict", 409
	case errors.Is(err, usdtlink.ErrFloatLow):
		code, status = "relayer_float_low", 503
	case errors.Is(err, usdtlink.ErrGasCredit):
		code, status = "gas_credit_exhausted", 503
	case errors.Is(err, usdtlink.ErrGasNotNeeded):
		code, status = "gas_not_needed", 409
	case errors.Is(err, relayer.ErrFloorReached):
		code, status = "gas_service_unavailable", 503
	case errors.Is(err, usdtlink.ErrNotFunded):
		code, status = "card_not_funded", 409
	case errors.Is(err, usdtlink.ErrRateLimited):
		code, status = "rate_limited", 429
	case errors.Is(err, usdtlink.ErrCapacity):
		code, status = "capacity_reached", 503
	case errors.Is(err, usdtlink.ErrUnavailable):
		code, status = "usdt_unavailable", 503
	case errors.Is(err, evm.ErrRPCUnavailable), errors.Is(err, chain.ErrUnavailable), errors.Is(err, relayer.ErrRelayerFunds):
		code, status = "chain_unavailable", 503
	case errors.Is(err, relayer.ErrJournal):
		code, status = "storage_unavailable", 503
	case errors.Is(err, relayer.ErrFeeTooLow), errors.Is(err, relayer.ErrSignatures), errors.Is(err, relayer.ErrCard), errors.Is(err, relayer.ErrAmount), errors.Is(err, relayer.ErrDestination):
		code = err.Error()
	case errors.Is(err, relayer.ErrRejected):
		code, status = "transaction_would_revert", 502
	case errors.Is(err, evm.ErrInvalidAddress):
		code = "invalid_address"
	case errors.Is(err, evm.ErrInvalidAmount), errors.Is(err, evm.ErrUnknownAsset):
		code = err.Error()
	}
	write(w, status, map[string]string{"error": code})
}

// ---- small per-IP limiter (the service sits behind a local proxy: only a loopback peer may name the client) ----

type ipLimiter struct {
	mu    sync.Mutex
	hits  map[string][]time.Time
	max   int
	every time.Duration
}

func newIPLimiter(max int, every time.Duration) *ipLimiter {
	return &ipLimiter{hits: map[string][]time.Time{}, max: max, every: every}
}

func (l *ipLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < l.every {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	if len(l.hits) > 50000 { // a bound on memory: forget idle keys
		for k, v := range l.hits {
			if len(v) == 0 || now.Sub(v[len(v)-1]) >= l.every {
				delete(l.hits, k)
			}
		}
	}
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if c := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); c != "" && net.ParseIP(c) != nil {
			return c
		}
	}
	return host
}

type usdtHTTP struct {
	svc      *usdtlink.Service
	origins  map[string]bool
	sessions *ipLimiter
	reads    *ipLimiter
	writes   *ipLimiter
	durable  func() bool // e-mail codes are really delivered and survive a restart
}

// originOK: every browser write must come from a page of an allowed origin (together with SameSite=Strict on the cookie).
func (h *usdtHTTP) originOK(r *http.Request) bool { return h.origins[r.Header.Get("Origin")] }

func sessionOf(r *http.Request) string {
	if c, err := r.Cookie(usdtCookie); err == nil {
		return c.Value
	}
	return ""
}

func (h *usdtHTTP) browser(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/usdt/"), "/")
	parts := strings.Split(rest, "/")
	ip := clientIP(r)
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()
	switch {
	case r.Method == http.MethodGet && len(parts) == 2 && parts[0] == "cards":
		if !h.reads.allow(ip) {
			write(w, 429, map[string]string{"error": "rate_limited"})
			return
		}
		p, err := h.svc.Refresh(ctx, parts[1])
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 200, p)
		return
	case r.Method != http.MethodPost:
		write(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !h.originOK(r) {
		write(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	if parts[0] == "sessions" && len(parts) == 1 {
		if !h.sessions.allow(ip) {
			write(w, 429, map[string]string{"error": "rate_limited"})
			return
		}
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			write(w, 500, map[string]string{"error": "internal"})
			return
		}
		http.SetCookie(w, &http.Cookie{Name: usdtCookie, Value: hex.EncodeToString(raw), Path: "/", Secure: true, HttpOnly: true,
			SameSite: http.SameSiteStrictMode, MaxAge: int(usdtlink.SessionLife / time.Second)})
		write(w, 201, map[string]any{"expires_in": int(usdtlink.SessionLife / time.Second)})
		return
	}
	if !h.writes.allow(ip) {
		write(w, 429, map[string]string{"error": "rate_limited"})
		return
	}
	session := sessionOf(r)
	switch {
	case len(parts) == 3 && parts[0] == "cards" && parts[2] == "email-checks":
		var in struct {
			Idempotency string `json:"idempotency_key"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		p, err := h.svc.StartCheck(ctx, session, parts[1], in.Idempotency)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 201, p)
	case len(parts) == 3 && parts[0] == "cards" && parts[2] == "authorizations":
		var in struct {
			Destination string `json:"destination"`
			Idempotency string `json:"idempotency_key"`
			CheckID     string `json:"check_id"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		p, err := h.svc.Prepare(ctx, session, parts[1], in.Destination, in.Idempotency, in.CheckID)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 201, p)
	case (len(parts) == 3 && parts[0] == "email-checks" && parts[2] == "verify" || len(parts) == 5 && parts[0] == "cards" && parts[2] == "email-checks" && parts[4] == "code") && !h.durable():
		write(w, 503, map[string]string{"error": "durable_email_otp_required"})
	case len(parts) == 5 && parts[0] == "cards" && parts[2] == "email-checks" && parts[4] == "code":
		var in struct {
			Signature string `json:"signature"`
			Lang      string `json:"lang"`
		}
		sig, ok := decodeBody(w, r, &in, func() (string, bool) { return in.Signature, true })
		if !ok {
			return
		}
		if err := h.svc.ProveLink(ctx, session, parts[1], parts[3], sig, in.Lang); err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 202, map[string]any{"status": "code_sent", "resend_after": 20})
	case len(parts) == 3 && parts[0] == "email-checks" && parts[2] == "verify":
		var in struct {
			Code string `json:"code"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		until, err := h.svc.VerifyEmail(ctx, session, parts[1], in.Code)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 200, map[string]any{"valid_until": until})
	case len(parts) == 3 && parts[0] == "authorizations" && parts[2] == "grant":
		var in struct {
			Signature string `json:"signature"`
		}
		sig, ok := decodeBody(w, r, &in, func() (string, bool) { return in.Signature, true })
		if !ok {
			return
		}
		grant, until, err := h.svc.Authorize(ctx, session, parts[1], sig)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 200, map[string]any{"grant": grant, "expires_at": until})
	case len(parts) == 3 && parts[0] == "authorizations" && parts[2] == "signature":
		var in struct {
			Grant string `json:"grant"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		owner, sig, err := h.svc.CosignClaim(ctx, session, parts[1], in.Grant)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 200, map[string]string{"owner": owner.Hex(), "signature": "0x" + hex.EncodeToString(sig)})
	case len(parts) == 3 && parts[0] == "authorizations" && parts[2] == "claims":
		var in struct {
			Grant      string `json:"grant"`
			Signatures string `json:"signatures"`
		}
		sigs, ok := decodeBody(w, r, &in, func() (string, bool) { return in.Signatures, true })
		if !ok {
			return
		}
		tx, state, err := h.svc.ClaimAuthorized(ctx, session, parts[1], in.Grant, sigs)
		if err != nil {
			// A transport or timeout failure after the transaction was journaled is not a failure: the chain decides.
			if tx != "" {
				write(w, 202, map[string]string{"status": "verifying", "tx": tx})
				return
			}
			usdtlinkError(w, err)
			return
		}
		if state != usdtlink.StatePaid { // sent is not paid: the page says "being verified" until the card is deep and verified
			write(w, 202, map[string]string{"status": "verifying", "tx": tx})
			return
		}
		write(w, 200, map[string]string{"status": "paid", "tx": tx})
	default:
		write(w, 404, map[string]string{"error": "not_found"})
	}
}

// decodeBody reads the JSON body into in, then turns the hex string chosen by pick into bytes.
func decodeBody(w http.ResponseWriter, r *http.Request, in any, pick func() (string, bool)) ([]byte, bool) {
	if decode(w, r, in) != nil {
		write(w, 400, map[string]string{"error": "invalid_json"})
		return nil, false
	}
	text, _ := pick()
	raw, err := hex.DecodeString(strings.TrimPrefix(text, "0x"))
	if err != nil || len(raw) > 3*65 {
		write(w, 400, map[string]string{"error": "invalid_request"})
		return nil, false
	}
	return raw, true
}

// gasStatus answers {asset} -> {available, reason}: what the sender's payment screen shows as a green or red light.
func (h *usdtHTTP) gasStatus(s *server, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.issuerAuthorized(r) {
		write(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	var in struct {
		Asset string `json:"asset"`
	}
	if decode(w, r, &in) != nil {
		write(w, 400, map[string]string{"error": "invalid_json"})
		return
	}
	ok, reason := h.svc.GasStatus(r.Context(), in.Asset)
	write(w, 200, map[string]any{"available": ok, "reason": reason})
}

func (h *usdtHTTP) issuer(s *server, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		write(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !s.issuerAuthorized(r) {
		write(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/usdt/issuer/cards"), "/")
	parts := strings.Split(rest, "/")
	if rest == "" {
		var in struct {
			Asset       string `json:"asset"`
			CardID      string `json:"card_id"`
			LinkOwner   string `json:"link_owner"`
			Amount      string `json:"amount"`
			FeeUnits    uint64 `json:"fee_units"`
			Email       string `json:"email"`
			Message     string `json:"message"`
			SenderEmail string `json:"sender_email"`
			SenderLang  string `json:"sender_lang"`
		}
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		created, err := h.svc.Create(usdtlink.Input{Asset: in.Asset, CardID: in.CardID, LinkOwner: in.LinkOwner, Amount: in.Amount, FeeUnits: in.FeeUnits,
			Protected: true, Email: in.Email, Message: in.Message, SenderEmail: in.SenderEmail, SenderLang: in.SenderLang})
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 201, created)
		return
	}
	if len(parts) != 2 {
		write(w, 404, map[string]string{"error": "not_found"})
		return
	}
	id := parts[0]
	var in struct {
		SenderToken string `json:"sender_token"`
		To          string `json:"to"`
		Signatures  string `json:"signatures"`
	}
	switch parts[1] {
	case "status":
		st, err := h.svc.Status(ctx, id)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 200, st)
	case "cancel":
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		p, err := h.svc.Cancel(id, in.SenderToken)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 200, p)
	case "gas":
		var g struct {
			SenderToken   string `json:"sender_token"`
			SenderAddress string `json:"sender_address"`
		}
		if decode(w, r, &g) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		res, err := h.svc.GasTopUp(ctx, id, g.SenderToken, g.SenderAddress)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 200, map[string]any{"tx": res.Tx, "wei": res.WeiText, "already": res.Already})
	case "refund":
		if decode(w, r, &in) != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		owner, sig, units, err := h.svc.Refund(ctx, id, in.SenderToken, in.To)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		nonce, _ := h.svc.RefundNonce(id)
		fee, _ := h.svc.RefundFeeUnits(id)
		write(w, 200, map[string]any{"owner": owner.Hex(), "signature": "0x" + hex.EncodeToString(sig), "units": units, "nonce": nonce, "fee_units": fee})
	case "refund-claim":
		sigs, ok := decodeBody(w, r, &in, func() (string, bool) { return in.Signatures, true })
		if !ok {
			return
		}
		res, err := h.svc.RefundClaim(ctx, id, in.SenderToken, sigs)
		if err != nil {
			if res.ExecTx != "" {
				write(w, 202, map[string]string{"status": "verifying", "tx": res.ExecTx})
				return
			}
			usdtlinkError(w, err)
			return
		}
		if h.svc.InFlight(id) {
			write(w, 202, map[string]string{"status": "verifying", "tx": res.ExecTx})
			return
		}
		write(w, 200, map[string]string{"status": "refunded", "tx": res.ExecTx})
	default:
		write(w, 404, map[string]string{"error": "not_found"})
	}
}

// ---- startup ----

// keyFile returns the secret stored at path, creating it (32 random bytes, hex, mode 0600) the first time.
func keyFile(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, err
		}
		_, werr := f.WriteString(hex.EncodeToString(b) + "\n")
		if cerr := f.Close(); werr != nil || cerr != nil {
			return nil, errors.Join(werr, cerr)
		}
		return b, nil
	}
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(path); err != nil || st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("key file must be private (mode 0600): " + path)
	}
	b, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(b) != 32 {
		return nil, errors.New("key file must hold 32 bytes in hex: " + path)
	}
	return b, nil
}

func gwei(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1_000_000_000)) }

func wei(decimal string) *big.Int {
	v, ok := new(big.Int).SetString(decimal, 10)
	if !ok {
		panic("bad wei constant")
	}
	return v
}

var usdtlinkNetworks = map[string]struct {
	asset  string
	minTip *big.Int
	minFee uint64
	drip   *big.Int // the most native coin one sender is given (gas.go)
	floor  *big.Int // the float the relayer keeps
}{
	"polygon":      {evm.AssetUSDTPolygon, gwei(30), 50_000, wei("300000000000000000"), wei("200000000000000000")},        // drip up to 0.3 POL, keep 0.2 POL
	"ethereum":     {evm.AssetUSDTEthereum, wei("100000000"), 400_000, wei("5000000000000000"), wei("10000000000000000")}, // drip up to 0.005 ETH, keep 0.01 ETH
	"amoy":         {evm.AssetUSDCAmoyTest, gwei(30), 50_000, wei("300000000000000000"), wei("200000000000000000")},       // test network, test coins
	"polygon-usdc": {evm.AssetUSDCPolygon, gwei(30), 50_000, wei("300000000000000000"), wei("200000000000000000")},        // USDC on Polygon (shares the Polygon relayer and its key)
	"sepolia":      {evm.AssetUSDCSepoliaTest, gwei(1), 50_000, wei("5000000000000000"), wei("10000000000000000")},        // test network, test coins
}

// setupUSDTLink starts the USDT cashlink service when NIMIQ_CLAIM_USDTLINK_CHAINS names at least one chain
// ("polygon", "ethereum"). Secrets live in <state>/usdtlink/: master.key (co-signer, BACK IT UP: losing it blocks every
// protected card), auth.key (grants and bindings), relayer-<chain>.key (pays gas: fund its address with POL/ETH).
func setupUSDTLink(s *server, stateDir string) (*usdtHTTP, error) {
	if os.Getenv("NIMIQ_CLAIM_TESTNET") == "1" {
		evm.EnableTestnets() // a separate test instance only; log it loudly so it is never mistaken for production
		log.Print("TEST NETWORKS ENABLED (Polygon Amoy, test coins): this instance is not production")
	}
	names := strings.Split(strings.ToLower(os.Getenv("NIMIQ_CLAIM_USDTLINK_CHAINS")), ",")
	dir := filepath.Join(stateDir, "usdtlink")
	var chains []usdtlink.Chain
	for _, name := range names {
		name = strings.TrimSpace(name)
		spec, ok := usdtlinkNetworks[name]
		if name == "" {
			continue
		}
		if !ok {
			return nil, errors.New("unknown USDT chain: " + name)
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		n, err := evm.Lookup(spec.asset)
		if err != nil {
			return nil, err
		}
		key, err := keyFile(filepath.Join(dir, "relayer-"+strings.TrimSuffix(name, "-usdc")+".key")) // one relayer (gas wallet) per chain
		if err != nil {
			return nil, err
		}
		node := chain.NewHTTP(n.ChainID, n.RPCs, spec.minTip)
		if onlyLocalNodes(n.RPCs) { // the operator's own node on this machine may speak plain http; any other node must use https
			node.AllowHTTPForTests()
		}
		token, err := cl.ParseAddress(n.Token)
		if err != nil {
			return nil, err
		}
		rl, err := relayer.New(relayer.Config{Chain: node, Key: key, Token: token, MinFee: new(big.Int).SetUint64(spec.minFee)})
		if err != nil {
			return nil, err
		}
		log.Printf("USDT cashlink %s: relayer address %s (fund it with the chain's native coin for gas)", name, rl.Address().Hex())
		chainCfg := usdtlink.Chain{Network: n, Node: node, Relayer: rl, MinFee: spec.minFee, MinFloat: spec.floor}
		// The minimum float below which no NEW card is taken can be lowered (even to 0) while the relayer is still being funded:
		// only for a service whose users are the owner alone.
		// NIMIQ_CLAIM_USDTLINK_MIN_FLOAT_WEI_<NAME> (ETHEREUM, POLYGON, POLYGON_USDC) sets one chain, the unsuffixed variable all of them.
		for _, key := range []string{"NIMIQ_CLAIM_USDTLINK_MIN_FLOAT_WEI_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")), "NIMIQ_CLAIM_USDTLINK_MIN_FLOAT_WEI"} {
			if raw := os.Getenv(key); raw != "" {
				if v, ok := new(big.Int).SetString(raw, 10); ok && v.Sign() >= 0 {
					chainCfg.MinFloat = v
					log.Printf("USDT cashlink %s: minimum relayer float set to %s wei by %s", name, v, key)
					break
				}
			}
		}
		if os.Getenv("NIMIQ_CLAIM_USDTLINK_GAS") == "on" { // gas money for token-only senders is its own switch, OFF by default
			chainCfg.DripCap, chainCfg.Floor = spec.drip, spec.floor
			if v, err := strconv.Atoi(os.Getenv("NIMIQ_CLAIM_USDTLINK_GAS_SEED")); err == nil && v > 0 {
				chainCfg.GasSeed = v
			}
			log.Printf("USDT cashlink %s: gas money for senders ENABLED (cap %s wei per card, float floor %s wei)", name, spec.drip, spec.floor)
		}
		chains = append(chains, chainCfg)
	}
	if len(chains) == 0 {
		return nil, nil
	}
	if raw := os.Getenv("NIMIQ_CLAIM_USDTLINK_FEE_BPS"); raw != "" { // the service fee, in basis points of the amount (default 600 = 6%)
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil && v <= 2000 {
			usdtlink.FeeBps = v
		}
	}
	master, err := keyFile(filepath.Join(dir, "master.key"))
	if err != nil {
		return nil, err
	}
	authKey, err := keyFile(filepath.Join(dir, "auth.key"))
	if err != nil {
		return nil, err
	}
	svc, err := usdtlink.New(dir, master, chains)
	if err != nil {
		return nil, err
	}
	if err := svc.SetCodes(s.otp, authKey); err != nil {
		return nil, err
	}
	// the only addresses a relayer withdrawal may go to (comma-separated); none configured: withdrawals are off
	svc.SetWithdrawAddresses(strings.Split(os.Getenv("NIMIQ_CLAIM_USDTLINK_WITHDRAW_TO"), ","))
	origins := map[string]bool{}
	list := os.Getenv("NIMIQ_CLAIM_ALLOWED_ORIGINS")
	if strings.TrimSpace(list) == "" {
		list = "https://claim.phoenixmail.ca"
	}
	for _, o := range strings.Split(list, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins[o] = true
		}
	}
	go svc.Run(context.Background(), 30*time.Second)
	return &usdtHTTP{svc: svc, origins: origins, sessions: newIPLimiter(50, 15*time.Minute), reads: newIPLimiter(60, time.Minute), writes: newIPLimiter(120, time.Minute),
		durable: func() bool { return !s.devConsole && s.otp.Durable() }}, nil
}

func anyTrue(m map[string]bool) bool {
	for _, v := range m {
		if v {
			return true
		}
	}
	return false
}

// relayers: what each relayer holds, and the owner's way to take it all out. Issuer key only (PhoenixMail's administrator).
func (h *usdtHTTP) relayers(s *server, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !s.issuerAuthorized(r) {
		write(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/usdt/issuer/relayers"), "/")
	if rest == "" {
		if r.Method != http.MethodGet && r.Method != http.MethodPost { // PhoenixMail's bridge only posts: a read, whichever the verb
			write(w, 405, map[string]string{"error": "method_not_allowed"})
			return
		}
		infos, err := h.svc.RelayerInfos(ctx)
		if err != nil {
			usdtlinkError(w, err)
			return
		}
		write(w, 200, map[string]any{"relayers": infos, "withdraw_to": h.svc.WithdrawAddresses()})
		return
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "withdraw" || r.Method != http.MethodPost {
		write(w, 404, map[string]string{"error": "not_found"})
		return
	}
	var in struct {
		What string `json:"what"`
		To   string `json:"to"`
	}
	if decode(w, r, &in) != nil {
		write(w, 400, map[string]string{"error": "invalid_json"})
		return
	}
	res, err := h.svc.RelayerWithdraw(ctx, parts[0], in.What, in.To)
	if err != nil {
		usdtlinkError(w, err)
		return
	}
	write(w, 200, res)
}

// onlyLocalNodes: every endpoint is a plain-http node on this very machine.
func onlyLocalNodes(endpoints []string) bool {
	if len(endpoints) == 0 {
		return false
	}
	for _, e := range endpoints {
		if !strings.HasPrefix(e, "http://127.0.0.1:") && !strings.HasPrefix(e, "http://localhost:") {
			return false
		}
	}
	return true
}
