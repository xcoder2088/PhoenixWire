package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/escrow"
	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

type confirmedFixture struct{}

func (confirmedFixture) ConfirmedSpendable(context.Context, string, string, uint64) (bool, error) {
	return true, nil
}

func TestHTTPClaimSealAndOTPReleaseSurviveRestartWithoutRecreatingPayment(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	state := filepath.Join(root, "state")
	private := filepath.Join(root, "vault")
	master := bytes.Repeat([]byte{0x28}, 32)
	sender := &testSender{}
	open := func() *server {
		t.Helper()
		s, err := newServer(sender, "", false)
		if err != nil {
			t.Fatal(err)
		}
		s.claims, err = claim.NewPersistentStore(state)
		if err != nil {
			t.Fatal(err)
		}
		s.otp, err = otp.NewPersistent(sender, state, master)
		if err != nil {
			t.Fatal(err)
		}
		v, err := vault.New(master, "fixture")
		if err != nil {
			t.Fatal(err)
		}
		s.cashlinkVault, err = vault.NewFileStore(private, v)
		if err != nil {
			t.Fatal(err)
		}
		s.escrow, err = escrow.NewAdmission(s.claims, confirmedFixture{}, nimiq.AddressDeriver{}, s.cashlinkVault)
		if err != nil {
			t.Fatal(err)
		}
		s.creationKey = strings.Repeat("issuer-fixture", 4)
		s.chainReady = func(context.Context) bool { return true }
		return s
	}
	call := func(s *server, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		data, _ := json.Marshal(body)
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
		r.Header.Set("X-Claim-Service-Key", s.creationKey)
		w := httptest.NewRecorder()
		s.handler(w, r)
		return w
	}
	s := open()
	payload := map[string]string{"email": "receiver@example.test", "amount": "2", "network": "mainnet", "request_id": "same-original-cashlink"}
	created := call(s, "/api/v1/claims", payload)
	if created.Code != 201 {
		t.Fatal(created.Code, created.Body)
	}
	var result struct {
		Claim claim.Public `json:"claim"`
	}
	json.Unmarshal(created.Body.Bytes(), &result)
	id := result.Claim.ID
	s = open()
	repeated := call(s, "/api/v1/claims", payload)
	if repeated.Code != 201 {
		t.Fatal(repeated.Code, repeated.Body)
	}
	var retry struct {
		Claim claim.Public `json:"claim"`
	}
	json.Unmarshal(repeated.Body.Bytes(), &retry)
	if retry.Claim.ID != id {
		t.Fatal("retry created another claim")
	}
	seed := bytes.Repeat([]byte{0x19}, 32)
	address, err := (nimiq.AddressDeriver{}).CashlinkAddress(seed)
	if err != nil {
		t.Fatal(err)
	}
	sealed := map[string]string{"seed": base64.RawURLEncoding.EncodeToString(seed), "address": address, "message": "hello"}
	for i := 0; i < 2; i++ {
		r := call(s, "/api/v1/claims/"+id+"?action=seal", sealed)
		if r.Code != 200 {
			t.Fatal("idempotent seal", r.Code, r.Body)
		}
	}
	sent := call(s, "/api/v1/claims/"+id+"?action=request", map[string]string{})
	if sent.Code != 200 {
		t.Fatal(sent.Code, sent.Body)
	}
	code := sender.code
	s = open()
	w := httptest.NewRecorder()
	s.handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/claims/"+id, nil))
	if strings.Contains(w.Body.String(), sealed["seed"]) || strings.Contains(w.Body.String(), "cashlink/#") {
		t.Fatal("public claim leaked spending secret")
	}
	released := call(s, "/api/v1/claims/"+id+"?action=release", map[string]string{"code": code})
	if released.Code != 200 || !strings.Contains(released.Body.String(), "https://hub.nimiq.com/cashlink/#") {
		t.Fatal(released.Code, released.Body)
	}
	s = open()
	replay := call(s, "/api/v1/claims/"+id+"?action=release", map[string]string{"code": code})
	if replay.Code != 409 {
		t.Fatal("restart revived consumed code", replay.Code, replay.Body)
	}
	r := call(s, "/api/v1/claims/"+id+"?action=request", map[string]string{})
	if r.Code != 429 {
		t.Fatal("restart reset resend limit", r.Code)
	}
	s.devConsole = true
	if r := call(s, "/api/v1/claims/"+id+"?action=release", map[string]string{"code": code}); r.Code != 503 {
		t.Fatal("console mode allowed funded secret release", r.Code)
	}
}

func TestStateDirectoryRejectsConcurrentWriter(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	close, err := lockPrivateState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := lockPrivateState(dir); err == nil {
		other()
		t.Fatal("concurrent state writer allowed")
	}
	close()
	again, err := lockPrivateState(dir)
	if err != nil {
		t.Fatal(err)
	}
	again()
}
