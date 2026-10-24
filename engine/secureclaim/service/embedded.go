package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/escrow"
	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

// Engine embeds the existing claim/OTP implementation. The Wire gateway must
// check persisted funding and ownership before calling PublicHandler or Recovery.
type Engine struct {
	server    *server
	provision *escrow.Provisioner
	unlock    func()
	secrets   *vault.Vault
}
type Config struct {
	Directory, RPCURL string
	Master            []byte
	Sender            otp.Sender
}
type Claim = claim.Public
type MailerSend = otp.MailerSend

func WithEmailLanguage(ctx context.Context, language string) context.Context {
	return otp.WithEmailLanguage(ctx, language)
}

var ErrOTPRateLimited = otp.ErrRateLimited

func LockDirectory(path string) (func(), error) { return lockPrivateState(path) }

func New(c Config) (*Engine, error) {
	unlock, err := lockPrivateState(c.Directory)
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			unlock()
		}
	}()
	claims, err := claim.NewPersistentStore(c.Directory)
	if err != nil {
		return nil, err
	}
	manager, err := otp.NewPersistent(c.Sender, c.Directory, c.Master)
	if err != nil {
		return nil, err
	}
	secrets, err := vault.New(c.Master, "wire-v1")
	if err != nil {
		return nil, err
	}
	private, err := vault.NewFileStore(filepath.Join(c.Directory, "vault"), secrets)
	if err != nil {
		secrets.Close()
		return nil, err
	}
	verifier, err := nimiq.NewMainnetVerifier(c.RPCURL)
	if err != nil {
		secrets.Close()
		return nil, err
	}
	provision, err := escrow.NewProvisioner(claims, nimiq.AddressDeriver{}, private, verifier)
	if err != nil {
		secrets.Close()
		return nil, err
	}
	s := &server{claims: claims, otp: manager, cashlinkVault: private, chainReady: func(ctx context.Context) bool { return verifier.NetworkReady(ctx) == nil }, chain: verifier, payoutDir: filepath.Join(c.Directory, "payouts")}
	// USDC / USDT cashlinks run in the same process when the operator names the chains (NIMIQ_CLAIM_USDTLINK_CHAINS). Their keys
	// (co-signer master, relayers) are generated in <state>/usdtlink and belong to Wire alone. The gateway reaches the issuer routes
	// in-process with a key that exists only in this process's memory.
	if strings.TrimSpace(os.Getenv("NIMIQ_CLAIM_USDTLINK_CHAINS")) != "" {
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			secrets.Close()
			return nil, err
		}
		s.creationKey = hex.EncodeToString(raw)
		handler, err := setupUSDTLink(s, c.Directory)
		if err != nil {
			secrets.Close()
			return nil, err
		}
		s.usdt = handler
	}
	ok = true
	return &Engine{s, provision, unlock, secrets}, nil
}
func (e *Engine) Close()                         { e.secrets.Close(); e.unlock() }
func (e *Engine) Ready(ctx context.Context) bool { return e.server.chainReady(ctx) }
func (e *Engine) Create(id, email, amount string) (claim.Public, error) {
	return e.server.claims.CreateOnce(id, email, amount, "mainnet")
}
func (e *Engine) Provision(id string) (string, error) { return e.provision.Provision(id) }
func (e *Engine) Confirm(ctx context.Context, id string) (bool, error) {
	return e.provision.FundingConfirmed(ctx, id)
}
func (e *Engine) PublicHandler() http.Handler { return http.HandlerFunc(e.server.handler) }

// USDTReady says whether the USDC / USDT cashlinks are running in this process.
func (e *Engine) USDTReady() bool { return e.server.usdt != nil }

// USDTHandler serves every /api/v1/usdt/ route (recipient routes and issuer routes). The gateway must expose only the recipient
// routes to the public: the issuer routes need the in-process key below and are reached through IssuerCall.
func (e *Engine) USDTHandler() http.Handler { return http.HandlerFunc(e.server.usdtRoutes) }

// IssuerCall performs an issuer request in-process (create a card, read its status, cancel, refund, relayer balances...). It never
// crosses the network and the key never leaves the process.
func (e *Engine) IssuerCall(ctx context.Context, method, path string, body []byte) (int, []byte, error) {
	if e.server.usdt == nil {
		return 503, []byte(`{"error":"usdt_unavailable"}`), nil
	}
	req, err := http.NewRequestWithContext(ctx, method, path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Claim-Service-Key", e.server.creationKey)
	rec := httptest.NewRecorder()
	e.server.usdtRoutes(rec, req)
	return rec.Code, rec.Body.Bytes(), nil
}

// CardState says where a NIM card's coins are, from the node and the payout journal, never from what a page claims: "unclaimed" (the card
// still holds them), "claiming" (a payout to an address was sent and is not in a block yet) or "claimed" (they have left the card: paid
// to the recipient's address, or taken with the Hub link, or recovered by the sender: the chain cannot tell these apart). Only a funded
// card should be asked: an empty one that was never funded would read as "claimed".
func (e *Engine) CardState(ctx context.Context, id string) (string, error) {
	s := e.server
	if s.chain == nil || s.cashlinkVault == nil || !s.cashlinkVault.Has(id) {
		return "", errors.New("card_state_unavailable")
	}
	if rec, err := s.loadPayout(id); err != nil {
		return "", err
	} else if rec != nil {
		if found, confirmations, executed := s.chain.TxState(ctx, rec.Hash); found && executed && confirmations >= 1 {
			return "claimed", nil
		}
		return "claiming", nil
	}
	seed, address, err := s.cashlinkKey(id)
	if err != nil {
		return "", err
	}
	clear(seed)
	balance, err := s.chain.Balance(ctx, address)
	if err != nil {
		return "", err
	}
	if balance == 0 {
		return "claimed", nil
	}
	return "unclaimed", nil
}
func (e *Engine) Recovery(id string) (string, error) {
	seed, err := e.server.cashlinkVault.Open(id)
	if err != nil {
		return "", err
	}
	defer clear(seed)
	if len(seed) != 32 {
		return "", errors.New("invalid_recovery")
	}
	// Recovery must remain possible after the 24-hour recipient claim expiry.
	// Amount is supplied by the owning gateway's immutable durable record below.
	return base64.RawURLEncoding.EncodeToString(seed), nil
}
func RecoveryLink(seed64 string, luna uint64) (string, error) {
	seed, err := base64.RawURLEncoding.DecodeString(seed64)
	if err != nil || len(seed) != 32 {
		clear(seed)
		return "", errors.New("invalid_recovery")
	}
	defer clear(seed)
	payload := append([]byte(nil), seed...)
	payload = binary.BigEndian.AppendUint64(payload, luna)
	defer clear(payload)
	return "https://hub.nimiq.com/cashlink/#" + base64.URLEncoding.EncodeToString(payload), nil
}

// Reuse the claim engine's amount parser and sender OTP implementation.
func ParseAmount(raw string) (string, uint64, error) { return claim.ParseAmount(raw) }
func NewSenderOTP(sender otp.Sender, dir string, master []byte) (*otp.Manager, error) {
	return otp.NewPersistent(sender, dir, master)
}

// RefundExpired pays a NIM card back to where it came from once its time is over: the sender of the transaction that funded it. It is
// asked by the gateway for each expired payment it knows (amount = what the card was made for) and answers:
//
//	"pending"  a refund was sent (or is being followed) and is not in a block yet
//	"done"     the refund is in a block
//	"claimed"  nothing to refund: the card is empty (the recipient took it, or it was refunded another way)
//	"released" the recipient already has the Hub link (they opened it before the end): the coins are theirs to take, nothing is sent
//	"started"  the recipient's own payout to an address began: it is not touched
//	"unknown"  where the money came from cannot be read yet: asked again later
//	"ambiguous" more than one address put money on the card (anyone can send the funding mark to its public address): nothing is sent, the sender
//	           recovers it by hand
//
// The destination is only ever the funding transaction's sender, recorded in the journal BEFORE anything is broadcast, so a retry can only
// repeat this very refund. The network fee of a basic transaction is nothing; a NIM card carries no service fee.
func (e *Engine) RefundExpired(ctx context.Context, id string, luna uint64) (string, error) {
	s := e.server
	if s.chain == nil || s.cashlinkVault == nil || s.payoutDir == "" || !s.cashlinkVault.Has(id) {
		return "", errors.New("refund_unavailable")
	}
	s.payoutMu.Lock()
	defer s.payoutMu.Unlock()
	rec, err := s.loadPayout(id)
	if err != nil {
		return "", err
	}
	if rec != nil {
		if rec.Kind != "refund" {
			return "started", nil
		}
		answer, status := s.followPayout(ctx, id, rec)
		if status != 200 {
			return "pending", nil
		}
		if answer["status"] == "payout_refunded" {
			if found, confirmations, executed := s.chain.TxState(ctx, rec.Hash); found && executed && confirmations >= 1 {
				return "done", nil
			}
		}
		return "pending", nil
	}
	view, err := s.claims.Peek(id)
	if err != nil {
		return "", err
	}
	if view.EmailVerified {
		return "released", nil
	}
	seed, own, err := s.cashlinkKey(id)
	if err != nil {
		return "", err
	}
	clear(seed)
	balance, err := s.chain.Balance(ctx, own)
	if err != nil {
		return "", err
	}
	if balance == 0 {
		return "claimed", nil
	}
	origin, err := s.chain.FundingSender(ctx, own)
	if errors.Is(err, nimiq.ErrAmbiguousFunding) {
		return "ambiguous", nil // more than one sender put money on this card (or its history is too long): by hand, never automatic
	}
	if err != nil {
		return "", err
	}
	if origin == "" {
		return "unknown", nil
	}
	height, err := s.chain.Height(ctx)
	if err != nil {
		return "", err
	}
	fresh, err := s.signPayout(ctx, id, origin, balance, height)
	if err != nil {
		return "", err
	}
	fresh.Kind = "refund"
	if err := s.savePayout(id, fresh); err != nil { // journal first, broadcast second
		return "", err
	}
	_, _ = s.chain.Broadcast(ctx, fresh.Raw)
	return "pending", nil
}

// ForgetNIM ends the retention of a NIM card: its claim (with the recipient's e-mail address) is erased, but only when the card holds nothing and no payout
// or refund is under way. The card's key stays sealed in the vault: a payment that someone sends to its address later could still be recovered.
// It answers "forgotten" or "keep".
func (e *Engine) ForgetNIM(ctx context.Context, id string) (string, error) {
	state, err := e.CardState(ctx, id)
	if err != nil {
		return "", err
	}
	if state != "claimed" { // "claimed" is also what an empty card reads: nothing is on it
		return "keep", nil
	}
	if err := e.server.claims.Forget(id); err != nil {
		return "", err
	}
	return "forgotten", nil
}
