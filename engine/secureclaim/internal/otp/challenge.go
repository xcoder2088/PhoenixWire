// Package otp provides an independent, claim-bound email OTP mechanism.
// NewPersistent stores encrypted challenges for the single-process local service.
package otp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	ErrRateLimited     = errors.New("otp_rate_limited")
	ErrInvalid         = errors.New("otp_invalid_or_expired")
	ErrAlreadyVerified = errors.New("otp_already_verified")
	ErrUnavailable     = errors.New("otp_not_configured")
)

// A new code may be requested after 20 seconds; 8 code requests and 5 wrong guesses are allowed per 10 minutes.
const (
	resendDelay   = 20 * time.Second
	maxRequests   = 8
	requestWindow = 10 * time.Minute
)

// RateLimit is ErrRateLimited that also says how long to wait: the page shows a countdown.
type RateLimit struct{ RetryAfter time.Duration }

func (e *RateLimit) Error() string          { return ErrRateLimited.Error() }
func (e *RateLimit) Is(target error) bool   { return target == ErrRateLimited }
func (e *RateLimit) WaitFor() time.Duration { return e.RetryAfter }

type Sender interface {
	Send(context.Context, string, string) error
}

type challenge struct {
	email    string
	digest   [32]byte
	expires  time.Time
	lastSent time.Time
	window   time.Time
	requests int
	attempts int
	verified bool
}

type Manager struct {
	mu            sync.Mutex
	key           [32]byte
	sender        Sender
	challenges    map[string]*challenge
	now           func() time.Time
	persist       func() error
	storageFailed bool
}

func New(sender Sender) (*Manager, error) {
	if sender == nil {
		return nil, ErrUnavailable
	}
	m := &Manager{sender: sender, challenges: make(map[string]*challenge), now: time.Now}
	if _, err := rand.Read(m.key[:]); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) hash(id, email, code string) [32]byte {
	mac := hmac.New(sha256.New, m.key[:])
	mac.Write([]byte("NimiqSecureClaim otp v1\x00"))
	mac.Write([]byte(id))
	mac.Write([]byte{0})
	mac.Write([]byte(email))
	mac.Write([]byte{0})
	mac.Write([]byte(code))
	var result [32]byte
	copy(result[:], mac.Sum(nil))
	return result
}

func (m *Manager) Issue(ctx context.Context, id, email string) error {
	return m.issue(ctx, id, email, false)
}

// IssueClaim permits a fresh OTP to reopen the same funded Cashlink after a
// lost response or a restart. It never reuses a verified code or funds again.
func (m *Manager) IssueClaim(ctx context.Context, id, email string) error {
	return m.issue(ctx, id, email, true)
}

func (m *Manager) issue(ctx context.Context, id, email string, recoverClaim bool) error {
	if id == "" || email == "" {
		return ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.storageFailed {
		return ErrUnavailable
	}
	now := m.now()
	c := m.challenges[id]
	if c == nil {
		if len(m.challenges) >= 10000 {
			for k, old := range m.challenges {
				if now.After(old.window.Add(requestWindow)) && now.After(old.expires) {
					delete(m.challenges, k)
				}
			}
			if len(m.challenges) >= 10000 {
				return ErrRateLimited
			}
		}
		c = &challenge{email: email}
		m.challenges[id] = c
	}
	if c.email != email {
		return ErrInvalid
	}
	if c.verified && !recoverClaim {
		return ErrAlreadyVerified
	}
	if !c.window.IsZero() && now.Sub(c.window) >= requestWindow {
		c.window = time.Time{}
		c.requests, c.attempts = 0, 0
	}
	if !c.lastSent.IsZero() && now.Sub(c.lastSent) < resendDelay || c.requests >= maxRequests || c.attempts >= 5 {
		var wait time.Duration
		if c.requests >= maxRequests || c.attempts >= 5 {
			wait = c.window.Add(requestWindow).Sub(now)
		} else {
			wait = c.lastSent.Add(resendDelay).Sub(now)
		}
		return &RateLimit{RetryAfter: wait}
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	if c.window.IsZero() {
		c.window = now
	}
	c.requests++
	c.lastSent = now
	c.verified = false
	// Invalidate an older code before dispatching its replacement.
	c.digest = [32]byte{}
	c.expires = time.Time{}
	// Reserve resend/attempt limits durably before contacting the mail provider.
	if err := m.saveLocked(); err != nil {
		return err
	}
	if err := m.sender.Send(ctx, email, code); err != nil {
		return err
	}
	c.digest = m.hash(id, email, code)
	c.expires = now.Add(10 * time.Minute)
	return m.saveLocked()
}

func (m *Manager) Verify(id, email, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.storageFailed {
		return ErrUnavailable
	}
	c := m.challenges[id]
	if c == nil || c.email != email {
		return ErrInvalid
	}
	if c.verified {
		return ErrAlreadyVerified
	}
	if !m.now().Before(c.expires) || c.expires.IsZero() || c.attempts >= 5 {
		return ErrInvalid
	}
	c.attempts++
	// Every attempted code is durable, including malformed codes.
	if err := m.saveLocked(); err != nil {
		return err
	}
	if len(code) != 6 {
		return ErrInvalid
	}
	for i := range code {
		if code[i] < '0' || code[i] > '9' {
			return ErrInvalid
		}
	}
	digest := m.hash(id, email, code)
	if subtle.ConstantTimeCompare(digest[:], c.digest[:]) != 1 {
		return ErrInvalid
	}
	c.verified = true
	c.digest = [32]byte{}
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	if m.persist == nil {
		return nil
	}
	if err := m.persist(); err != nil {
		m.storageFailed = true
		return ErrUnavailable
	}
	return nil
}

func (m *Manager) Durable() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.persist != nil && !m.storageFailed
}

// Verified reflects only email OTP verification, NEVER authorization to spend funds.
func (m *Manager) Verified(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.challenges[id]
	return c != nil && c.verified
}
