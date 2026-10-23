package web

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"

	claimweb "github.com/xcoder2088/NimiqSecureClaim/web"
)

// The recipient's side of a USDC / USDT card: the claim page is served from Wire's own origin (so the link key in its fragment
// never leaves the browser) with a Content-Security-Policy of its own, and the camera permission it needs to read a wallet QR code.

var claimPaths = map[string]bool{"/usdt.html": true, "/cashlink.js": true, "/jsqr.js": true, "/usdc-coin.png": true, "/usdt-coin.png": true, "/phoenixmail-official-emblem.webp": true}

var inlineScript = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

// claimCSP allows exactly the inline scripts the claim page ships (by hash) and no other inline code.
func claimCSP(page []byte) string {
	var hashes []string
	for _, m := range inlineScript.FindAllSubmatch(page, -1) {
		sum := sha256.Sum256(m[1])
		hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return "default-src 'self'; script-src 'self' " + strings.Join(hashes, " ") + "; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; media-src 'self' blob:; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'"
}

func serveClaimAssets(mux *http.ServeMux) {
	page, err := claimweb.Claim.ReadFile("usdt.html")
	if err != nil {
		panic(err)
	}
	csp := claimCSP(page)
	for name := range claimPaths {
		name = strings.TrimPrefix(name, "/")
		body, err := claimweb.Claim.ReadFile(name)
		if err != nil {
			panic(err)
		}
		kind := map[string]string{".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8", ".png": "image/png", ".webp": "image/webp"}[name[strings.LastIndex(name, "."):]]
		mux.HandleFunc("GET /"+name, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", kind)
			w.Header().Set("Content-Security-Policy", csp)
			w.Header().Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")
			w.Header().Set("Referrer-Policy", "no-referrer")
			_, _ = w.Write(body)
		})
	}
}
