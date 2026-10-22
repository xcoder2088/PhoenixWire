package service

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net/http"
	"strings"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

// This file adds NO Nimiq funding, signing, broadcast, or second payment engine.
// PhoenixMail creates and funds the original Cashlink using its existing Hub flow.
// The backend only verifies the already-funded Cashlink, encrypts its 32-byte
// seed, and releases the Hub-compatible link after email OTP verification.
func (s *server) issuerAuthorized(r *http.Request) bool {
	supplied := r.Header.Get("X-Claim-Service-Key")
	return len(s.creationKey) >= 32 && len(supplied) == len(s.creationKey) &&
		subtle.ConstantTimeCompare([]byte(supplied), []byte(s.creationKey)) == 1
}

func (s *server) admitExistingCashlink(w http.ResponseWriter, r *http.Request, id string) {
	if !s.issuerAuthorized(r) {
		write(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	if s.devConsole || !s.otp.Durable() || !s.claims.Durable() {
		write(w, 503, map[string]string{"error": "durable_email_otp_required"})
		return
	}
	if s.escrow == nil || s.cashlinkVault == nil {
		write(w, 503, map[string]string{"error": "cashlink_vault_unavailable"})
		return
	}
	var input struct {
		Seed    string `json:"seed"`
		Address string `json:"address"`
		Message string `json:"message"`
	}
	if decode(w, r, &input) != nil {
		write(w, 400, map[string]string{"error": "invalid_json"})
		return
	}
	seed, err := base64.RawURLEncoding.DecodeString(input.Seed)
	if err != nil || len(seed) != 32 {
		write(w, 400, map[string]string{"error": "invalid_cashlink_seed"})
		return
	}
	defer clear(seed)
	// Admission derives the Cashlink address from the seed and independently
	// checks confirmed spendable funds on Mainnet before sealing ciphertext.
	if err := s.escrow.Admit(r.Context(), id, seed, input.Address, input.Message); err != nil {
		// Retries may follow an acknowledged funding or a lost HTTP response.
		// Only the exact same original seed and message may reuse this claim.
		if errors.Is(err, vault.ErrAlreadyStored) {
			old, openErr := s.cashlinkVault.Open(id)
			defer clear(old)
			expected := append([]byte(nil), seed...)
			defer clear(expected)
			if input.Message != "" {
				expected = append(expected, byte(len(input.Message)))
				expected = append(expected, []byte(input.Message)...)
			}
			if openErr == nil && subtle.ConstantTimeCompare(old, expected) == 1 {
				write(w, 200, map[string]any{"status": "funded_secret_sealed", "claim_path": "/?claim=" + id})
				return
			}
		}
		write(w, 409, map[string]string{"error": "cashlink_not_confirmed_or_invalid"})
		return
	}
	write(w, 200, map[string]any{"status": "funded_secret_sealed", "claim_path": "/?claim=" + id})
}

// The OTP is submitted together with release. A previously verified public
// claim ID alone never authorizes retrieval of the spending secret.
func (s *server) releaseExistingCashlink(w http.ResponseWriter, r *http.Request, id string) {
	if s.devConsole || !s.otp.Durable() || !s.claims.Durable() {
		write(w, 503, map[string]string{"error": "durable_email_otp_required"})
		return
	}
	if s.cashlinkVault == nil {
		write(w, 503, map[string]string{"error": "cashlink_vault_unavailable"})
		return
	}
	view, err := s.claims.Get(id)
	if err != nil {
		write(w, statusFromError(err), map[string]string{"error": "claim_unavailable"})
		return
	}
	if view.Network != "mainnet" {
		write(w, 409, map[string]string{"error": "claim_not_available"})
		return
	}
	// Check encrypted storage BEFORE consuming a one-time code.
	if !s.cashlinkVault.Has(id) {
		write(w, 409, map[string]string{"error": "cashlink_not_funded"})
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	if decode(w, r, &input) != nil {
		write(w, 400, map[string]string{"error": "invalid_json"})
		return
	}
	recipient, err := s.claims.Recipient(id)
	if err != nil {
		write(w, statusFromError(err), map[string]string{"error": "claim_unavailable"})
		return
	}
	// Authenticate before decrypting any spending material, even for a
	// malformed, expired or replayed code.
	if err = s.otp.Verify(id, recipient, input.Code); err != nil {
		status := 403
		if errors.Is(err, otp.ErrUnavailable) {
			status = 503
		}
		if errors.Is(err, otp.ErrAlreadyVerified) {
			status = 409
		}
		write(w, status, map[string]string{"error": "invalid_or_expired_code"})
		return
	}
	// Persist one-time consumption so a service restart cannot reuse this code.
	if !view.EmailVerified {
		if _, err := s.claims.VerifyEmail(id); err != nil && !errors.Is(err, claim.ErrAlreadyVerified) {
			write(w, 503, map[string]string{"error": "claim_state_unavailable_request_new_code"})
			return
		}
	}
	// The vault stores only the original Cashlink's seed. Reconstruct the
	// standard Nimiq Hub link (seed + amount) without creating a new payment.
	seed, err := s.cashlinkVault.Open(id)
	defer clear(seed)
	if err != nil || len(seed) < 32 || len(seed) > 97 {
		write(w, 503, map[string]string{"error": "cashlink_unavailable"})
		return
	}
	if len(seed) > 32 && (len(seed) < 34 || int(seed[32]) != len(seed)-33) {
		clear(seed)
		write(w, 503, map[string]string{"error": "cashlink_unavailable"})
		return
	}
	_, luna, err := claim.ParseAmount(view.Amount)
	if err != nil {
		write(w, 503, map[string]string{"error": "cashlink_unavailable"})
		return
	}
	payload := make([]byte, 0, 40)
	payload = append(payload, seed[:32]...)
	payload = binary.BigEndian.AppendUint64(payload, luna)
	if len(seed) > 32 {
		payload = append(payload, seed[32:]...)
	}
	clear(seed)
	defer clear(payload)
	link := "https://hub.nimiq.com/cashlink/#" + base64.URLEncoding.EncodeToString(payload)
	// Never cache, log, email or redirect through a third party with this link.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	write(w, 200, map[string]string{"status": "otp_verified", "cashlink": link})
}

func (s *server) otpClaimAction(w http.ResponseWriter, r *http.Request, id, action string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch strings.TrimSpace(action) {
	case "seal":
		s.admitExistingCashlink(w, r, id)
		return true
	case "release":
		s.releaseExistingCashlink(w, r, id)
		return true
	default:
		return false
	}
}
