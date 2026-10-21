package otp

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPersistentOTPResistsRestartAttemptsReplayAndRecovery(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	key := bytes.Repeat([]byte{7}, 32)
	sender := &fakeSender{}
	open := func() *Manager {
		t.Helper()
		m, err := NewPersistent(sender, dir, key)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	m := open()
	now := time.Now()
	m.now = func() time.Time { return now }
	if err := m.IssueClaim(context.Background(), "claim", "recipient@example.test"); err != nil {
		t.Fatal(err)
	}
	code := sender.code
	data, err := os.ReadFile(filepath.Join(dir, "otp-challenges.sealed"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(code)) || bytes.Contains(data, []byte("recipient@example.test")) {
		t.Fatal("OTP state was not encrypted")
	}
	m = open()
	m.now = func() time.Time { return now }
	if err := m.IssueClaim(context.Background(), "claim", "recipient@example.test"); !errors.Is(err, ErrRateLimited) {
		t.Fatal("restart reset resend cooldown", err)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 4; i++ {
		if err := m.Verify("claim", "recipient@example.test", wrong); !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}
	m = open()
	m.now = func() time.Time { return now }
	if err := m.Verify("claim", "recipient@example.test", code); err != nil {
		t.Fatal("valid persisted OTP rejected", err)
	}
	m = open()
	m.now = func() time.Time { return now }
	if err := m.Verify("claim", "recipient@example.test", code); !errors.Is(err, ErrAlreadyVerified) {
		t.Fatal("consumed OTP survived as valid after restart", err)
	}
	// A lost release response is recoverable only with a freshly emailed code.
	now = now.Add(16 * time.Minute)
	if err := m.IssueClaim(context.Background(), "claim", "recipient@example.test"); err != nil {
		t.Fatal("recovery issuance", err)
	}
	recovery := sender.code
	m = open()
	m.now = func() time.Time { return now }
	if err := m.Verify("claim", "recipient@example.test", recovery); err != nil {
		t.Fatal("recovery OTP", err)
	}
}

func TestPersistentOTPAttemptLimitExpiryAndStorageFailures(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	key := bytes.Repeat([]byte{8}, 32)
	sender := &fakeSender{}
	m, err := NewPersistent(sender, dir, key)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Issue(context.Background(), "c", "r@example.test"); err != nil {
		t.Fatal(err)
	}
	code := sender.code
	wrong := "abcdef"
	for i := 0; i < 5; i++ {
		if err = m.Verify("c", "r@example.test", wrong); !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}
	m, err = NewPersistent(sender, dir, key)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Verify("c", "r@example.test", code); !errors.Is(err, ErrInvalid) {
		t.Fatal("restart reset attempts", err)
	}
	if _, err = NewPersistent(sender, dir, bytes.Repeat([]byte{9}, 32)); err == nil {
		t.Fatal("wrong master accepted")
	}
	// An unwritable persistence path must fail closed, including readiness.
	if err = os.Remove(filepath.Join(dir, "otp-challenges.sealed")); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(dir, "otp-challenges.sealed"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = m.Issue(context.Background(), "next", "r@example.test"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("storage failure ignored", err)
	}
	if m.Durable() {
		t.Fatal("failed OTP store advertised readiness")
	}
}
