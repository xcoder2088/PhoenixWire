// Package claim manages opaque, immutable mainnet-only draft claims.
// It deliberately does not hold funds, blockchain keys, signing capabilities,
// or an endpoint to mark funding based on a browser assertion.
package claim

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalid         = errors.New("invalid_claim")
	ErrNotFound        = errors.New("claim_not_found")
	ErrExpired         = errors.New("claim_expired")
	ErrCapacity        = errors.New("claim_capacity_reached")
	ErrAlreadyVerified = errors.New("claim_already_verified")
)

const (
	Lifetime       = 24 * time.Hour
	MaxDraftClaims = 10000
)

var amountRE = regexp.MustCompile(`^(0|[1-9][0-9]{0,6})(?:\.([0-9]{1,5}))?$`)

// Public describes a claim without exposing the destination email or any secret.
// Status never says funded: there is no funding implementation in this module.
type Public struct {
	ID            string    `json:"id"`
	Amount        string    `json:"amount"`
	Network       string    `json:"network"`
	Palette       int       `json:"palette"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type record struct {
	public Public
	email  string
	luna   uint64
}

type Store struct {
	mu      sync.Mutex
	values  map[string]*record
	now     func() time.Time
	persist func(map[string]*record) error
}

func NewStore() *Store { return &Store{values: make(map[string]*record), now: time.Now} }

func (s *Store) Durable() bool { return s != nil && s.persist != nil }

// CreateOnce binds a trusted issuer's deterministic request ID to one immutable
// claim. A retry after a lost response reuses the same funding/protection record.
func (s *Store) CreateOnce(id, email, amount, network string) (Public, error) {
	if len(id) != 48 || network != "mainnet" || !validEmail(email) {
		return Public{}, ErrInvalid
	}
	if _, err := hex.DecodeString(id); err != nil {
		return Public{}, ErrInvalid
	}
	canonical, luna, err := ParseAmount(amount)
	if err != nil {
		return Public{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.values[id]; ok {
		if old.email != email || old.public.Amount != canonical || old.public.Network != network {
			return Public{}, ErrInvalid
		}
		if !s.now().Before(old.public.ExpiresAt) {
			return Public{}, ErrExpired
		}
		return old.public, nil
	}
	if len(s.values) >= MaxDraftClaims {
		return Public{}, ErrCapacity
	}
	p := s.createLocked(id, email, canonical, luna, s.now().UTC())
	if s.persist != nil {
		if err := s.persist(s.values); err != nil {
			delete(s.values, id)
			return Public{}, err
		}
	}
	return p, nil
}

// ParseAmount returns the canonical amount and its integer Luna value.
// This conservative draft limit is not a supported payment limit.
func ParseAmount(input string) (string, uint64, error) {
	matches := amountRE.FindStringSubmatch(input)
	if matches == nil {
		return "", 0, ErrInvalid
	}
	whole, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil || whole > 1000 {
		return "", 0, ErrInvalid
	}
	fraction := matches[2]
	padded := fraction + strings.Repeat("0", 5-len(fraction))
	frac, err := strconv.ParseUint(padded, 10, 64)
	if err != nil {
		return "", 0, ErrInvalid
	}
	luna := whole*100000 + frac
	if luna == 0 || luna > 1000*100000 {
		return "", 0, ErrInvalid
	}
	canonical := strconv.FormatUint(luna/100000, 10)
	if rest := luna % 100000; rest > 0 {
		canonical += "." + strings.TrimRight(leftPad5(rest), "0")
	}
	return canonical, luna, nil
}

func leftPad5(v uint64) string {
	s := strconv.FormatUint(v, 10)
	return strings.Repeat("0", 5-len(s)) + s
}

func validEmail(raw string) bool {
	if len(raw) < 3 || len(raw) > 254 || strings.ContainsAny(raw, "\r\n\t ") {
		return false
	}
	addr, err := mail.ParseAddress(raw)
	return err == nil && addr.Address == raw && strings.Contains(raw, "@")
}

func (s *Store) createLocked(id, email, amount string, luna uint64, now time.Time) Public {
	var pal int
	if len(id) >= 2 {
		n, _ := strconv.ParseUint(id[:2], 16, 8)
		pal = int(n) % 12
	}
	p := Public{ID: id, Amount: amount, Network: "mainnet", Palette: pal, Status: "draft_unfunded", ExpiresAt: now.Add(Lifetime)}
	s.values[id] = &record{public: p, email: email, luna: luna}
	return p
}

// SeedDemo creates the explicitly local demo, not a payable claim.
func (s *Store) SeedDemo(email string) error {
	if !validEmail(email) {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createLocked("demo", email, "5", 500000, s.now().UTC())
	return nil
}

// Create requires a trusted caller at the HTTP layer. No wallet funding is
// accepted here; this creates only a time-limited, unfunded draft.
func (s *Store) Create(email, amount, network string) (Public, error) {
	if network != "mainnet" || !validEmail(email) {
		return Public{}, ErrInvalid
	}
	canonical, luna, err := ParseAmount(amount)
	if err != nil {
		return Public{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	if len(s.values) >= MaxDraftClaims {
		return Public{}, ErrCapacity
	}
	for i := 0; i < 3; i++ {
		var key [24]byte
		if _, err = rand.Read(key[:]); err != nil {
			return Public{}, err
		}
		id := hex.EncodeToString(key[:])
		if _, ok := s.values[id]; !ok {
			created := s.createLocked(id, email, canonical, luna, now)
			if s.persist != nil {
				if err := s.persist(s.values); err != nil {
					delete(s.values, id)
					return Public{}, err
				}
			}
			return created, nil
		}
	}
	return Public{}, ErrCapacity
}

func (s *Store) lookupLocked(id string) (*record, error) {
	r, ok := s.values[id]
	if !ok {
		return nil, ErrNotFound
	}
	if !s.now().Before(r.public.ExpiresAt) {
		return nil, ErrExpired
	}
	return r, nil
}

// Peek is Get without the expiry: what the service itself may read about a claim whose time is over (to refund it).
func (s *Store) Peek(id string) (Public, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.values[id]
	if !ok {
		return Public{}, ErrNotFound
	}
	return r.public, nil
}

func (s *Store) Get(id string) (Public, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.lookupLocked(id)
	if err != nil {
		return Public{}, err
	}
	return r.public, nil
}

// Recipient is available only to the service when sending OTP.
func (s *Store) Recipient(id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.lookupLocked(id)
	if err != nil {
		return "", err
	}
	return r.email, nil
}

// VerifyEmail marks *only* proof of email ownership. It does not release funds.
func (s *Store) VerifyEmail(id string) (Public, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.lookupLocked(id)
	if err != nil {
		return Public{}, err
	}
	if r.public.EmailVerified {
		return Public{}, ErrAlreadyVerified
	}
	r.public.EmailVerified = true
	r.public.Status = "email_verified_unfunded"
	if s.persist != nil {
		if err := s.persist(s.values); err != nil {
			r.public.EmailVerified = false
			r.public.Status = "draft_unfunded"
			return Public{}, err
		}
	}
	return r.public, nil
}
