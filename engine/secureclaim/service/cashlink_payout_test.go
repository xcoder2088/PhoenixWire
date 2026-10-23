package service

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
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

// fakeChain is a node that knows one account, whatever it is told, and names transactions by a digest of their bytes.
type fakeChain struct {
	height     uint32
	balance    uint64
	funded     bool
	known      map[string]bool
	confirms   int64
	executed   bool
	broadcasts []string
}

func (f *fakeChain) Height(context.Context) (uint32, error)                { return f.height, nil }
func (f *fakeChain) Balance(context.Context, string) (uint64, error)       { return f.balance, nil }
func (f *fakeChain) RawHash(_ context.Context, raw string) (string, error) { return fakeHash(raw), nil }
func (f *fakeChain) ConfirmedSpendable(context.Context, string, string, uint64) (bool, error) {
	return f.funded, nil
}
func (f *fakeChain) Broadcast(_ context.Context, raw string) (string, error) {
	f.broadcasts = append(f.broadcasts, raw)
	return fakeHash(raw), nil
}
func (f *fakeChain) TxState(_ context.Context, hash string) (bool, int64, bool) {
	return f.known[hash], f.confirms, f.executed
}
func fakeHash(raw string) string {
	b := []byte(raw)
	out := make([]byte, 32)
	for i, c := range b {
		out[i%32] ^= c + byte(i)
	}
	return hex.EncodeToString(out)
}

type payoutFixture struct {
	s      *server
	sender *testSender
	chain  *fakeChain
	id     string
	seed   []byte
	to     string
}

func newPayoutFixture(t *testing.T) *payoutFixture {
	t.Helper()
	sender := &testSender{}
	s, err := newServer(sender, "", false)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	master := bytes.Repeat([]byte{0x35}, 32)
	if s.claims, err = claim.NewPersistentStore(dir); err != nil {
		t.Fatal(err)
	}
	if s.otp, err = otp.NewPersistent(sender, dir, master); err != nil {
		t.Fatal(err)
	}
	c, err := s.claims.Create("receiver@example.test", "3.25", "mainnet")
	if err != nil {
		t.Fatal(err)
	}
	v, err := vault.New(master, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "vault")
	os.Mkdir(root, 0700)
	os.Chmod(root, 0700)
	if s.cashlinkVault, err = vault.NewFileStore(root, v); err != nil {
		t.Fatal(err)
	}
	seed := bytes.Repeat([]byte{0x19}, 32)
	if err := s.cashlinkVault.Put(c.ID, seed); err != nil {
		t.Fatal(err)
	}
	chain := &fakeChain{height: 1000, balance: 325000, funded: true, known: map[string]bool{}, executed: true}
	s.chain = chain
	s.payoutDir = filepath.Join(dir, "payouts")
	to, _ := (nimiq.AddressDeriver{}).CashlinkAddress(bytes.Repeat([]byte{0x42}, 32))
	return &payoutFixture{s, sender, chain, c.ID, seed, to}
}

func (f *payoutFixture) call(action, code, to string) (int, map[string]any) {
	body := map[string]string{}
	if code != "" {
		body["code"] = code
	}
	if to != "" {
		body["to"] = to
	}
	data, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	f.s.handler(w, httptest.NewRequest(http.MethodPost, "/api/v1/claims/"+f.id+"?action="+action, bytes.NewReader(data)))
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func (f *payoutFixture) code(t *testing.T) string {
	t.Helper()
	if got := request(f.s, f.id, "request", ""); got.Code != 200 {
		t.Fatal("code request", got.Code)
	}
	return f.sender.code
}

func TestPayoutNeedsTheCodeAndMovesNothingWithoutIt(t *testing.T) {
	f := newPayoutFixture(t)
	f.code(t)
	for name, code := range map[string]string{"no code": "", "wrong code": "000000"} {
		if f.sender.code == "000000" {
			code = "111111"
		}
		if status, _ := f.call("payout", code, f.to); status != 403 {
			t.Fatal(name, "accepted", status)
		}
	}
	if len(f.chain.broadcasts) != 0 {
		t.Fatal("something was sent without a valid code")
	}
	if status, _ := f.call("payout_status", "", ""); status != 404 {
		t.Fatal("a payout exists that nobody started", status)
	}
}

func TestPayoutSignsOnceJournalsAndFollows(t *testing.T) {
	f := newPayoutFixture(t)
	code := f.code(t)
	status, answer := f.call("payout", code, f.to)
	if status != 200 || answer["status"] != "payout_pending" || answer["to"] != strings.ReplaceAll(f.to, " ", "") {
		t.Fatal("payout", status, answer)
	}
	if len(f.chain.broadcasts) != 1 {
		t.Fatal("expected one broadcast", len(f.chain.broadcasts))
	}
	raw, _ := hex.DecodeString(f.chain.broadcasts[0])
	to, _ := nimiq.AddressBytes(f.to)
	if len(raw) != 139 || !bytes.Equal(raw[34:54], to[:]) {
		t.Fatal("the transaction does not pay the named address")
	}
	if got := uint64(raw[54])<<56 | uint64(raw[55])<<48 | uint64(raw[56])<<40 | uint64(raw[57])<<32 | uint64(raw[58])<<24 | uint64(raw[59])<<16 | uint64(raw[60])<<8 | uint64(raw[61]); got != 325000 {
		t.Fatal("it must pay everything the Cashlink holds", got)
	}
	if answer["hash"] != fakeHash(f.chain.broadcasts[0]) {
		t.Fatal("the journal must follow the transaction that was sent")
	}
	// The code is spent, and the Hub link is no longer a way into this claim.
	if status, _ := f.call("release", f.sender.code, ""); status != 409 {
		t.Fatal("release still open after a payout began", status)
	}
	// The same destination can ask again without a code: the node does not know the payment yet, so it is pushed again, identically.
	status, answer = f.call("payout", "", f.to)
	if status != 200 || answer["status"] != "payout_pending" || len(f.chain.broadcasts) != 2 || f.chain.broadcasts[1] != f.chain.broadcasts[0] {
		t.Fatal("retry must repeat the same signed transaction", status, answer)
	}
	// Spacing and case of the address do not matter.
	if status, _ = f.call("payout", "", strings.ToLower(f.to)); status != 200 {
		t.Fatal("same address written differently refused", status)
	}
	// Another destination is refused for good.
	other, _ := (nimiq.AddressDeriver{}).CashlinkAddress(bytes.Repeat([]byte{0x43}, 32))
	if status, _ = f.call("payout", "", other); status != 409 {
		t.Fatal("a second destination was accepted", status)
	}
	// Once the node has it in a block, the status says so.
	f.chain.known[answer["hash"].(string)] = true
	f.chain.confirms = 3
	if status, answer = f.call("payout_status", "", ""); status != 200 || answer["status"] != "payout_confirmed" {
		t.Fatal("status", status, answer)
	}
	count := len(f.chain.broadcasts)
	f.call("payout_status", "", "")
	if len(f.chain.broadcasts) != count {
		t.Fatal("a confirmed payment must not be pushed again")
	}
}

func TestPayoutRefusals(t *testing.T) {
	f := newPayoutFixture(t)
	code := f.code(t)
	own, _ := (nimiq.AddressDeriver{}).CashlinkAddress(f.seed)
	for name, to := range map[string]string{"empty": "", "garbage": "hello", "burn": nimiq.BurnAddress, "itself": own} {
		if status, _ := f.call("payout", code, to); status != 400 {
			t.Fatal(name, "accepted", status)
		}
	}
	// An unfunded Cashlink is refused WITHOUT spending the code.
	f.chain.funded = false
	if status, _ := f.call("payout", code, f.to); status != 409 {
		t.Fatal("unfunded payout accepted", status)
	}
	f.chain.funded = true
	if status, _ := f.call("payout", code, f.to); status != 200 {
		t.Fatal("the code was spent by a refused attempt", status)
	}
}

func TestPayoutSignsAgainOnlyWhenTheFirstCanNoLongerBeIncluded(t *testing.T) {
	f := newPayoutFixture(t)
	code := f.code(t)
	_, first := f.call("payout", code, f.to)
	// Still valid and unknown: pushed again, never re-signed.
	f.chain.height += payoutExpiry - 1
	if _, again := f.call("payout", "", f.to); again["hash"] != first["hash"] {
		t.Fatal("re-signed while the first transaction was still valid")
	}
	// Expired and the coins are still there: a fresh transaction for the SAME destination.
	f.chain.height += 5
	status, fresh := f.call("payout", "", f.to)
	if status != 200 || fresh["hash"] == first["hash"] || fresh["to"] != first["to"] {
		t.Fatal("expected a fresh transaction to the same address", status, fresh)
	}
	// Expired and the Cashlink is empty: it was paid, nothing is signed.
	f.chain.height += payoutExpiry + 5
	f.chain.balance = 0
	count := len(f.chain.broadcasts)
	if _, answer := f.call("payout", "", f.to); answer["status"] != "payout_pending" || len(f.chain.broadcasts) != count {
		t.Fatal("signed again for an empty Cashlink", answer)
	}
}
