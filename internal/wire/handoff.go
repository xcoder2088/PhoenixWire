package wire

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// The hand-over of ONE payment from the sender's browser to the Nimiq Pay app.
//
// The app is another browser (its own cookies). What crosses over is not the sender's session but a right to fund and follow one payment:
//
//   - the sender, already verified, has prepared the payment (amount, recipient, address are fixed by the server) and asks for a right;
//   - the right is an opaque token (43 random characters, only its hash is kept, in memory, never logged): one use, five minutes, tied to
//     that payment and to the session it came from (signing out cancels it);
//   - redeemed in the other browser, it opens a SCOPED session: it can fund and follow that payment and nothing else (no other payment, no
//     creation, no USDC / USDT card, no recovery key, no hand-over of its own), for two hours at most and never beyond the payment's own day
//     or the session it came from;
//   - it never replaces a session that is already signed in as someone else, and it never turns into the sender's general session.
//
// A lost answer to the redeem loses the right (it was used): the sender asks for another one from the first browser.
const (
	handoffTTL      = 5 * time.Minute
	scopedLifetime  = 2 * time.Hour
	handoffMax      = 1000
	handoffPerSpell = 6 // rights one sender may ask for in handoffSpell
	handoffSpell    = 10 * time.Minute
)

var handoffToken = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

// scopedActions are the only things a scoped session may do with its payment: its picture for the e-mail, starting its funding, asking the
// wallet again when the first request never showed, checking the funding, giving the card up while it is still unpaid (the payment's own checks
// refuse a card whose money is on it, "already_funded") and, if the e-mail failed, sending it again. Not the recovery key.
var scopedActions = map[string]bool{"art": true, "start": true, "checkout": true, "confirm": true, "resend": true, "cancel": true}

type handoff struct {
	Email     string
	PaymentID string
	Origin    string // key of the session it came from
	Expires   time.Time
}

func handoffKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// pruneHandoffs forgets what has expired. Called under a.mu.
func (a *API) pruneHandoffs() {
	now := a.now()
	for k, h := range a.handoffs {
		if !now.Before(h.Expires) {
			delete(a.handoffs, k)
		}
	}
	for email, times := range a.handoffAsks {
		kept := times[:0]
		for _, at := range times {
			if now.Sub(at) < handoffSpell {
				kept = append(kept, at)
			}
		}
		if len(kept) == 0 {
			delete(a.handoffAsks, email)
		} else {
			a.handoffAsks[email] = kept
		}
	}
}

// fundable says whether a payment can still be handed over: it is the sender's, prepared and not started, and within its day.
func (a *API) fundable(email, id string) (payment, int, string) {
	saved, ok := a.db.state.Payments[id]
	p := saved.Payment
	switch {
	case !ok || p.Sender != email:
		return p, 404, "not_found"
	case !a.now().Before(p.Created.Add(24 * time.Hour)):
		return p, 410, "claim_expired"
	case p.Stage != "ready":
		return p, 409, "payment_not_fundable" // funding already started (here or elsewhere), delivered, or not prepared: no second payment
	}
	return p, 0, ""
}

// handoffCreate: POST /api/handoff {payment_id}, for a verified sender. Called under a.mu.
func (a *API) handoffCreate(w http.ResponseWriter, r *http.Request, key string, s session) {
	var in struct {
		PaymentID string `json:"payment_id"`
	}
	if decode(w, r, &in) != nil || !idRE.MatchString(in.PaymentID) {
		failure(w, 400, "invalid_request")
		return
	}
	if _, status, code := a.fundable(s.Email, in.PaymentID); code != "" {
		failure(w, status, code)
		return
	}
	a.pruneHandoffs()
	if a.handoffs == nil {
		a.handoffs, a.handoffAsks = map[string]handoff{}, map[string][]time.Time{}
	}
	if len(a.handoffs) >= handoffMax || len(a.handoffAsks[s.Email]) >= handoffPerSpell {
		failure(w, 429, "rate_limited")
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	tok := base64.RawURLEncoding.EncodeToString(raw)
	now := a.now()
	a.handoffs[handoffKey(tok)] = handoff{Email: s.Email, PaymentID: in.PaymentID, Origin: key, Expires: now.Add(handoffTTL)}
	a.handoffAsks[s.Email] = append(a.handoffAsks[s.Email], now)
	respond(w, 200, map[string]any{"token": tok, "expires_in": int(handoffTTL / time.Second)})
}

// handoffRedeem: POST /api/handoff/redeem {token}, no session needed. Every failure about the token says the same thing. Called under a.mu.
func (a *API) handoffRedeem(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if decode(w, r, &in) != nil || !handoffToken.MatchString(in.Token) {
		failure(w, 404, "invalid_or_expired")
		return
	}
	a.pruneHandoffs()
	k := handoffKey(in.Token)
	h, ok := a.handoffs[k]
	now := a.now()
	// This browser may already be signed in (as the sender, or as someone else, or with another hand-over). Someone else is never replaced.
	_, current, signedIn := a.current(r)
	if ok && signedIn && current.Email != h.Email && (current.Verified || current.Scope != "") {
		failure(w, 409, "session_conflict") // the right is not used up: sign out here, then open the link again
		return
	}
	delete(a.handoffs, k) // one use, whatever follows
	origin, still := a.db.state.Sessions[h.Origin]
	if !ok || !now.Before(h.Expires) || !still || !origin.Verified || !now.Before(origin.Expires) || origin.Email != h.Email {
		failure(w, 404, "invalid_or_expired")
		return
	}
	p, status, code := a.fundable(h.Email, h.PaymentID)
	if code != "" {
		failure(w, status, code)
		return
	}
	if signedIn && current.Verified && current.Email == h.Email { // the same sender is already here: nothing to hand over
		respond(w, 200, map[string]any{"payment": p, "already_signed_in": true})
		return
	}
	expires := now.Add(scopedLifetime)
	for _, limit := range []time.Time{p.Created.Add(24 * time.Hour), origin.Expires} {
		if limit.Before(expires) {
			expires = limit
		}
	}
	raw := token()
	a.db.state.Sessions[sessionKey(raw)] = session{Email: h.Email, Verified: false, Scope: h.PaymentID, Expires: expires}
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	a.setCookie(w, raw)
	respond(w, 200, map[string]any{"payment": p, "scoped": true, "expires_at": expires.UTC()})
}

// scoped serves the other browser of a hand-over: its one payment, and nothing else. Called under a.mu.
func (a *API) scoped(w http.ResponseWriter, r *http.Request, s session) {
	path := r.URL.Path
	switch {
	case path == "/api/payments" && r.Method == "GET":
		list := []payment{}
		if saved, ok := a.db.state.Payments[s.Scope]; ok && saved.Payment.Sender == s.Email {
			list = append(list, saved.Payment)
		}
		respond(w, 200, map[string]any{"payments": list})
		return
	case strings.HasPrefix(path, "/api/payments/") && r.Method == "POST":
		parts := strings.Split(strings.TrimPrefix(path, "/api/payments/"), "/")
		if len(parts) == 2 && parts[0] == s.Scope && scopedActions[parts[1]] {
			// the owner's name for the payment's own checks; the session itself stays unverified and scoped
			a.paymentAction(w, r, session{Email: s.Email, Verified: true, Expires: s.Expires}, path)
			return
		}
	}
	failure(w, 403, "handoff_scope")
}
