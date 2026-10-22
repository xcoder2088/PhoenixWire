package web

import (
	"context"
	"embed"
	"encoding/json"
	"github.com/xcoder2088/PhoenixWire/internal/wire"
	"io/fs"
	"net/http"
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
			writeJSON(w, 200, map[string]any{"mode": "integrated", "payments_enabled": ready, "email_enabled": true, "otp_enabled": true, "assets": []string{"NIM"}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"mode": "preview", "payments_enabled": false, "email_enabled": false, "otp_enabled": false, "assets": []string{"NIM", "USDT_ETHEREUM", "USDT0_POLYGON"}})
	})
	// Fail closed. No seed, wallet, payment state, or delivery provider is loaded.
	if api != nil {
		for _, path := range []string{"/api/payments", "/api/payments/", "/api/sender/", "/api/session", "/api/logout", "/api/claims/"} {
			mux.Handle(path, api.Handler())
		}
	} else {
		mux.HandleFunc("POST /api/payments", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"code": "payments_not_connected"})
		})
	}
	files, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	static := http.FileServer(http.FS(files))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// An explicit allowlist prevents directory listings and SPA fallbacks on unknown APIs.
		switch r.URL.Path {
		case "/", "/app.js", "/flow.js", "/flow-text.js", "/model.js", "/card-mirror.js", "/hub.js", "/iqons.js", "/style.css", "/flow.css", "/design.css", "/brand/phoenix-wire.webp", "/brand/phoenix-emblem.webp", "/brand/nimiq-logo.svg", "/brand/usdt.svg":
			static.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: blob:; connect-src 'self'; frame-src https://hub.nimiq.com; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
