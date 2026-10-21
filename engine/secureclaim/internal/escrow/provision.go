package escrow

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

// SecretVault exposes private methods to trusted server-side components only.
type SecretVault interface {
	Put(claimID string, seed []byte) error
	Open(claimID string) ([]byte, error)
}

// Provisioner is the correct secure Cashlink creation order:
// generate secret SERVER-SIDE -> seal BEFORE funding -> return PUBLIC address.
// The Cashlink spending seed is NEVER sent to the sender's browser.
// A server restart can recover the same address from the encrypted vault.
type Provisioner struct {
	mu       sync.Mutex
	claims   *claim.Store
	derive   AddressDeriver
	private  SecretVault
	verifier ChainVerifier
}

func NewProvisioner(store *claim.Store, derive AddressDeriver, private SecretVault, verifier ChainVerifier) (*Provisioner, error) {
	if store == nil || derive == nil || private == nil || verifier == nil {
		return nil, ErrUnavailable
	}
	return &Provisioner{claims: store, derive: derive, private: private, verifier: verifier}, nil
}

// Provision returns only a public Nimiq address. It is idempotent for a claim.
// The caller must be authenticated and authorized to manage the claim.
// Never publish this method as an anonymous HTTP endpoint.
func (p *Provisioner) Provision(claimID string) (string, error) {
	if p == nil {
		return "", ErrUnavailable
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	record, err := p.claims.Get(claimID)
	if err != nil {
		return "", err
	}
	if record.Network != "mainnet" || record.Status != "draft_unfunded" {
		return "", ErrUnavailable
	}
	old, err := p.private.Open(claimID)
	if err == nil {
		if len(old) != 32 {
			clear(old)
			return "", ErrMismatched
		}
		address, e := p.derive.CashlinkAddress(old)
		clear(old)
		return address, e
	}
	if !errors.Is(err, vault.ErrMissing) {
		return "", ErrUnavailable
	}
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return "", ErrUnavailable
	}
	address, err := p.derive.CashlinkAddress(seed)
	if err != nil {
		clear(seed)
		return "", ErrUnavailable
	}
	err = p.private.Put(claimID, seed)
	clear(seed)
	if errors.Is(err, vault.ErrAlreadyStored) {
		old, err := p.private.Open(claimID)
		if err != nil || len(old) != 32 {
			clear(old)
			return "", ErrUnavailable
		}
		existing, e := p.derive.CashlinkAddress(old)
		clear(old)
		return existing, e
	}
	if err != nil {
		return "", ErrUnavailable
	}
	return address, nil
}

// FundingConfirmed reads the sealed seed internally to recompute the
// funding address, and independently checks the Mainnet chain. It does not
// authorize payout or change claim state; a durable settlement state machine
// is required before step 2 can release any NIM.
func (p *Provisioner) FundingConfirmed(ctx context.Context, claimID string) (bool, error) {
	if p == nil || ctx == nil {
		return false, ErrUnavailable
	}
	record, err := p.claims.Get(claimID)
	if err != nil {
		return false, err
	}
	if record.Network != "mainnet" {
		return false, ErrUnavailable
	}
	seed, err := p.private.Open(claimID)
	if err != nil {
		return false, ErrUnavailable
	}
	if len(seed) != 32 {
		clear(seed)
		return false, ErrMismatched
	}
	address, err := p.derive.CashlinkAddress(seed)
	clear(seed)
	if err != nil {
		return false, ErrMismatched
	}
	_, luna, err := claim.ParseAmount(record.Amount)
	if err != nil {
		return false, ErrMismatched
	}
	return p.verifier.ConfirmedSpendable(ctx, "mainnet", address, luna)
}
