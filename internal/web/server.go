package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
)

//go:embed static
var assets embed.FS

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "mode": "preview"})
	})
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"mode": "preview", "payments_enabled": false, "email_enabled": false, "otp_enabled": false, "assets": []string{"NIM", "USDT_ETHEREUM", "USDT0_POLYGON"}})
	})
	// Fail closed. No seed, wallet, payment state, or delivery provider is loaded.
	mux.HandleFunc("POST /api/payments", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"code": "payments_not_connected"})
	})
	files, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	static := http.FileServer(http.FS(files))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// An explicit allowlist prevents directory listings and SPA fallbacks on unknown APIs.
		switch r.URL.Path {
		case "/", "/app.js", "/model.js", "/style.css", "/brand/phoenix-wire.webp":
			static.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'")
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
