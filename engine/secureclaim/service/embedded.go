package service

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net/http"
	"path/filepath"

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
	s := &server{claims: claims, otp: manager, cashlinkVault: private, chainReady: func(ctx context.Context) bool { return verifier.NetworkReady(ctx) == nil }}
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
