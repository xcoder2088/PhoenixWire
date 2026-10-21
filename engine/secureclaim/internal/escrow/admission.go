// Package escrow connects immutable OTP claims to encrypted Cashlink secrets.
// Funding verification and Cashlink address derivation MUST be implemented by a
// trusted server-side Nimiq Mainnet adapter before this component is used with
// any money. No HTTP exposure, no release API, and no private key in emails.
package escrow

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

var (
	ErrUnavailable = errors.New("secureclaim_escrow_unavailable")
	ErrUnfunded    = errors.New("secureclaim_unfunded")
	ErrMismatched  = errors.New("secureclaim_cashlink_mismatch")
)

// ChainVerifier is an internal trusted RPC/node adapter, not client input.
// Implementations must verify *confirmed and currently spendable* Mainnet funds,
// not merely that a submitted tx hash exists.
type ChainVerifier interface {
	ConfirmedSpendable(ctx context.Context, network, address string, exactLuna uint64) (bool, error)
}

// AddressDeriver independently computes the public Cashlink address from the
// private 32-byte Ed25519 seed according to the Nimiq protocol. Never trust a
// sender-provided address as proof that the seed controls the funded account.
type AddressDeriver interface {
	CashlinkAddress(seed []byte) (string, error)
}

// PrivateStore is a private ciphertext-only destination. The interface
// intentionally provides no Open/read method to the admission controller.
type PrivateStore interface {
	Put(claimID string, seed []byte) error
}

// Admission is a backend-only admission gate. Locking prevents double
// admission in this process; the vault's create-once disk publication also
// rejects duplicate claim IDs. For production, persist a transactional state
// machine with durable locking and recovery across processes.
type Admission struct {
	mu       sync.Mutex
	claims   *claim.Store
	verifier ChainVerifier
	derive   AddressDeriver
	private  PrivateStore
}

func NewAdmission(store *claim.Store, verifier ChainVerifier, derive AddressDeriver, private PrivateStore) (*Admission, error) {
	if store == nil || verifier == nil || derive == nil || private == nil {
		return nil, ErrUnavailable
	}
	return &Admission{claims: store, verifier: verifier, derive: derive, private: private}, nil
}

// Admit accepts an actual Cashlink spending SEED, never the public cashlink
// URL or its QR. It fails closed when the sender's seed does not control the
// stated public address or the chain cannot verify the exact claim amount.
// It does NOT mark the claim funded, enable OTP payout, or expose any secret.
func (a *Admission) Admit(ctx context.Context, claimID string, seed []byte, expectedAddress string, message ...string) error {
	if a == nil || ctx == nil || len(seed) != 32 || expectedAddress == "" || len(expectedAddress) > 128 {
		return ErrMismatched
	}
	if err := ctx.Err(); err != nil {
		return ErrUnavailable
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	record, err := a.claims.Get(claimID)
	if err != nil {
		return err
	}
	if record.Network != "mainnet" || (record.Status != "draft_unfunded" && record.Status != "email_verified_unfunded") {
		return ErrUnavailable
	}
	derived, err := a.derive.CashlinkAddress(seed)
	if err != nil || derived == "" || subtle.ConstantTimeCompare([]byte(derived), []byte(expectedAddress)) != 1 {
		return ErrMismatched
	}
	_, luna, err := claim.ParseAmount(record.Amount)
	if err != nil {
		return ErrMismatched
	}
	confirmed, err := a.verifier.ConfirmedSpendable(ctx, record.Network, derived, luna)
	if err != nil {
		return fmt.Errorf("%w: chain verification unavailable", ErrUnavailable)
	}
	if !confirmed {
		return ErrUnfunded
	}
	if err := ctx.Err(); err != nil {
		return ErrUnavailable
	}
	// Disk stores ONLY authenticated ciphertext. Caller must have already
	// authenticated the sender; this internal component does not grant access.
	// Preserve the original Cashlink message inside the same encrypted envelope.
	// The secret remains a Nimiq seed; this only restores the original Hub URL.
	payload := append([]byte(nil), seed...)
	if len(message) > 0 && message[0] != "" {
		m := message[0]
		if len(message) != 1 || len([]byte(m)) > 64 || strings.ContainsAny(m, "\r\n") {
			return ErrMismatched
		}
		payload = append(payload, byte(len(m)))
		payload = append(payload, m...)
	}
	if err := a.private.Put(claimID, payload); err != nil {
		if errors.Is(err, vault.ErrAlreadyStored) {
			return vault.ErrAlreadyStored
		}
		return ErrUnavailable
	}
	return nil
}
