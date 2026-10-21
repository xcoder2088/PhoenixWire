package claim

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistentClaimSurvivesRestart(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	s, err := NewPersistentStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Create("receiver@example.com", "3.25000", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	if c.Amount != "3.25" || c.Status != "draft_unfunded" {
		t.Fatal("bad claim")
	}
	stat, err := os.Stat(filepath.Join(dir, "claims-private.json"))
	if err != nil || stat.Mode().Perm() != 0600 {
		t.Fatalf("private state file permissions: %v %v", stat, err)
	}
	s2, err := NewPersistentStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s2.Get(c.ID)
	if err != nil || loaded.ID != c.ID || loaded.Status != "draft_unfunded" {
		t.Fatal("draft lost after restart", err)
	}
	recipient, err := s2.Recipient(c.ID)
	if err != nil || recipient != "receiver@example.com" {
		t.Fatal("immutable recipient lost")
	}
	if _, err := s2.VerifyEmail(c.ID); err != nil {
		t.Fatal(err)
	}
	s3, err := NewPersistentStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := s3.Get(c.ID)
	if err != nil || !verified.EmailVerified || verified.Status != "email_verified_unfunded" {
		t.Fatal("verification lost after restart", err)
	}
}
func TestPersistentStateRejectsCorruptionAndInsecureDirectory(t *testing.T) {
	root := t.TempDir()
	insecure := filepath.Join(root, "insecure")
	if err := os.Mkdir(insecure, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentStore(insecure); err == nil {
		t.Fatal("world-readable state directory accepted")
	}
	private := filepath.Join(root, "private")
	s, err := NewPersistentStore(private)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Create("recipient@example.com", "1", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(private, "claims-private.json")
	if err := os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentStore(private); err == nil {
		t.Fatal("corrupt state silently accepted")
	}
}
