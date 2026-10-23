package wire

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/xcoder2088/NimiqSecureClaim/service"
	"github.com/xcoder2088/PhoenixWire/internal/cards"
)

// The hand-over from the browser to the Nimiq Pay app. The app is another browser (its own cookies), so what the sender was doing is
// carried by an opaque token instead of by the address: the draft stays here, the link holds nothing but 43 random characters.
//
//   - one use: the first redeem deletes it, whatever happens next; two minutes at most;
//   - it opens a session for the SAME verified sender, never longer than the session it came from, and only while that session still exists
//     (signing out cancels what was handed over);
//   - only its hash is kept, in memory (a restart forgets it: it was going to expire anyway), it is never logged.
const (
	handoffTTL      = 2 * time.Minute
	handoffMax      = 1000
	handoffPerSpell = 6 // tokens one sender may ask for in handoffSpell
	handoffSpell    = 10 * time.Minute
)

var handoffToken = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

type handoffDraft struct {
	Recipient string `json:"recipient"`
	Amount    string `json:"amount"`
	Message   string `json:"message"`
	Language  string `json:"language"`
}

type handoff struct {
	Email          string
	Draft          handoffDraft
	Origin         string // key of the session it came from
	SessionExpires time.Time
	Expires        time.Time
}

func handoffKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// prune forgets what has expired. Called under a.mu.
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

// handoffCreate: POST /api/handoff, for a verified sender. Called under a.mu.
func (a *API) handoffCreate(w http.ResponseWriter, r *http.Request, key string, s session) {
	var in handoffDraft
	if decode(w, r, &in) != nil || utf8.RuneCountInString(in.Message) > 160 {
		failure(w, 400, "invalid_request")
		return
	}
	recipient, err := email(in.Recipient)
	if err != nil {
		failure(w, 400, "invalid_request")
		return
	}
	amount, _, err := service.ParseAmount(in.Amount)
	if err != nil {
		failure(w, 400, "invalid_request")
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
	a.handoffs[handoffKey(tok)] = handoff{Email: s.Email, Draft: handoffDraft{Recipient: recipient, Amount: amount, Message: in.Message, Language: cards.Language(in.Language)},
		Origin: key, SessionExpires: s.Expires, Expires: now.Add(handoffTTL)}
	a.handoffAsks[s.Email] = append(a.handoffAsks[s.Email], now)
	respond(w, 200, map[string]any{"token": tok, "expires_in": int(handoffTTL / time.Second)})
}

// handoffRedeem: POST /api/handoff/redeem, no session needed. Every failure says the same thing. Called under a.mu.
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
	delete(a.handoffs, k) // one use, whatever follows
	now := a.now()
	origin, still := a.db.state.Sessions[h.Origin]
	if !ok || !now.Before(h.Expires) || !still || !origin.Verified || !now.Before(origin.Expires) || origin.Email != h.Email {
		failure(w, 404, "invalid_or_expired")
		return
	}
	raw := token()
	expires := now.Add(24 * time.Hour)
	if origin.Expires.Before(expires) { // a hand-over never makes a sign-in last longer than it already would
		expires = origin.Expires
	}
	a.db.state.Sessions[sessionKey(raw)] = session{Email: h.Email, Verified: true, Expires: expires}
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	a.setCookie(w, raw)
	respond(w, 200, map[string]any{"email": h.Email, "draft": h.Draft})
}
