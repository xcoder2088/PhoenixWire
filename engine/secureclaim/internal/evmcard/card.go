// Package evmcard is the USDT (Ethereum / Polygon) card. It holds no money and no key.
//
// The sender creates a card (amount, optional message, optional recipient email for a protected card) and puts it in an
// email. The recipient opens the claim link, proves his email with a code when the card is protected, and gives HIS OWN
// address. The sender is told, and makes ONE ordinary USDT transfer from his wallet straight to that address (his wallet
// adds the network fee, exactly as for any transfer). The service only watches the chain to see that transfer and say
// "paid" on both sides. Nothing is custodied, nobody advances any fee.
package evmcard

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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
)

var (
	ErrNotFound    = errors.New("claim_not_found")
	ErrExpired     = errors.New("claim_expired")
	ErrNotVerified = errors.New("email_not_verified")
	ErrForbidden   = errors.New("forbidden")
	ErrCapacity    = errors.New("capacity_reached")
	ErrAddressSet  = errors.New("address_already_given")
	ErrCancelled   = errors.New("card_cancelled")
	ErrSameAddress = errors.New("same_address")
)

const (
	CardLifetime = 7 * 24 * time.Hour // to give an address
	PayWindow    = 7 * 24 * time.Hour // to pay once the address is known
	MaxCards     = 20000
	scanChunk    = 2999
)

// States of a card.
const (
	StateAwaiting  = "awaiting_address"
	StateReady     = "ready"
	StatePaid      = "paid"
	StateCancelled = "cancelled"
	StateExpired   = "expired"
)

type Record struct {
	ID            string         `json:"id"`
	Asset         string         `json:"asset"`
	Units         uint64         `json:"units"`
	Amount        string         `json:"amount"`
	Message       string         `json:"message,omitempty"`
	Protected     bool           `json:"protected"`
	Email         string         `json:"email,omitempty"` // recipient, protected cards only
	SenderEmail   string         `json:"sender_email,omitempty"`
	SenderLang    string         `json:"sender_lang,omitempty"`    // language of the notice sent to the sender
	SenderAddress string         `json:"sender_address,omitempty"` // when given, only a transfer from it counts
	Created       time.Time      `json:"created"`
	Verified      bool           `json:"verified"`
	SenderHash    string         `json:"sender_hash"`
	State         string         `json:"state"`
	Recipient     string         `json:"recipient,omitempty"`
	ReadyAt       time.Time      `json:"ready_at,omitempty"`
	FromBlock     uint64         `json:"from_block,omitempty"`
	Scanned       uint64         `json:"scanned,omitempty"`
	Doubt         map[string]int `json:"doubt,omitempty"` // candidates the second node did not confirm yet (a lagging node), by hash
	PaidTx        string         `json:"paid_tx,omitempty"`
	PaidBlock     uint64         `json:"paid_block,omitempty"`
	PaidAt        time.Time      `json:"paid_at,omitempty"`
}

// Public is what anyone holding the claim link may see. No email, no address.
type Public struct {
	ID           string    `json:"id"`
	Asset        string    `json:"asset"`
	Chain        string    `json:"chain"`
	Amount       string    `json:"amount"`
	Symbol       string    `json:"symbol"`
	Palette      int       `json:"palette"`
	Protected    bool      `json:"protected"`
	Verified     bool      `json:"email_verified"`
	State        string    `json:"state"`
	HasRecipient bool      `json:"has_recipient"`
	TxHash       string    `json:"tx_hash,omitempty"`
	TxURL        string    `json:"tx_url,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Message      string    `json:"message,omitempty"`
}

// Issuer is what the sender's PhoenixMail may see: the recipient's address once it is known.
type Issuer struct {
	Public
	Units         uint64 `json:"units"`
	Token         string `json:"token"`
	ChainID       uint64 `json:"chain_id"`
	Recipient     string `json:"recipient,omitempty"`
	SenderAddress string `json:"sender_address,omitempty"`
	Confirmations uint64 `json:"confirmations"`
}

type Service struct {
	dir    string
	now    func() time.Time
	client func(evm.Network) *evm.Client
	// OnReady is told (in its own goroutine) when a recipient has given his address: PhoenixMail's sender is notified.
	OnReady func(Record)

	mu      sync.Mutex
	records map[string]*Record
	locks   map[string]*sync.Mutex
	clients map[string]*evm.Client
}

func New(dir string) (*Service, error) {
	if !filepath.IsAbs(dir) {
		return nil, errors.New("invalid_evm_state")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if st, err := os.Lstat(dir); err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("evm_state_not_private")
	}
	s := &Service{dir: dir, now: time.Now, records: map[string]*Record{}, locks: map[string]*sync.Mutex{}, clients: map[string]*evm.Client{}}
	s.client = func(n evm.Network) *evm.Client {
		s.mu.Lock()
		defer s.mu.Unlock()
		if c, ok := s.clients[n.Asset]; ok {
			return c
		}
		c := evm.NewClient(n)
		s.clients[n.Asset] = c
		return c
	}
	raw, err := os.ReadFile(filepath.Join(dir, "cards.json"))
	if err == nil {
		var snapshot struct {
			Version int       `json:"version"`
			Records []*Record `json:"records"`
		}
		if json.Unmarshal(raw, &snapshot) != nil || snapshot.Version != 3 {
			return nil, errors.New("evm_state_corrupt")
		}
		for _, r := range snapshot.Records {
			s.records[r.ID] = r
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

// UseClients replaces how RPC clients are built. Tests point them at a fake node; production never calls it.
func (s *Service) UseClients(f func(evm.Network) *evm.Client) { s.client = f }

func (s *Service) persistLocked() error {
	snapshot := struct {
		Version int       `json:"version"`
		Records []*Record `json:"records"`
	}{Version: 3}
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
	if d, err := os.Open(s.dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
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

// Has reports whether this ID is a USDT card (so the HTTP layer can route it).
func (s *Service) Has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.records[id]
	return ok
}

func (s *Service) get(id string) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := *r
	return &c, nil
}

func (s *Service) update(r *Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.records[r.ID]
	if !ok {
		return ErrNotFound
	}
	prev := *old
	*old = *r
	if err := s.persistLocked(); err != nil {
		*old = prev
		return err
	}
	return nil
}

// deadline is when the card stops being usable: a week to give an address, then a week to pay.
func (r *Record) deadline() time.Time {
	if r.State == StateReady && !r.ReadyAt.IsZero() {
		return r.ReadyAt.Add(PayWindow)
	}
	return r.Created.Add(CardLifetime)
}

func (s *Service) state(r *Record) string {
	if r.State != StatePaid && r.State != StateCancelled && !s.now().Before(r.deadline()) {
		return StateExpired
	}
	return r.State
}

func (s *Service) view(r *Record) Public {
	n, _ := evm.Lookup(r.Asset)
	pal, _ := strconv.ParseUint(r.ID[:2], 16, 8)
	p := Public{ID: r.ID, Asset: r.Asset, Chain: n.Chain, Amount: r.Amount, Symbol: evm.Symbol(r.Asset), Palette: int(pal) % 12, Protected: r.Protected, Verified: r.Verified,
		State: s.state(r), HasRecipient: r.Recipient != "", ExpiresAt: r.deadline(), Message: r.Message}
	if r.PaidTx != "" {
		p.TxHash, p.TxURL = r.PaidTx, n.Explorer+r.PaidTx
	}
	return p
}

func (s *Service) issuerView(r *Record) Issuer {
	n, _ := evm.Lookup(r.Asset)
	return Issuer{Public: s.view(r), Units: r.Units, Token: n.Token, ChainID: n.ChainID, Recipient: r.Recipient, SenderAddress: r.SenderAddress, Confirmations: n.Confirmations}
}

// Public returns the non-secret view of a card.
func (s *Service) Public(id string) (Public, error) {
	r, err := s.get(id)
	if err != nil {
		return Public{}, err
	}
	return s.view(r), nil
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
	if s.state(r) == StateExpired {
		return "", ErrExpired
	}
	return r.Email, nil
}

type Input struct {
	Asset         string
	Amount        string
	Email         string // recipient, protected cards only
	Protected     bool
	Message       string
	SenderEmail   string // told when the recipient is ready (optional)
	SenderLang    string // en fr es de zh
	SenderAddress string // the wallet the sender will pay from (optional; makes the check strict)
}

// Created is what the issuer gets back once.
type Created struct {
	Issuer
	SenderToken string `json:"sender_token"`
}

func normalizeLang(l string) string {
	switch l {
	case "fr", "es", "de", "zh":
		return l
	}
	return "en"
}

func validEmail(e string) bool {
	if len(e) < 3 || len(e) > 254 || strings.ContainsAny(e, "\r\n\t ,;<>") {
		return false
	}
	a, err := mail.ParseAddress(e)
	return err == nil && a.Address == e
}

func (s *Service) Create(in Input) (Created, error) {
	n, err := evm.Lookup(in.Asset)
	if err != nil {
		return Created{}, err
	}
	units, canonical, err := evm.ParseAmount(in.Amount)
	if err != nil {
		return Created{}, err
	}
	if len([]rune(in.Message)) > 140 || strings.ContainsAny(in.Message, "\r\n") {
		return Created{}, errors.New("invalid_message")
	}
	if in.Protected && !validEmail(in.Email) {
		return Created{}, errors.New("invalid_email")
	}
	if !in.Protected {
		in.Email = ""
	}
	if in.SenderEmail != "" && !validEmail(in.SenderEmail) {
		return Created{}, errors.New("invalid_email")
	}
	if in.SenderAddress != "" {
		if in.SenderAddress, err = evm.NormalizeAddress(in.SenderAddress); err != nil {
			return Created{}, err
		}
	}
	s.mu.Lock()
	full := len(s.records) >= MaxCards
	s.mu.Unlock()
	if full {
		return Created{}, ErrCapacity
	}
	idBytes := make([]byte, 24)
	token := make([]byte, 32)
	if _, err := rand.Read(idBytes); err != nil {
		return Created{}, err
	}
	if _, err := rand.Read(token); err != nil {
		return Created{}, err
	}
	sum := sha256.Sum256(token)
	rec := &Record{ID: hex.EncodeToString(idBytes), Asset: n.Asset, Units: units, Amount: canonical, Message: in.Message, Protected: in.Protected, Email: in.Email,
		SenderEmail: in.SenderEmail, SenderLang: normalizeLang(in.SenderLang), SenderAddress: in.SenderAddress, Created: s.now().UTC(), SenderHash: hex.EncodeToString(sum[:]), State: StateAwaiting}
	s.mu.Lock()
	s.records[rec.ID] = rec
	if err := s.persistLocked(); err != nil {
		delete(s.records, rec.ID)
		s.mu.Unlock()
		return Created{}, err
	}
	s.mu.Unlock()
	return Created{Issuer: s.issuerView(rec), SenderToken: base64.RawURLEncoding.EncodeToString(token)}, nil
}

// MarkVerified records that the recipient proved the email code.
func (s *Service) MarkVerified(id string) error {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return err
	}
	if !r.Protected {
		return ErrForbidden
	}
	if s.state(r) == StateExpired {
		return ErrExpired
	}
	if r.Verified {
		return nil
	}
	r.Verified = true
	return s.update(r)
}

// SetRecipient records the address the recipient chose. It is final: the sender's QR will be built from it.
func (s *Service) SetRecipient(ctx context.Context, id, address string) (Public, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return Public{}, err
	}
	address, err = evm.NormalizeAddress(address)
	if err != nil {
		return Public{}, err
	}
	n, _ := evm.Lookup(r.Asset)
	switch s.state(r) {
	case StateExpired:
		return Public{}, ErrExpired
	case StateCancelled:
		return Public{}, ErrCancelled
	case StateReady, StatePaid:
		return Public{}, ErrAddressSet
	}
	if r.Protected && !r.Verified {
		return Public{}, ErrNotVerified
	}
	if address == r.SenderAddress || strings.EqualFold(address, n.Token) {
		return Public{}, ErrSameAddress
	}
	head, err := s.client(n).BlockNumber(ctx)
	if err != nil {
		return Public{}, err
	}
	r.State, r.Recipient, r.ReadyAt, r.FromBlock, r.Scanned = StateReady, address, s.now().UTC(), head, 0
	if err := s.update(r); err != nil {
		return Public{}, err
	}
	if s.OnReady != nil {
		go s.OnReady(*r)
	}
	return s.view(r), nil
}

// detect looks for the sender's transfer in the blocks that are old enough to count.
func (s *Service) detect(ctx context.Context, r *Record) error {
	if r.State != StateReady || s.state(r) == StateExpired {
		return nil
	}
	n, _ := evm.Lookup(r.Asset)
	c := s.client(n)
	head, err := c.BlockNumber(ctx)
	if err != nil || head <= n.Confirmations {
		return err
	}
	settled := head - n.Confirmations
	from := r.Scanned + 1
	if from < r.FromBlock {
		from = r.FromBlock
	}
	need := new(big.Int).SetUint64(r.Units)
	changed := false
	for from <= settled {
		to := from + scanChunk
		if to > settled {
			to = settled
		}
		transfers, err := c.Transfers(ctx, from, to, r.Recipient, r.SenderAddress)
		if err != nil {
			break
		}
		for _, t := range transfers {
			if t.Value.Cmp(need) < 0 {
				continue
			}
			// One node reported it; two must agree on the transaction itself before it counts.
			confirmed, ok, err := c.Confirm(ctx, t.Hash, r.Recipient, need)
			if err != nil {
				return err // look again later, do not skip it
			}
			if ok {
				r.State, r.PaidTx, r.PaidBlock, r.PaidAt = StatePaid, confirmed.Hash, confirmed.Block, s.now().UTC()
				return s.update(r)
			}
			// Not confirmed: a node may simply lag. Keep looking at this range a few times before giving up on the candidate.
			if r.Doubt == nil {
				r.Doubt = map[string]int{}
			}
			if r.Doubt[t.Hash]++; r.Doubt[t.Hash] < 6 {
				_ = s.update(r)
				return errors.New("unconfirmed")
			}
		}
		r.Scanned, changed = to, true
		from = to + 1
	}
	if changed {
		return s.update(r)
	}
	return err
}

// Status is the issuer's view (the recipient's address once it is known); it looks at the chain first.
func (s *Service) Status(ctx context.Context, id string) (Issuer, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return Issuer{}, err
	}
	_ = s.detect(ctx, r)
	if r, err = s.get(id); err != nil {
		return Issuer{}, err
	}
	return s.issuerView(r), nil
}

// Refresh is the public (claim page) view, after a look at the chain.
func (s *Service) Refresh(ctx context.Context, id string) (Public, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return Public{}, err
	}
	_ = s.detect(ctx, r)
	if r, err = s.get(id); err != nil {
		return Public{}, err
	}
	return s.view(r), nil
}

// Cancel withdraws a card that has not been paid (the sender's token proves who asks).
func (s *Service) Cancel(id, token string) (Public, error) {
	defer s.lock(id)()
	r, err := s.get(id)
	if err != nil {
		return Public{}, err
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	sum := sha256.Sum256(raw)
	want, _ := hex.DecodeString(r.SenderHash)
	if err != nil || subtle.ConstantTimeCompare(sum[:], want) != 1 {
		return Public{}, ErrForbidden
	}
	if r.State == StatePaid {
		return Public{}, ErrForbidden
	}
	r.State = StateCancelled
	if err := s.update(r); err != nil {
		return Public{}, err
	}
	return s.view(r), nil
}

// Run watches the cards that wait for a payment while nobody has a page open, until ctx ends.
func (s *Service) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.mu.Lock()
			var ids []string
			for id, r := range s.records {
				if r.State == StateReady {
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
