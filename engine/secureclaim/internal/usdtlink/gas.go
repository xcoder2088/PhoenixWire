package usdtlink

// Gas money for a sender who holds only the token. On Polygon and Ethereum every token transfer is paid in the chain's native
// coin (POL, ETH), so a sender with only USDT/USDC could not even fund his card. The relayer therefore gives him, once per card,
// the few cents of native coin that his single transfer needs; it is recouped from the card's fee reserve, which already
// covers the relayer's work. Bounds that make this safe to offer:
//
//   - only for a card that exists, is unpaid, not cancelled or expired and not funded yet;
//   - only up to what ONE transfer needs (a cap per chain), never more than the sender is missing;
//   - once per card, at most once per address per day, and at most MaxDripsPerHour over the whole service;
//   - never below the relayer's float floor (what claims and refunds still need);
//   - the transaction hash is on disk before it is broadcast, like every payout.

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
)

var (
	ErrGasNotNeeded = errors.New("gas_not_needed")
	ErrGasCredit    = errors.New("gas_credit_exhausted")
)

// The gas money can never cost the operator more than its seed. Each confirmed payout of a card (a claim or a refund) earns
// CreditsPerPayout gifts, because the card's fee reserve is far larger than what a gift costs; each gift spends one. With no credit
// left, nothing is given until real cards complete. The seed (Chain.GasSeed, default DefaultGasSeed) is the worst case, whatever
// happens to cancelled or abandoned cards and whatever the exchange rate does.
const (
	CreditsPerPayout = 10
	DefaultGasSeed   = 5
)

const (
	// Pilot limits, deliberately small: they bound what the relayer can lose to gas money if cards are abandoned.
	MaxDripsPerHour = 2
	MaxDripsPerDay  = 10
	dripPerAddress  = 24 * time.Hour
	dripPending     = 10 * time.Minute // drips this recent are counted as still spending the relayer's float
	transferGas     = 100_000          // headroom over a token transfer (about 45-65k)
)

// dripMu serializes every GasTopUp of the service: quotas and the float floor are checked and RESERVED under it, so concurrent
// requests (different cards, different assets of the same chain) can never both pass the same quota or the same floor.
var dripMu sync.Mutex

// Drip is the gas money given for a card.
type Drip struct {
	To  string    `json:"to"`
	Wei string    `json:"wei"`
	Tx  string    `json:"tx,omitempty"`
	At  time.Time `json:"at"`
}

// DripResult is what the caller learns.
type DripResult struct {
	Tx      string   `json:"tx"`
	Wei     *big.Int `json:"-"`
	WeiText string   `json:"wei"`
	Already bool     `json:"already"` // this card was already given its gas money
}

// GasTopUp gives `sender` the native coin one token transfer needs, for this card.
func (s *Service) GasTopUp(ctx context.Context, cardID, senderToken, sender string) (DripResult, error) {
	defer s.lock(cardID)()
	r, err := s.get(cardID)
	if err != nil {
		return DripResult{}, err
	}
	if !s.senderOK(r, senderToken) {
		return DripResult{}, ErrForbidden // only the card's sender may ask for gas money for it
	}
	c := s.chains[r.Asset]
	if c.DripCap == nil || c.Floor == nil || c.Relayer == nil {
		return DripResult{}, ErrUnavailable
	}
	to, err := cl.ParseAddress(sender)
	if err != nil || s.badDestination(r, to) {
		return DripResult{}, evm.ErrInvalidAddress
	}
	if r.Drip != nil { // once per card: the same answer again
		w, _ := new(big.Int).SetString(r.Drip.Wei, 10)
		return DripResult{Tx: r.Drip.Tx, Wei: w, WeiText: r.Drip.Wei, Already: true}, nil
	}
	if err := s.refresh(ctx, r); err != nil {
		return DripResult{}, err
	}
	if inFlight(r) || s.state(r) != StateAwaitingFunds {
		return DripResult{}, ErrAuthState // only an unfunded, live card needs the sender's payment
	}
	// Quotas over ALL cards and assets, and the gas already promised but not yet mined; checked and reserved under dripMu.
	dripMu.Lock()
	defer dripMu.Unlock()
	s.mu.Lock()
	perHour, perDay, sameAddress, pending := 0, 0, false, new(big.Int)
	for _, o := range s.records {
		if o.Drip == nil {
			continue
		}
		age := s.now().Sub(o.Drip.At)
		if age < time.Hour {
			perHour++
		}
		if age < 24*time.Hour {
			perDay++
		}
		if age < dripPending && s.chains[o.Asset].Network.ChainID == c.Network.ChainID {
			if w, ok := new(big.Int).SetString(o.Drip.Wei, 10); ok {
				pending.Add(pending, w)
			}
		}
		if strings.EqualFold(o.Drip.To, to.Hex()) && age < dripPerAddress {
			sameAddress = true
		}
	}
	s.mu.Unlock()
	if perHour >= MaxDripsPerHour || perDay >= MaxDripsPerDay || sameAddress {
		return DripResult{}, ErrRateLimited
	}
	if s.gasCredits(c.Network.ChainID, c.GasSeed) <= 0 {
		return DripResult{}, ErrGasCredit
	}
	// What the sender is missing: enough for one transfer at the current price, with a margin.
	_, maxFee, err := c.Node.Fees(ctx)
	if err != nil {
		return DripResult{}, evm.ErrRPCUnavailable
	}
	need := new(big.Int).Mul(maxFee, big.NewInt(transferGas))
	need.Mul(need, big.NewInt(3)).Div(need, big.NewInt(2))
	have, err := c.Node.NativeBalance(ctx, to)
	if err != nil {
		return DripResult{}, evm.ErrRPCUnavailable
	}
	missing := new(big.Int).Sub(need, have)
	if missing.Sign() <= 0 {
		return DripResult{}, ErrGasNotNeeded
	}
	if missing.Cmp(c.DripCap) > 0 {
		missing = new(big.Int).Set(c.DripCap)
	}
	// Reserve it BEFORE anything is sent (on disk), then send; a refusal releases the reservation.
	drip := &Drip{To: to.Hex(), Wei: missing.String(), At: s.now().UTC()}
	r.Drip = drip
	if err := s.update(r); err != nil {
		return DripResult{}, err
	}
	floor := new(big.Int).Add(c.Floor, pending) // the float, plus what earlier drips are still going to spend
	hash, err := c.Relayer.SendNative(ctx, to, missing, floor, func(h string) error {
		drip.Tx = h
		return s.update(r) // the hash is on disk before it is broadcast
	})
	if err != nil {
		if hash != "" { // broadcast answer lost: the hash is journaled, the chain decides
			return DripResult{Tx: hash, Wei: missing, WeiText: missing.String()}, nil
		}
		if r.Drip != nil { // the journal was written but nothing was sent (refused before): release it
			r.Drip = nil
			_ = s.update(r)
		}
		return DripResult{}, err
	}
	return DripResult{Tx: hash, Wei: missing, WeiText: missing.String()}, nil
}

// gasCredits is the number of gifts still available on a chain: the seed, plus what completed payouts earned, minus what was
// given. It is derived from the records themselves, so nothing extra can drift or be forgotten.
func (s *Service) gasCredits(chainID uint64, seed int) int {
	if seed <= 0 {
		seed = DefaultGasSeed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	credits := seed
	for _, r := range s.records {
		if s.chains[r.Asset].Network.ChainID != chainID {
			continue
		}
		credits += int(r.Settled) * CreditsPerPayout
		if r.Drip != nil {
			credits--
		}
	}
	return credits
}

// GasStatus tells whether PhoenixWire can pay the gas of a sender's payment right now: ok, or the reason it cannot (off: the
// feature is not enabled; credit: no gift is earned yet; float: the relayer's reserve is too low; busy: the hourly or daily limit).
// It is what the green/red light of the payment screen shows.
func (s *Service) GasStatus(ctx context.Context, asset string) (bool, string) {
	c, ok := s.chains[asset]
	if !ok || c.DripCap == nil || c.Floor == nil || c.Relayer == nil {
		return false, "off"
	}
	if s.gasCredits(c.Network.ChainID, c.GasSeed) <= 0 {
		return false, "credit"
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	have, err := c.Node.NativeBalance(ctx, c.Relayer.Address())
	if err != nil || have.Cmp(new(big.Int).Add(c.Floor, c.DripCap)) < 0 {
		return false, "float"
	}
	s.mu.Lock()
	perHour, perDay := 0, 0
	for _, o := range s.records {
		if o.Drip == nil {
			continue
		}
		if age := s.now().Sub(o.Drip.At); age < time.Hour {
			perHour++
			perDay++
		} else if age < 24*time.Hour {
			perDay++
		}
	}
	s.mu.Unlock()
	if perHour >= MaxDripsPerHour || perDay >= MaxDripsPerDay {
		return false, "busy"
	}
	return true, "ok"
}
