package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
)

// Direct payout: a recipient who already has a wallet (Nimiq Pay, or any address) can have the Cashlink's coins sent there instead of
// opening the Hub link. The seed never leaves the engine: after the e-mail code is verified, the engine signs ONE transaction that moves
// everything the Cashlink holds to the address the recipient named, and hands it to the node. The destination is written to a journal
// before anything is broadcast, so a retry can only ever repeat that same payment (same address, same signed transaction), never start
// another one. The Hub link stays available as the other choice; once a payout has begun it is the only path left for that claim.
//
// A payout needs the e-mail code once. After it, the same destination may be asked again without a code (to follow the payment, or to
// push it again if the node lost it): that costs the asker nothing and can only repeat what the recipient already authorized.

// payoutExpiry is how long a signed transaction stays valid, less a margin (the network accepts 7200 blocks from its start height).
const payoutExpiry = 7000

type payoutChain interface {
	Height(ctx context.Context) (uint32, error)
	Balance(ctx context.Context, address string) (uint64, error)
	Broadcast(ctx context.Context, rawHex string) (string, error)
	RawHash(ctx context.Context, rawHex string) (string, error)
	TxState(ctx context.Context, hash string) (found bool, confirmations int64, executed bool)
	ConfirmedSpendable(ctx context.Context, network, address string, exactLuna uint64) (bool, error)
}

type payoutRecord struct {
	To       string `json:"to"`
	Hash     string `json:"hash"`
	Raw      string `json:"raw"`
	Value    uint64 `json:"value"`
	Validity uint32 `json:"validity"`
	Created  int64  `json:"created"`
}

func (s *server) payoutFile(id string) (string, bool) {
	if s.payoutDir == "" || len(id) != 48 || strings.Trim(id, "0123456789abcdef") != "" {
		return "", false
	}
	return filepath.Join(s.payoutDir, id+".json"), true
}

func (s *server) loadPayout(id string) (*payoutRecord, error) {
	if s.payoutDir == "" {
		return nil, nil // payouts are not set up: only the Hub link exists
	}
	path, ok := s.payoutFile(id)
	if !ok {
		return nil, errors.New("payout_unavailable")
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec payoutRecord
	if json.Unmarshal(raw, &rec) != nil || rec.Hash == "" || rec.To == "" {
		return nil, errors.New("payout_unavailable")
	}
	return &rec, nil
}

func (s *server) savePayout(id string, rec *payoutRecord) error {
	path, ok := s.payoutFile(id)
	if !ok {
		return errors.New("payout_unavailable")
	}
	if err := os.MkdirAll(s.payoutDir, 0700); err != nil {
		return err
	}
	encoded, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.payoutDir, ".pending-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(encoded); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

// cashlinkKey opens the seed of a claim and returns it with the Cashlink's own address.
func (s *server) cashlinkKey(id string) (seed []byte, address string, err error) {
	opened, err := s.cashlinkVault.Open(id)
	if err != nil || len(opened) < 32 {
		clear(opened)
		return nil, "", errors.New("cashlink_unavailable")
	}
	seed = append([]byte(nil), opened[:32]...)
	clear(opened)
	address, err = (nimiq.AddressDeriver{}).CashlinkAddress(seed)
	if err != nil {
		clear(seed)
		return nil, "", errors.New("cashlink_unavailable")
	}
	return seed, address, nil
}

// signPayout signs and journals the one transaction that moves `value` luna to `to`, valid from `validity`.
func (s *server) signPayout(ctx context.Context, id, to string, value uint64, validity uint32) (*payoutRecord, error) {
	seed, _, err := s.cashlinkKey(id)
	if err != nil {
		return nil, err
	}
	defer clear(seed)
	raw, err := nimiq.SignBasic(seed, to, value, 0, validity, nimiq.NetworkMainAlbatross)
	if err != nil {
		return nil, err
	}
	encoded := hex.EncodeToString(raw)
	// The node names the transaction (parsing it moves nothing): that hash is what the journal follows.
	hash, err := s.chain.RawHash(ctx, encoded)
	if err != nil {
		return nil, err
	}
	return &payoutRecord{To: to, Hash: hash, Raw: encoded, Value: value, Validity: validity, Created: time.Now().Unix()}, nil
}

func payoutAnswer(rec *payoutRecord, status string, confirmations int64) map[string]any {
	return map[string]any{"status": status, "hash": rec.Hash, "to": rec.To, "value": rec.Value, "confirmations": confirmations}
}

// follow looks the journaled payment up and, when the node does not know it, pushes it again (or, once it can no longer be included and the
// Cashlink still holds its coins, signs it afresh for the same destination).
func (s *server) followPayout(ctx context.Context, id string, rec *payoutRecord) (map[string]any, int) {
	height, err := s.chain.Height(ctx)
	if err != nil {
		return map[string]any{"error": "chain_unavailable"}, http.StatusServiceUnavailable
	}
	if found, confirmations, executed := s.chain.TxState(ctx, rec.Hash); found {
		switch {
		case !executed:
			return payoutAnswer(rec, "payout_failed", confirmations), http.StatusOK
		case confirmations >= 1:
			return payoutAnswer(rec, "payout_confirmed", confirmations), http.StatusOK
		}
		return payoutAnswer(rec, "payout_pending", confirmations), http.StatusOK
	}
	if height < rec.Validity+payoutExpiry {
		_, _ = s.chain.Broadcast(ctx, rec.Raw) // already known or refused for now: the next look will tell
		return payoutAnswer(rec, "payout_pending", 0), http.StatusOK
	}
	// The signed transaction can no longer be included. If the coins are still there, sign again for the same destination.
	_, own, err := s.cashlinkKey(id)
	if err != nil {
		return map[string]any{"error": "cashlink_unavailable"}, http.StatusServiceUnavailable
	}
	balance, err := s.chain.Balance(ctx, own)
	if err != nil {
		return map[string]any{"error": "chain_unavailable"}, http.StatusServiceUnavailable
	}
	if balance == 0 {
		return payoutAnswer(rec, "payout_pending", 0), http.StatusOK
	}
	fresh, err := s.signPayout(ctx, id, rec.To, balance, height)
	if err != nil {
		return map[string]any{"error": "payout_unavailable"}, http.StatusServiceUnavailable
	}
	hash, err := s.chain.Broadcast(ctx, fresh.Raw)
	if err != nil {
		return map[string]any{"error": "chain_unavailable"}, http.StatusServiceUnavailable
	}
	fresh.Hash = hash
	if err := s.savePayout(id, fresh); err != nil {
		return map[string]any{"error": "payout_unavailable"}, http.StatusServiceUnavailable
	}
	return payoutAnswer(fresh, "payout_pending", 0), http.StatusOK
}

// payoutExistingCashlink serves the "payout" action (code + destination, first time; destination alone afterwards) and "payout_status"
// (no body: where the payment stands).
func (s *server) payoutExistingCashlink(w http.ResponseWriter, r *http.Request, id string, statusOnly bool) {
	if s.devConsole || !s.otp.Durable() || !s.claims.Durable() {
		write(w, 503, map[string]string{"error": "durable_email_otp_required"})
		return
	}
	if s.cashlinkVault == nil || s.chain == nil || s.payoutDir == "" {
		write(w, 503, map[string]string{"error": "payout_unavailable"})
		return
	}
	view, err := s.claims.Get(id)
	if err != nil {
		write(w, statusFromError(err), map[string]string{"error": "claim_unavailable"})
		return
	}
	if view.Network != "mainnet" || !s.cashlinkVault.Has(id) {
		write(w, 409, map[string]string{"error": "cashlink_not_funded"})
		return
	}
	var input struct {
		Code string `json:"code"`
		To   string `json:"to"`
	}
	if !statusOnly && decode(w, r, &input) != nil {
		write(w, 400, map[string]string{"error": "invalid_json"})
		return
	}
	s.payoutMu.Lock()
	defer s.payoutMu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	rec, err := s.loadPayout(id)
	if err != nil {
		write(w, 503, map[string]string{"error": "payout_unavailable"})
		return
	}
	if statusOnly {
		if rec == nil {
			write(w, 404, map[string]string{"error": "payout_not_started"})
			return
		}
		answer, status := s.followPayout(ctx, id, rec)
		write(w, status, answer)
		return
	}
	to := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(input.To), " ", ""))
	target, err := nimiq.AddressBytes(to)
	burn, _ := nimiq.AddressBytes(nimiq.BurnAddress)
	if err != nil || target == burn {
		write(w, 400, map[string]string{"error": "invalid_destination"})
		return
	}
	if rec != nil {
		// A payout is already under way: only the same destination may ask again, and it needs no new code.
		if previous, err := nimiq.AddressBytes(rec.To); err != nil || previous != target {
			write(w, 409, map[string]string{"error": "payout_already_started"})
			return
		}
		answer, status := s.followPayout(ctx, id, rec)
		write(w, status, answer)
		return
	}
	if input.Code == "" {
		write(w, 403, map[string]string{"error": "invalid_or_expired_code"}) // no payment yet: the code is needed, and no node is asked without one
		return
	}
	seed, own, err := s.cashlinkKey(id)
	if err != nil {
		write(w, 503, map[string]string{"error": "cashlink_unavailable"})
		return
	}
	clear(seed)
	if own, _ := nimiq.AddressBytes(own); own == target {
		write(w, 400, map[string]string{"error": "invalid_destination"})
		return
	}
	_, luna, err := claim.ParseAmount(view.Amount)
	if err != nil {
		write(w, 503, map[string]string{"error": "cashlink_unavailable"})
		return
	}
	// Funding is checked BEFORE the one-time code is spent.
	if ok, err := s.chain.ConfirmedSpendable(ctx, "mainnet", own, luna); err != nil || !ok {
		write(w, 409, map[string]string{"error": "cashlink_not_confirmed_or_invalid"})
		return
	}
	recipient, err := s.claims.Recipient(id)
	if err != nil {
		write(w, statusFromError(err), map[string]string{"error": "claim_unavailable"})
		return
	}
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
	if !view.EmailVerified {
		if _, err := s.claims.VerifyEmail(id); err != nil && !errors.Is(err, claim.ErrAlreadyVerified) {
			write(w, 503, map[string]string{"error": "claim_state_unavailable_request_new_code"})
			return
		}
	}
	height, err := s.chain.Height(ctx)
	if err != nil {
		write(w, 503, map[string]string{"error": "chain_unavailable"})
		return
	}
	balance, err := s.chain.Balance(ctx, own)
	if err != nil || balance < luna {
		write(w, 503, map[string]string{"error": "chain_unavailable"})
		return
	}
	rec, err = s.signPayout(ctx, id, to, balance, height)
	if err != nil {
		write(w, 503, map[string]string{"error": "payout_unavailable"})
		return
	}
	// Journal first, broadcast second: from here on a retry can only repeat this very payment.
	if err := s.savePayout(id, rec); err != nil {
		write(w, 503, map[string]string{"error": "payout_unavailable"})
		return
	}
	_, _ = s.chain.Broadcast(ctx, rec.Raw) // if the node refuses it now, the next look pushes it again
	w.Header().Set("Cache-Control", "no-store")
	write(w, 200, payoutAnswer(rec, "payout_pending", 0))
}
