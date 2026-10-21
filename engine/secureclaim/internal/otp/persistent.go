package otp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

const maxOTPState = 8 * 1024 * 1024

type savedChallenge struct {
	Email    string    `json:"email"`
	Digest   [32]byte  `json:"digest"`
	Expires  time.Time `json:"expires"`
	LastSent time.Time `json:"last_sent"`
	Window   time.Time `json:"window"`
	Requests int       `json:"requests"`
	Attempts int       `json:"attempts"`
	Verified bool      `json:"verified"`
}
type otpSnapshot struct {
	Version    int                       `json:"version"`
	Challenges map[string]savedChallenge `json:"challenges"`
}

// NewPersistent encrypts OTP digests, recipient bindings, expiry, resend
// counters and consumed state. The 32-byte master is held outside this file.
// The service must hold its exclusive state-directory process lock.
func NewPersistent(sender Sender, directory string, master []byte) (*Manager, error) {
	if len(master) != 32 || !filepath.IsAbs(directory) {
		return nil, ErrUnavailable
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, ErrUnavailable
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, ErrUnavailable
	}
	derive := func(label string) []byte {
		h := hmac.New(sha256.New, master)
		h.Write([]byte(label))
		return h.Sum(nil)
	}
	encryptionKey := derive("NimiqSecureClaim OTP storage v1")
	block, err := aes.NewCipher(encryptionKey)
	clear(encryptionKey)
	if err != nil {
		return nil, ErrUnavailable
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrUnavailable
	}
	m, err := New(sender)
	if err != nil {
		return nil, err
	}
	hashKey := derive("NimiqSecureClaim OTP digest v1")
	copy(m.key[:], hashKey)
	clear(hashKey)
	path := filepath.Join(directory, "otp-challenges.sealed")
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > maxOTPState {
			return nil, ErrUnavailable
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, ErrUnavailable
		}
		raw, err := io.ReadAll(io.LimitReader(f, maxOTPState+1))
		closeErr := f.Close()
		if err != nil || closeErr != nil || len(raw) > maxOTPState || len(raw) < aead.NonceSize()+aead.Overhead() {
			return nil, ErrUnavailable
		}
		plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte("otp-state-v1"))
		if err != nil {
			return nil, ErrUnavailable
		}
		defer clear(plain)
		var state otpSnapshot
		if json.Unmarshal(plain, &state) != nil || state.Version != 1 || len(state.Challenges) > 10000 {
			return nil, ErrUnavailable
		}
		for id, c := range state.Challenges {
			if id == "" || len(id) > 128 || c.Email == "" || len(c.Email) > 254 || c.Requests < 0 || c.Requests > 5 || c.Attempts < 0 || c.Attempts > 5 {
				return nil, ErrUnavailable
			}
			m.challenges[id] = &challenge{email: c.Email, digest: c.Digest, expires: c.Expires, lastSent: c.LastSent, window: c.Window, requests: c.Requests, attempts: c.Attempts, verified: c.Verified}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, ErrUnavailable
	}
	m.persist = func() error {
		state := otpSnapshot{Version: 1, Challenges: make(map[string]savedChallenge, len(m.challenges))}
		for id, c := range m.challenges {
			state.Challenges[id] = savedChallenge{Email: c.email, Digest: c.digest, Expires: c.expires, LastSent: c.lastSent, Window: c.window, Requests: c.requests, Attempts: c.attempts, Verified: c.verified}
		}
		plain, err := json.Marshal(state)
		if err != nil {
			return err
		}
		defer clear(plain)
		nonce := make([]byte, aead.NonceSize())
		if _, err = rand.Read(nonce); err != nil {
			return err
		}
		raw := aead.Seal(nonce, nonce, plain, []byte("otp-state-v1"))
		if len(raw) > maxOTPState {
			return ErrUnavailable
		}
		tmp, err := os.CreateTemp(directory, ".otp-*.sealed")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		if err = tmp.Chmod(0600); err != nil {
			tmp.Close()
			return err
		}
		if _, err = tmp.Write(raw); err != nil {
			tmp.Close()
			return err
		}
		if err = tmp.Sync(); err != nil {
			tmp.Close()
			return err
		}
		if err = tmp.Close(); err != nil {
			return err
		}
		if err = os.Rename(tmp.Name(), path); err != nil {
			return err
		}
		dir, err := os.Open(directory)
		if err != nil {
			return err
		}
		defer dir.Close()
		return dir.Sync()
	}
	// Validate writable durable storage before advertising any OTP capability.
	if err = m.saveLocked(); err != nil {
		return nil, err
	}
	return m, nil
}
