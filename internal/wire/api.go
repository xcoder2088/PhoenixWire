package wire

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/service"
	"github.com/xcoder2088/PhoenixWire/internal/cards"
)

type Engine interface {
	Ready(context.Context) bool
	Create(string, string, string) (service.Claim, error)
	Provision(string) (string, error)
	Confirm(context.Context, string) (bool, error)
	PublicHandler() http.Handler
	Recovery(string) (string, error)
}
type senderOTP interface {
	IssueClaim(context.Context, string, string) error
	Verify(string, string, string) error
}
type Config struct {
	Directory, Origin string
	Master            []byte
	Engine            Engine
	Mail              Delivery
}
type API struct {
	mu     sync.Mutex
	db     *diskStore
	otp    senderOTP
	engine Engine
	mail   Delivery
	origin string
	master []byte
	unlock func()
	now    func() time.Time
	// the public gas light: remembered for a short while, with its own lock (it asks the chain, so it never waits behind a.mu)
	gasMu    sync.Mutex
	gasCache map[string]gasEntry
	// the hand-over to the Nimiq Pay app (handoff.go), under mu
	handoffs    map[string]handoff
	handoffAsks map[string][]time.Time
	// the same for a recipient's code (claimhandoff.go), under mu
	claimHandoffs map[string]claimHandoff
	claimAsks     map[string][]time.Time
}

var idRE = regexp.MustCompile(`^[a-f0-9]{48}$`)
var nonceRE = regexp.MustCompile(`^[a-f0-9]{32}$`)
var codeRE = regexp.MustCompile(`^[0-9]{6}$`)

func New(c Config) (*API, error) {
	u, err := url.Parse(c.Origin)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" || u.Host == "" {
		return nil, errors.New("invalid_public_origin")
	}
	local := u.Hostname() == "localhost" || net.ParseIP(u.Hostname()) != nil && net.ParseIP(u.Hostname()).IsLoopback()
	if u.Scheme != "https" && !(u.Scheme == "http" && local) || len(c.Master) != 32 || c.Engine == nil || c.Mail == nil {
		return nil, errors.New("invalid_wire_configuration")
	}
	unlock, err := service.LockDirectory(c.Directory)
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			unlock()
		}
	}()
	db, err := newStore(c.Directory, c.Master)
	if err != nil {
		return nil, err
	}
	otp, err := service.NewSenderOTP(c.Mail, filepath.Join(c.Directory, "sender-otp"), digest(c.Master, "Wire sender OTP v1"))
	if err != nil {
		return nil, err
	}
	ok = true
	return &API{db: db, otp: otp, engine: c.Engine, mail: c.Mail, origin: strings.TrimSuffix(c.Origin, "/"), master: append([]byte(nil), c.Master...), unlock: unlock, now: func() time.Time { return time.Now().UTC() }}, nil
}
func (a *API) Close() { clear(a.master); a.unlock() }

// USDTEngine is what the embedded engine offers when the USDC / USDT cashlinks run in the same process.
type USDTEngine interface {
	USDTReady() bool
	USDTHandler() http.Handler
	IssuerCall(ctx context.Context, method, path string, body []byte) (int, []byte, error)
}

// USDT returns the USDC / USDT engine, or nil when it is not configured.
func (a *API) USDT() USDTEngine {
	if e, ok := a.engine.(USDTEngine); ok && e.USDTReady() {
		return e
	}
	return nil
}

func (a *API) Ready(ctx context.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return !a.db.failed && a.engine.Ready(ctx)
}
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func failure(w http.ResponseWriter, status int, code string) {
	respond(w, status, map[string]string{"code": code})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		return errors.New("json_required")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("invalid_json")
	}
	return nil
}
func email(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	m, err := mail.ParseAddress(s)
	if err != nil || m.Address != s || len(s) > 254 || strings.ContainsAny(s, "\r\n") {
		return "", errors.New("invalid_email")
	}
	return s, nil
}
func (a *API) cookieName() string {
	if strings.HasPrefix(a.origin, "https:") {
		return "__Host-wire_session"
	}
	return "wire_session"
}
func (a *API) setCookie(w http.ResponseWriter, raw string) {
	http.SetCookie(w, &http.Cookie{Name: a.cookieName(), Value: raw, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(a.origin, "https:"), SameSite: http.SameSiteLaxMode, MaxAge: 86400})
}
func (a *API) current(r *http.Request) (string, session, bool) {
	c, err := r.Cookie(a.cookieName())
	if err != nil || !idRE.MatchString(c.Value) {
		return "", session{}, false
	}
	key := sessionKey(c.Value)
	s, ok := a.db.state.Sessions[key]
	return key, s, ok && a.now().Before(s.Expires)
}
func (a *API) Handler() http.Handler { return http.HandlerFunc(a.serve) }
func (a *API) serve(w http.ResponseWriter, r *http.Request) {
	// No CORS. Every mutation must originate at the configured Wire origin.
	if r.Method == "POST" && (r.Header.Get("Origin") != a.origin || r.Header.Get("Sec-Fetch-Site") == "cross-site") {
		failure(w, 403, "invalid_origin")
		return
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		failure(w, 403, "invalid_origin")
		return
	}
	if r.URL.Path == "/api/gas" && r.Method == "GET" { // public: the light is shown before anyone signs in
		a.gasPublic(w)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.db.failed {
		failure(w, 503, "state_unavailable")
		return
	}
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/claims/") {
		a.claim(w, r)
		return
	}
	switch path {
	case "/api/sender/request":
		if r.Method == "POST" {
			a.requestSender(w, r)
			return
		}
	case "/api/sender/verify":
		if r.Method == "POST" {
			a.verifySender(w, r)
			return
		}
	case "/api/handoff/redeem":
		if r.Method == "POST" {
			a.handoffRedeem(w, r)
			return
		}
	case "/api/session":
		if r.Method == "GET" {
			_, s, ok := a.current(r)
			out := map[string]any{"verified": ok && s.Verified, "email": func() string {
				if ok && s.Verified {
					return s.Email
				}
				return ""
			}()}
			if ok && s.Scope != "" { // a hand-over: not signed in as a sender, but allowed to fund and follow this one payment
				out["handoff"] = map[string]string{"payment_id": s.Scope}
			}
			respond(w, 200, out)
			return
		}
	case "/api/logout":
		if r.Method == "POST" {
			key, _, _ := a.current(r)
			delete(a.db.state.Sessions, key)
			if a.db.save() != nil {
				failure(w, 503, "state_unavailable")
				return
			}
			a.setCookie(w, "")
			respond(w, 200, map[string]bool{"ok": true})
			return
		}
	}
	sessKey, s, ok := a.current(r)
	if ok && s.Scope != "" { // the other browser of a hand-over: one payment, nothing else
		a.scoped(w, r, s)
		return
	}
	if !ok || !s.Verified {
		failure(w, 401, "sender_verification_required")
		return
	}
	if path == "/api/handoff" && r.Method == "POST" {
		a.handoffCreate(w, r, sessKey, s)
		return
	}
	if strings.HasPrefix(path, "/api/usdt/") {
		a.usdtServe(w, r, s)
		return
	}
	if path == "/api/payments" {
		if r.Method == "POST" {
			a.create(w, r, s)
			return
		}
		if r.Method == "GET" {
			list := []payment{}
			for _, saved := range a.db.state.Payments {
				p := saved.Payment
				if p.Sender == s.Email {
					list = append(list, p)
				}
			}
			sort.Slice(list, func(i, j int) bool { return list[i].Created.After(list[j].Created) })
			if len(list) > 50 {
				list = list[:50]
			}
			respond(w, 200, map[string]any{"payments": list})
			return
		}
	}
	a.paymentAction(w, r, s, path)
}

// paymentAction is what a sender does with one payment: POST /api/payments/<id>/<action>. The hand-over session reaches it only for its own
// payment and only for the actions of funding it (scoped).
func (a *API) paymentAction(w http.ResponseWriter, r *http.Request, s session, path string) {
	parts := strings.Split(strings.TrimPrefix(path, "/api/payments/"), "/")
	if len(parts) != 2 || !idRE.MatchString(parts[0]) || r.Method != "POST" {
		failure(w, 404, "not_found")
		return
	}
	saved, found := a.db.state.Payments[parts[0]]
	p := saved.Payment
	p.LastDelivery = saved.LastDelivery
	p.Deliveries = saved.Deliveries
	p.Nimicon = saved.Nimicon
	if !found || p.Sender != s.Email {
		failure(w, 404, "not_found")
		return
	}
	switch parts[1] {
	case "art":
		var input struct {
			PNG string `json:"png"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 95000)
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if d.Decode(&input) != nil || d.Decode(new(any)) != io.EOF {
			failure(w, 400, "invalid_nimicon")
			return
		}
		image, err := cards.CleanNimicon(input.PNG)
		if err != nil || p.Address == "" {
			failure(w, 400, "invalid_nimicon")
			return
		}
		if p.Nimicon == "" {
			p.Nimicon = base64.StdEncoding.EncodeToString(image)
			if !a.savePayment(w, p) {
				return
			}
		}
		respond(w, 200, map[string]bool{"saved": true})
	case "prepare":
		if p.Stage != "preparing" {
			respond(w, 200, map[string]any{"payment": p})
			return
		}
		a.prepare(w, p)
	case "start":
		if !a.now().Before(p.Created.Add(24 * time.Hour)) {
			failure(w, 410, "claim_expired")
			return
		}
		if p.Stage != "ready" {
			failure(w, 409, "funding_already_started")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		if !a.engine.Ready(ctx) {
			failure(w, 503, "chain_unavailable")
			return
		}
		p.Stage = "funding_started"
		started := a.now().UTC()
		p.StartedAt = &started
		if !a.savePayment(w, p) {
			return
		}
		respond(w, 200, map[string]any{"payment": p, "checkout": checkoutOf(p)})
	case "checkout":
		// Ask the wallet again for a payment whose funding started but never showed up. Never while the first request could still be on its way,
		// and never when the money is already there: a second payment would put twice the amount on the card.
		if !a.now().Before(p.Created.Add(24 * time.Hour)) {
			failure(w, 410, "claim_expired")
			return
		}
		if p.Stage != "funding_started" {
			failure(w, 409, "funding_not_started")
			return
		}
		if p.StartedAt != nil && a.now().Sub(*p.StartedAt) < checkoutRetryAfter {
			failure(w, 409, "retry_too_soon")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if ok, err := a.engine.Confirm(ctx, p.ID); err != nil {
			failure(w, 503, "chain_unavailable")
			return
		} else if ok {
			failure(w, 409, "already_funded")
			return
		}
		respond(w, 200, map[string]any{"payment": p, "checkout": checkoutOf(p)})
	case "confirm":
		a.confirm(w, r, p, false)
	case "resend":
		a.confirm(w, r, p, true)
	case "recovery":
		seed, err := a.engine.Recovery(p.ID)
		if err != nil {
			failure(w, 503, "recovery_unavailable")
			return
		}
		link, err := service.RecoveryLink(seed, p.Luna)
		if err != nil {
			failure(w, 503, "recovery_unavailable")
			return
		}
		respond(w, 200, map[string]string{"cashlink": link})
	default:
		failure(w, 404, "not_found")
	}
}
func (a *API) requestSender(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Language string `json:"language"`
	}
	if decode(w, r, &input) != nil {
		failure(w, 400, "invalid_email")
		return
	}
	to, err := email(input.Email)
	if err != nil {
		failure(w, 400, "invalid_email")
		return
	}
	now := a.now()
	for k, s := range a.db.state.Sessions {
		if !now.Before(s.Expires) {
			delete(a.db.state.Sessions, k)
		}
	}
	if len(a.db.state.Sessions) >= 10000 {
		failure(w, 429, "wait_before_resend")
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	rateKey := hex.EncodeToString(digest(a.master, "ip:"+host))
	limit := a.db.state.Rates[rateKey]
	if now.After(limit.Until) {
		limit = rate{Until: now.Add(time.Hour)}
	}
	if limit.Count >= 20 {
		failure(w, 429, "wait_before_resend")
		return
	}
	for k, v := range a.db.state.Rates {
		if now.After(v.Until) {
			delete(a.db.state.Rates, k)
		}
	}
	if len(a.db.state.Rates) >= 10000 {
		failure(w, 429, "wait_before_resend")
		return
	}
	limit.Count++
	a.db.state.Rates[rateKey] = limit
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	// Per-email persistent counters cannot be bypassed by changing cookies.
	challenge := hex.EncodeToString(digest(a.master, "sender:"+to))
	if err = a.otp.IssueClaim(service.WithEmailLanguage(r.Context(), cards.Language(input.Language)), challenge, to); err != nil {
		status := 502
		if errors.Is(err, service.ErrOTPRateLimited) {
			status = 429
		}
		failure(w, status, "wait_or_delivery_unavailable")
		return
	}
	raw := token()
	a.db.state.Sessions[sessionKey(raw)] = session{Email: to, Expires: now.Add(15 * time.Minute)}
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	a.setCookie(w, raw)
	respond(w, 200, map[string]bool{"sent": true})
}
func (a *API) verifySender(w http.ResponseWriter, r *http.Request) {
	key, s, ok := a.current(r)
	if !ok || s.Verified {
		failure(w, 403, "invalid_or_expired_code")
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	if decode(w, r, &input) != nil || !codeRE.MatchString(input.Code) {
		failure(w, 403, "invalid_or_expired_code")
		return
	}
	challenge := hex.EncodeToString(digest(a.master, "sender:"+s.Email))
	if a.otp.Verify(challenge, s.Email, input.Code) != nil {
		failure(w, 403, "invalid_or_expired_code")
		return
	}
	delete(a.db.state.Sessions, key)
	raw := token()
	s.Verified = true
	s.Expires = a.now().Add(24 * time.Hour)
	a.db.state.Sessions[sessionKey(raw)] = s
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	a.setCookie(w, raw)
	respond(w, 200, map[string]any{"verified": true, "email": s.Email})
}
func (a *API) savePayment(w http.ResponseWriter, p payment) bool {
	a.db.state.Payments[p.ID] = savedPayment{Payment: p, LastDelivery: p.LastDelivery, Deliveries: p.Deliveries, Nimicon: p.Nimicon}
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return false
	}
	return true
}
func (a *API) create(w http.ResponseWriter, r *http.Request, s session) {
	var in struct {
		Recipient string `json:"recipient"`
		Amount    string `json:"amount"`
		Currency  string `json:"currency"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
		Language  string `json:"language"`
	}
	if decode(w, r, &in) != nil || in.Currency != "NIM" || !nonceRE.MatchString(in.RequestID) || len([]rune(in.Message)) > 160 {
		failure(w, 400, "invalid_payment")
		return
	}
	recipient, err := email(in.Recipient)
	if err != nil {
		failure(w, 400, "invalid_payment")
		return
	}
	amount, luna, err := service.ParseAmount(in.Amount)
	if err != nil {
		failure(w, 400, "invalid_amount")
		return
	}
	id := hex.EncodeToString(digest(a.master, "payment:"+s.Email+":"+in.RequestID)[:24])
	if saved, ok := a.db.state.Payments[id]; ok {
		p := saved.Payment
		if p.Recipient != recipient || p.Amount != amount || p.Message != in.Message {
			failure(w, 409, "idempotency_conflict")
			return
		}
		if p.Stage != "preparing" {
			respond(w, 200, map[string]any{"payment": p})
			return
		}
	}
	if len(a.db.state.Payments) >= 10000 {
		failure(w, 429, "payment_capacity")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	if !a.engine.Ready(ctx) {
		failure(w, 503, "chain_unavailable")
		return
	}
	p := payment{ID: id, Sender: s.Email, Recipient: recipient, Amount: amount, Luna: luna, Message: in.Message, Language: in.Language, Stage: "preparing", Created: a.now().UTC()}
	if existing, ok := a.db.state.Payments[id]; ok {
		p = existing.Payment
	}
	if !a.savePayment(w, p) {
		return
	}
	a.prepare(w, p)
}
func (a *API) prepare(w http.ResponseWriter, p payment) {
	if !a.now().Before(p.Created.Add(24 * time.Hour)) {
		failure(w, 410, "claim_expired")
		return
	}
	id := p.ID
	if _, err := a.engine.Create(id, p.Recipient, p.Amount); err != nil {
		failure(w, 503, "claim_unavailable")
		return
	}
	address, err := a.engine.Provision(id)
	if err != nil {
		failure(w, 503, "claim_unavailable")
		return
	}
	p.Address = address
	p.Stage = "ready"
	if !a.savePayment(w, p) {
		return
	}
	respond(w, 201, map[string]any{"payment": p})
}
func (a *API) confirm(w http.ResponseWriter, r *http.Request, p payment, resend bool) {
	if p.Stage == "sent" && !resend {
		respond(w, 200, map[string]any{"payment": p})
		return
	}
	if p.Stage == "funding_started" {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		ok, err := a.engine.Confirm(ctx, p.ID)
		if err != nil || !ok {
			failure(w, 409, "funding_pending")
			return
		}
		p.Stage = "funded"
		if !a.savePayment(w, p) {
			return
		}
	}
	if p.Stage != "funded" && p.Stage != "delivery_pending" && p.Stage != "delivery_failed" && p.Stage != "sent" {
		failure(w, 409, "funding_not_started")
		return
	}
	if !a.now().Before(p.Created.Add(24 * time.Hour)) {
		failure(w, 410, "claim_expired")
		return
	}
	if p.Stage != "funded" && !resend {
		respond(w, 200, map[string]any{"payment": p})
		return
	}
	if p.Deliveries >= 5 || !p.LastDelivery.IsZero() && a.now().Sub(p.LastDelivery) < time.Minute {
		failure(w, 429, "wait_before_resend")
		return
	}
	p.Deliveries++
	p.LastDelivery = a.now()
	p.Stage = "delivery_pending"
	if !a.savePayment(w, p) {
		return
	}
	if err := a.mail.Invite(r.Context(), p, a.origin+"/?claim="+p.ID); err != nil {
		p.Stage = "delivery_failed"
		if a.savePayment(w, p) {
			failure(w, 502, "email_delivery_unavailable")
		}
		return
	}
	p.Stage = "sent"
	if p.SentAt == nil { // the first success only: a resend does not move it
		sent := a.now().UTC()
		p.SentAt = &sent
	}
	if !a.savePayment(w, p) {
		return
	}
	respond(w, 200, map[string]any{"payment": p})
}
func (a *API) claim(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/claims/"), "/")
	if len(parts) < 1 || len(parts) > 3 || !idRE.MatchString(parts[0]) || len(parts) == 3 && (parts[1] != "handoff" || parts[2] != "redeem") {
		failure(w, 404, "claim_unavailable")
		return
	}
	saved, ok := a.db.state.Payments[parts[0]]
	p := saved.Payment
	if !ok || (p.Stage != "funded" && p.Stage != "delivery_pending" && p.Stage != "delivery_failed" && p.Stage != "sent") {
		failure(w, 404, "claim_unavailable")
		return
	}
	if r.Method == "GET" && len(parts) == 1 {
		respond(w, 200, map[string]any{"id": p.ID, "amount": p.Amount, "sender": p.Sender, "message": p.Message, "address": p.Address, "expires_at": p.Created.Add(24 * time.Hour)})
		return
	}
	if r.Method == "POST" && len(parts) >= 2 && parts[1] == "handoff" {
		if len(parts) == 3 {
			a.claimHandoffRedeem(w, r, p.ID)
		} else {
			a.claimHandoffCreate(w, r, p.ID)
		}
		return
	}
	if r.Method != "POST" || len(parts) != 2 || (parts[1] != "request" && parts[1] != "release" && parts[1] != "payout" && parts[1] != "payout_status") {
		failure(w, 404, "not_found")
		return
	}
	if parts[1] == "release" {
		var input struct {
			Code string `json:"code"`
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
		if err != nil || len(raw) > 8192 || json.Unmarshal(raw, &input) != nil || !codeRE.MatchString(input.Code) {
			failure(w, 403, "invalid_or_expired_code")
			return
		}
		r.Body = io.NopCloser(strings.NewReader(string(raw)))
	}
	if parts[1] == "payout" {
		// The code is needed once, to start the payment; the same destination may ask again without it (see the engine). Only these two
		// fields ever reach the engine.
		var input struct {
			Code string `json:"code"`
			To   string `json:"to"`
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
		if err != nil || len(raw) > 8192 || json.Unmarshal(raw, &input) != nil || len(input.To) > 64 || input.Code != "" && !codeRE.MatchString(input.Code) {
			failure(w, 403, "invalid_or_expired_code")
			return
		}
		clean, _ := json.Marshal(map[string]string{"code": input.Code, "to": input.To})
		r.Body = io.NopCloser(strings.NewReader(string(clean)))
	}
	// Route only public request/release to the original, audited OTP handler.
	clone := r.Clone(service.WithEmailLanguage(r.Context(), cards.Language(p.Language)))
	u := *r.URL
	clone.URL = &u
	clone.URL.Path = "/api/v1/claims/" + p.ID
	clone.URL.RawQuery = "action=" + parts[1]
	a.engine.PublicHandler().ServeHTTP(w, clone)
}

// checkoutRetryAfter is how long a started funding is given before the wallet may be asked again (a first payment shows within a minute or two).
const checkoutRetryAfter = 3 * time.Minute

func checkoutOf(p payment) map[string]any {
	return map[string]any{"appName": "Phoenix Wire", "recipient": p.Address, "value": p.Luna, "recipientData": []int{0, 130, 128, 146, 135}}
}
