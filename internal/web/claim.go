package web

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"

	claimweb "github.com/xcoder2088/NimiqSecureClaim/web"
)

// The recipient claim page reads and uses the link key locally; recipient API calls carry proofs and signatures.
// Separately, the sender submits the full link once for invitation delivery; Wire validates and e-mails it without persisting it.
// This page has its own Content-Security-Policy and camera permission for wallet QR codes.

var claimPaths = map[string]bool{"/proxycode.js": true, "/usdt.html": true, "/cashlink.js": true, "/jsqr.js": true, "/usdc-coin.png": true, "/usdt-coin.png": true, "/phoenixmail-official-emblem.webp": true}

var proxyCode = regexp.MustCompile(`PROXY_CREATION_CODE='(0x[0-9a-f]+)'`)

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
	// The Safe proxy creation code is the one constant the sender's page needs besides cashlink.js: it is taken from the claim page,
	// where it already lives, instead of being copied a second time.
	proxy := proxyCode.FindSubmatch(page)
	if proxy == nil {
		panic("proxy creation code not found in the claim page")
	}
	proxyJS := []byte("window.PHOENIX_PROXY_CREATION_CODE='" + string(proxy[1]) + "';\n")
	for name := range claimPaths {
		name = strings.TrimPrefix(name, "/")
		var body []byte
		if name == "proxycode.js" {
			body = proxyJS
		} else if body, err = claimweb.Claim.ReadFile(name); err != nil {
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
