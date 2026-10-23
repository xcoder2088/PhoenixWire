package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testSender struct {
	code, to string
	ctx      context.Context // the context of the last send: what the code e-mail would be dressed with
}

func (s *testSender) Send(ctx context.Context, to, code string) error {
	s.to, s.code, s.ctx = to, code, ctx
	return nil
}

func request(s *server, id, action, code string) *httptest.ResponseRecorder {
	data, _ := json.Marshal(map[string]string{"code": code})
	req := httptest.NewRequest("POST", "/api/v1/claims/"+id+"?action="+action, bytes.NewReader(data))
	w := httptest.NewRecorder()
	s.handler(w, req)
	return w
}
func TestDemoOTPOnly(t *testing.T) {
	sender := &testSender{}
	s, e := newServer(sender, "recipient@example.test", false)
	if e != nil {
		t.Fatal(e)
	}
	if got := request(s, "missing", "request", ""); got.Code != 404 {
		t.Fatalf("unknown: %d", got.Code)
	}
	if got := request(s, "demo", "request", ""); got.Code != 200 {
		t.Fatalf("send: %d %s", got.Code, got.Body)
	} else {
		var response map[string]any
		if err := json.Unmarshal(got.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if delay, ok := response["dispatch_response_ms"].(float64); !ok || delay < 0 {
			t.Fatalf("missing service dispatch metric: %+v", response)
		}
	}
	if len(sender.code) != 6 || sender.to != "recipient@example.test" {
		t.Fatal("OTP recipient")
	}
	if got := request(s, "demo", "request", ""); got.Code != 429 {
		t.Fatal("rate limit", got.Code)
	}
	if got := request(s, "demo", "verify", "wrong"); got.Code != 403 {
		t.Fatal("accepted wrong code", got.Code)
	}
	if got := request(s, "demo", "verify", sender.code); got.Code != 200 {
		t.Fatal("verify", got.Code, got.Body)
	}
	status, _ := s.claims.Get("demo")
	if !status.EmailVerified || status.Status != "email_verified_unfunded" {
		t.Fatalf("not email verified: %+v", status)
	}
	if got := request(s, "demo", "verify", sender.code); got.Code != 409 {
		t.Fatal("replay", got.Code)
	}
	if got := request(s, "demo", "redeem", ""); got.Code != 404 {
		t.Fatal("payout should not exist", got.Code)
	}
}

func TestAuthenticatedDraftLifecycle(t *testing.T) {
	sender := &testSender{}
	s, e := newServer(sender, "recipient@example.test", false)
	if e != nil {
		t.Fatal(e)
	}
	s.creationKey = strings.Repeat("a", 64)
	create := func(auth, network, email string) *httptest.ResponseRecorder {
		data, _ := json.Marshal(map[string]string{"email": email, "amount": "3.25", "network": network})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/claims", bytes.NewReader(data))
		req.Header.Set("X-Claim-Service-Key", auth)
		w := httptest.NewRecorder()
		s.handler(w, req)
		return w
	}
	if got := create("wrong", "mainnet", "valid@example.com"); got.Code != 401 {
		t.Fatal("unauthorized", got.Code)
	}
	if got := create(s.creationKey, "testnet", "valid@example.com"); got.Code != 400 {
		t.Fatal("mainnet accepted", got.Code)
	}
	if got := create(s.creationKey, "mainnet", "X <valid@example.com>"); got.Code != 400 {
		t.Fatal("invalid email accepted", got.Code)
	}
	good := create(s.creationKey, "mainnet", "valid@example.com")
	if good.Code != 201 {
		t.Fatalf("create failed: %d %s", good.Code, good.Body)
	}
	var data map[string]any
	if e := json.Unmarshal(good.Body.Bytes(), &data); e != nil {
		t.Fatal(e)
	}
	claimView := data["claim"].(map[string]any)
	id := claimView["id"].(string)
	if len(id) != 48 || claimView["status"] != "draft_unfunded" || claimView["amount"] != "3.25" {
		t.Fatal("incorrect draft", data)
	}
	if strings.Contains(good.Body.String(), "valid@example.com") {
		t.Fatal("recipient leaked into response")
	}
	if got := request(s, id, "request", ""); got.Code != 503 {
		t.Fatal("unfunded claim sent OTP", got.Code)
	}
	if sender.to != "" {
		t.Fatal("unfunded OTP dispatched")
	}
	get := httptest.NewRecorder()
	s.handler(get, httptest.NewRequest(http.MethodGet, "/api/v1/claims/"+id, nil))
	if get.Code != 200 || !strings.Contains(get.Body.String(), "draft_unfunded") {
		t.Fatal("public draft state", get.Code, get.Body)
	}
	if got := request(s, id, "funded", ""); got.Code != 404 {
		t.Fatal("browser allowed to mark funding", got.Code)
	}
	if got := request(s, id, "wallet", ""); got.Code != 404 {
		t.Fatal("browser allowed wallet payout", got.Code)
	}
	if got := request(s, id, "redeem", ""); got.Code != 404 {
		t.Fatal("browser allowed payout", got.Code)
	}
}

func TestNoIssuerCreationWithoutConfiguredSecret(t *testing.T) {
	s, _ := newServer(&testSender{}, "recipient@example.test", true)
	req := httptest.NewRequest("POST", "/api/v1/claims", strings.NewReader(`{"email":"a@example.com","amount":"1","network":"mainnet"}`))
	w := httptest.NewRecorder()
	s.handler(w, req)
	if w.Code != 503 {
		t.Fatalf("unexpected unconfigured issuer status %d", w.Code)
	}
}

func TestMailerConfigurationReusesPhoenixMailEnvironment(t *testing.T) {
	t.Setenv("NIMIQ_CLAIM_MAILERSEND_API_KEY", "")
	t.Setenv("NIMIQ_CLAIM_MAILERSEND_FROM", "")
	t.Setenv("MAILERSEND_API_KEY", "unit-test-mailer-token")
	t.Setenv("MAILERSEND_FROM", "otp@example.test")
	if got := firstConfiguredEnv("NIMIQ_CLAIM_MAILERSEND_API_KEY", "MAILERSEND_API_KEY"); got != "unit-test-mailer-token" {
		t.Fatalf("generic MailerSend API key not reused: %q", got)
	}
	if got := firstConfiguredEnv("NIMIQ_CLAIM_MAILERSEND_FROM", "MAILERSEND_FROM"); got != "otp@example.test" {
		t.Fatalf("generic MailerSend sender not reused: %q", got)
	}
	t.Setenv("NIMIQ_CLAIM_MAILERSEND_API_KEY", "claim-specific-token")
	if got := firstConfiguredEnv("NIMIQ_CLAIM_MAILERSEND_API_KEY", "MAILERSEND_API_KEY"); got != "claim-specific-token" {
		t.Fatalf("claim-specific setting must take precedence: %q", got)
	}
}
