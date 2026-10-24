package otp

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeSender struct {
	code, to string
	err      error
	count    int
}

func (f *fakeSender) Send(_ context.Context, to, code string) error {
	f.count++
	f.to, f.code = to, code
	return f.err
}

func TestOTPValidOnlyOnceAndBoundToEmail(t *testing.T) {
	sender := &fakeSender{}
	m, err := New(sender)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Issue(context.Background(), "claim1", "one@example.com"); err != nil {
		t.Fatal(err)
	}
	if sender.to != "one@example.com" || len(sender.code) != 6 {
		t.Fatal("OTP delivery mismatch")
	}
	if err = m.Verify("claim1", "other@example.com", sender.code); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong recipient allowed: %v", err)
	}
	if m.Verified("claim1") {
		t.Fatal("email was not verified")
	}
	if err = m.Verify("claim1", "one@example.com", sender.code); err != nil {
		t.Fatal(err)
	}
	if err = m.Verify("claim1", "one@example.com", sender.code); !errors.Is(err, ErrAlreadyVerified) {
		t.Fatalf("replay allowed: %v", err)
	}
}

func TestOTPRateLimitsExpiryAndFailedDelivery(t *testing.T) {
	sender := &fakeSender{}
	m, _ := New(sender)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	if err := m.Issue(context.Background(), "a", "a@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := m.Issue(context.Background(), "a", "a@example.com"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("resend allowed: %v", err)
	}
	now = now.Add(11 * time.Minute)
	if err := m.Verify("a", "a@example.com", sender.code); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expired OTP allowed: %v", err)
	}
	if err := m.Issue(context.Background(), "a", "a@example.com"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		bad := "000000"
		if sender.code == bad {
			bad = "111111"
		}
		if err := m.Verify("a", "a@example.com", bad); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad OTP accepted: %v", err)
		}
	}
	if err := m.Verify("a", "a@example.com", sender.code); !errors.Is(err, ErrInvalid) {
		t.Fatalf("guesses not locked: %v", err)
	}
	sender.err = errors.New("provider rejected")
	if err := m.Issue(context.Background(), "b", "b@example.com"); err == nil {
		t.Fatal("delivery failure ignored")
	}
	if m.Verified("b") {
		t.Fatal("failed delivery verified")
	}
}

func TestRateLimitSaysHowLongToWait(t *testing.T) {
	m, _ := New(&fakeSender{})
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	for i := 0; i < maxRequests; i++ {
		if err := m.Issue(context.Background(), "a", "a@example.com"); err != nil {
			t.Fatal(i, err)
		}
		now = now.Add(resendDelay)
	}
	err := m.Issue(context.Background(), "a", "a@example.com")
	var limited *RateLimit
	if !errors.Is(err, ErrRateLimited) || !errors.As(err, &limited) {
		t.Fatalf("the ninth request must be limited: %v", err)
	}
	// the window opened at the first request and lasts requestWindow
	if want := requestWindow - maxRequests*resendDelay; limited.RetryAfter != want {
		t.Fatalf("wait %v, want %v", limited.RetryAfter, want)
	}
	now = now.Add(limited.RetryAfter)
	if err := m.Issue(context.Background(), "a", "a@example.com"); err != nil {
		t.Fatalf("a request after the announced wait must pass: %v", err)
	}
}
