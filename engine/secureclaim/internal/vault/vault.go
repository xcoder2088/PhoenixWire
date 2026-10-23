// Package vault seals Cashlink spending secrets under a separately managed
// master key. The vault is a crypto primitive, NOT an escrow/payment service.
// This package exposes no HTTP endpoint or Nimiq wallet signing operation.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
)

const (
	FormatVersion = 1
	KeyBytes      = 32
	NonceBytes    = 12
	// A real Nimiq spending seed is short. Bound input sizes to avoid unbounded
	// allocations and make accidental ingestion of emails/files impossible.
	MaxSecretBytes = 256
)

var (
	ErrInvalid   = errors.New("invalid_vault_input")
	ErrOpen      = errors.New("vault_open_failed")
	claimPattern = regexp.MustCompile(`^[a-f0-9]{48}$`)
)

// Envelope contains ciphertext ONLY. It may be stored alongside the claim
// metadata, but MUST NEVER include the master key or an unencrypted seed.
// KeyID is an operational rotation label, not a cryptographic key.
type Envelope struct {
	Version    int    `json:"version"`
	KeyID      string `json:"key_id"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type Vault struct {
	master [KeyBytes]byte
	keyID  string
}

// New requires exactly 32 random bytes from a secret manager or a protected
// configuration file. Repeated passwords are NOT suitable master keys.
func New(master []byte, keyID string) (*Vault, error) {
	if len(master) != KeyBytes || len(keyID) < 1 || len(keyID) > 64 {
		return nil, ErrInvalid
	}
	for _, r := range keyID {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return nil, ErrInvalid
		}
	}
	v := &Vault{keyID: keyID}
	copy(v.master[:], master)
	return v, nil
}

// DecodeMasterHex accepts a secret provided from outside code or a repository.
// Do not print or log this value, including on failure.
func DecodeMasterHex(raw string) ([]byte, error) {
	if len(raw) != KeyBytes*2 {
		return nil, ErrInvalid
	}
	b, err := hex.DecodeString(raw)
	if err != nil || len(b) != KeyBytes {
		return nil, ErrInvalid
	}
	return b, nil
}

// Per-claim derivation separates keys cryptographically from one another,
// though compromise of the master still compromises all encrypted claims.
func (v *Vault) claimKey(claimID string) []byte {
	mac := hmac.New(sha256.New, v.master[:])
	_, _ = mac.Write([]byte("NimiqSecureClaim:cashlink:wrap-key:v1\x00"))
	_, _ = mac.Write([]byte(claimID))
	return mac.Sum(nil)
}
func associatedData(id, keyID string) []byte {
	return []byte("NimiqSecureClaim:cashlink:v1\x00" + id + "\x00" + keyID)
}
func aead(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func validClaimID(id string) bool { return claimPattern.MatchString(id) }

// Seal encrypts an actual Cashlink private spending secret while binding the
// ciphertext to the precise claim ID and key version. It does NOT validate
// funding, the Nimiq network, sender authorization or a wallet destination.
// Callers must enforce all of those BEFORE enabling any funded escrow operation.
func (v *Vault) Seal(claimID string, secret []byte) (Envelope, error) {
	if v == nil || !validClaimID(claimID) || len(secret) == 0 || len(secret) > MaxSecretBytes {
		return Envelope{}, ErrInvalid
	}
	key := v.claimKey(claimID)
	g, err := aead(key)
	clear(key)
	if err != nil {
		return Envelope{}, err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return Envelope{}, fmt.Errorf("secure random unavailable: %w", err)
	}
	ciphertext := g.Seal(nil, nonce, secret, associatedData(claimID, v.keyID))
	return Envelope{Version: FormatVersion, KeyID: v.keyID, Nonce: hex.EncodeToString(nonce), Ciphertext: hex.EncodeToString(ciphertext)}, nil
}

// Open is for future trusted server-side settlement code ONLY; do not expose
// it to the browser or to the ordinary OTP verification response.
func (v *Vault) Open(claimID string, env Envelope) ([]byte, error) {
	if v == nil || !validClaimID(claimID) || env.Version != FormatVersion || env.KeyID != v.keyID ||
		len(env.Nonce) != NonceBytes*2 || len(env.Ciphertext) < (aes.BlockSize)*2 ||
		len(env.Ciphertext) > (MaxSecretBytes+aes.BlockSize)*2 {
		return nil, ErrOpen
	}
	nonce, e1 := hex.DecodeString(env.Nonce)
	body, e2 := hex.DecodeString(env.Ciphertext)
	if e1 != nil || e2 != nil || len(nonce) != NonceBytes {
		return nil, ErrOpen
	}
	key := v.claimKey(claimID)
	g, err := aead(key)
	clear(key)
	if err != nil {
		return nil, ErrOpen
	}
	plain, err := g.Open(nil, nonce, body, associatedData(claimID, env.KeyID))
	if err != nil {
		return nil, ErrOpen
	}
	return plain, nil
}

// Close zeros the in-process copy of a key. It cannot guarantee wiping old
// copies held by the Go runtime; limit process access and avoid swap.
func (v *Vault) Close() {
	if v == nil {
		return
	}
	clear(v.master[:])
}
