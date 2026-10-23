// NimiqSecureClaim local server: durable email OTP for the original Nimiq Cashlink.
package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/escrow"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evmcard"
	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

func firstConfiguredEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

type server struct {
	claims        *claim.Store
	otp           *otp.Manager
	devConsole    bool
	creationKey   string
	escrow        *escrow.Admission
	cashlinkVault *vault.FileStore
	chainReady    func(context.Context) bool
	evm           *evmcard.Service
	usdt          *usdtHTTP // USDT cashlinks (usdtlink_http.go); nil unless NIMIQ_CLAIM_USDTLINK_CHAINS is set
}

func newServer(sender otp.Sender, email string, devConsole bool) (*server, error) {
	manager, err := otp.New(sender)
	if err != nil {
		return nil, err
	}
	claims := claim.NewStore()
	if email != "" {
		if err := claims.SeedDemo(email); err != nil {
			return nil, err
		}
	}
	return &server{claims: claims, otp: manager, devConsole: devConsole}, nil
}

func write(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("trailing_data")
	}
	return nil
}

func statusFromError(err error) int {
	switch {
	case errors.Is(err, claim.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, claim.ErrExpired):
		return http.StatusGone
	case errors.Is(err, claim.ErrCapacity):
		return http.StatusTooManyRequests
	default:
		return http.StatusBadRequest
	}
}

// The issuer API is designed for a future server-to-server PhoenixMail adapter.
// It must never accept anonymous internet-origin creation or client-provided
// claims of blockchain funding. This build binds to loopback and has no payout.
func (s *server) create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		write(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	if len(s.creationKey) < 32 {
		write(w, 503, map[string]string{"error": "issuer_not_configured"})
		return
	}
	supplied := r.Header.Get("X-Claim-Service-Key")
	if len(supplied) != len(s.creationKey) || subtle.ConstantTimeCompare([]byte(supplied), []byte(s.creationKey)) != 1 {
		write(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	var input struct {
		Email     string `json:"email"`
		Amount    string `json:"amount"`
		Network   string `json:"network"`
		RequestID string `json:"request_id"`
	}
	if err := decode(w, r, &input); err != nil {
		write(w, 400, map[string]string{"error": "invalid_json"})
		return
	}
	var created claim.Public
	var err error
	if input.RequestID != "" {
		if len(input.RequestID) > 128 {
			write(w, 400, map[string]string{"error": "invalid_request_id"})
			return
		}
		h := hmac.New(sha256.New, []byte(s.creationKey))
		h.Write([]byte("PhoenixMail cashlink claim v1\x00"))
		h.Write([]byte(input.RequestID))
		created, err = s.claims.CreateOnce(hex.EncodeToString(h.Sum(nil)[:24]), input.Email, input.Amount, input.Network)
	} else {
		created, err = s.claims.Create(input.Email, input.Amount, input.Network)
	}
	if err != nil {
		write(w, statusFromError(err), map[string]string{"error": err.Error()})
		return
	}
	write(w, http.StatusCreated, map[string]any{"claim": created, "claim_path": "/?claim=" + created.ID, "funding": "not_implemented"})
}

// Public, non-secret capability report used by PhoenixMail before it offers
// a funded OTP transfer. A configured issuer alone is not sufficient:
// encrypted vault and independent Mainnet chain admission must both exist.
func (s *server) capability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		write(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	chainReady := false
	if s.chainReady != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		chainReady = s.chainReady(ctx)
		cancel()
	}
	ready := len(s.creationKey) >= 32 && s.escrow != nil && s.cashlinkVault != nil && s.otp.Durable() && s.claims.Durable() && !s.devConsole && chainReady
	delivery := "email"
	if s.devConsole {
		delivery = "dev_console"
	}
	write(w, http.StatusOK, map[string]any{
		"service": "nimiq-secureclaim", "version": 1, "network": "mainnet",
		"issuer_ready": len(s.creationKey) >= 32,
		"vault_ready":  s.cashlinkVault != nil && s.escrow != nil,
		"otp_delivery": delivery, "protected_cashlink_ready": ready,
		"mainnet_enabled":       ready,
		"chain_ready":           chainReady,
		"otp_persistence_ready": s.otp.Durable(), "claims_persistence_ready": s.claims.Durable(),
		"usdt_ready": s.evm != nil && len(s.creationKey) >= 32, "usdt_protected_ready": s.evm != nil && len(s.creationKey) >= 32 && s.otp.Durable() && !s.devConsole,
		"usdt_assets": evm.Assets(), "usdt_cashlink_ready": s.usdt != nil, "usdt_cashlink_float_low": s.usdt != nil && anyTrue(s.usdt.svc.FloatStatus(r.Context())),
	})
}

func (s *server) handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/health" {
		s.capability(w, r)
		return
	}
	if r.URL.Path == "/api/v1/claims" {
		s.create(w, r)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "claims" || parts[3] == "" {
		write(w, 404, map[string]string{"error": "not_found"})
		return
	}
	id := parts[3]
	if s.evm != nil && s.evm.Has(id) {
		s.evmClaim(w, r, id)
		return
	}
	view, err := s.claims.Get(id)
	if err != nil {
		write(w, statusFromError(err), map[string]string{"error": err.Error()})
		return
	}
	if r.Method == http.MethodGet {
		if r.URL.Query().Get("action") != "" {
			write(w, 400, map[string]string{"error": "invalid_action"})
			return
		}
		// Publish only the derived public address for the original Cashlink's
		// Nimicon. Neither the spending seed nor its bearer URL is serialized.
		response := struct {
			claim.Public
			Address string `json:"address,omitempty"`
		}{Public: view}
		if id != "demo" && s.cashlinkVault != nil && s.cashlinkVault.Has(id) {
			secret, openErr := s.cashlinkVault.Open(id)
			if openErr != nil || len(secret) < 32 {
				clear(secret)
				write(w, 503, map[string]string{"error": "cashlink_unavailable"})
				return
			}
			address, deriveErr := (nimiq.AddressDeriver{}).CashlinkAddress(secret[:32])
			clear(secret)
			if deriveErr != nil {
				write(w, 503, map[string]string{"error": "cashlink_unavailable"})
				return
			}
			response.Address = address
			view.Status = "otp_required"
			if view.EmailVerified {
				view.Status = "otp_released"
			}
		}
		response.Public = view
		write(w, 200, response)
		return
	}
	if r.Method != http.MethodPost {
		write(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.otpClaimAction(w, r, id, r.URL.Query().Get("action")) {
		return
	}
	switch r.URL.Query().Get("action") {
	case "request":
		if id != "demo" && (s.devConsole || !s.otp.Durable() || !s.claims.Durable()) {
			write(w, 503, map[string]string{"error": "durable_email_otp_required"})
			return
		}
		if id != "demo" && (s.cashlinkVault == nil || !s.cashlinkVault.Has(id)) {
			write(w, 409, map[string]string{"error": "cashlink_not_funded"})
			return
		}
		if id == "demo" && view.EmailVerified {
			write(w, 409, map[string]string{"error": "already_verified"})
			return
		}
		recipient, err := s.claims.Recipient(id)
		if err != nil {
			write(w, statusFromError(err), map[string]string{"error": err.Error()})
			return
		}
		dispatchStart := time.Now()
		issue := s.otp.IssueClaim
		if id == "demo" {
			issue = s.otp.Issue
		}
		if err := issue(r.Context(), id, recipient); err != nil {
			if errors.Is(err, otp.ErrRateLimited) {
				write(w, 429, map[string]string{"error": "wait_before_resend"})
				return
			}
			if errors.Is(err, otp.ErrAlreadyVerified) {
				write(w, 409, map[string]string{"error": "already_verified"})
				return
			}
			log.Print("OTP delivery unavailable")
			write(w, 502, map[string]string{"error": "otp_delivery_unavailable"})
			return
		}
		status := "sent"
		if s.devConsole {
			status = "sent_dev_console"
		}
		write(w, 200, map[string]any{"status": status, "dispatch_response_ms": time.Since(dispatchStart).Milliseconds()})
	case "verify":
		// Real Cashlinks use the atomic OTP release action; legacy verify is demo-only.
		if id != "demo" {
			write(w, 404, map[string]string{"error": "unknown_action"})
			return
		}
		if view.EmailVerified {
			write(w, 409, map[string]string{"error": "already_verified"})
			return
		}
		var input struct {
			Code string `json:"code"`
		}
		if err := decode(w, r, &input); err != nil {
			write(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		recipient, err := s.claims.Recipient(id)
		if err != nil {
			write(w, statusFromError(err), map[string]string{"error": err.Error()})
			return
		}
		if err := s.otp.Verify(id, recipient, input.Code); err != nil {
			write(w, 403, map[string]string{"error": "invalid_or_expired_code"})
			return
		}
		verified, err := s.claims.VerifyEmail(id)
		if err != nil {
			write(w, statusFromError(err), map[string]string{"error": err.Error()})
			return
		}
		write(w, 200, map[string]any{"status": "email_verified_unfunded", "claim": verified, "note": "Email verified. No NIM have been funded; wallet payout is disabled."})
	default:
		// No funding, wallet redemption, cashlink seed, or chain broadcast endpoint.
		write(w, 404, map[string]string{"error": "unknown_action"})
	}
}

func LegacyMain() {
	key := firstConfiguredEnv("NIMIQ_CLAIM_MAILERSEND_API_KEY", "MAILERSEND_API_KEY")
	from := firstConfiguredEnv("NIMIQ_CLAIM_MAILERSEND_FROM", "MAILERSEND_FROM")
	email := strings.TrimSpace(os.Getenv("NIMIQ_CLAIM_DEMO_EMAIL"))
	if (key == "") != (from == "") {
		log.Fatal("Set BOTH NIMIQ_CLAIM_MAILERSEND_API_KEY and NIMIQ_CLAIM_MAILERSEND_FROM")
	}
	var sender otp.Sender = otp.ConsoleSender{}
	console := true
	if key != "" {
		mailer := otp.MailerSend{APIKey: key, From: from}
		if err := mailer.Validate(); err != nil {
			log.Fatal("Invalid MailerSend configuration for SecureClaim OTP")
		}
		sender = mailer
		console = false
	} else {
		email = "recipient@example.test"
	}
	s, err := newServer(sender, email, console)
	if err != nil {
		log.Fatal(err)
	}
	if directory := os.Getenv("NIMIQ_CLAIM_STATE_DIR"); directory != "" {
		unlock, err := lockPrivateState(directory)
		if err != nil {
			log.Fatal("SecureClaim state is locked by another process or is not private")
		}
		defer unlock()
		persistent, err := claim.NewPersistentStore(directory)
		if err != nil {
			log.Fatalf("private claim state unavailable: %v", err)
		}
		if email != "" {
			if err := persistent.SeedDemo(email); err != nil {
				log.Fatal(err)
			}
		}
		s.claims = persistent
		log.Print("SecureClaim private claim persistence enabled")
	}
	rpcURL := os.Getenv("NIMIQ_CLAIM_MAINNET_RPC_URL")
	vaultDir := os.Getenv("NIMIQ_CLAIM_VAULT_DIR")
	masterHex := os.Getenv("NIMIQ_CLAIM_MASTER_HEX")
	keyID := os.Getenv("NIMIQ_CLAIM_KEY_ID")
	if rpcURL != "" || vaultDir != "" || masterHex != "" || keyID != "" {
		if os.Getenv("NIMIQ_CLAIM_STATE_DIR") == "" {
			log.Fatal("SecureClaim escrow admission requires durable claim metadata directory")
		}
		admission, err := escrow.NewMainnetAdmission(s.claims, rpcURL, vaultDir, masterHex, keyID)
		if err != nil {
			log.Fatal("SecureClaim Mainnet escrow admission initialization failed")
		}
		s.escrow = admission
		verifier, err := nimiq.NewMainnetVerifier(rpcURL)
		if err != nil {
			log.Fatal("Invalid Mainnet RPC")
		}
		s.chainReady = func(ctx context.Context) bool { return verifier.NetworkReady(ctx) == nil }
		master, err := vault.DecodeMasterHex(masterHex)
		if err != nil {
			log.Fatal("Invalid encrypted Cashlink key")
		}
		persistentOTP, err := otp.NewPersistent(sender, os.Getenv("NIMIQ_CLAIM_STATE_DIR"), master)
		if err != nil {
			clear(master)
			log.Fatal("Encrypted OTP persistence unavailable")
		}
		s.otp = persistentOTP
		v, err := vault.New(master, keyID)
		clear(master)
		if err != nil {
			log.Fatal("Invalid encrypted Cashlink vault")
		}
		private, err := vault.NewFileStore(vaultDir, v)
		if err != nil {
			log.Fatal("Encrypted Cashlink vault unavailable")
		}
		s.cashlinkVault = private
		log.Print("Mainnet existing-Cashlink OTP admission and encrypted vault initialized")
	}
	// USDT cards hold no money and no key: a small state file is all they need.
	if stateDir := os.Getenv("NIMIQ_CLAIM_STATE_DIR"); stateDir != "" && os.Getenv("NIMIQ_CLAIM_EVM") != "off" {
		cards, err := evmcard.New(filepath.Join(stateDir, "evm"))
		if err != nil {
			log.Fatalf("USDT card state unavailable: %v", err)
		}
		if m, ok := sender.(otp.MailerSend); ok {
			appURL := strings.TrimSpace(os.Getenv("NIMIQ_CLAIM_APP_URL"))
			if appURL == "" {
				appURL = "https://mail.phoenixmail.ca"
			}
			cards.OnReady = evmNotice(m, appURL)
		}
		s.evm = cards
		go cards.Run(context.Background(), 30*time.Second)
		log.Print("USDT (Ethereum, Polygon) cards initialized")
	}
	if stateDir := os.Getenv("NIMIQ_CLAIM_STATE_DIR"); stateDir != "" {
		h, err := setupUSDTLink(s, stateDir)
		if err != nil {
			log.Fatalf("USDT cashlink unavailable: %v", err)
		}
		s.usdt = h
		if h != nil {
			log.Print("USDT cashlinks (relayed claim, protected cards) initialized")
		}
	}
	s.creationKey = firstConfiguredEnv("NIMIQ_CLAIM_ISSUER_KEY", "NIMIQ_CLAIM_DEV_CREATE_KEY")
	if s.creationKey != "" && len(s.creationKey) < 32 {
		log.Fatal("NIMIQ_CLAIM_DEV_CREATE_KEY must be at least 32 characters")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handler)
	mux.HandleFunc("/api/v1/claims", s.handler)
	mux.HandleFunc("/api/v1/claims/", s.handler)
	mux.HandleFunc("/api/v1/evm/cards", s.evmIssuer)
	mux.HandleFunc("/api/v1/evm/cards/", s.evmIssuer)
	mux.HandleFunc("/api/v1/usdt/", s.usdtRoutes)
	files := http.FileServer(http.Dir("./web"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		files.ServeHTTP(w, r)
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8099"
	}
	n, e := strconv.Atoi(port)
	if e != nil || n < 1 || n > 65535 {
		log.Fatal("invalid PORT")
	}
	address := fmt.Sprintf("127.0.0.1:%d", n)
	log.Printf("NimiqSecureClaim local Mainnet service at http://%s (OTP email sender: %t, issuer enabled: %t, encrypted vault configured: %t, encrypted OTP persistence: %t)", address, !console, len(s.creationKey) >= 32, s.escrow != nil && s.cashlinkVault != nil, s.otp.Durable())
	log.Fatal(http.ListenAndServe(address, mux))
}

// usdtRoutes serves every /api/v1/usdt/ route: the recipient's browser routes and the issuer routes (X-Claim-Service-Key). The
// standalone server and the embedded Wire engine share it.
func (s *server) usdtRoutes(w http.ResponseWriter, r *http.Request) {
	if s.usdt == nil {
		write(w, 503, map[string]string{"error": "usdt_unavailable"})
		return
	}
	if r.URL.Path == "/api/v1/usdt/issuer/relayers" || strings.HasPrefix(r.URL.Path, "/api/v1/usdt/issuer/relayers/") {
		s.usdt.relayers(s, w, r)
		return
	}
	if r.URL.Path == "/api/v1/usdt/issuer/gas-status" {
		s.usdt.gasStatus(s, w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/usdt/issuer/cards") {
		s.usdt.issuer(s, w, r)
		return
	}
	s.usdt.browser(w, r)
}
