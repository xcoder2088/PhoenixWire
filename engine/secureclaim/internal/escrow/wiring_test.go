package escrow

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

// This is a fully local integration test: real seed/address derivation, mocked
// Nimiq Mainnet RPC, real AES-256-GCM and real private disk publication.
// It does NOT prove interoperability with a live Nimiq node or wallet.
func TestLocalFundingAdmissionToEncryptedVault(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}
	address, err := (nimiq.AddressDeriver{}).CashlinkAddress(seed)
	if err != nil {
		t.Fatal(err)
	}
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		var result any
		switch request.Method {
		case "getBlockNumber":
			result = 100
		case "getBlockByNumber":
			result = map[string]any{"network": "MainAlbatross"}
		case "getAccountByAddress":
			result = map[string]any{"balance": 250001}
		case "getTransactionsByAddress":
			result = []any{map[string]any{"to": address, "value": 250000, "recipientData": "0082809287", "confirmations": 3, "executionResult": true}}
		default:
			t.Errorf("unexpected RPC method %q", request.Method)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"data": result}})
	}))
	defer rpc.Close()
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	vaultDir := filepath.Join(root, "vault")
	store, err := claim.NewPersistentStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := store.Create("recipient@example.com", "2.5", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	masterHex := strings.Repeat("a1", 32)
	admission, err := NewMainnetAdmission(store, rpc.URL, vaultDir, masterHex, "test-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := admission.Admit(context.Background(), draft.ID, seed, address); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(vaultDir, draft.ID+".sealed.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, seed) || bytes.Contains(raw, []byte(hex.EncodeToString(seed))) {
		t.Fatal("private Cashlink seed leaked into persisted envelope")
	}
	key, err := vault.DecodeMasterHex(masterHex)
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := vault.New(key, "test-key-1")
	if err != nil {
		t.Fatal(err)
	}
	private, err := vault.NewFileStore(vaultDir, cipher)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := private.Open(draft.ID)
	if err != nil || !bytes.Equal(restored, seed) {
		t.Fatal("encrypted vault roundtrip failed", err)
	}
	reloaded, err := claim.NewPersistentStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	view, err := reloaded.Get(draft.ID)
	if err != nil || view.Status != "draft_unfunded" {
		t.Fatal("admission must not authorize payout", err)
	}
}
