package wire

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWireNeverUsesClaimServiceDirectories(t *testing.T) {
	safe := t.TempDir()
	for _, reserved := range []string{"/etc/nimiqsecureclaim", "/etc/nimiqsecureclaim/wire", "/var/lib/nimiqsecureclaim/vault", "/etc"} {
		if separateWireDirectories(safe, reserved) == nil || separateWireDirectories(reserved, safe) == nil {
			t.Fatal("shared directory accepted")
		}
	}
	alias := filepath.Join(safe, "alias")
	if err := os.Symlink("/var/lib/nimiqsecureclaim", alias); err != nil {
		t.Fatal(err)
	}
	if separateWireDirectories(safe, filepath.Join(alias, "wire")) == nil {
		t.Fatal("symlink into claim state accepted")
	}
	if err := separateWireDirectories(filepath.Join(safe, "config"), filepath.Join(safe, "state")); err != nil {
		t.Fatal(err)
	}
}
