package wire

import (
	"context"
	"log"
	"time"

	"github.com/xcoder2088/PhoenixWire/internal/cards"
)

// The end of a card. Every card lives 24 hours from its creation.
//
//   - An hour before the end, the SENDER gets one reminder if the card is still unopened (NIM and USDC / USDT).
//   - NIM: when the time is over and nobody opened the card, Wire pays the money back by itself, to the address that funded the card, and tells
//     the sender. The card then has the stage "refunded" ("refunding" while the payment is not in a block). A card the recipient already opened (its
//     Hub link was released) or one that is empty is left alone.
//   - USDC / USDT: the card is a Safe that needs the sender's link key to move, which only the sender's browser has: Wire cannot send the money back by
//     itself and never will without that key. When the time is over the sender gets one e-mail saying the money is still on the card and Wire's page
//     takes it back for him when he opens it.
//
// A refund keeps the relayer's fee (a NIM card has none). All of this is done by a sweep every few minutes; each step is written down before and
// each notice is sent once.
const (
	cardLifetime     = 24 * time.Hour
	reminderBefore   = time.Hour
	sweepEvery       = 5 * time.Minute
	sweepFirstDelay  = time.Minute
	refundRetryAfter = time.Hour // a card the engine would not refund (opened, empty, origin not shown) is asked about again after this
)

func (a *API) sweepLoop() {
	defer a.looks.Done()
	timer := time.NewTimer(sweepFirstDelay)
	defer timer.Stop()
	for {
		select {
		case <-a.stop:
			return
		case <-timer.C:
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			go func() { // a sweep ends with the process
				select {
				case <-a.stop:
					cancel()
				case <-ctx.Done():
				}
			}()
			a.sweepExpiry(ctx)
			cancel()
			timer.Reset(sweepEvery)
		}
	}
}

// noticeSent says whether a notice of this kind was sent for this card. Called under a.mu.
func (a *API) noticeSent(id, kind string) bool {
	_, ok := a.db.state.Notices[id][kind]
	return ok
}

// markNotice writes down that a notice was sent (before the next one can be: once per kind). Called under a.mu.
func (a *API) markNotice(id, kind string) {
	if a.db.state.Notices == nil {
		a.db.state.Notices = map[string]map[string]time.Time{}
	}
	if a.db.state.Notices[id] == nil {
		a.db.state.Notices[id] = map[string]time.Time{}
	}
	a.db.state.Notices[id][kind] = a.now()
	_ = a.db.save()
}

func (a *API) sendNotice(ctx context.Context, to, kind, symbol, amount, recipient, lang string, auto bool) error {
	return a.mail.Notice(ctx, to, cards.Notice{Kind: kind, Amount: amount, Symbol: symbol, Recipient: recipient, URL: a.origin + "/", Auto: auto}, lang)
}

// sweepExpiry does one pass over every card Wire knows.
func (a *API) sweepExpiry(ctx context.Context) {
	a.mu.Lock()
	now := a.now()
	var nim []payment
	for _, saved := range a.db.state.Payments {
		switch saved.Payment.Stage {
		case "funded", "delivery_pending", "delivery_failed", "sent", "refunding":
			nim = append(nim, saved.Payment)
		}
	}
	var stable []usdtCard
	for _, c := range a.db.state.USDT {
		stable = append(stable, c)
	}
	a.mu.Unlock()
	for _, p := range nim {
		if ctx.Err() != nil {
			return
		}
		a.sweepNIM(ctx, p, now)
	}
	for _, c := range stable {
		if ctx.Err() != nil {
			return
		}
		a.sweepUSDT(ctx, c, now)
	}
}

func (a *API) sweepNIM(ctx context.Context, p payment, now time.Time) {
	age := now.Sub(p.Created)
	a.mu.Lock()
	reminded, refundedMailed := a.noticeSent(p.ID, "reminder"), a.noticeSent(p.ID, "refunded")
	skipUntil := a.refundSkip[p.ID]
	a.mu.Unlock()
	switch {
	case age >= cardLifetime || p.Stage == "refunding":
		if now.Before(skipUntil) {
			return
		}
		rctx, cancel := context.WithTimeout(ctx, 40*time.Second)
		state, err := a.engine.RefundExpired(rctx, p.ID, p.Luna)
		cancel()
		if err != nil {
			return // the node or the engine is not ready: next pass
		}
		switch state {
		case "pending":
			if p.Stage != "refunding" {
				p.Stage = "refunding"
				a.storePayment(p)
			}
		case "done":
			if p.Stage != "refunded" {
				p.Stage = "refunded"
				a.storePayment(p)
			}
			if !refundedMailed {
				if err := a.sendNotice(ctx, p.Sender, "refunded", "NIM", p.Amount, p.Recipient, p.Language, true); err == nil {
					a.mu.Lock()
					a.markNotice(p.ID, "refunded")
					a.mu.Unlock()
				}
			}
		default: // claimed, released, started, unknown: not Wire's to refund now
			a.mu.Lock()
			if a.refundSkip == nil {
				a.refundSkip = map[string]time.Time{}
			}
			a.refundSkip[p.ID] = now.Add(refundRetryAfter)
			a.mu.Unlock()
		}
	case age >= cardLifetime-reminderBefore && !reminded && p.Stage != "refunded":
		cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		state, err := a.engine.CardState(cctx, p.ID)
		cancel()
		if err != nil || state != "unclaimed" { // only a card that still holds its money and was not opened is worth a reminder
			return
		}
		if err := a.sendNotice(ctx, p.Sender, "reminder", "NIM", p.Amount, p.Recipient, p.Language, true); err != nil {
			log.Printf("expiry: reminder not sent for a NIM card: %v", err)
			return
		}
		a.mu.Lock()
		a.markNotice(p.ID, "reminder")
		a.mu.Unlock()
	}
}

func (a *API) sweepUSDT(ctx context.Context, c usdtCard, now time.Time) {
	engine := a.USDT()
	if engine == nil || !c.Invited { // a card that was never e-mailed has no recipient to wait for
		return
	}
	age := now.Sub(c.Created)
	a.mu.Lock()
	reminded, expiredMailed := a.noticeSent(c.ID, "reminder"), a.noticeSent(c.ID, "expired")
	a.mu.Unlock()
	if age < cardLifetime-reminderBefore || age < cardLifetime && reminded || age >= cardLifetime && expiredMailed {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, status, code := engineAnswer(engine.IssuerCall(cctx, "POST", "/api/v1/usdt/issuer/cards/"+c.ID+"/status", []byte("{}")))
	if code != "" || status != 0 {
		return
	}
	state := str(out, "state")
	switch {
	case age < cardLifetime && !reminded && state == "ready":
		if err := a.sendNotice(ctx, c.Sender, "reminder", c.Symbol, c.Amount, c.Recipient, c.Language, false); err != nil {
			log.Printf("expiry: reminder not sent for a stable card: %v", err)
			return
		}
		a.mu.Lock()
		a.markNotice(c.ID, "reminder")
		a.mu.Unlock()
	case age >= cardLifetime && !expiredMailed && (state == "expired" || state == "ready"):
		if err := a.sendNotice(ctx, c.Sender, "expired", c.Symbol, c.Amount, c.Recipient, c.Language, false); err != nil {
			log.Printf("expiry: end notice not sent for a stable card: %v", err)
			return
		}
		a.mu.Lock()
		a.markNotice(c.ID, "expired")
		a.mu.Unlock()
	}
}

// storePayment saves a payment outside a request. Called without a.mu.
func (a *API) storePayment(p payment) {
	a.mu.Lock()
	defer a.mu.Unlock()
	saved := a.db.state.Payments[p.ID]
	saved.Payment = p
	a.db.state.Payments[p.ID] = saved
	if err := a.db.save(); err != nil {
		log.Printf("expiry: state not saved: %v", err)
	}
}
