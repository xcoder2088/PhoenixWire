package wire

import (
	"context"
	"sort"
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
//
// The reads of the chain are few per list (cardStateLooks), and they go first to the cards whose state is not known yet and then to the ones
// asked about the longest ago, so a long history is read in turns and no card is left unknown for ever.
func (a *API) withCardStates(list []payment) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if a.cardStates == nil {
		a.cardStates = map[string]cardState{}
	}
	now := a.now()
	type due struct {
		id string
		at time.Time
	}
	var wanting []due
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
		if cancelled && entry.State == "claimed" {
			p.ClaimState = "" // nothing was ever paid to it: an empty card is not a received one
		}
		final := entry.State == "claimed" && !cancelled
		a.cardStates[p.ID] = entry
		if final || entry.Looking || entry.State != "" && now.Sub(entry.At) < cardStateTTL {
			continue
		}
		wanting = append(wanting, due{p.ID, entry.At}) // zero time: never read, first in line
	}
	sort.SliceStable(wanting, func(i, j int) bool { return wanting[i].at.Before(wanting[j].at) })
	if len(wanting) > cardStateLooks {
		wanting = wanting[:cardStateLooks]
	}
	for _, w := range wanting {
		entry := a.cardStates[w.id]
		entry.Looking = true
		a.cardStates[w.id] = entry
		a.looks.Add(1)
		go func(id string) {
			defer a.looks.Done()
			a.lookCardState(id, now)
		}(w.id)
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

// cardStateNow answers where a funded card's coins are, asking the chain now when nothing fresh is remembered ("claimed" is final, so it is
// never asked twice). Called under a.mu by the recipient's routes, which can afford one look: the list of the sender cannot (withCardStates).
func (a *API) cardStateNow(id string) string { return a.cardStateAsk(id, false) }

// cardStateAsk is cardStateNow; fresh asks the chain even when a recent answer is remembered (before a code is sent, a card emptied a few
// seconds ago must not read "unclaimed" from the cache).
func (a *API) cardStateAsk(id string, fresh bool) string {
	a.stateMu.Lock()
	entry := a.cardStates[id]
	a.stateMu.Unlock()
	if entry.State == "claimed" && !entry.Cancelled || !fresh && entry.State != "" && a.now().Sub(entry.At) < cardStateTTL {
		return entry.State
	}
	a.lookCardState(id, a.now())
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.cardStates[id].State
}
