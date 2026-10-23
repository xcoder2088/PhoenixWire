package usdtlink

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/relayer"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
)

// What the owner sees and does with the relayers: how much each one holds (the native coin that pays the gas, and the fees it
// earned in USDC/USDT) and a way to take it all out. Only the owner's PhoenixMail administrator reaches this (issuer key).

// TokenBalance is one token held by a relayer.
type TokenBalance struct {
	Asset  string `json:"asset"`
	Symbol string `json:"symbol"`
	Units  string `json:"units"` // token units (6 decimals), as text
}

// RelayerInfo is one relayer (one per chain; both Polygon assets share one).
type RelayerInfo struct {
	Chain        string         `json:"chain"`
	ChainID      uint64         `json:"chain_id"`
	Address      string         `json:"address"`
	NativeSymbol string         `json:"native_symbol"`
	NativeWei    string         `json:"native_wei"`
	Tokens       []TokenBalance `json:"tokens"`
	PendingCards int            `json:"pending_cards"` // funded cards not yet paid out: they still need gas from this relayer
	LiveCards    int            `json:"live_cards"`    // cards that may still need gas: waiting for their payment, funded, or being paid
	ReserveWei   string         `json:"reserve_wei"`   // the native coin a withdrawal must leave: nothing when no card is live
	WithdrawWei  string         `json:"withdrawable_wei"`
	MinFloatWei  string         `json:"min_float_wei,omitempty"`
}

// gasPerCard is the native coin kept back for each live card (its deployment and its payout, with a margin).
func gasPerCard(chainID uint64) *big.Int {
	if chainID == 137 {
		return big.NewInt(150_000_000_000_000_000) // 0.15 POL
	}
	return big.NewInt(1_000_000_000_000_000) // 0.001 ETH
}

// reserveFor is what a withdrawal of the native coin must leave on the relayer: nothing when no card can still need gas (the
// owner may then empty it, and the service simply takes no new card), otherwise the minimum float plus gas for every live card.
func reserveFor(chainID uint64, minFloat *big.Int, live int) *big.Int {
	if live == 0 {
		return new(big.Int)
	}
	r := new(big.Int).Mul(gasPerCard(chainID), big.NewInt(int64(live)))
	if minFloat != nil {
		r.Add(r, minFloat)
	}
	return r
}

func nativeSymbol(chainID uint64) string {
	if chainID == 137 {
		return "POL"
	}
	return "ETH"
}

// relayerChains groups the served assets by chain: the assets of one chain share one relayer.
func (s *Service) relayerChains() map[uint64][]string {
	out := map[uint64][]string{}
	for asset, c := range s.chains {
		if c.Relayer != nil {
			out[c.Network.ChainID] = append(out[c.Network.ChainID], asset)
		}
	}
	for _, assets := range out {
		sort.Strings(assets)
	}
	return out
}

// RelayerInfos reads every relayer's balances from the chain.
func (s *Service) RelayerInfos(ctx context.Context) ([]RelayerInfo, error) {
	groups := s.relayerChains()
	var ids []uint64
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var out []RelayerInfo
	for _, id := range ids {
		assets := groups[id]
		first := s.chains[assets[0]]
		addr := first.Relayer.Address()
		native, err := first.Node.NativeBalance(ctx, addr)
		if err != nil {
			return nil, evm.ErrRPCUnavailable
		}
		info := RelayerInfo{Chain: first.Network.Chain, ChainID: id, Address: addr.Hex(), NativeSymbol: nativeSymbol(id), NativeWei: native.String()}
		if first.MinFloat != nil {
			info.MinFloatWei = first.MinFloat.String()
		}
		for _, asset := range assets {
			c := s.chains[asset]
			token, err := cl.ParseAddress(c.Network.Token)
			if err != nil {
				return nil, ErrInvalid
			}
			out2, err := c.Node.Call(ctx, cl.Address{}, token, append([]byte{0x70, 0xa0, 0x82, 0x31}, addr.Word()...))
			if err != nil || len(out2) < 32 {
				return nil, evm.ErrRPCUnavailable
			}
			info.Tokens = append(info.Tokens, TokenBalance{Asset: asset, Symbol: c.Network.Symbol, Units: new(big.Int).SetBytes(out2[:32]).String()})
		}
		_ = s.refreshHeld(ctx, id) // for display; a withdrawal below checks it
		info.PendingCards, info.LiveCards = s.cardsNeedingGas(id)
		reserve := reserveFor(id, first.MinFloat, info.LiveCards)
		info.ReserveWei = reserve.String()
		if w := new(big.Int).Sub(native, reserve); w.Sign() > 0 {
			info.WithdrawWei = w.String()
		} else {
			info.WithdrawWei = "0"
		}
		out = append(out, info)
	}
	return out, nil
}

// needsGas says whether this card may still need gas from its relayer, whatever its displayed state: it waits for its payment, it is
// funded or being paid, a refund was prepared and not yet done (preparing one marks the card cancelled at once), or it HOLDS money
// (a part-payment, a leftover, an expired or cancelled card that was funded: all recoverable). What a card holds that was never read
// is unknown, and an unknown is counted: a terminal state is not an empty Safe.
func (s *Service) needsGas(r *Record) bool {
	switch s.state(r) {
	case StateAwaitingFunds, StateReady, StateVerifying:
		return true
	}
	if inFlight(r) {
		return true
	}
	if r.Refund != nil && !(r.Op != nil && r.Op.Kind == "refund" && r.Op.Done && !r.Op.Failed) {
		return true
	}
	return r.Funded || r.Held > 0 || r.HeldAt.IsZero()
}

// ErrBalanceUnknown: a balance a withdrawal depends on could be neither read nor recorded as unknown.
var ErrBalanceUnknown = errors.New("balance_unreadable")

// refreshHeld reads, for every card of this chain, what it holds NOW (the latest block, not only the deep one), so that the reserve
// of a withdrawal rests on balances and not on displayed states. A card whose balance cannot be read is recorded as UNKNOWN (an old
// reading, even a zero, is not kept): unknown counts as needing gas. If even that cannot be recorded, it says so and the caller
// must not go on.
func (s *Service) refreshHeld(ctx context.Context, chainID uint64) error {
	s.mu.Lock()
	var recs []*Record
	for _, r := range s.records {
		if c, ours := s.chains[r.Asset]; ours && c.Network.ChainID == chainID {
			recs = append(recs, r.clone())
		}
	}
	s.mu.Unlock()
	var failed error
	for _, r := range recs {
		st := s.state(r)
		if (st == StateAwaitingFunds || st == StateReady) && !r.HeldAt.IsZero() && s.now().Sub(r.HeldAt) < time.Minute {
			continue // read a moment ago
		}
		if !r.HeldAt.IsZero() && r.Held == 0 && (st == StatePaid || st == StateWithdrawn || st == StateSettledOther) {
			continue // finished and empty: it does not need reading again
		}
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		bal, err := s.balance(cctx, r)
		cancel()
		unlock := s.lock(r.ID)
		if fresh, gerr := s.get(r.ID); gerr == nil {
			if err != nil {
				fresh.HeldAt = time.Time{} // the old reading is no longer believed
			} else {
				fresh.Held, fresh.HeldAt = heldUnits(bal), s.now().UTC()
			}
			if uerr := s.update(fresh); uerr != nil {
				failed = ErrBalanceUnknown // nothing recorded: what the card holds is not known
			}
		}
		unlock()
	}
	return failed
}

// cardsNeedingGas counts, for one chain, the funded cards not yet paid out (pending) and every card that may still need gas (live).
func (s *Service) cardsNeedingGas(chainID uint64) (pending, live int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.records {
		c, ours := s.chains[r.Asset]
		if !ours || c.Network.ChainID != chainID || !s.needsGas(r) {
			continue
		}
		live++
		if st := s.state(r); st == StateReady || st == StateVerifying {
			pending++
		}
	}
	return pending, live
}

// WithdrawResult is what a withdrawal answers. Status is "sent" (the node accepted the transaction) or "unconfirmed" (the hash
// exists and was written down, but the broadcast answer was an error: the transaction may or may not be on the network).
type WithdrawResult struct {
	Tx       string `json:"tx"`
	Amount   string `json:"amount"` // token units, or wei for the native coin
	Explorer string `json:"explorer"`
	Status   string `json:"status"`
}

var (
	// ErrWithdrawDisabled: no destination address was authorized by the operator (NIMIQ_CLAIM_USDTLINK_WITHDRAW_TO).
	ErrWithdrawDisabled = errors.New("withdraw_not_configured")
	// ErrWithdrawDestination: the address is not on the operator's list.
	ErrWithdrawDestination = errors.New("withdraw_destination_not_allowed")
)

// SetWithdrawAddresses sets the ONLY addresses a withdrawal may go to. None configured: no withdrawal at all. A stolen
// administrator session can therefore never send the relayer's money to an address of its own choosing.
func (s *Service) SetWithdrawAddresses(addrs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.withdrawTo = map[string]bool{}
	for _, a := range addrs {
		if p, err := cl.ParseAddress(strings.TrimSpace(a)); err == nil {
			s.withdrawTo[strings.ToLower(p.Hex())] = true
		}
	}
}

// WithdrawAddresses lists the authorized destinations.
func (s *Service) WithdrawAddresses() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for a := range s.withdrawTo {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// ErrReserveRequired: the gas of this very withdrawal would eat into what the live cards need.
var ErrReserveRequired = errors.New("reserve_required")

// ErrNothingToWithdraw: the relayer holds none of what was asked for.
var ErrNothingToWithdraw = errors.New("nothing_to_withdraw")

var withdrawMu sync.Mutex

// RelayerWithdraw takes money out of a relayer, to an address the operator authorized: what is "native" (the coin that pays the
// gas: only what is above the reserve the live cards need, or everything when no card is live) or an asset id (ALL the fees earned
// in that token). Every withdrawal is written to withdrawals.log before it is broadcast.
func (s *Service) RelayerWithdraw(ctx context.Context, chainName, what, to string) (WithdrawResult, error) {
	dest, err := cl.ParseAddress(to)
	if err != nil {
		return WithdrawResult{}, evm.ErrInvalidAddress
	}
	s.mu.Lock()
	configured, allowed := len(s.withdrawTo) > 0, s.withdrawTo[strings.ToLower(dest.Hex())]
	s.mu.Unlock()
	if !configured {
		return WithdrawResult{}, ErrWithdrawDisabled
	}
	if !allowed {
		return WithdrawResult{}, ErrWithdrawDestination
	}
	var group []string
	for _, assets := range s.relayerChains() {
		if s.chains[assets[0]].Network.Chain == chainName {
			group = assets
			break
		}
	}
	if len(group) == 0 {
		return WithdrawResult{}, ErrNotFound
	}
	c := s.chains[group[0]]
	withdrawMu.Lock()
	defer withdrawMu.Unlock()
	journal := func(kind, amountText string) func(hash string) error {
		return func(hash string) error {
			line, _ := json.Marshal(map[string]string{"at": s.now().UTC().Format(time.RFC3339), "chain": chainName, "what": kind, "to": dest.Hex(), "tx": hash, "amount": amountText})
			f, err := os.OpenFile(filepath.Join(s.dir, "withdrawals.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := f.Write(append(line, '\n')); err != nil {
				return err
			}
			return f.Sync()
		}
	}
	// A broadcast that returned an error but left a hash is NOT a success and NOT a failure: the transaction may be on the network.
	finish := func(hash, amount, explorer string, err error) (WithdrawResult, error) {
		if err != nil && hash == "" {
			return WithdrawResult{}, err
		}
		status := "sent"
		if err != nil {
			status = "unconfirmed"
		}
		log.Printf("relayer withdrawal: %s %s to %s tx %s (%s)", chainName, amount, dest.Hex(), hash, status)
		return WithdrawResult{Tx: hash, Amount: amount, Explorer: explorer, Status: status}, nil
	}
	if what == "native" {
		native, err := c.Node.NativeBalance(ctx, c.Relayer.Address())
		if err != nil {
			return WithdrawResult{}, evm.ErrRPCUnavailable
		}
		if err := s.refreshHeld(ctx, c.Network.ChainID); err != nil { // balances, not displayed states
			return WithdrawResult{}, err
		}
		_, live := s.cardsNeedingGas(c.Network.ChainID)
		reserve := reserveFor(c.Network.ChainID, c.MinFloat, live)
		if reserve.Sign() == 0 { // no card can still need gas: everything may go
			hash, wei, err := c.Relayer.SweepNative(ctx, dest, journal("native", "all"))
			if errors.Is(err, relayer.ErrAmount) {
				return WithdrawResult{}, ErrNothingToWithdraw
			}
			amount := "0"
			if wei != nil {
				amount = wei.String()
			}
			return finish(hash, amount, c.Network.Explorer, err)
		}
		_, maxFee, err := c.Node.Fees(ctx)
		if err != nil {
			return WithdrawResult{}, evm.ErrRPCUnavailable
		}
		wei := new(big.Int).Sub(new(big.Int).Sub(native, reserve), new(big.Int).Mul(maxFee, big.NewInt(21000)))
		if wei.Sign() <= 0 {
			return WithdrawResult{}, ErrNothingToWithdraw // everything above is kept for the cards that may still need gas
		}
		hash, err := c.Relayer.SendNative(ctx, dest, wei, reserve, journal("native", wei.String()))
		if errors.Is(err, relayer.ErrFloorReached) {
			return WithdrawResult{}, ErrNothingToWithdraw
		}
		return finish(hash, wei.String(), c.Network.Explorer, err)
	}
	asset := what
	cc, ok := s.chains[asset]
	if !ok || cc.Network.ChainID != c.Network.ChainID || !strings.EqualFold(cc.Network.Chain, chainName) {
		return WithdrawResult{}, evm.ErrUnknownAsset
	}
	token, err := cl.ParseAddress(cc.Network.Token)
	if err != nil {
		return WithdrawResult{}, ErrInvalid
	}
	addr := cc.Relayer.Address()
	out, err := cc.Node.Call(ctx, cl.Address{}, token, append([]byte{0x70, 0xa0, 0x82, 0x31}, addr.Word()...))
	if err != nil || len(out) < 32 {
		return WithdrawResult{}, evm.ErrRPCUnavailable
	}
	units := new(big.Int).SetBytes(out[:32])
	if units.Sign() <= 0 {
		return WithdrawResult{}, ErrNothingToWithdraw
	}
	// Sending a token costs gas too: its worst-case cost, at the fees read under the relayer's send lock, may not cross the reserve
	// the live cards need (the relayer itself enforces it with the floor given here).
	if err := s.refreshHeld(ctx, c.Network.ChainID); err != nil {
		return WithdrawResult{}, err
	}
	_, live := s.cardsNeedingGas(c.Network.ChainID)
	reserve := reserveFor(c.Network.ChainID, c.MinFloat, live)
	hash, err := cc.Relayer.SendToken(ctx, token, dest, units, reserve, journal(asset, units.String()))
	if errors.Is(err, relayer.ErrFloorReached) {
		return WithdrawResult{}, ErrReserveRequired
	}
	return finish(hash, units.String(), cc.Network.Explorer, err)
}
