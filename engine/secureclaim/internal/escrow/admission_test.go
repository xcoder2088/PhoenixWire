package escrow

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

type fakeDeriver struct {
	address string
	err     error
}

func (d fakeDeriver) CashlinkAddress(_ []byte) (string, error) { return d.address, d.err }

type fakeChain struct {
	approved bool
	err      error
	calls    atomic.Int32
	want     uint64
}

func (c *fakeChain) ConfirmedSpendable(_ context.Context, n, a string, amount uint64) (bool, error) {
	c.calls.Add(1)
	if n != "mainnet" || a != "NQ_TEST_ADDR" || amount != c.want {
		return false, ErrMismatched
	}
	return c.approved, c.err
}

type fakePrivate struct {
	mu     sync.Mutex
	called int
	saved  []byte
}

func (p *fakePrivate) Put(id string, seed []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.called != 0 {
		return vault.ErrAlreadyStored
	}
	p.called++
	p.saved = append([]byte(nil), seed...)
	return nil
}
func prep(t *testing.T, approve bool) (*Admission, *fakeChain, *fakePrivate, string) {
	t.Helper()
	s := claim.NewStore()
	rec, err := s.Create("recipient@example.com", "2.5", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	chain := &fakeChain{approved: approve, want: 250000}
	storage := &fakePrivate{}
	a, err := NewAdmission(s, chain, fakeDeriver{address: "NQ_TEST_ADDR"}, storage)
	if err != nil {
		t.Fatal(err)
	}
	return a, chain, storage, rec.ID
}
func TestFundingIsVerifiedBeforeCiphertextAdmission(t *testing.T) {
	a, c, p, id := prep(t, true)
	seed := bytes.Repeat([]byte{7}, 32)
	if err := a.Admit(context.Background(), id, seed, "NQ_TEST_ADDR"); err != nil {
		t.Fatal(err)
	}
	if c.calls.Load() != 1 || p.called != 1 || !bytes.Equal(p.saved, seed) {
		t.Fatal("funding admission failed")
	}
	// No state says 'funded': admission and on-chain payout remain separate.
	view, err := a.claims.Get(id)
	if err != nil || view.Status != "draft_unfunded" {
		t.Fatal("dangerous funded state prematurely exposed")
	}
	if err := a.Admit(context.Background(), id, seed, "NQ_TEST_ADDR"); !errors.Is(err, vault.ErrAlreadyStored) {
		t.Fatal("repeated admission replaced secret", err)
	}
}
func TestNoDepositIfUnfundedWrongAmountOrWallet(t *testing.T) {
	a, c, p, id := prep(t, false)
	seed := bytes.Repeat([]byte{3}, 32)
	if err := a.Admit(context.Background(), id, seed, "NQ_TEST_ADDR"); !errors.Is(err, ErrUnfunded) {
		t.Fatal("unfunded accepted", err)
	}
	if p.called != 0 {
		t.Fatal("saved unverified funds")
	}
	c.approved = true
	if err := a.Admit(context.Background(), id, seed, "NQ_ANOTHER_ADDRESS"); !errors.Is(err, ErrMismatched) {
		t.Fatal("wrong key accepted", err)
	}
	if p.called != 0 {
		t.Fatal("saved a mismatched key")
	}
	if err := a.Admit(context.Background(), id, seed[:3], "NQ_TEST_ADDR"); !errors.Is(err, ErrMismatched) {
		t.Fatal("short seed accepted", err)
	}
	c.want = 100000
	if err := a.Admit(context.Background(), id, seed, "NQ_TEST_ADDR"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("incorrect amount accepted", err)
	}
	if p.called != 0 {
		t.Fatal("saved incorrect amount")
	}
	c.want = 250000
	c.err = errors.New("rpc timeout")
	if err := a.Admit(context.Background(), id, seed, "NQ_TEST_ADDR"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("RPC failure accepted", err)
	}
	if p.called != 0 {
		t.Fatal("saved despite unavailable chain")
	}
}
func TestEscrowDisabledWithoutTrustedAdapters(t *testing.T) {
	s := claim.NewStore()
	if _, err := NewAdmission(s, nil, nil, nil); !errors.Is(err, ErrUnavailable) {
		t.Fatal("nil chain verifier accepted")
	}
	if _, err := NewAdmission(nil, &fakeChain{}, fakeDeriver{}, &fakePrivate{}); !errors.Is(err, ErrUnavailable) {
		t.Fatal("nil claim store accepted")
	}
}
func TestConcurrentAdmissionOnlyPublishesOnce(t *testing.T) {
	a, _, p, id := prep(t, true)
	seed := bytes.Repeat([]byte{7}, 32)
	var wg sync.WaitGroup
	var successes atomic.Int32
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if a.Admit(context.Background(), id, seed, "NQ_TEST_ADDR") == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || p.called != 1 {
		t.Fatalf("admitted %d times, vault writes %d", successes.Load(), p.called)
	}
}
