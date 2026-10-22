package service

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/internal/escrow"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

func TestOTPReadinessRequiresAllFundsProtectionServices(t *testing.T) {
	s, err := newServer(&testSender{}, "recipient@example.test", true)
	if err != nil {
		t.Fatal(err)
	}
	check := func() map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		s.handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
		if w.Code != 200 {
			t.Fatalf("status %d", w.Code)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("capabilities were cached")
		}
		var data map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(w.Body.String(), s.creationKey) && len(s.creationKey) > 0 {
			t.Fatal("issuer key leaked")
		}
		if data["network"] != "mainnet" {
			t.Fatal(data)
		}
		return data
	}
	if d := check(); d["protected_cashlink_ready"] != false || d["otp_delivery"] != "dev_console" {
		t.Fatal(d)
	}
	s.creationKey = strings.Repeat("k", 64)
	if d := check(); d["protected_cashlink_ready"] != false {
		t.Fatal("key alone enabled OTP", d)
	}
	s.escrow = &escrow.Admission{}
	if d := check(); d["protected_cashlink_ready"] != false {
		t.Fatal("missing vault enabled OTP", d)
	}
	s.cashlinkVault = &vault.FileStore{}
	if d := check(); d["protected_cashlink_ready"] != false || d["vault_ready"] != true {
		t.Fatal("complete services not reported", d)
	}
	state := t.TempDir()
	os.Chmod(state, 0700)
	persistent, err := claim.NewPersistentStore(state)
	if err != nil {
		t.Fatal(err)
	}
	s.claims = persistent
	manager, err := otp.NewPersistent(&testSender{}, state, bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	s.otp = manager
	if d := check(); d["protected_cashlink_ready"] != false {
		t.Fatal("console enabled Mainnet", d)
	}
	s.devConsole = false
	s.chainReady = func(context.Context) bool { return true }
	if d := check(); d["protected_cashlink_ready"] != true || d["mainnet_enabled"] != true {
		t.Fatal("durable email OTP not ready", d)
	}
	s.chainReady = func(context.Context) bool { return false }
	if d := check(); d["mainnet_enabled"] != false {
		t.Fatal("wrong/unavailable chain enabled wallet funding", d)
	}
	w := httptest.NewRecorder()
	s.handler(w, httptest.NewRequest(http.MethodPost, "/api/v1/health", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal("POST exposed", w.Code)
	}
}
