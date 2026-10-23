package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/service"
	claimweb "github.com/xcoder2088/NimiqSecureClaim/web"
	"github.com/xcoder2088/PhoenixWire/internal/wire"
)

func TestCapabilitiesFailClosed(t *testing.T) {
	r := httptest.NewRecorder()
	NewHandler().ServeHTTP(r, httptest.NewRequest("GET", "/api/config", nil))
	var config struct {
		Mode     string `json:"mode"`
		Payments bool   `json:"payments_enabled"`
		Email    bool   `json:"email_enabled"`
		OTP      bool   `json:"otp_enabled"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if r.Code != 200 || config.Mode != "preview" || config.Payments || config.Email || config.OTP {
		t.Fatalf("unsafe config: %s", r.Body.String())
	}
	r = httptest.NewRecorder()
	NewHandler().ServeHTTP(r, httptest.NewRequest("POST", "/api/payments", strings.NewReader(`{"amount":"25"}`)))
	if r.Code != http.StatusServiceUnavailable || !strings.Contains(r.Body.String(), "payments_not_connected") {
		t.Fatalf("payment should be unavailable: %d %s", r.Code, r.Body.String())
	}
}

func TestRoutesAndHeaders(t *testing.T) {
	for path, status := range map[string]int{"/": 200, "/api/health": 200, "/app.js": 200, "/model.js": 200, "/style.css": 200, "/brand/phoenix-wire.webp": 200, "/brand/": 404, "/api/unknown": 404, "/.env": 404, "/LICENSE": 404} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRecorder()
			NewHandler().ServeHTTP(r, httptest.NewRequest("GET", path, nil))
			if r.Code != status {
				t.Fatalf("got %d want %d", r.Code, status)
			}
			if !strings.Contains(r.Header().Get("Content-Security-Policy"), "form-action 'none'") || r.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing safety headers")
			}
		})
	}
}

func TestPageBrandingAndNoClaimableContent(t *testing.T) {
	r := httptest.NewRecorder()
	NewHandler().ServeHTTP(r, httptest.NewRequest("GET", "/", nil))
	for _, text := range []string{"Powered by", "Phoenix Mail", "Phoenix Wire", "type=\"email\"", "Protégé"} {
		if !strings.Contains(r.Body.String(), text) {
			t.Fatalf("missing %q", text)
		}
	}
	for _, text := range []string{"cashlink#", "nimiq://", "seed", "data:image"} {
		if strings.Contains(r.Body.String(), text) {
			t.Fatalf("unexpected claim material %q", text)
		}
	}
}

func TestPreviewServesNoClaimPageAndNoUSDTRoutes(t *testing.T) {
	for _, path := range []string{"/usdt.html", "/cashlink.js", "/jsqr.js", "/api/v1/usdt/cards/00112233445566778899aabbccddeeff", "/api/v1/usdt/issuer/relayers"} {
		r := httptest.NewRecorder()
		NewHandler().ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		if r.Code != http.StatusNotFound {
			t.Fatalf("%s must not exist in preview: %d", path, r.Code)
		}
	}
}

func TestClaimPolicyAllowsOnlyTheInlineScriptsThePageShips(t *testing.T) {
	page, err := claimweb.Claim.ReadFile("usdt.html")
	if err != nil {
		t.Fatal(err)
	}
	csp := claimCSP(page)
	scripts := len(inlineScript.FindAll(page, -1))
	if scripts == 0 || strings.Count(csp, "'sha256-") != scripts {
		t.Fatalf("one hash per inline script (%d): %s", scripts, csp)
	}
	if strings.Contains(strings.SplitN(strings.SplitN(csp, "script-src", 2)[1], ";", 2)[0], "unsafe-inline") {
		t.Fatal("inline scripts must be allowed by hash only")
	}
	if !strings.Contains(csp, "form-action 'none'") || !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatal("the claim page keeps Wire's other restrictions")
	}
}

func TestUSDTRoutesArePublicOnlyForTheRecipient(t *testing.T) {
	t.Setenv("NIMIQ_CLAIM_USDTLINK_CHAINS", "polygon-usdc")
	t.Setenv("NIMIQ_CLAIM_ALLOWED_ORIGINS", "https://wire.example.test")
	dir := t.TempDir()
	mail := wire.MailerSend{APIKey: "fixture", From: "no-reply@example.test"}
	engine, err := service.New(service.Config{Directory: filepath.Join(dir, "engine"), RPCURL: "http://127.0.0.1:1", Master: bytes.Repeat([]byte{7}, 32), Sender: service.MailerSend{APIKey: mail.APIKey, From: mail.From}})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	api, err := wire.New(wire.Config{Directory: filepath.Join(dir, "gateway"), Origin: "https://wire.example.test", Master: bytes.Repeat([]byte{9}, 32), Engine: engine, Mail: mail})
	if err != nil {
		t.Fatal(err)
	}
	defer api.Close()
	if api.USDT() == nil {
		t.Fatal("the USDC engine did not start")
	}
	h := NewHandler(api)
	get := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		return r
	}
	page := get("/usdt.html")
	if page.Code != 200 || !strings.Contains(page.Header().Get("Permissions-Policy"), "camera=(self)") || !strings.Contains(page.Header().Get("Content-Security-Policy"), "'sha256-") {
		t.Fatalf("claim page: %d %v", page.Code, page.Header())
	}
	if other := get("/"); !strings.Contains(other.Header().Get("Permissions-Policy"), "camera=()") {
		t.Fatal("the camera stays off everywhere else")
	}
	for _, path := range []string{"/cashlink.js", "/jsqr.js", "/usdc-coin.png"} {
		if r := get(path); r.Code != 200 {
			t.Fatalf("%s: %d", path, r.Code)
		}
	}
	// the issuer routes are never reachable from outside, with or without any key
	for _, path := range []string{"/api/v1/usdt/issuer/cards", "/api/v1/usdt/issuer/relayers", "/api/v1/usdt/issuer/gas-status"} {
		r := httptest.NewRecorder()
		req := httptest.NewRequest("POST", path, strings.NewReader("{}"))
		req.Header.Set("X-Claim-Service-Key", strings.Repeat("a", 64))
		h.ServeHTTP(r, req)
		if r.Code != http.StatusNotFound {
			t.Fatalf("%s must not be public: %d", path, r.Code)
		}
	}
	// the recipient's routes answer (an unknown card is a plain 404 from the engine, not from the router)
	if r := get("/api/v1/usdt/cards/00112233445566778899aabbccddeeff"); r.Code != 404 || !strings.Contains(r.Body.String(), "card_not_found") {
		t.Fatalf("recipient route: %d %s", r.Code, r.Body.String())
	}
	// and the gateway reaches the issuer routes in-process
	status, body, err := api.USDT().IssuerCall(context.Background(), "POST", "/api/v1/usdt/issuer/relayers", []byte("{}"))
	if err != nil || status != 200 && status != 503 {
		t.Fatalf("in-process issuer call: %d %s %v", status, body, err)
	}
}
