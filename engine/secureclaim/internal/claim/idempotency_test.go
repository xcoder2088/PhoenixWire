package claim

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCreateOnceSurvivesRestartAndDoesNotReplaceExpiredClaim(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	open := func() *Store {
		t.Helper()
		s, e := NewPersistentStore(dir)
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	s := open()
	id := strings.Repeat("1", 48)
	a, e := s.CreateOnce(id, "r@example.test", "1.25", "mainnet")
	if e != nil {
		t.Fatal(e)
	}
	s = open()
	b, e := s.CreateOnce(id, "r@example.test", "1.25000", "mainnet")
	if e != nil || a != b {
		t.Fatal("retry created or changed claim", e)
	}
	if _, e = s.CreateOnce(id, "other@example.test", "1.25", "mainnet"); e == nil {
		t.Fatal("recipient was rebound")
	}
	s.now = func() time.Time { return a.ExpiresAt.Add(time.Second) }
	if _, e = s.CreateOnce(id, "r@example.test", "1.25", "mainnet"); !errors.Is(e, ErrExpired) {
		t.Fatal("expired id was recycled", e)
	}
}
