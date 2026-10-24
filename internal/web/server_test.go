package web

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
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

func TestShortcutAssetsKeepThePagePolicy(t *testing.T) {
	h := NewHandler()
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest("GET", "/", nil))
	policy := page.Header().Get("Content-Security-Policy")
	for path, size := range map[string]int{"/brand/wire-icon-32.png": 32, "/brand/wire-icon-180.png": 180, "/brand/wire-icon-192.png": 192, "/brand/wire-icon-512.png": 512, "/favicon.ico": 0, "/site.webmanifest": 0} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRecorder()
			h.ServeHTTP(r, httptest.NewRequest("GET", path, nil))
			if r.Code != 200 || r.Body.Len() == 0 || r.Header().Get("Content-Security-Policy") != policy {
				t.Fatalf("asset unavailable or policy changed: %s %d", path, r.Code)
			}
			if size > 0 {
				config, err := png.DecodeConfig(bytes.NewReader(r.Body.Bytes()))
				if err != nil || config.Width != size || config.Height != size {
					t.Fatalf("invalid icon dimensions: %s %+v %v", path, config, err)
				}
			}
			if path == "/favicon.ico" && !bytes.HasPrefix(r.Body.Bytes(), []byte{0, 0, 1, 0}) {
				t.Fatal("invalid favicon")
			}
			if path == "/site.webmanifest" {
				if r.Header().Get("Content-Type") != "application/manifest+json" {
					t.Fatal("incorrect manifest content type")
				}
				var manifest struct {
					Name  string `json:"name"`
					Start string `json:"start_url"`
				}
				if err := json.Unmarshal(r.Body.Bytes(), &manifest); err != nil || manifest.Name != "Phoenix Wire" || manifest.Start != "/" {
					t.Fatal("invalid shortcut manifest")
				}
			}
		})
	}
	for _, path := range []string{"/", "/classic.html", "/?claim=fixture"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		for _, text := range []string{"<title>Phoenix Wire</title>", "rel=\"apple-touch-icon\"", "rel=\"manifest\"", "/favicon.ico?v=0.4.1"} {
			if !strings.Contains(r.Body.String(), text) {
				t.Fatalf("%s missing %s", path, text)
			}
		}
	}
}

func TestPageBrandingAndNoClaimableContent(t *testing.T) {
	r := httptest.NewRecorder()
	NewHandler().ServeHTTP(r, httptest.NewRequest("GET", "/", nil))
	for _, text := range []string{"Powered by", "Phoenix Mail", "Phoenix Wire", "type=\"email\"", "/pm/nimiq-payments.js"} {
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

func TestTheNimClaimLinkStillOpensTheEarlierPage(t *testing.T) {
	h := NewHandler()
	get := func(path string) string {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		return r.Body.String()
	}
	if claim := get("/?claim=" + strings.Repeat("a", 48)); !strings.Contains(claim, "Protégé") || strings.Contains(claim, "/pm/nimiq-payments.js") {
		t.Fatal("a NIM claim link must open the page that knows how to claim")
	}
	if !strings.Contains(get("/classic.html"), "Protégé") {
		t.Fatal("the earlier page is kept for NIM")
	}
}

func TestPaymentWindowFilesAreMirroredAndServed(t *testing.T) {
	h := NewHandler()
	for path := range payFiles {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", "/pm/"+path, nil))
		if r.Code != 200 || r.Body.Len() < 100 {
			t.Fatalf("/pm/%s: %d", path, r.Code)
		}
	}
	if r := httptest.NewRecorder(); func() int { h.ServeHTTP(r, httptest.NewRequest("GET", "/pm/LICENSE-NIMIQ-MODULE", nil)); return r.Code }() == 200 {
		t.Fatal("only the window's own files are served")
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

// What a recipient sees is Wire's, and every card of Wire has the one gradient (the card colour, then #e2eeff, then #ffe8f4).
func TestClaimPageWearsWireBrandAndTheCommonGradient(t *testing.T) {
	page, err := claimweb.Claim.ReadFile("usdt.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(page)
	for _, must := range []string{"Phoenix<strong>Wire</strong>", "/brand/phoenix-emblem.webp", `alt="Phoenix Wire"`, "<title>Phoenix Wire</title>", "--card-mid:#e2eeff", "--card-end:#ffe8f4",
		"linear-gradient(125deg,var(--card-start) 0%,var(--card-mid) 56%,var(--card-end) 100%)"} {
		if !strings.Contains(text, must) {
			t.Fatalf("claim page: missing %q", must)
		}
	}
	for _, mustNot := range []string{"Phoenix<strong>Mail</strong>", `alt="PhoenixMail"`, "phoenixmail-official-emblem.webp\"", "radial-gradient(ellipse at 4% 8%", "mx(p[0]"} {
		if strings.Contains(text, mustNot) {
			t.Fatalf("claim page still has %q", mustNot)
		}
	}
}

// The gas light is shown before anyone signs in: the web server hands /api/gas to the gateway, without a session.
func TestGasLightRouteIsReachableWithoutASession(t *testing.T) {
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
	r := httptest.NewRecorder()
	NewHandler(api).ServeHTTP(r, httptest.NewRequest("GET", "/api/gas", nil))
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"polygon"`) || strings.Contains(r.Body.String(), "sender_verification_required") {
		t.Fatalf("/api/gas must answer without a session: %d %s", r.Code, r.Body)
	}
}

// The hand-over routes are reachable through the web server (the redeem needs no session: the other browser has none).
func TestHandoffRoutesAreReachable(t *testing.T) {
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
	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://wire.example.test")
		r := httptest.NewRecorder()
		NewHandler(api).ServeHTTP(r, req)
		return r
	}
	if r := post("/api/handoff/redeem", `{"token":"`+strings.Repeat("A", 43)+`"}`); r.Code != 404 || !strings.Contains(r.Body.String(), "invalid_or_expired") {
		t.Fatalf("redeem must reach the gateway: %d %s", r.Code, r.Body)
	}
	if r := post("/api/admin/access", `{"open":true}`); r.Code != 401 { // the administrator's panel is reached through the gateway and asks for a verified sender
		t.Fatal("the admin route is not served by the gateway", r.Code)
	}
	if r := post("/api/handoff", `{"recipient":"a@b.co","amount":"1"}`); r.Code != 401 {
		t.Fatalf("a token needs a verified sender: %d %s", r.Code, r.Body)
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
	proxy := get("/proxycode.js")
	if proxy.Code != 200 || !strings.HasPrefix(proxy.Body.String(), "window.PHOENIX_PROXY_CREATION_CODE='0x") || !strings.Contains(proxy.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("proxy code for the sender's browser: %d %q", proxy.Code, proxy.Body.String()[:20])
	}
	for _, path := range []string{"/usdt-flow.js", "/usdt-text.js", "/qrcode.js", "/brand/usdc-coin.png"} {
		if r := get(path); r.Code != 200 {
			t.Fatalf("%s: %d", path, r.Code)
		}
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

// Every script, style and image a page names must be served: a file added to a page and not to the server's list breaks the page for everyone.
func TestEveryFileAPageNamesIsServed(t *testing.T) {
	h := NewHandler()
	refs := regexp.MustCompile(`(?:src|href)="(/?[A-Za-z0-9_./-]+\.(?:js|css|png|webp|svg|ico|webmanifest))(?:\?[^"]*)?"`)
	for _, page := range []string{"/", "/classic.html"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", page, nil))
		if r.Code != 200 {
			t.Fatalf("%s: %d", page, r.Code)
		}
		found := refs.FindAllStringSubmatch(r.Body.String(), -1)
		if len(found) < 3 {
			t.Fatalf("%s names too few files (%d): the test is not reading the page", page, len(found))
		}
		for _, m := range found {
			path := m[1]
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			f := httptest.NewRecorder()
			h.ServeHTTP(f, httptest.NewRequest("GET", path, nil))
			if f.Code != 200 {
				t.Errorf("%s names %s, which the server answers with %d", page, path, f.Code)
			}
		}
	}
}

// A file that is shipped with the page (the top of the static folder) is served. A new file that is not on the server's list answers 404 and breaks
// whatever loads it, including the files a script loads by itself, which no page names.
func TestEveryShippedFileIsServed(t *testing.T) {
	h := NewHandler()
	entries, err := assets.ReadDir("static")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), "LICENSE") || e.Name() == "index.html" { // index.html is served at "/"
			continue
		}
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", "/"+e.Name(), nil))
		if r.Code != 200 {
			t.Errorf("/%s is shipped with the page and the server answers %d", e.Name(), r.Code)
		}
	}
}
