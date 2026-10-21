package claim

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// NewPersistentStore keeps private claim metadata across process restarts.
// It does NOT persist OTP challenges (step 2), does NOT mark funds as confirmed,
// and MUST be used by a single process on a trusted local filesystem.
// Never store a spending seed in this file.
func NewPersistentStore(directory string) (*Store, error) {
	if !filepath.IsAbs(directory) {
		return nil, ErrInvalid
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	st, err := os.Lstat(directory)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil, ErrInvalid
	}
	path := filepath.Join(directory, "claims-private.json")
	s := NewStore()
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 4*1024*1024 {
			return nil, ErrInvalid
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || len(data) > 4*1024*1024 {
			return nil, ErrInvalid
		}
		var snapshot struct {
			Version int `json:"version"`
			Claims  []struct {
				Public Public `json:"public"`
				Email  string `json:"email"`
			} `json:"claims"`
		}
		if json.Unmarshal(data, &snapshot) != nil || snapshot.Version != 1 || len(snapshot.Claims) > MaxDraftClaims {
			return nil, ErrInvalid
		}
		validID := regexp.MustCompile(`^[a-f0-9]{48}$`)
		for _, entry := range snapshot.Claims {
			p := entry.Public
			canonical, luna, e := ParseAmount(p.Amount)
			if !validID.MatchString(p.ID) || e != nil || p.Amount != canonical || p.Network != "mainnet" || !validEmail(entry.Email) ||
				p.ExpiresAt.IsZero() || p.ExpiresAt.After(time.Now().Add(Lifetime+time.Minute)) ||
				(p.Status != "draft_unfunded" && p.Status != "email_verified_unfunded") ||
				(p.EmailVerified != (p.Status == "email_verified_unfunded")) || p.Palette < 0 || p.Palette >= 12 {
				return nil, ErrInvalid
			}
			if _, exists := s.values[p.ID]; exists {
				return nil, ErrInvalid
			}
			s.values[p.ID] = &record{public: p, email: entry.Email, luna: luna}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	s.persist = func(records map[string]*record) error {
		snapshot := struct {
			Version int `json:"version"`
			Claims  []struct {
				Public Public `json:"public"`
				Email  string `json:"email"`
			} `json:"claims"`
		}{Version: 1}
		for id, r := range records {
			if id == "demo" {
				continue
			}
			snapshot.Claims = append(snapshot.Claims, struct {
				Public Public `json:"public"`
				Email  string `json:"email"`
			}{Public: r.public, Email: r.email})
		}
		encoded, err := json.Marshal(snapshot)
		if err != nil || len(encoded) > 4*1024*1024 {
			return ErrInvalid
		}
		temp, err := os.CreateTemp(directory, ".claims-*.tmp")
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
		if err := os.Rename(temp.Name(), path); err != nil {
			return fmt.Errorf("claim_state_write_failed: %w", err)
		}
		if dir, err := os.Open(directory); err == nil {
			_ = dir.Sync()
			_ = dir.Close()
		}
		return nil
	}
	return s, nil
}
