package web

import (
	"context"
	"embed"
	"encoding/json"
	"github.com/xcoder2088/PhoenixWire/internal/wire"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed static
var assets embed.FS

func NewHandler(gateway ...*wire.API) http.Handler {
	var api *wire.API
	if len(gateway) > 0 {
		api = gateway[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if api != nil {
			writeJSON(w, 200, map[string]string{"status": "ok", "mode": "integrated"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "mode": "preview"})
	})
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		if api != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
			defer cancel()
			ready := api.Ready(ctx)
			assetList := []string{"NIM"}
			if api.USDT() != nil {
				assetList = append(assetList, "USDC_POLYGON", "USDT_POLYGON", "USDT_ETHEREUM")
			}
			writeJSON(w, 200, map[string]any{"mode": "integrated", "payments_enabled": ready, "email_enabled": true, "otp_enabled": true, "invite_only": api.InviteOnly(), "assets": assetList})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"mode": "preview", "payments_enabled": false, "email_enabled": false, "otp_enabled": false, "assets": []string{"NIM", "USDC_POLYGON", "USDT_POLYGON", "USDT_ETHEREUM"}})
	})
	// Fail closed. No seed, wallet, payment state, or delivery provider is loaded.
	if api != nil {
		for _, path := range []string{"/api/payments", "/api/payments/", "/api/sender/", "/api/session", "/api/logout", "/api/claims/", "/api/usdt/", "/api/gas", "/api/handoff", "/api/handoff/"} {
			mux.Handle(path, api.Handler())
		}
	} else {
		mux.HandleFunc("POST /api/payments", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"code": "payments_not_connected"})
		})
	}
	// USDC / USDT cards: the recipient's routes of the engine are public; its issuer routes never are (the gateway reaches them in-process).
	if api != nil && api.USDT() != nil {
		usdt := api.USDT().USDTHandler()
		mux.HandleFunc("/api/v1/usdt/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/usdt/issuer") {
				http.NotFound(w, r)
				return
			}
			usdt.ServeHTTP(w, r)
		})
		serveClaimAssets(mux)
	}
	files, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	static := http.FileServer(http.FS(files))
	// No method in the pattern: "GET /" would conflict with the method-less API patterns above (Go 1.22 mux) and stop the integrated server.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// An explicit allowlist prevents directory listings and SPA fallbacks on unknown APIs.
		// The NIM claim link (/?claim=…) still opens the earlier page, which knows how to claim; every other visit gets PhoenixMail's payment window.
		if r.URL.Path == "/" && r.URL.Query().Has("claim") {
			page, err := fs.ReadFile(files, "classic.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(page)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/pm/") && payFiles[strings.TrimPrefix(r.URL.Path, "/pm/")] {
			static.ServeHTTP(w, r)
			return
		}
		switch r.URL.Path {
		case "/site.webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
			static.ServeHTTP(w, r)
		case "/favicon.ico", "/brand/wire-icon-32.png", "/brand/wire-icon-180.png", "/brand/wire-icon-192.png", "/brand/wire-icon-512.png", "/", "/classic.html", "/wire-pay.js", "/wire-pay.css", "/brand/usdt-coin.png", "/brand/phoenixmail-approved-icon-transparent.webp", "/app.js", "/flow.js", "/flow-text.js", "/model.js", "/card-mirror.js", "/hub.js", "/iqons.js", "/style.css", "/flow.css", "/design.css", "/usdt-flow.js", "/usdt-text.js", "/qrcode.js", "/brand/phoenix-wire.webp", "/brand/phoenix-emblem.webp", "/brand/nimiq-logo.svg", "/brand/usdt.svg", "/brand/usdc-coin.png":
			static.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !claimPaths[r.URL.Path] { // the claim page brings its own policy and camera permission (claim.go)
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: blob:; connect-src 'self'; frame-src https://hub.nimiq.com; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

// payFiles are PhoenixMail's payment window, mirrored unchanged under /pm/.
var payFiles = map[string]bool{"app.css": true, "nimiq-payments.css": true, "theme-light.css": true, "nimiq-hub.js": true, "phoenix-nimiq.js": true, "nimiq-iqons.js": true, "usdt-cashlink.js": true, "usdt-payments.js": true, "nimiq-payments.js": true}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
