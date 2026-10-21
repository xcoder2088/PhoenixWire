package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

func TestExistingCashlinkOTPReleaseWithoutNewPaymentEngine(t *testing.T) {
	sender := &testSender{}
	s, err := newServer(sender, "recipient@example.test", false)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	os.Chmod(directory, 0700)
	s.claims, err = claim.NewPersistentStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	s.otp, err = otp.NewPersistent(sender, directory, bytes.Repeat([]byte{0x35}, 32))
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.claims.Create("receiver@example.test", "3.25", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	master := bytes.Repeat([]byte{0x35}, 32)
	v, err := vault.New(master, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "private-vault")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := vault.NewFileStore(root, v)
	if err != nil {
		t.Fatal(err)
	}
	s.cashlinkVault = store
	if got := request(s, c.ID, "request", ""); got.Code != 409 {
		t.Fatal("unfunded OTP accepted", got.Code)
	}
	seed := bytes.Repeat([]byte{0x19}, 32)
	if err := store.Put(c.ID, append(append([]byte(nil), seed...), append([]byte{5}, []byte("Salut")...)...)); err != nil {
		t.Fatal(err)
	}
	// Opening or scanning a public claim never returns spending material.
	public := httptest.NewRecorder()
	s.handler(public, httptest.NewRequest(http.MethodGet, "/api/v1/claims/"+c.ID, nil))
	var presentation struct {
		Address  string `json:"address"`
		Status   string `json:"status"`
		Cashlink string `json:"cashlink"`
	}
	if public.Code != 200 || json.Unmarshal(public.Body.Bytes(), &presentation) != nil {
		t.Fatal("public presentation unavailable", public.Code)
	}
	address, _ := (nimiq.AddressDeriver{}).CashlinkAddress(seed)
	if presentation.Address != address || presentation.Status != "otp_required" || presentation.Cashlink != "" {
		t.Fatal("wrong public presentation")
	}
	if strings.Contains(public.Body.String(), base64.RawURLEncoding.EncodeToString(seed)) || strings.Contains(public.Body.String(), "hub.nimiq") {
		t.Fatal("seed or spending URL exposed before OTP")
	}
	viaGet := httptest.NewRecorder()
	s.handler(viaGet, httptest.NewRequest(http.MethodGet, "/api/v1/claims/"+c.ID+"?action=release", nil))
	if viaGet.Code != 400 || strings.Contains(viaGet.Body.String(), "cashlink/#") {
		t.Fatal("public GET bypassed OTP")
	}
	if got := request(s, c.ID, "release", ""); got.Code != 403 || strings.Contains(got.Body.String(), "cashlink/#") {
		t.Fatal("empty code bypassed OTP")
	}
	if got := request(s, c.ID, "request", ""); got.Code != 200 {
		t.Fatal("OTP request", got.Code, got.Body)
	}
	if sender.to != "receiver@example.test" || len(sender.code) != 6 {
		t.Fatal("OTP recipient or code")
	}
	bad := "000000"
	if sender.code == bad {
		bad = "111111"
	}
	if got := request(s, c.ID, "release", bad); got.Code != 403 {
		t.Fatal("wrong code accepted", got.Code)
	}
	good := request(s, c.ID, "release", sender.code)
	if good.Code != 200 {
		t.Fatal("OTP release", good.Code, good.Body)
	}
	var result struct {
		Cashlink string `json:"cashlink"`
	}
	if json.Unmarshal(good.Body.Bytes(), &result) != nil {
		t.Fatal("invalid response")
	}
	payload := append([]byte(nil), seed...)
	payload = binary.BigEndian.AppendUint64(payload, 325000)
	payload = append(payload, 5)
	payload = append(payload, []byte("Salut")...)
	expected := "https://hub.nimiq.com/cashlink/#" + base64.URLEncoding.EncodeToString(payload)
	if result.Cashlink != expected {
		t.Fatal("original Cashlink format changed")
	}
	if got := request(s, c.ID, "release", sender.code); got.Code != 409 {
		t.Fatal("OTP replay succeeded", got.Code)
	}
	if got := request(s, c.ID, "verify", sender.code); got.Code != 404 {
		t.Fatal("alternate OTP replay succeeded", got.Code)
	}
}

func TestReleaseVerifiesOTPBeforeOpeningVault(t *testing.T) {
	sender := &testSender{}
	s, err := newServer(sender, "", false)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	s.claims, err = claim.NewPersistentStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	s.otp, err = otp.NewPersistent(sender, directory, bytes.Repeat([]byte{0x41}, 32))
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.claims.Create("receiver@example.test", "1", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	key, err := vault.New(bytes.Repeat([]byte{0x42}, 32), "test-key")
	if err != nil {
		t.Fatal(err)
	}
	store, err := vault.NewFileStore(root, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(c.ID, bytes.Repeat([]byte{0x43}, 32)); err != nil {
		t.Fatal(err)
	}
	// The ciphertext exists, but this key cannot decrypt it. An OTP failure
	// must still return its normal error rather than reaching vault.Open.
	wrongKey, err := vault.New(bytes.Repeat([]byte{0x44}, 32), "test-key")
	if err != nil {
		t.Fatal(err)
	}
	s.cashlinkVault, err = vault.NewFileStore(root, wrongKey)
	if err != nil {
		t.Fatal(err)
	}
	if got := request(s, c.ID, "request", ""); got.Code != 200 {
		t.Fatal("code request failed", got.Code)
	}
	bad := "000000"
	if sender.code == bad {
		bad = "111111"
	}
	for _, code := range []string{"", bad} {
		got := request(s, c.ID, "release", code)
		if got.Code != 403 || !strings.Contains(got.Body.String(), `"error":"invalid_or_expired_code"`) {
			t.Fatal("unauthenticated attempt reached the unreadable vault", got.Code)
		}
	}
	got := request(s, c.ID, "release", sender.code)
	if got.Code != 503 || !strings.Contains(got.Body.String(), `"error":"cashlink_unavailable"`) {
		t.Fatal("verified attempt did not reach the vault failure", got.Code)
	}
	view, err := s.claims.Get(c.ID)
	if err != nil || !view.EmailVerified {
		t.Fatal("claim verification must be persisted before opening the vault", err)
	}
	if got := request(s, c.ID, "release", sender.code); got.Code != 409 || !strings.Contains(got.Body.String(), `"error":"invalid_or_expired_code"`) {
		t.Fatal("consumed OTP reached the unreadable vault", got.Code)
	}
}

func TestOTPSealRequiresIssuerAndConfirmedFunding(t *testing.T) {
	s, _ := newServer(&testSender{}, "recipient@example.test", false)
	c, err := s.claims.Create("receiver@example.test", "1", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	s.creationKey = strings.Repeat("a", 64)
	body, _ := json.Marshal(map[string]string{"seed": base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)), "address": "NQ"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/claims/"+c.ID+"?action=seal", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handler(rec, req)
	if rec.Code != 401 {
		t.Fatal("unauthorized seal accepted", rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/claims/"+c.ID+"?action=seal", bytes.NewReader(body))
	req.Header.Set("X-Claim-Service-Key", s.creationKey)
	rec = httptest.NewRecorder()
	s.handler(rec, req)
	if rec.Code != 503 {
		t.Fatal("unconfigured escrow accepted", rec.Code)
	}
}
