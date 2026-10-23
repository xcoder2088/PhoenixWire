package wire

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"
)

// The hand-over of ONE card's code from the recipient's browser to the Nimiq Pay app, so the code is asked once.
//
// The recipient types the e-mailed code where the card was opened (an ordinary browser) and chooses Nimiq Pay. The app is another browser
// with nothing in common, so the code crosses over as a right, like the sender's hand-over (handoff.go): an opaque token (43 random
// characters; only its hash is kept, in memory, never logged), one use, five minutes, tied to that card. The code itself is held in memory
// for those five minutes at most and is never written to disk, never put in a link. The token only gives the code back: spending it still
// needs the destination, which the app takes from the person's own wallet.
//
// A lost answer to the redeem loses the right (it was used): the person types the code again in the app.
type claimHandoff struct {
	ClaimID string
	Code    string
	Expires time.Time
}

const (
	claimHandoffMax      = 1000
	claimHandoffPerSpell = 6 // rights asked for one card in handoffSpell
)

func (a *API) pruneClaimHandoffs() {
	now := a.now()
	for k, h := range a.claimHandoffs {
		if !now.Before(h.Expires) {
			delete(a.claimHandoffs, k)
		}
	}
	for id, times := range a.claimAsks {
		kept := times[:0]
		for _, at := range times {
			if now.Sub(at) < handoffSpell {
				kept = append(kept, at)
			}
		}
		if len(kept) == 0 {
			delete(a.claimAsks, id)
		} else {
			a.claimAsks[id] = kept
		}
	}
}

// claimHandoffCreate: POST /api/claims/<id>/handoff {code}. Called under a.mu, for a card that is open to its recipient.
func (a *API) claimHandoffCreate(w http.ResponseWriter, r *http.Request, id string) {
	var in struct {
		Code string `json:"code"`
	}
	if decode(w, r, &in) != nil || !codeRE.MatchString(in.Code) {
		failure(w, 400, "invalid_request")
		return
	}
	a.pruneClaimHandoffs()
	if a.claimHandoffs == nil {
		a.claimHandoffs, a.claimAsks = map[string]claimHandoff{}, map[string][]time.Time{}
	}
	if len(a.claimHandoffs) >= claimHandoffMax || len(a.claimAsks[id]) >= claimHandoffPerSpell {
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
	a.claimHandoffs[handoffKey(tok)] = claimHandoff{ClaimID: id, Code: in.Code, Expires: now.Add(handoffTTL)}
	a.claimAsks[id] = append(a.claimAsks[id], now)
	respond(w, 200, map[string]any{"token": tok, "expires_in": int(handoffTTL / time.Second)})
}

// claimHandoffRedeem: POST /api/claims/<id>/handoff/redeem {token}. Every failure says the same thing. Called under a.mu.
func (a *API) claimHandoffRedeem(w http.ResponseWriter, r *http.Request, id string) {
	var in struct {
		Token string `json:"token"`
	}
	if decode(w, r, &in) != nil || !handoffToken.MatchString(in.Token) {
		failure(w, 404, "invalid_or_expired")
		return
	}
	a.pruneClaimHandoffs()
	k := handoffKey(in.Token)
	h, ok := a.claimHandoffs[k]
	delete(a.claimHandoffs, k) // one use, whatever follows
	if !ok || h.ClaimID != id || !a.now().Before(h.Expires) {
		failure(w, 404, "invalid_or_expired")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	respond(w, 200, map[string]string{"code": h.Code})
}
