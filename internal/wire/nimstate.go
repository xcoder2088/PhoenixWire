package wire

import (
	"context"
	"time"
)

// Where a NIM card's coins are, so the sender's list can say "received" and stop offering to take back what has already left the card.
//
// The answer comes from the chain (Engine.CardState), never from the page. Asking the chain takes a moment and a.mu is held for the whole
// list request, so the list only READS what is remembered; when it is missing or stale it starts one background look and the next refresh of
// the list carries the answer. "claimed" is final for a funded card (coins do not come back to a card); the other answers are asked again
// after cardStateTTL.
//
// A card cancelled before it was paid is different: an empty card there means nothing was ever paid, not that it was received. It is watched
// for two days, because a payment that was still on its way when it was cancelled can arrive afterwards: the card then reads "unclaimed"
// (money on a card nobody will open) and the sender can take it back.
type cardState struct {
	State     string
	At        time.Time
	Looking   bool
	Cancelled bool
}

const (
	cardStateTTL   = 20 * time.Second
	cardStateLooks = 12 // cards looked at per list, newest first: the rest of a long history is not asked about
	cancelledLooks = 48 * time.Hour
)

// withCardStates fills ClaimState on the copies of a list (already sorted, newest first). Called under a.mu.
func (a *API) withCardStates(list []payment) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if a.cardStates == nil {
		a.cardStates = map[string]cardState{}
	}
	now := a.now()
	looks := 0
	for i := range list {
		p := &list[i]
		cancelled := p.Stage == "cancelled"
		if cancelled && now.Sub(p.Created) > cancelledLooks || !cancelled && p.Stage != "funded" && p.Stage != "delivery_pending" && p.Stage != "delivery_failed" && p.Stage != "sent" {
			continue
		}
		entry := a.cardStates[p.ID]
		entry.Cancelled = cancelled
		if !cancelled || entry.State == "unclaimed" || entry.State == "claiming" {
			p.ClaimState = entry.State
		}
		final := entry.State == "claimed" && !cancelled
		if final || entry.Looking || looks >= cardStateLooks || entry.State != "" && now.Sub(entry.At) < cardStateTTL {
			a.cardStates[p.ID] = entry
			continue
		}
		looks++
		entry.Looking = true
		a.cardStates[p.ID] = entry
		a.looks.Add(1)
		go func(id string) {
			defer a.looks.Done()
			a.lookCardState(id, now)
		}(p.ID)
	}
}

func (a *API) lookCardState(id string, now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	state, err := a.engine.CardState(ctx, id)
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if a.cardStates == nil {
		a.cardStates = map[string]cardState{}
	}
	entry := a.cardStates[id]
	entry.Looking = false
	if err == nil && (entry.State != "claimed" || entry.Cancelled) {
		entry.State, entry.At = state, now
	}
	a.cardStates[id] = entry
}
