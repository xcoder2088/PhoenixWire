package wire

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

type session struct {
	Email    string
	Verified bool
	Expires  time.Time
	// Scope, when set, is the id of the ONE payment this session may fund and follow (the hand-over to the Nimiq Pay app). Such a session is never
	// Verified: it cannot list other payments, create any, reach USDC / USDT cards, ask for a recovery key or hand anything over.
	Scope string `json:",omitempty"`
}
type payment struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Amount    string    `json:"amount"`
	Luna      uint64    `json:"luna"`
	Message   string    `json:"message"`
	Language  string    `json:"language"`
	Address   string    `json:"address"`
	Stage     string    `json:"stage"`
	Created   time.Time `json:"created"`
	// SentAt is when the first invitation was accepted by the mail service; a failed attempt gives none, a resend never replaces it,
	// and a card that was sent before this was kept has none (it is not made up).
	SentAt *time.Time `json:"sent_at,omitempty"`
	// StartedAt is when the funding was started (the wallet was asked): a second request to a wallet waits long enough for the first one to show.
	StartedAt *time.Time `json:"started_at,omitempty"`
	// ClaimState is where the coins are, read from the chain (nimstate.go): "unclaimed", "claiming" or "claimed". Set on the copies of the list
	// only, never stored; absent while it is not known yet.
	ClaimState   string    `json:"claim_state,omitempty"`
	LastDelivery time.Time `json:"-"`
	Deliveries   int       `json:"-"`
	Nimicon      string    `json:"-"`
}

// Internal snapshot fields for delivery counters must persist across restart.
type savedPayment struct {
	Payment      payment
	LastDelivery time.Time
	Deliveries   int
	Nimicon      string
}
type rate struct {
	Until time.Time
	Count int
}
type snapshot struct {
	Version  int
	Sessions map[string]session
	Payments map[string]savedPayment
	Rates    map[string]rate
	Notices  map[string]map[string]time.Time `json:",omitempty"` // card id -> notice kind -> when it was sent to the sender (expiry.go)
	USDT     map[string]usdtCard             `json:",omitempty"` // USDC / USDT cards of the senders (their claim links are never kept)
	Access   accessPolicy                    `json:",omitempty"` // who may send, as the administrator set it in the app (senders.go)
}
type diskStore struct {
	state  snapshot
	aead   cipher.AEAD
	path   string
	failed bool
}

func digest(master []byte, label string) []byte {
	h := hmac.New(sha256.New, master)
	h.Write([]byte(label))
	return h.Sum(nil)
}
func token() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic("secure random unavailable")
	}
	return hex.EncodeToString(b)
}
func sessionKey(raw string) string { h := sha256.Sum256([]byte(raw)); return hex.EncodeToString(h[:]) }
func newStore(dir string, master []byte) (*diskStore, error) {
	key := digest(master, "Phoenix Wire state v1")
	block, err := aes.NewCipher(key)
	clear(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	d := &diskStore{aead: aead, path: filepath.Join(dir, "wire-state.sealed"), state: snapshot{Version: 1, Sessions: map[string]session{}, Payments: map[string]savedPayment{}, Rates: map[string]rate{}, USDT: map[string]usdtCard{}}}
	info, err := os.Lstat(d.path)
	if errors.Is(err, os.ErrNotExist) {
		return d, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 8*1024*1024 {
		return nil, errors.New("private_state_unavailable")
	}
	raw, err := os.ReadFile(d.path)
	if err != nil || len(raw) < aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("state_unavailable")
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte("wire-state-v1"))
	if err != nil {
		return nil, errors.New("state_key_mismatch")
	}
	defer clear(plain)
	if json.Unmarshal(plain, &d.state) != nil || d.state.Version != 1 || d.state.Sessions == nil || d.state.Payments == nil || d.state.Rates == nil || len(d.state.Payments) > 10000 || len(d.state.Sessions) > 10000 {
		return nil, errors.New("invalid_state")
	}
	if d.state.USDT == nil { // a state file made before the USDC / USDT cards
		d.state.USDT = map[string]usdtCard{}
	}
	return d, nil
}
func (d *diskStore) save() error {
	if d.failed {
		return errors.New("state_unavailable")
	}
	err := d.write()
	if err != nil {
		d.failed = true
	}
	return err
}
func (d *diskStore) write() error {
	plain, err := json.Marshal(d.state)
	if err != nil {
		return err
	}
	defer clear(plain)
	if len(plain) > 8*1024*1024 {
		return errors.New("state_full")
	}
	nonce := make([]byte, d.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	sealed := d.aead.Seal(nonce, nonce, plain, []byte("wire-state-v1"))
	f, err := os.CreateTemp(filepath.Dir(d.path), ".wire-*.sealed")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(sealed)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(f.Name(), d.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(d.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
