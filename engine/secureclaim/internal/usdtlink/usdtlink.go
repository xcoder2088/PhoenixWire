// Package usdtlink is the USDT cashlink (Ethereum / Polygon): the card is a small Safe wallet that HOLDS the money, the
// recipient claims it with one transaction and pays nothing. Every card is protected (see below): there is no open card.
//
// What this service is, and is not:
//
//   - The sender's browser creates the card (a random id and a locally generated link key) and pays amount + a small fee
//     reserve to the card's address. Only the card's id, the PUBLIC address of the link key, the amount and the optional
//     recipient email reach card creation. Separately, Wire receives the full claim link once to validate and e-mail the
//     invitation without persisting the link. Recipient claim and sender refund routes receive signatures, not the key.
//   - A card has two owners: the link key and a co-signer key derived HERE from a master secret. It pays out only if this
//     service co-signs, and it co-signs only after the recipient proved his email with a code, and only for ONE destination
//     address. Whoever merely holds the link cannot claim.
//   - The claim transaction is sent by this service's relayer, which pays the gas in POL/ETH and takes the fee reserve (USDT)
//     from the card. The relayer cannot redirect money: it rebuilds the one payout the signatures allow (see
//     internal/cashlink/relayer). The recipient pays nothing and needs no gas.
//   - The normal protected-card claim needs the link-key signature AND this service's signature after the recipient
//     email proof. Invitation delivery transiently exposes the full link to Wire and its mail provider; it is not a
//     guarantee that the operator can never see the link key.
//
// The Safe/relayer/co-signer code is the vendored PhoenixCashlink library (internal/cashlink, mirror: do not edit it there).
package usdtlink

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/chain"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/cosigner"
	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/relayer"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/safe"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
)

var (
	ErrNotFound    = errors.New("claim_not_found")
	ErrExpired     = errors.New("claim_expired")
	ErrCancelled   = errors.New("card_cancelled")
	ErrNotVerified = errors.New("email_not_verified")
	ErrForbidden   = errors.New("forbidden")
	ErrNotFunded   = errors.New("card_not_funded")
	ErrAddressSet  = errors.New("address_already_given")
	ErrPaid        = errors.New("already_paid")
	ErrCapacity    = errors.New("capacity_reached")
	ErrUnavailable = errors.New("usdt_unavailable")
	ErrDuplicate   = errors.New("card_exists")
	ErrInvalid     = errors.New("invalid_request")
	ErrPending     = errors.New("payment_in_progress")
	ErrFloatLow    = errors.New("relayer_float_low")
)

const (
	// Lifetime is how long the claim link works. After it the sender simply takes his money back with his own key.
	Lifetime = 14 * 24 * time.Hour
	MaxCards = 20000
)

// States of a card. Paid and withdrawn come only from a confirmed receipt, never from a balance.
const (
	StateAwaitingFunds = "awaiting_funds"    // created; the sender has not paid yet
	StateReady         = "ready"             // funded: the recipient can claim
	StatePaid          = "paid"              // the recipient was paid
	StateWithdrawn     = "withdrawn"         // the sender took the money back (a refund we relayed)
	StateVerifying     = "verifying"         // a payment was sent and its outcome is being confirmed: never shown as failed
	StateSettledOther  = "settled_elsewhere" // the card was spent by a signature this service did not relay (see refresh)
	StateCancelled     = "cancelled"         // the sender asked to cancel: a protected card is no longer co-signed
	StateExpired       = "expired"
)

// DefaultFee is the fee reserve (USDT units) a card pays the relayer for the gas of its deployment and its claim. It is the
// default the sender's page proposes; the relayer's own floor (Network.MinFee) always applies.
var DefaultFee = map[string]uint64{evm.AssetUSDTPolygon: 100_000, evm.AssetUSDTEthereum: 750_000, evm.AssetUSDCAmoyTest: 100_000, evm.AssetUSDCSepoliaTest: 100_000, evm.AssetUSDCPolygon: 100_000}

// FeeBps is the service fee, in basis points of the card's amount (600 = 6%). It is ADDED to the gas of the card (fee.go): a quiet
// network does not lower it and a busy one does not eat it. The sender sees ONE amount to pay, never a percentage.
var FeeBps uint64 = 600

// MaxFeeFor is the highest fee a card of this amount may carry: 10 token units or a tenth of the amount, whichever is larger.
func MaxFeeFor(units uint64) uint64 {
	if m := units / 10; m > 10_000_000 {
		return m
	}
	return 10_000_000
}

// refundFee is what a REFUND pays the relayer. Cards made since the fee followed the price of gas keep their whole fee (gas and
// percentage: RefundFeeUnits, fixed at creation); older cards, which have none, pay only the flat gas fee.
func (s *Service) refundFee(r *Record) uint64 {
	if r.RefundFeeUnits > 0 && r.RefundFeeUnits <= r.FeeUnits {
		return r.RefundFeeUnits
	}
	if flat := DefaultFee[r.Asset]; flat > 0 && flat < r.FeeUnits {
		return flat
	}
	return r.FeeUnits
}

// heldUnits is a balance as token units; a balance beyond 64 bits is clamped (it is "a lot", never "nothing").
func heldUnits(b *big.Int) uint64 {
	if b == nil || b.Sign() <= 0 {
		return 0
	}
	if !b.IsUint64() {
		return ^uint64(0)
	}
	return b.Uint64()
}

// Chain is one supported network: its public description, the node client, and its relayer.
type Chain struct {
	Network evm.Network
	Node    chain.Chain
	Relayer *relayer.Relayer
	MinFee  uint64
	// DripCap and Floor enable GasTopUp (gas.go): the most native coin one sender is given, and the float the relayer keeps.
	DripCap *big.Int
	Floor   *big.Int
	// MinFloat is the native coin the relayer must hold for the service to accept NEW cards: below it, claims and refunds could not
	// be paid, so no new card is taken (cards already funded keep their refund route as long as some gas remains).
	MinFloat *big.Int
	// FloatMin, when set, makes the float follow the price of gas (see minFloat): the service keeps enough native coin for FloatCards cards at
	// today's gas price, never less than FloatMin and never more than MinFloat. Unset: MinFloat is the floor, whatever gas costs.
	FloatMin *big.Int
	// PriceFeed is the address of the chain's USD price feed for its coin (Chainlink) and FallbackUSD a deliberately high price used when the
	// feed cannot be read: the fee of a new card follows the price of gas (fee.go).
	PriceFeed   string
	CoinGeckoID string // asked when the feed cannot be read
	FallbackUSD float64
	// GasSeed is how many gifts of gas money the service may give before any payout has completed (default DefaultGasSeed).
	GasSeed int
}

// Record does not persist the link key. Wire transiently receives the full link during invitation delivery.
// The co-signer key is derived from the master secret when needed and is not stored in the record.
type Record struct {
	ID             string    `json:"id"` // 32 hex characters: the card id chosen by the sender's browser
	Asset          string    `json:"asset"`
	Units          uint64    `json:"units"` // paid to the recipient
	FeeUnits       uint64    `json:"fee_units"`
	RefundFeeUnits uint64    `json:"refund_fee_units,omitempty"` // what a refund pays the relayer, fixed at creation (absent on older cards: the flat fee)
	Amount         string    `json:"amount"`
	Message        string    `json:"message,omitempty"`
	Protected      bool      `json:"protected"`
	Email          string    `json:"email,omitempty"` // recipient, protected cards only
	SenderEmail    string    `json:"sender_email,omitempty"`
	SenderLang     string    `json:"sender_lang,omitempty"`
	LinkOwner      string    `json:"link_owner"` // address of the link key (public): the one a link proof must come from
	Owners         []string  `json:"owners"`     // sorted checksum addresses: the link key's, and the co-signer's when protected
	Safe           string    `json:"safe"`       // the card's address
	Created        time.Time `json:"created"`
	SenderHash     string    `json:"sender_hash"`
	Verified       bool      `json:"verified"`
	Recipient      string    `json:"recipient,omitempty"` // the one destination this card may pay (set when it is co-signed or claimed)
	Funded         bool      `json:"funded,omitempty"`    // the card holds the full amount NOW (as last read)
	Held           uint64    `json:"held,omitempty"`      // token units the card holds, as last read (a part-payment or a leftover is recoverable money)
	HeldAt         time.Time `json:"held_at,omitempty"`   // when Held was last read; zero: never read, so what the card holds is UNKNOWN
	Cancelled      bool      `json:"cancelled,omitempty"`
	ClaimTx        string    `json:"claim_tx,omitempty"`      // the confirmed payout to the recipient (kept when a later refund recovers a surplus)
	ClaimedAt      time.Time `json:"claimed_at,omitempty"`    // when the block that holds that payout was mined; zero: not known (never made up from the time of a look)
	Settled        uint64    `json:"settled,omitempty"`       // payouts of ours confirmed on chain: the Safe nonce we expect
	SettledOther   bool      `json:"settled_other,omitempty"` // the Safe nonce moved without any payout of ours
	// Op is the one operation (claim or refund) in flight or last settled. It is written to disk BEFORE the transaction is
	// broadcast, hash included (Journal), so that whatever happens next (crash, lost answer, failed write) the chain can say
	// what became of it. The Safe's own nonce on chain is the other source of truth.
	Op *Operation `json:"op,omitempty"`
	// Refund is the sender's refund intent: ONE destination, fixed once chosen, until that refund is settled.
	Refund *Intent `json:"refund,omitempty"`
	// Late are operations released after dropAfter without any receipt. They stay watched: the Safe nonce makes a second
	// execution of the same payout harmless, but a late first one must still be recognised when it mines.
	Late []*Operation `json:"late,omitempty"`
	// Anomaly is set when a payout succeeded on chain but did not emit the transfers it must (see refresh): never shown as paid.
	Anomaly string `json:"anomaly,omitempty"`
	// Drip is the gas money the relayer gave the sender for this card (gas.go).
	Drip *Drip `json:"drip,omitempty"`
}

// Operation is a payout this service sent (or is sending) for the card.
type Operation struct {
	Kind    string    `json:"kind"` // "claim" | "refund"
	To      string    `json:"to"`
	Units   uint64    `json:"units"`
	Nonce   uint64    `json:"nonce"` // the Safe nonce the signatures were made for
	Started time.Time `json:"started"`
	Tx      string    `json:"tx,omitempty"`     // hash of the exec transaction, journaled before it was broadcast
	Deploy  string    `json:"deploy,omitempty"` // hash of the deployment, when this operation deployed the card
	Done    bool      `json:"done,omitempty"`   // settled from the chain receipt
	Failed  bool      `json:"failed,omitempty"` // mined and reverted, or never executed: nothing was paid
}

// Intent is a refund the sender asked for: where the money goes and how much, for a given Safe nonce.
type Intent struct {
	To    string `json:"to"`
	Units uint64 `json:"units"`
	Nonce uint64 `json:"nonce"`
}

// Public is what anyone holding the link may read: no email, no recipient address.
type Public struct {
	ID        string    `json:"id"`
	Asset     string    `json:"asset"`
	Chain     string    `json:"chain"`
	ChainID   uint64    `json:"chain_id"`
	Amount    string    `json:"amount"`
	Units     uint64    `json:"units"`
	FeeUnits  uint64    `json:"fee_units"`
	RefundFee uint64    `json:"refund_fee_units"` // what a refund of this card pays the relayer: the flat gas fee, fixed at creation
	Symbol    string    `json:"symbol"`
	Protected bool      `json:"protected"`
	Verified  bool      `json:"email_verified"`
	State     string    `json:"state"`
	Owners    []string  `json:"owners"`
	Threshold uint64    `json:"threshold"`
	Safe      string    `json:"safe"`
	Token     string    `json:"token"`
	Relayer   string    `json:"relayer"`
	TxHash    string    `json:"tx_hash,omitempty"`
	TxURL     string    `json:"tx_url,omitempty"`
	TxKind    string    `json:"tx_kind,omitempty"` // "claim" or "refund"
	Explorer  string    `json:"explorer"`          // the network explorer's transaction URL prefix
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message,omitempty"`
}

// Issuer is what the sender's app may read: the public view plus the recipient's destination once known.
type Issuer struct {
	Public
	Recipient    string `json:"recipient,omitempty"`
	Anomaly      string `json:"anomaly,omitempty"`
	Recovering   bool   `json:"recovering,omitempty"` // a refund is in flight, even if the card shows as paid
	BalanceUnits uint64 `json:"balance_units"`        // what the card holds now, last read (a surplus or a late deposit is visible and refundable)
	Stale        bool   `json:"stale,omitempty"`      // the chain could not be read: the state shown is the last known one
	// ClaimedAt is when the recipient's payout was mined (the block's own timestamp), once that payout is confirmed and verified. A refund is
	// never a claim; a card paid before this was kept has it filled in from the receipt of its payout, or not at all.
	ClaimedAt *time.Time `json:"claimed_at,omitempty"`
}

// Input is a card to create.
type Input struct {
	Asset       string
	CardID      string // 32 hex characters, chosen by the sender's browser
	LinkOwner   string // address of the link key (public)
	Amount      string // decimal USDT, paid to the recipient
	FeeUnits    uint64 // 0 = the default of the chain
	Protected   bool
	Email       string
	Message     string
	SenderEmail string
	SenderLang  string
}

// Created is returned once, to the issuer.
type Created struct {
	Issuer
	SenderToken string `json:"sender_token"`
}

// Service keeps the cards and does the protected-card and relay work.
type Service struct {
	dir    string
	now    func() time.Time
	chains map[string]Chain // by asset
	cosign *cosigner.Signer // nil: protected cards unavailable
	price  pricer           // the price of each chain's coin (fee.go)
	// OnPaid is told (in its own goroutine) when a recipient was paid: the sender is notified.
	OnPaid func(Record)

	mu         sync.Mutex
	records    map[string]*Record
	authsDirty bool            // authorizations were pruned in memory and the file still has to be rewritten (retried at every tick)
	withdrawTo map[string]bool // the only destinations of an administrator's withdrawal (lower case)
	locks      map[string]*sync.Mutex
	auths      map[string]*Authorization // see auth.go
	codes      CodeService
	secret     []byte
}

// New opens (or creates) the state in dir. master is the co-signer master secret (at least 32 bytes; nil disables protected
// cards). chains lists the networks to serve; each needs a relayer.
func New(dir string, master []byte, chains []Chain) (*Service, error) {
	if !filepath.IsAbs(dir) {
		return nil, errors.New("invalid_usdtlink_state")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if st, err := os.Lstat(dir); err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("usdtlink_state_not_private")
	}
	s := &Service{dir: dir, now: time.Now, chains: map[string]Chain{}, records: map[string]*Record{}, locks: map[string]*sync.Mutex{}, auths: map[string]*Authorization{}}
	for _, c := range chains {
		s.chains[c.Network.Asset] = c
	}
	if len(master) > 0 {
		cs, err := cosigner.New(master)
		if err != nil {
			return nil, err
		}
		s.cosign = cs
	}
	raw, err := os.ReadFile(filepath.Join(dir, "cards.json"))
	if err == nil {
		var snapshot struct {
			Version int       `json:"version"`
			Records []*Record `json:"records"`
		}
		if json.Unmarshal(raw, &snapshot) != nil || snapshot.Version != 2 {
			return nil, errors.New("usdtlink_state_unknown_format") // 1 was never deployed; no migration exists, so refuse rather than guess
		}
		safes := map[string]bool{}
		for _, r := range snapshot.Records {
			if raw, err := hex.DecodeString(r.ID); err != nil || len(raw) != 16 || r.ID != hex.EncodeToString(raw) || safes[r.Safe] {
				return nil, errors.New("usdtlink_state_inconsistent") // an alias or a duplicate Safe: never merged silently
			}
			safes[r.Safe] = true
			if c, ok := s.chains[r.Asset]; ok && s.cosign != nil {
				// The cards must still belong to THIS master secret and to the Safe they were created for.
				id, _ := hex.DecodeString(r.ID)
				co, err := s.cosign.Address(c.Network.ChainID, id)
				card, cerr := r.card()
				addr, aerr := card.Address(c.Network.ChainID)
				if err != nil || cerr != nil || aerr != nil || addr.Hex() != r.Safe || !containsOwner(r.Owners, co.Hex()) {
					return nil, errors.New("usdtlink_master_secret_mismatch")
				}
			}
			s.records[r.ID] = r
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := s.loadAuths(); err != nil {
		return nil, err
	}
	return s, nil
}

// Assets lists the networks that are served.
func (s *Service) Assets() []string {
	var out []string
	for _, a := range evm.Assets() {
		if _, ok := s.chains[a]; ok {
			out = append(out, a)
		}
	}
	return out
}

// ProtectedReady reports whether protected cards can be created (a co-signer master secret is present).
func (s *Service) ProtectedReady() bool { return s.cosign != nil }

func (s *Service) persistLocked() error {
	snapshot := struct {
		Version int       `json:"version"`
		Records []*Record `json:"records"`
	}{Version: 2}
	for _, r := range s.records {
		snapshot.Records = append(snapshot.Records, r)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.dir, ".cards-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(encoded); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(temp.Name(), filepath.Join(s.dir, "cards.json")); err != nil {
		return err
	}
	d, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync() // the rename itself must be durable: a journal that may vanish on power loss is no journal
}

func (s *Service) lock(id string) func() {
	s.mu.Lock()
	l, ok := s.locks[id]
	if !ok {
		l = &sync.Mutex{}
		s.locks[id] = l
	}
	s.mu.Unlock()
	l.Lock()
	return l.Unlock
}

// Has reports whether this id is a USDT cashlink (so the HTTP layer can route it).
func (s *Service) Has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.records[id]
	return ok
}

// clone is a deep copy: the in-memory record and any working copy never share Operation, Intent or slices, so a failed write
// can always be rolled back completely.
func (r *Record) clone() *Record {
	c := *r
	c.Owners = append([]string(nil), r.Owners...)
	if r.Op != nil {
		op := *r.Op
		c.Op = &op
	}
	if r.Refund != nil {
		in := *r.Refund
		c.Refund = &in
	}
	if r.Drip != nil {
		d := *r.Drip
		c.Drip = &d
	}
	c.Late = nil
	for _, l := range r.Late {
		op := *l
		c.Late = append(c.Late, &op)
	}
	return &c
}

func (s *Service) get(id string) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return r.clone(), nil
}

func (s *Service) update(r *Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.records[r.ID]
	if !ok {
		return ErrNotFound
	}
	prev := old.clone()
	*old = *r.clone()
	if err := s.persistLocked(); err != nil {
		*old = *prev
		return err
	}
	return nil
}

func validEmail(e string) bool {
	if len(e) < 3 || len(e) > 254 || strings.ContainsAny(e, "\r\n\t ,;<>") {
		return false
	}
	a, err := mail.ParseAddress(e)
	return err == nil && a.Address == e
}

func normalizeLang(l string) string {
	switch l {
	case "fr", "es", "de", "zh":
		return l
	}
	return "en"
}

// card rebuilds the safe.Card of a record.
func (r *Record) card() (safe.Card, error) {
	id, err := hex.DecodeString(r.ID)
	if err != nil || len(id) != 16 {
		return safe.Card{}, ErrInvalid
	}
	var owners []cl.Address
	for _, o := range r.Owners {
		a, err := cl.ParseAddress(o)
		if err != nil {
			return safe.Card{}, ErrInvalid
		}
		owners = append(owners, a)
	}
	return safe.Card{ID: id, Owners: owners, Threshold: uint64(len(owners))}, nil
}

func (s *Service) state(r *Record) string {
	op := r.Op
	switch {
	case r.ClaimTx != "":
		return StatePaid
	case op != nil && op.Done && !op.Failed && op.Kind == "refund":
		return StateWithdrawn
	case r.SettledOther:
		return StateSettledOther
	case op != nil && !op.Done && !op.Failed:
		return StateVerifying
	case r.Cancelled:
		return StateCancelled
	case s.now().After(r.Created.Add(Lifetime)):
		return StateExpired
	case r.Funded:
		return StateReady
	}
	return StateAwaitingFunds
}

func (s *Service) public(r *Record) Public {
	c := s.chains[r.Asset]
	st := s.state(r)
	p := Public{ID: r.ID, Asset: r.Asset, Chain: c.Network.Chain, ChainID: c.Network.ChainID, Amount: r.Amount, Units: r.Units, FeeUnits: r.FeeUnits, RefundFee: s.refundFee(r),
		Symbol: evm.Symbol(r.Asset), Protected: r.Protected, Verified: r.Verified, State: st, Owners: r.Owners, Threshold: uint64(len(r.Owners)), Safe: r.Safe,
		Token: c.Network.Token, ExpiresAt: r.Created.Add(Lifetime), Message: r.Message, Explorer: c.Network.Explorer}
	if c.Relayer != nil {
		p.Relayer = c.Relayer.Address().Hex()
	}
	if r.ClaimTx != "" {
		p.TxHash, p.TxURL, p.TxKind = r.ClaimTx, c.Network.Explorer+r.ClaimTx, "claim"
	} else if r.Op != nil && r.Op.Tx != "" && !r.Op.Failed {
		p.TxHash, p.TxURL, p.TxKind = r.Op.Tx, c.Network.Explorer+r.Op.Tx, r.Op.Kind
	}
	return p
}

func (s *Service) issuer(r *Record) Issuer {
	return Issuer{Public: s.public(r), Recipient: r.Recipient}
}

// Create registers a card. The sender's browser has chosen the id and holds the link key; for a protected card this
// service adds its own co-signer to the owners and returns the resulting card address, which the browser compares with
// its own computation before showing the payment QR.
func (s *Service) Create(in Input) (Created, error) {
	n, err := evm.Lookup(in.Asset)
	c, served := s.chains[in.Asset]
	if err != nil || !served {
		return Created{}, evm.ErrUnknownAsset
	}
	units, canonical, err := evm.ParseAmount(in.Amount)
	if err != nil {
		return Created{}, err
	}
	qctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	fee, refundFee, qerr := s.quote(qctx, in.Asset, c, units)
	cancel()
	if qerr != nil {
		return Created{}, qerr
	}
	if in.FeeUnits != 0 { // a page that names its own fee may not go below what the service asks
		if in.FeeUnits < fee {
			return Created{}, errors.New("invalid_fee")
		}
		fee, refundFee = in.FeeUnits, in.FeeUnits
	}
	if fee < c.MinFee || fee > MaxFeeFor(units) { // a typo must not burn money
		return Created{}, errors.New("invalid_fee")
	}
	id, err := hex.DecodeString(in.CardID)
	// One identity per card: the id must be written exactly as hex.EncodeToString writes it (lower case). Two spellings of
	// the same 16 bytes would address the SAME Safe under two different records (two different emails).
	if err != nil || len(id) != 16 || in.CardID != hex.EncodeToString(id) {
		return Created{}, ErrInvalid
	}
	link, err := cl.ParseAddress(in.LinkOwner)
	if err != nil {
		return Created{}, ErrInvalid
	}
	if len([]rune(in.Message)) > 140 || strings.ContainsAny(in.Message, "\r\n") {
		return Created{}, errors.New("invalid_message")
	}
	// Every card is protected: no open (link-only) card exists. The recipient must prove his email, and the card pays out
	// only with this service's co-signature.
	if !in.Protected {
		return Created{}, errors.New("protection_required")
	}
	if !validEmail(in.Email) {
		return Created{}, errors.New("invalid_email")
	}
	if s.cosign == nil {
		return Created{}, ErrUnavailable
	}
	if in.SenderEmail != "" && !validEmail(in.SenderEmail) {
		return Created{}, errors.New("invalid_email")
	}
	co, err := s.cosign.Address(n.ChainID, id)
	if err != nil {
		return Created{}, err
	}
	owners := []cl.Address{link, co}
	card := safe.Card{ID: id, Owners: owners, Threshold: uint64(len(owners))}
	addr, err := card.Address(n.ChainID)
	if err != nil {
		return Created{}, ErrInvalid
	}
	var sorted []string
	for _, o := range sortOwners(owners) {
		sorted = append(sorted, o.Hex())
	}
	if low, err := s.FloatLow(context.Background(), in.Asset); err != nil {
		return Created{}, evm.ErrRPCUnavailable
	} else if low {
		return Created{}, ErrFloatLow // never take a card the relayer could not pay to claim or refund
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return Created{}, err
	}
	sum := sha256.Sum256(token)
	rec := &Record{ID: in.CardID, Asset: n.Asset, Units: units, FeeUnits: fee, RefundFeeUnits: refundFee, Amount: canonical, Message: in.Message, Protected: in.Protected, Email: in.Email,
		SenderEmail: in.SenderEmail, SenderLang: normalizeLang(in.SenderLang), LinkOwner: link.Hex(), Owners: sorted, Safe: addr.Hex(), Created: s.now().UTC(), SenderHash: hex.EncodeToString(sum[:])}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[rec.ID]; exists {
		return Created{}, ErrDuplicate
	}
	for _, other := range s.records { // and one record per Safe, whatever the id looks like
		if other.Safe == rec.Safe {
			return Created{}, ErrDuplicate
		}
	}
	if len(s.records) >= MaxCards {
		return Created{}, ErrCapacity
	}
	s.records[rec.ID] = rec
	if err := s.persistLocked(); err != nil {
		delete(s.records, rec.ID)
		return Created{}, err
	}
	return Created{Issuer: s.issuer(rec), SenderToken: base64.RawURLEncoding.EncodeToString(token)}, nil
}

func containsOwner(owners []string, a string) bool {
	for _, o := range owners {
		if strings.EqualFold(o, a) {
			return true
		}
	}
	return false
}

func sortOwners(in []cl.Address) []cl.Address {
	out := append([]cl.Address(nil), in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Less(out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Recipient is the protected card's email, for the code mail. Only the service itself uses it.
func (s *Service) Recipient(id string) (string, error) {
	r, err := s.get(id)
	if err != nil {
		return "", err
	}
	if !r.Protected || r.Email == "" {
		return "", ErrForbidden
	}
	switch s.state(r) {
	case StateExpired:
		return "", ErrExpired
	case StateCancelled:
		return "", ErrCancelled
	case StatePaid:
		return "", ErrPaid
	}
	return r.Email, nil
}

// balance reads the card's USDT balance (units).
func (s *Service) balance(ctx context.Context, r *Record) (*big.Int, error) {
	c := s.chains[r.Asset]
	token, err := cl.ParseAddress(c.Network.Token)
	if err != nil {
		return nil, err
	}
	safeAddr, err := cl.ParseAddress(r.Safe)
	if err != nil {
		return nil, err
	}
	data := append([]byte{0x70, 0xa0, 0x82, 0x31}, safeAddr.Word()...) // balanceOf(address)
	out, err := c.Node.Call(ctx, cl.Address{}, token, data)
	if err != nil || len(out) < 32 {
		return nil, evm.ErrRPCUnavailable
	}
	return new(big.Int).SetBytes(out[:32]), nil
}

// inFlight reports an operation that was sent (or is being sent) and whose outcome is unknown. It is read from the operation
// itself, never from the displayed state: a card that was Paid can still have a surplus refund in flight.
func inFlight(r *Record) bool { return r.Op != nil && !r.Op.Done && !r.Op.Failed }

// Refresh reads the chain and updates the card's state.
func (s *Service) Refresh(ctx context.Context, id string) (Public, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return Public{}, err
	}
	if err := s.refresh(ctx, r); err != nil {
		return Public{}, err
	}
	return s.public(r), nil
}

// nonce reads the Safe's own nonce: how many payouts the card has executed. Not deployed yet means zero.
func (s *Service) nonce(ctx context.Context, r *Record) (uint64, error) {
	c := s.chains[r.Asset]
	addr, err := cl.ParseAddress(r.Safe)
	if err != nil {
		return 0, err
	}
	out, err := c.Node.Call(ctx, cl.Address{}, addr, safe.NonceCall)
	if err != nil {
		return 0, evm.ErrRPCUnavailable
	}
	if len(out) < 32 {
		return 0, nil // no code at the address yet
	}
	return new(big.Int).SetBytes(out[:32]).Uint64(), nil
}

// confirmedBalance is the card's balance at a block that is CONFIRMATIONS deep (when the node client can tell), so that a
// deposit that a reorganisation could still remove never makes a card claimable. Without that capability it is the latest balance.
func (s *Service) confirmedBalance(ctx context.Context, r *Record) (*big.Int, error) {
	c := s.chains[r.Asset]
	f, ok := c.Node.(chain.Finality)
	if !ok || c.Network.Confirmations == 0 {
		return s.balance(ctx, r)
	}
	bn, err := f.BlockNumber(ctx)
	if err != nil {
		return nil, evm.ErrRPCUnavailable
	}
	if bn < c.Network.Confirmations {
		return new(big.Int), nil
	}
	token, err := cl.ParseAddress(c.Network.Token)
	safeAddr, err2 := cl.ParseAddress(r.Safe)
	if err != nil || err2 != nil {
		return nil, ErrInvalid
	}
	out, err := f.CallAt(ctx, cl.Address{}, token, append([]byte{0x70, 0xa0, 0x82, 0x31}, safeAddr.Word()...), bn-c.Network.Confirmations)
	if err != nil || len(out) < 32 {
		return nil, evm.ErrRPCUnavailable
	}
	return new(big.Int).SetBytes(out[:32]), nil
}

// deepEnough decides whether a successful receipt may be believed: it must be CONFIRMATIONS blocks deep and it must show the
// transfers the payout was meant to make. done=false, anomaly=false means "not deep enough yet".
func (s *Service) deepEnough(ctx context.Context, r *Record, op *Operation, rc chain.Receipt) (done, anomaly bool, err error) {
	c := s.chains[r.Asset]
	if f, ok := c.Node.(chain.Finality); ok && c.Network.Confirmations > 0 {
		bn, err := f.BlockNumber(ctx)
		if err != nil {
			return false, false, evm.ErrRPCUnavailable
		}
		if bn+1 < rc.Block+c.Network.Confirmations {
			return false, false, nil
		}
	}
	token, e1 := cl.ParseAddress(c.Network.Token)
	card, e2 := cl.ParseAddress(r.Safe)
	to, e3 := cl.ParseAddress(op.To)
	if errors.Join(e1, e2, e3) != nil {
		return false, true, nil
	}
	expectedFee := r.FeeUnits
	if op.Kind == "refund" {
		expectedFee = s.refundFee(r)
	}
	if relayer.VerifyPayout(rc, token, card, to, c.Relayer.Address(), new(big.Int).SetUint64(op.Units), new(big.Int).SetUint64(expectedFee)) != nil {
		return false, true, nil
	}
	return true, false, nil
}

// blockTime dates a block with its own timestamp. A node that cannot say gives the zero time (the date stays unknown), a node that
// fails gives an error (the payout is settled on the next look, not now, so that its date is not lost).
func (s *Service) blockTime(ctx context.Context, node chain.Chain, block uint64) (time.Time, error) {
	bt, ok := node.(chain.BlockTimer)
	if !ok {
		return time.Time{}, nil
	}
	return bt.BlockTime(ctx, block)
}

// dropAfter is how long a transaction that never shows up on chain is given before the operation is released.
const dropAfter = 2 * time.Hour

// refresh settles the card from the chain, in this order of authority: the receipt of the journaled transaction, the Safe's
// nonce, then the balance. A balance alone never concludes anything except "funded now or not".
func (s *Service) refresh(ctx context.Context, r *Record) error {
	c := s.chains[r.Asset]
	n, err := s.nonce(ctx, r)
	if err != nil {
		return err
	}
	changed, paidNow := false, false
	settle := func(op *Operation, at time.Time) { // a payout of ours is confirmed on chain: deep enough, and the right transfers
		op.Done, changed = true, true
		r.Settled++
		if op.Kind == "claim" {
			r.ClaimTx, paidNow = op.Tx, true
			if !at.IsZero() {
				r.ClaimedAt = at // the date of the block that holds the payout, not of this look
			}
		}
	}
	dateOf := func(op *Operation, rc chain.Receipt) (time.Time, error) {
		if op.Kind != "claim" {
			return time.Time{}, nil // a refund is not a claim: it gets no claim date
		}
		return s.blockTime(ctx, c.Node, rc.Block)
	}
	if op := r.Op; inFlight(r) {
		switch {
		case op.Tx == "":
			op.Failed, changed = true, true // no payout transaction was ever signed: the process stopped before sending
		default:
			rc, err := c.Node.Receipt(ctx, op.Tx)
			if err != nil {
				return err
			}
			switch {
			case rc.Found && rc.Success:
				done, anomaly, err := s.deepEnough(ctx, r, op, rc)
				if err != nil {
					return err
				}
				if anomaly && r.Anomaly == "" {
					r.Anomaly, changed = "payout_transfers_not_found", true // succeeded on chain without the expected transfers: never "paid"
				}
				if done {
					at, err := dateOf(op, rc)
					if err != nil {
						return err
					}
					settle(op, at)
				}
			case rc.Found:
				op.Failed, changed = true, true // mined and reverted: nothing was paid
			case n > op.Nonce:
				op.Failed, r.SettledOther, changed = true, true, true // the nonce was used by a transaction that is not ours
			case s.now().After(op.Started.Add(dropAfter)):
				// Never mined and the nonce is still free. This is NOT proof that it will never mine: the operation is released
				// so that the same payout can be sent again (the Safe nonce makes a second execution harmless), but the old
				// hash stays watched in Late and is recognised if it mines after all.
				late := *op
				op.Failed, changed = true, true
				r.Late = append(r.Late, &late)
			}
		}
	}
	for i := 0; i < len(r.Late); {
		rc, err := c.Node.Receipt(ctx, r.Late[i].Tx)
		if err != nil {
			return err
		}
		if rc.Found && rc.Success {
			done, anomaly, err := s.deepEnough(ctx, r, r.Late[i], rc)
			if err != nil {
				return err
			}
			if anomaly && r.Anomaly == "" {
				r.Anomaly, changed = "payout_transfers_not_found", true
			}
			if !done {
				i++
				continue
			}
			at, err := dateOf(r.Late[i], rc)
			if err != nil {
				return err
			}
			settle(r.Late[i], at)
			r.Op = r.Late[i] // the late operation is the card's last one, whatever its kind
			r.Late = append(r.Late[:i], r.Late[i+1:]...)
			continue
		}
		if rc.Found { // reverted: nothing to watch
			r.Late = append(r.Late[:i], r.Late[i+1:]...)
			changed = true
			continue
		}
		i++
	}
	if r.ClaimTx != "" && r.ClaimedAt.IsZero() {
		// A card paid before its date was kept: the receipt of its confirmed payout says in which block it was mined. If the node cannot say,
		// the date stays unknown.
		if rc, err := c.Node.Receipt(ctx, r.ClaimTx); err == nil && rc.Found && rc.Success {
			if at, err := s.blockTime(ctx, c.Node, rc.Block); err == nil && !at.IsZero() {
				r.ClaimedAt, changed = at, true
			}
		}
	}
	if (r.Op == nil || r.Op.Done || r.Op.Failed) && n > r.Settled && !r.SettledOther {
		// The card executed a payout that we did not relay (a signature we issued, relayed elsewhere).
		r.SettledOther, changed = true, true
	}
	if changed { // the receipt's outcome is written BEFORE the optional balance read: a failing node must not lose it
		if err := s.update(r); err != nil {
			return err
		}
		changed = false
		if paidNow && s.OnPaid != nil {
			go s.OnPaid(*r.clone()) // the sender is told only when the payment is deep and verified
		}
	}
	bal, err := s.confirmedBalance(ctx, r)
	if err != nil {
		return err
	}
	funded := bal.Cmp(new(big.Int).SetUint64(r.Units+r.FeeUnits)) >= 0
	if funded != r.Funded {
		r.Funded, changed = funded, true
	}
	if held := heldUnits(bal); held != r.Held || r.HeldAt.IsZero() {
		r.Held, r.HeldAt, changed = held, s.now().UTC(), true
	}
	if changed {
		return s.update(r)
	}
	return nil
}

// Status is the issuer's view: the state read from the chain, the balance, and the recipient once known.
func (s *Service) Status(ctx context.Context, id string) (Issuer, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return Issuer{}, err
	}
	in := Issuer{}
	err = s.refresh(ctx, r)
	if bal, berr := s.balance(ctx, r); berr == nil && bal.IsUint64() {
		in.BalanceUnits = bal.Uint64()
	}
	in.Public, in.Recipient, in.Stale, in.Recovering, in.Anomaly = s.public(r), r.Recipient, err != nil, inFlight(r) && r.Op.Kind == "refund", r.Anomaly // an unreadable chain is said, not hidden
	if !r.ClaimedAt.IsZero() && r.ClaimTx != "" && r.Anomaly == "" {
		at := r.ClaimedAt.UTC()
		in.ClaimedAt = &at
	}
	return in, nil
}

// Cancel asks that the recipient no longer be co-signed (the sender's token proves who asks). It does not move money: the
// sender takes it back with Refund.
func (s *Service) Cancel(id, token string) (Public, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return Public{}, err
	}
	if !s.senderOK(r, token) {
		return Public{}, ErrForbidden
	}
	if inFlight(r) {
		return Public{}, ErrPending // a payout is in flight: its outcome comes first
	}
	if s.state(r) == StatePaid {
		return Public{}, ErrPaid
	}
	r.Cancelled = true
	if err := s.update(r); err != nil {
		return Public{}, err
	}
	return s.public(r), nil
}

func (s *Service) senderOK(r *Record, token string) bool {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(raw)
	want, _ := hex.DecodeString(r.SenderHash)
	return subtle.ConstantTimeCompare(sum[:], want) == 1
}

// badDestination refuses addresses that can never be a recipient: the zero address, the card itself, the token contract and
// the relayer (the relayer's address is where the FEE goes; paying it as recipient would be a confusion, not a claim).
func (s *Service) badDestination(r *Record, dest cl.Address) bool {
	c := s.chains[r.Asset]
	var zero cl.Address
	if dest == zero || strings.EqualFold(dest.Hex(), r.Safe) || strings.EqualFold(dest.Hex(), c.Network.Token) {
		return true
	}
	return c.Relayer != nil && dest == c.Relayer.Address()
}

// usable checks that a card may still be claimed by this destination.
func (s *Service) usable(r *Record, to cl.Address) error {
	if err := s.usableState(r); err != nil {
		return err
	}
	if r.Recipient != "" && !strings.EqualFold(r.Recipient, to.Hex()) {
		return ErrAddressSet // one destination per card, once chosen
	}
	return nil
}

// usableState is usable without a destination: the card itself can still be claimed (not paid, cancelled, expired, in flight).
func (s *Service) usableState(r *Record) error {
	if inFlight(r) {
		return ErrPending
	}
	switch s.state(r) {
	case StatePaid:
		return ErrPaid
	case StateCancelled, StateWithdrawn, StateSettledOther:
		return ErrCancelled
	case StateExpired:
		return ErrExpired
	}
	return nil
}

// run sends one operation through the relayer with the journal wired in: the operation is on disk before anything is
// signed for the chain, its transaction hashes are on disk before they are broadcast, and a journal write that fails stops
// the send. Afterwards the outcome is recorded, but nothing depends on that write: refresh settles from the chain.
func (s *Service) run(ctx context.Context, r *Record, op *Operation, req relayer.Request) (relayer.Result, error) {
	if inFlight(r) {
		return relayer.Result{}, ErrPending // never replace the tracking of an operation whose outcome is unknown
	}
	c := s.chains[r.Asset]
	r.Op = op
	if err := s.update(r); err != nil {
		return relayer.Result{}, err
	}
	req.Journal = func(kind, hash string) error {
		if kind == "deploy" {
			op.Deploy = hash
		} else {
			op.Tx = hash
		}
		return s.update(r)
	}
	res, err := c.Relayer.Submit(ctx, req)
	switch {
	case err == nil:
		// The first receipt is not the end: the operation stays "verifying" until refresh finds it deep enough and checks the
		// transfers it must have made.
	case op.Tx == "":
		op.Failed = true // refused before any payout transaction existed (a deployment alone moves no money)
	}
	// op.Tx != "" with an error: the transaction may exist. It stays in flight until the chain says what it became.
	_ = s.update(r)
	return res, err
}

// claim has the relayer pay the recipient (reached only through ClaimAuthorized). The browser sends only the destination and the owners' signatures; this service
// rebuilds the amount and the fee from the card's record, so a client can never change them, and the relayer checks the rest.
// Nothing is recorded until the request has been fully verified (Check): a rejected request leaves the card untouched.
func (s *Service) claim(ctx context.Context, id, to string, signatures []byte) (relayer.Result, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return relayer.Result{}, err
	}
	dest, err := cl.ParseAddress(to)
	if err != nil || s.badDestination(r, dest) {
		return relayer.Result{}, evm.ErrInvalidAddress
	}
	if !r.Verified {
		return relayer.Result{}, ErrNotVerified
	}
	if err := s.refresh(ctx, r); err != nil {
		return relayer.Result{}, err
	}
	if err := s.usable(r, dest); err != nil {
		return relayer.Result{}, err
	}
	c := s.chains[r.Asset]
	card, err := r.card()
	if err != nil {
		return relayer.Result{}, err
	}
	req := relayer.Request{Card: card, Recipient: dest, Amount: new(big.Int).SetUint64(r.Units), Fee: new(big.Int).SetUint64(r.FeeUnits), Signatures: signatures}
	if _, _, _, err := c.Relayer.Check(ctx, req); err != nil { // signatures, funding, fee: nothing is sent, nothing is recorded
		return relayer.Result{}, err
	}
	r.Recipient = dest.Hex() // from here this card can only ever pay this address
	return s.run(ctx, r, &Operation{Kind: "claim", To: dest.Hex(), Units: r.Units, Nonce: 0, Started: s.now().UTC()}, req)
}

// Refund co-signs a refund of a card to the sender's own address (the sender's token proves who asks). A refund is a full
// sweep (balance minus the fee) made for the Safe's CURRENT nonce, so it also recovers a surplus or a late deposit after the
// card was paid. The destination is fixed by the first intent until that refund is settled. Asking for it stops any NEW
// co-signature for the recipient; a claim signature already handed out stays valid on chain until the nonce is used:
// whichever transaction is mined first wins, and the loser's signatures become void.
func (s *Service) Refund(ctx context.Context, id, senderToken, to string) (cl.Address, []byte, uint64, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return cl.Address{}, nil, 0, err
	}
	if !s.senderOK(r, senderToken) {
		return cl.Address{}, nil, 0, ErrForbidden
	}
	if s.cosign == nil {
		return cl.Address{}, nil, 0, ErrForbidden
	}
	dest, err := cl.ParseAddress(to)
	if err != nil || s.badDestination(r, dest) {
		return cl.Address{}, nil, 0, evm.ErrInvalidAddress
	}
	if err := s.refresh(ctx, r); err != nil {
		return cl.Address{}, nil, 0, err
	}
	if inFlight(r) {
		return cl.Address{}, nil, 0, ErrPending
	}
	n, err := s.nonce(ctx, r)
	if err != nil {
		return cl.Address{}, nil, 0, err
	}
	bal, err := s.balance(ctx, r)
	if err != nil {
		return cl.Address{}, nil, 0, err
	}
	fee := new(big.Int).SetUint64(s.refundFee(r)) // a refund pays the relayer the gas, not the service fee
	intent := r.Refund
	if intent == nil || intent.Nonce != n { // a new intent: a full sweep of what the card holds now
		amount := new(big.Int).Sub(bal, fee)
		if amount.Sign() <= 0 || amount.Cmp(new(big.Int).SetUint64(evm.MaxUnits)) > 0 {
			return cl.Address{}, nil, 0, ErrNotFunded // nothing to take back, or above the relayer's ceiling
		}
		intent = &Intent{To: dest.Hex(), Units: amount.Uint64(), Nonce: n}
	} else if !strings.EqualFold(intent.To, dest.Hex()) {
		return cl.Address{}, nil, 0, ErrAddressSet // the first refund address stays: one intent, no concurrent ones
	}
	if bal.Cmp(new(big.Int).Add(new(big.Int).SetUint64(intent.Units), fee)) < 0 {
		return cl.Address{}, nil, 0, ErrNotFunded
	}
	c := s.chains[r.Asset]
	card, err := r.card()
	if err != nil {
		return cl.Address{}, nil, 0, err
	}
	token, _ := cl.ParseAddress(c.Network.Token)
	signed, err := s.cosign.SignPayout(c.Network.ChainID, card, token, dest, c.Relayer.Address(), new(big.Int).SetUint64(intent.Units), fee, n)
	if err != nil {
		return cl.Address{}, nil, 0, err
	}
	r.Refund = intent
	if n == 0 {
		r.Cancelled = true // the card was never paid: no new co-signature for the recipient
	}
	if err := s.update(r); err != nil { // the intent is durable BEFORE the signature leaves
		return cl.Address{}, nil, 0, err
	}
	return signed.Owner, signed.Sig.Bytes(), intent.Units, nil
}

// RefundClaim relays the refund that Refund prepared, and only that one (same address, same amount, same nonce).
func (s *Service) RefundClaim(ctx context.Context, id, senderToken string, signatures []byte) (relayer.Result, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return relayer.Result{}, err
	}
	if !s.senderOK(r, senderToken) {
		return relayer.Result{}, ErrForbidden
	}
	if r.Refund == nil {
		return relayer.Result{}, ErrForbidden // no refund was prepared
	}
	if err := s.refresh(ctx, r); err != nil {
		return relayer.Result{}, err
	}
	if inFlight(r) {
		return relayer.Result{}, ErrPending
	}
	n, err := s.nonce(ctx, r)
	if err != nil {
		return relayer.Result{}, err
	}
	if n != r.Refund.Nonce {
		return relayer.Result{}, ErrInvalid // the card moved on since this refund was prepared: prepare a new one
	}
	dest, err := cl.ParseAddress(r.Refund.To)
	if err != nil {
		return relayer.Result{}, ErrInvalid
	}
	card, err := r.card()
	if err != nil {
		return relayer.Result{}, err
	}
	req := relayer.Request{Card: card, Recipient: dest, Amount: new(big.Int).SetUint64(r.Refund.Units), Fee: new(big.Int).SetUint64(s.refundFee(r)), Signatures: signatures}
	res, err := s.run(ctx, r, &Operation{Kind: "refund", To: dest.Hex(), Units: r.Refund.Units, Nonce: n, Started: s.now().UTC()}, req)
	if err == nil {
		r.Refund = nil
		_ = s.update(r)
	}
	return res, err
}

// RefundNonce is the Safe nonce the prepared refund of a card was made for (the browser signs the same batch with it).
// RefundFeeUnits is what a refund of this card pays the relayer (the gas fee, never the service fee).
func (s *Service) RefundFeeUnits(id string) (uint64, bool) {
	r, err := s.get(id)
	if err != nil {
		return 0, false
	}
	return s.refundFee(r), true
}

func (s *Service) RefundNonce(id string) (uint64, bool) {
	r, err := s.get(id)
	if err != nil || r.Refund == nil {
		return 0, false
	}
	return r.Refund.Nonce, true
}

// Run follows, until ctx ends, the cards whose outcome nobody may be watching: payouts in flight (they become "paid" only when deep
// and verified), released operations, and cards still waiting for their funding to become deep enough.
func (s *Service) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.pruneAuths()
			s.mu.Lock()
			var ids []string
			for id, r := range s.records {
				if inFlight(r) || len(r.Late) > 0 || (s.state(r) == StateAwaitingFunds && s.now().Before(r.Created.Add(Lifetime))) {
					ids = append(ids, id)
				}
			}
			s.mu.Unlock()
			for _, id := range ids {
				c, cancel := context.WithTimeout(ctx, 40*time.Second)
				_, _ = s.Refresh(c, id)
				cancel()
			}
		}
	}
}

// InFlight reports whether a payout of this card is still being verified.
func (s *Service) InFlight(id string) bool {
	r, err := s.get(id)
	return err == nil && inFlight(r)
}

// FloatLow reports whether the relayer of this asset's chain holds less native coin than MinFloat (when one is configured).
func (s *Service) FloatLow(ctx context.Context, asset string) (bool, error) {
	c, ok := s.chains[asset]
	if !ok || c.MinFloat == nil || c.Relayer == nil {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	have, err := c.Node.NativeBalance(ctx, c.Relayer.Address())
	if err != nil {
		return false, err
	}
	return have.Cmp(s.minFloat(ctx, c)) < 0, nil
}

// FloatStatus lists, per served asset, whether the relayer's float is below its minimum (for the health route and the operator).
func (s *Service) FloatStatus(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	for asset := range s.chains {
		low, err := s.FloatLow(ctx, asset)
		out[asset] = low || err != nil
	}
	return out
}

// FloatCards is how many cards' gas the relayer must be able to pay for (claim or refund) before it takes a NEW one.
var FloatCards int64 = 10

// minFloat is the native coin the relayer must hold to take a new card. With FloatMin set it follows the price of gas: FloatCards times the
// gas one card uses at the node's current fee estimate, between FloatMin and MinFloat. If the node cannot say, MinFloat (the prudent
// value) is used.
func (s *Service) minFloat(ctx context.Context, c Chain) *big.Int {
	if c.FloatMin == nil || c.MinFloat == nil || c.Node == nil {
		return c.MinFloat
	}
	_, maxFee, err := c.Node.Fees(ctx)
	if err != nil || maxFee == nil || maxFee.Sign() <= 0 {
		return c.MinFloat
	}
	need := new(big.Int).Mul(maxFee, new(big.Int).SetUint64(CardGas))
	need.Mul(need, big.NewInt(FloatCards))
	if need.Cmp(c.FloatMin) < 0 {
		need = new(big.Int).Set(c.FloatMin)
	}
	if need.Cmp(c.MinFloat) > 0 {
		need = new(big.Int).Set(c.MinFloat)
	}
	return need
}
