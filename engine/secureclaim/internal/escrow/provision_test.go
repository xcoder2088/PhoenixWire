package escrow

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

type memoryVault struct{ seed map[string][]byte }

func (m *memoryVault) Put(id string, seed []byte) error {
	if _, ok := m.seed[id]; ok {
		return vault.ErrAlreadyStored
	}
	m.seed[id] = append([]byte(nil), seed...)
	return nil
}
func (m *memoryVault) Open(id string) ([]byte, error) {
	seed, ok := m.seed[id]
	if !ok {
		return nil, vault.ErrMissing
	}
	return append([]byte(nil), seed...), nil
}

type fixedFunding struct {
	address   string
	amount    uint64
	confirmed bool
}

func (f *fixedFunding) ConfirmedSpendable(_ context.Context, network, address string, amount uint64) (bool, error) {
	if network != "mainnet" || address != f.address || amount != f.amount {
		return false, ErrMismatched
	}
	return f.confirmed, nil
}
func TestProvisionBeforeFundingIsPrivateAndIdempotent(t *testing.T) {
	store := claim.NewStore()
	draft, err := store.Create("recipient@example.com", "1.25", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	private := &memoryVault{seed: map[string][]byte{}}
	chain := &fixedFunding{amount: 125000}
	provisioner, err := NewProvisioner(store, nimiq.AddressDeriver{}, private, chain)
	if err != nil {
		t.Fatal(err)
	}
	address, err := provisioner.Provision(draft.ID)
	if err != nil || len(address) != 36 || len(private.seed[draft.ID]) != 32 {
		t.Fatal("provision failed", err)
	}
	original := append([]byte(nil), private.seed[draft.ID]...)
	again, err := provisioner.Provision(draft.ID)
	if err != nil || again != address || !bytes.Equal(private.seed[draft.ID], original) {
		t.Fatal("provision changed Cashlink secret", err)
	}
	chain.address = address
	confirmed, err := provisioner.FundingConfirmed(context.Background(), draft.ID)
	if err != nil || confirmed {
		t.Fatal("unfunded accepted", err)
	}
	chain.confirmed = true
	confirmed, err = provisioner.FundingConfirmed(context.Background(), draft.ID)
	if err != nil || !confirmed {
		t.Fatal("funding not recognized", err)
	}
	if _, err = provisioner.Provision("missing"); !errors.Is(err, claim.ErrNotFound) {
		t.Fatal("missing claim accepted")
	}
}
