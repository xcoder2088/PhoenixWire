package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
