package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDraftCreationIsOpaqueUnfundedAndDeterministicallyColored(t *testing.T) {
	s := NewStore()
	a, err := s.Create("dest@example.com", "5.50000", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create("dest@example.com", "5.5", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID || len(a.ID) != 48 || a.Amount != "5.5" || a.Network != "mainnet" || a.Status != "draft_unfunded" || a.EmailVerified {
		t.Fatalf("incorrect draft: %+v / %+v", a, b)
	}
	if a.Palette < 0 || a.Palette >= 12 {
		t.Fatalf("bad palette %d", a.Palette)
	}
	visible, err := s.Get(a.ID)
	if err != nil || visible != a {
		t.Fatalf("draft changed: %+v / %v", visible, err)
	}
	recipient, err := s.Recipient(a.ID)
	if err != nil || recipient != "dest@example.com" {
		t.Fatal(err)
	}
	if strings.Contains(a.ID, "@") || strings.Contains(a.Amount, "@") {
		t.Fatal("recipient leaked")
	}
	v, err := s.VerifyEmail(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != "email_verified_unfunded" || !v.EmailVerified {
		t.Fatalf("wrong state %+v", v)
	}
	if _, err = s.VerifyEmail(a.ID); !errors.Is(err, ErrAlreadyVerified) {
		t.Fatalf("replay: %v", err)
	}
}

func TestRejectUnsafeDrafts(t *testing.T) {
	s := NewStore()
	for _, row := range []struct{ email, amount, network string }{
		{"alice@example.com", "5", "testnet"},
		{"Name <alice@example.com>", "5", "mainnet"},
		{"alice@example.com\r\nBcc:evil@example.com", "5", "mainnet"},
		{"alice@example.com", "0", "mainnet"},
		{"alice@example.com", "0.000001", "mainnet"},
		{"alice@example.com", "1e6", "mainnet"},
		{"alice@example.com", "1000.00001", "mainnet"},
		{"alice@example.com", "-1", "mainnet"},
		{"alice@example.com", "1.234567", "mainnet"},
	} {
		if _, err := s.Create(row.email, row.amount, row.network); !errors.Is(err, ErrInvalid) {
			t.Fatalf("accepted %#v: %v", row, err)
		}
	}
}

func TestExpiryPreventsReadOTPAndVerification(t *testing.T) {
	s := NewStore()
	now := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	a, e := s.Create("alice@example.com", "5", "mainnet")
	if e != nil {
		t.Fatal(e)
	}
	now = now.Add(Lifetime)
	if _, e = s.Get(a.ID); !errors.Is(e, ErrExpired) {
		t.Fatal("read after expiry", e)
	}
	if _, e = s.Recipient(a.ID); !errors.Is(e, ErrExpired) {
		t.Fatal("OTP after expiry", e)
	}
	if _, e = s.VerifyEmail(a.ID); !errors.Is(e, ErrExpired) {
		t.Fatal("verify after expiry", e)
	}
}

func TestParseAmountCanonicalLuna(t *testing.T) {
	cases := map[string]uint64{"5": 500000, "5.5": 550000, "0.00001": 1, "0.1": 10000, "1000": 100000000, "1.23450": 123450}
	for raw, want := range cases {
		_, got, err := ParseAmount(raw)
		if err != nil || got != want {
			t.Fatalf("%s => %d / %v", raw, got, err)
		}
	}
}

func TestForgetErasesAClaimAndIsHarmlessTwice(t *testing.T) {
	s := NewStore()
	c, err := s.Create("recipient@example.test", "1", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Forget(c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Peek(c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("the claim and its e-mail address are still there", err)
	}
	if err := s.Forget(c.ID); err != nil {
		t.Fatal("forgetting twice", err)
	}
}
