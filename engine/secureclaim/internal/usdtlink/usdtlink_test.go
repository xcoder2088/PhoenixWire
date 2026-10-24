package usdtlink

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/chain"
	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/proof"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/relayer"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/safe"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
)

// fakeNode answers only what the service asks outside a real claim: the card's USDT balance. Everything that would send a
// transaction is refused, so these tests can never spend anything.
type fakeNode struct {
	balance     *big.Int
	nonce       uint64
	receipts    map[string]chain.Receipt
	native      map[cl.Address]*big.Int // native-coin balances (absent: zero)
	sent        int                     // transactions broadcast
	block       uint64                  // the latest block (Finality)
	deep        *big.Int                // the balance at a deep block, when different from the latest
	estimate    uint64                  // gas the node estimates for any call (0: it refuses, as a call that would revert)
	failBalance bool                    // balanceOf calls fail (the node cannot be read)
	times       map[uint64]time.Time    // when each block was mined (BlockTimer)
	timeErr     error                   // the node cannot say when a block was mined
}

func (f *fakeNode) ChainID() uint64 { return 137 }
func (f *fakeNode) Call(_ context.Context, _, _ cl.Address, data []byte) ([]byte, error) {
	if bytes.Equal(data, safe.NonceCall) {
		return cl.Word32(new(big.Int).SetUint64(f.nonce)), nil // Safe.nonce()
	}
	if f.failBalance {
		return nil, errors.New("rpc unavailable")
	}
	return cl.Word32(f.balance), nil // balanceOf
}
func (f *fakeNode) EstimateGas(context.Context, cl.Address, cl.Address, []byte) (uint64, error) {
	if f.estimate > 0 {
		return f.estimate, nil
	}
	return 0, errors.New("no")
}
func (f *fakeNode) Code(context.Context, cl.Address) ([]byte, error)         { return nil, nil }
func (f *fakeNode) PendingNonce(context.Context, cl.Address) (uint64, error) { return 0, nil }
func (f *fakeNode) NativeBalance(_ context.Context, a cl.Address) (*big.Int, error) {
	if v, ok := f.native[a]; ok {
		return v, nil
	}
	return big.NewInt(0), nil
}
func (f *fakeNode) Fees(context.Context) (*big.Int, *big.Int, error) {
	return big.NewInt(1_000_000_000), big.NewInt(100_000_000_000), nil // 1 and 100 gwei
}
func (f *fakeNode) Send(context.Context, []byte) error          { f.sent++; return nil }
func (f *fakeNode) BlockNumber(context.Context) (uint64, error) { return f.block, nil }
func (f *fakeNode) CallAt(_ context.Context, _, _ cl.Address, _ []byte, _ uint64) ([]byte, error) {
	if f.deep != nil {
		return cl.Word32(f.deep), nil
	}
	return cl.Word32(f.balance), nil
}
func (f *fakeNode) Receipt(_ context.Context, h string) (chain.Receipt, error) {
	return f.receipts[h], nil
}
func (f *fakeNode) BlockTime(_ context.Context, block uint64) (time.Time, error) {
	if f.timeErr != nil {
		return time.Time{}, f.timeErr
	}
	return f.times[block], nil
}

// plainNode is a node that cannot say when a block was mined (no BlockTimer).
type plainNode struct{ chain.Chain }

// fakeCodes stands for the e-mail code manager: the right code is always 123456, and it remembers who was mailed.
type fakeCodes struct {
	mu       sync.Mutex
	sent     map[string]string // key -> e-mail the code went to
	verified map[string]bool
}

func (f *fakeCodes) IssueClaim(_ context.Context, key, email string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent[key] = email
	return nil
}
func (f *fakeCodes) Verify(key, email, code string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sent[key] != email || code != "123456" {
		return errors.New("invalid")
	}
	f.verified[key] = true
	return nil
}
func (f *fakeCodes) Verified(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.verified[key]
}

var linkKeys sync.Map // card id -> link private key (what the sender's browser would hold)

func codesOf(s *Service) *fakeCodes { return s.codes.(*fakeCodes) }

// payoutReceipt is the receipt of a real payout: the token emitted Transfer(card -> recipient) and Transfer(card -> relayer).
func payoutReceipt(c Created, to string, units uint64, block uint64) chain.Receipt {
	token, card, relayerAddr := cl.MustAddress(c.Token), cl.MustAddress(c.Safe), cl.MustAddress(c.Relayer)
	topic := cl.Keccak256([]byte("Transfer(address,address,uint256)"))
	log := func(toAddr cl.Address, v uint64) chain.Log {
		return chain.Log{Address: token, Topics: [][]byte{topic, card.Word(), toAddr.Word()}, Data: cl.Word32(new(big.Int).SetUint64(v))}
	}
	return chain.Receipt{Found: true, Success: true, Block: block, Logs: []chain.Log{log(cl.MustAddress(to), units), log(relayerAddr, c.FeeUnits)}}
}

func newService(t *testing.T) (*Service, *fakeNode) {
	t.Helper()
	n, err := evm.Lookup(evm.AssetUSDTPolygon)
	if err != nil {
		t.Fatal(err)
	}
	node := &fakeNode{balance: big.NewInt(0), receipts: map[string]chain.Receipt{}, native: map[cl.Address]*big.Int{}, block: 1000}
	key, _, err := cl.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	r, err := relayer.New(relayer.Config{Chain: node, Key: key, Token: cl.MustAddress(n.Token), MinFee: big.NewInt(50_000)})
	if err != nil {
		t.Fatal(err)
	}
	node.native[r.Address()] = new(big.Int).Mul(big.NewInt(5), big.NewInt(1_000_000_000_000_000_000)) // 5 POL: plenty
	master := make([]byte, 32)
	rand.Read(master)
	s, err := New(filepath.Join(t.TempDir(), "state"), master, []Chain{{Network: n, Node: node, Relayer: r, MinFee: 50_000, DripCap: big.NewInt(300_000_000_000_000_000), Floor: big.NewInt(200_000_000_000_000_000), MinFloat: big.NewInt(200_000_000_000_000_000)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetCodes(&fakeCodes{sent: map[string]string{}, verified: map[string]bool{}}, bytes.Repeat([]byte{7}, 32)); err != nil {
		t.Fatal(err)
	}
	return s, node
}

func newCard(t *testing.T, s *Service) Created {
	t.Helper()
	id := make([]byte, 16)
	rand.Read(id)
	key, link, _ := cl.GenerateKey()
	linkKeys.Store(hex.EncodeToString(id), key)
	in := Input{Asset: evm.AssetUSDTPolygon, CardID: hex.EncodeToString(id), LinkOwner: link, Amount: "2.50", Protected: true, Email: "friend@example.com"}
	c, err := s.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const dest1, dest2 = "0x1111111111111111111111111111111111111111", "0x2222222222222222222222222222222222222222"

// preparedChecks remembers what the browser was given by StartCheck: an e-mail check is stateless until its proof is valid, so the
// proof is built from those answers, exactly as the page does.
var preparedChecks sync.Map // check id -> checkAnswer

type checkAnswer struct {
	P    Prepared
	Card string
}

func startCheck(t *testing.T, s *Service, session, cardID, idem string) (Prepared, error) {
	t.Helper()
	p, err := s.StartCheck(context.Background(), session, cardID, idem)
	if err == nil {
		preparedChecks.Store(p.AuthorizationID, checkAnswer{p, cardID})
	}
	return p, err
}

// linkProof is what the recipient's browser does: sign the LinkAuthorization of an attempt with the link key.
func linkProof(t *testing.T, s *Service, authID string, key []byte) []byte {
	t.Helper()
	var p proof.LinkAuthorization
	var r *Record
	if a, rec, err := s.attemptNoSession(authID); err == nil {
		var perr error
		if p, perr = s.proofOf(a, rec); perr != nil {
			t.Fatal(perr)
		}
		r = rec
	} else if v, ok := preparedChecks.Load(authID); ok { // a check that is not stored yet: from what StartCheck answered
		ans := v.(checkAnswer)
		rec, err := s.get(ans.Card)
		if err != nil {
			t.Fatal(err)
		}
		r = rec
		id, _ := hex.DecodeString(rec.ID)
		aid, _ := hex.DecodeString(ans.P.AuthorizationID)
		ch, _ := hex.DecodeString(ans.P.Challenge)
		sb, _ := hex.DecodeString(ans.P.SessionBinding)
		eb, _ := hex.DecodeString(ans.P.EmailBinding)
		copy(p.CardID[:], id)
		copy(p.AuthorizationID[:], aid)
		copy(p.Challenge[:], ch)
		copy(p.SessionBinding[:], sb)
		copy(p.EmailBinding[:], eb)
		p.Action, p.Deadline = proof.ActionEmail, uint64(ans.P.Deadline)
	} else {
		t.Fatal("unknown attempt")
	}
	safeAddr, _ := cl.ParseAddress(r.Safe)
	sig, err := p.Sign(137, safeAddr, key)
	if err != nil {
		t.Fatal(err)
	}
	return sig.Bytes()
}

// emailCheck runs the e-mail check that comes first: link proof, code mailed, right code entered. It returns the check's id.
func emailCheck(t *testing.T, s *Service, c Created, session string) string {
	t.Helper()
	ctx := context.Background()
	p, err := startCheck(t, s, session, c.ID, randomHex(16))
	if err != nil {
		t.Fatalf("start check: %v", err)
	}
	key, _ := linkKeys.Load(c.ID)
	if err := s.ProveLink(ctx, session, c.ID, p.AuthorizationID, linkProof(t, s, p.AuthorizationID, key.([]byte)), "fr"); err != nil {
		t.Fatalf("prove link: %v", err)
	}
	if _, err := s.VerifyEmail(ctx, session, p.AuthorizationID, "123456"); err != nil {
		t.Fatalf("verify e-mail: %v", err)
	}
	return p.AuthorizationID
}

// authorize runs the whole flow up to the grant: e-mail check first, then the destination.
func authorize(t *testing.T, s *Service, c Created, dest string) (session, authID, grant string) {
	t.Helper()
	ctx := context.Background()
	session = randomHex(32)
	check := emailCheck(t, s, c, session)
	p, err := s.Prepare(ctx, session, c.ID, dest, randomHex(16), check)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	key, _ := linkKeys.Load(c.ID)
	grant, _, err = s.Authorize(ctx, session, p.AuthorizationID, linkProof(t, s, p.AuthorizationID, key.([]byte)))
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	return session, p.AuthorizationID, grant
}

func TestEveryCardIsProtected(t *testing.T) {
	s, _ := newService(t)
	prot := newCard(t, s)
	if len(prot.Owners) != 2 || prot.Threshold != 2 || !prot.Protected {
		t.Fatalf("owners %d, threshold %d", len(prot.Owners), prot.Threshold)
	}
	if prot.State != StateAwaitingFunds || prot.FeeUnits != 250_000 || prot.Units != 2_500_000 {
		t.Fatalf("unexpected card: %+v", prot.Public)
	}
	// The card address must be the one rebuilt from the stored owners (the browser checks the same thing).
	rec, _ := s.get(prot.ID)
	card, _ := rec.card()
	if addr, _ := card.Address(137); addr.Hex() != prot.Safe {
		t.Fatal("card address does not match its owners")
	}
	// An unprotected card cannot be created at all.
	_, link, _ := cl.GenerateKey()
	if _, err := s.Create(Input{Asset: evm.AssetUSDTPolygon, CardID: "00112233445566778899aabbccddeeff", LinkOwner: link, Amount: "1"}); err == nil {
		t.Fatal("an unprotected card was accepted")
	}
}

func TestCreateRejectsBadInput(t *testing.T) {
	s, _ := newService(t)
	_, link, _ := cl.GenerateKey()
	good := Input{Asset: evm.AssetUSDTPolygon, CardID: "00112233445566778899aabbccddeeff", LinkOwner: link, Amount: "1", Protected: true, Email: "friend@example.com"}
	for name, mutate := range map[string]func(*Input){
		"short id":      func(i *Input) { i.CardID = "00" },
		"bad owner":     func(i *Input) { i.LinkOwner = "0x12" },
		"fee too low":   func(i *Input) { i.FeeUnits = 1 },
		"fee too high":  func(i *Input) { i.FeeUnits = 99_000_000 },
		"above ceiling": func(i *Input) { i.Amount = "1000.01" },
		"no email":      func(i *Input) { i.Email = "" },
		"not protected": func(i *Input) { i.Protected = false },
		"unknown asset": func(i *Input) { i.Asset = "usdt_ethereum" }, // not served by this test service
	} {
		in := good
		mutate(&in)
		if _, err := s.Create(in); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := s.Create(good); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(good); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate id: %v", err)
	}
}

func TestAuthorizationFlowAndItsGates(t *testing.T) { // UL-03, with the e-mail code first
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	if _, err := startCheck(t, s, randomHex(32), c.ID, randomHex(16)); !errors.Is(err, ErrNotFunded) {
		t.Fatalf("unfunded card: %v", err)
	}
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	session := randomHex(32)
	idem := randomHex(16)
	ck, err := startCheck(t, s, session, c.ID, idem)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := startCheck(t, s, session, c.ID, idem); again.AuthorizationID != ck.AuthorizationID {
		t.Fatal("a repeated request must return the same check")
	}
	key, _ := linkKeys.Load(c.ID)
	good := linkProof(t, s, ck.AuthorizationID, key.([]byte))

	if _, err := s.VerifyEmail(ctx, session, ck.AuthorizationID, "123456"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a code check before the link proof (the check does not exist yet): %v", err)
	}
	if n := len(s.auths); n != 0 {
		t.Fatalf("StartCheck must store nothing, found %d attempts", n)
	}
	wrongKey, _, _ := cl.GenerateKey()
	if err := s.ProveLink(ctx, session, c.ID, ck.AuthorizationID, linkProof(t, s, ck.AuthorizationID, wrongKey), ""); !errors.Is(err, ErrLinkProof) {
		t.Fatalf("a proof from another key: %v", err)
	}
	if len(codesOf(s).sent) != 0 {
		t.Fatal("a code was mailed without a valid link proof")
	}
	if err := s.ProveLink(ctx, randomHex(32), c.ID, ck.AuthorizationID, good, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another session cannot even use the id of a check that is not stored: %v", err)
	}
	if err := s.ProveLink(ctx, session, c.ID, ck.AuthorizationID, good, "fr"); err != nil {
		t.Fatal(err)
	}
	// once stored, a resend from another session is forbidden, and the id of another card is not accepted
	if err := s.ProveLink(ctx, randomHex(32), c.ID, ck.AuthorizationID, good, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("another session touched a stored check: %v", err)
	}
	for _, email := range codesOf(s).sent {
		if email != "friend@example.com" {
			t.Fatalf("the code went to %q, not to the card's own recipient", email)
		}
	}
	if _, err := s.VerifyEmail(ctx, session, ck.AuthorizationID, "000000"); !errors.Is(err, ErrBadCode) {
		t.Fatalf("a wrong code: %v", err)
	}
	// Until the right code was entered no destination can be chosen.
	if _, err := s.Prepare(ctx, session, c.ID, dest1, randomHex(16), ck.AuthorizationID); !errors.Is(err, ErrEmailCheck) {
		t.Fatalf("a destination before the e-mail code: %v", err)
	}
	until, err := s.VerifyEmail(ctx, session, ck.AuthorizationID, "123456")
	if err != nil || until.IsZero() {
		t.Fatalf("verify e-mail: %v", err)
	}
	if again, err := s.VerifyEmail(ctx, session, ck.AuthorizationID, "123456"); err != nil || !again.Equal(until) {
		t.Fatalf("a repeated verification: %v", err)
	}
	// The check belongs to its session.
	if _, err := s.Prepare(ctx, randomHex(32), c.ID, dest1, randomHex(16), ck.AuthorizationID); !errors.Is(err, ErrEmailCheck) {
		t.Fatalf("a check used from another session: %v", err)
	}
	idem2 := randomHex(16)
	p, err := s.Prepare(ctx, session, c.ID, dest1, idem2, ck.AuthorizationID)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := s.Prepare(ctx, session, c.ID, dest1, idem2, ck.AuthorizationID); again.AuthorizationID != p.AuthorizationID {
		t.Fatal("a repeated request must return the same attempt")
	}
	// Nothing is reserved by a bare Prepare: knowing a card id cannot block the recipient's address.
	if r, _ := s.get(c.ID); r.Recipient != "" {
		t.Fatal("Prepare reserved the destination")
	}
	// A claim attempt never asks for a code: that is the e-mail check's business.
	if err := s.ProveLink(ctx, session, c.ID, p.AuthorizationID, linkProof(t, s, p.AuthorizationID, key.([]byte)), ""); !errors.Is(err, ErrAuthState) {
		t.Fatalf("a claim attempt asked for a code: %v", err)
	}
	if _, _, err := s.CosignClaim(ctx, session, p.AuthorizationID, "nope"); !errors.Is(err, ErrBadGrant) {
		t.Fatalf("no grant yet: %v", err)
	}
	claimProof := linkProof(t, s, p.AuthorizationID, key.([]byte))
	if _, _, err := s.Authorize(ctx, session, p.AuthorizationID, linkProof(t, s, p.AuthorizationID, wrongKey)); !errors.Is(err, ErrLinkProof) {
		t.Fatalf("a claim proof from another key: %v", err)
	}
	if _, _, err := s.Authorize(ctx, randomHex(32), p.AuthorizationID, claimProof); !errors.Is(err, ErrForbidden) {
		t.Fatalf("another session authorized the attempt: %v", err)
	}
	// The proof of the e-mail check is not a claim proof: it signs another action, no destination, no transaction.
	if _, _, err := s.Authorize(ctx, session, p.AuthorizationID, good); !errors.Is(err, ErrLinkProof) {
		t.Fatalf("the e-mail proof used as a claim proof: %v", err)
	}
	grant, until2, err := s.Authorize(ctx, session, p.AuthorizationID, claimProof)
	if err != nil || grant == "" || until2.IsZero() {
		t.Fatalf("authorize: %v", err)
	}
	if again, _, _ := s.Authorize(ctx, session, p.AuthorizationID, claimProof); again != grant {
		t.Fatal("a repeated authorization must return the same grant")
	}
	if r, _ := s.get(c.ID); r.Recipient != dest1 {
		t.Fatalf("destination not reserved by an authorized attempt: %q", r.Recipient)
	}
	// The e-mail check allowed ONE attempt: it is consumed.
	if _, err := s.Prepare(ctx, session, c.ID, dest1, randomHex(16), ck.AuthorizationID); !errors.Is(err, ErrEmailCheck) {
		t.Fatalf("a consumed check used again: %v", err)
	}
	if _, _, err := s.CosignClaim(ctx, session, p.AuthorizationID, "wrong grant"); !errors.Is(err, ErrBadGrant) {
		t.Fatalf("a wrong grant: %v", err)
	}
	if _, _, err := s.CosignClaim(ctx, randomHex(32), p.AuthorizationID, grant); !errors.Is(err, ErrForbidden) {
		t.Fatalf("the grant used from another session: %v", err)
	}
	owner, sig, err := s.CosignClaim(ctx, session, p.AuthorizationID, grant)
	if err != nil || len(sig) != 65 || owner.Hex() == "" {
		t.Fatalf("co-signature: %v (%d bytes)", err, len(sig))
	}
	if owner2, sig2, err := s.CosignClaim(ctx, session, p.AuthorizationID, grant); err != nil || owner2 != owner || !bytes.Equal(sig, sig2) {
		t.Fatalf("a repeated call must return the same signature: %v", err)
	}
	// One destination per card: another attempt for another address is refused.
	if _, err := s.Prepare(ctx, randomHex(32), c.ID, dest2, randomHex(16), ""); !errors.Is(err, ErrAddressSet) {
		t.Fatalf("a second destination: %v", err)
	}
	// The grant expires.
	s.now = func() time.Time { return time.Now().Add(GrantLife + time.Minute) }
	if _, _, err := s.CosignClaim(ctx, session, p.AuthorizationID, grant); !errors.Is(err, ErrAuthExpired) {
		t.Fatalf("an expired grant: %v", err)
	}
}

func TestAnEmailCheckExpires(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	session := randomHex(32)
	check := emailCheck(t, s, c, session)
	s.now = func() time.Time { return time.Now().Add(CheckLife + time.Minute) }
	if _, err := s.Prepare(context.Background(), session, c.ID, dest1, randomHex(16), check); !errors.Is(err, ErrEmailCheck) {
		t.Fatalf("an old check: %v", err)
	}
}

func TestKnowingACardIDCannotUseUpItsBudgetOrItsStorage(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	// a stranger starts thousands of e-mail checks without ever proving the link key: nothing is stored, nothing is counted
	for i := 0; i < 2000; i++ {
		if _, err := startCheck(t, s, randomHex(32), c.ID, randomHex(16)); err != nil {
			t.Fatalf("unproven check %d: %v", i, err)
		}
	}
	if len(s.auths) != 0 {
		t.Fatalf("StartCheck stored %d attempts", len(s.auths))
	}
	// the legitimate recipient is not blocked, however many there were
	if _, a, _ := authorize(t, s, c, dest1); a == "" {
		t.Fatal("the recipient could not start")
	}
}

func TestADerivedCheckIsBoundToItsCardItsSessionAndItsMinute(t *testing.T) {
	s, node := newService(t)
	c, c2 := newCard(t, s), newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	ctx := context.Background()
	session := randomHex(32)
	p, err := startCheck(t, s, session, c.ID, randomHex(16))
	if err != nil {
		t.Fatal(err)
	}
	key, _ := linkKeys.Load(c.ID)
	good := linkProof(t, s, p.AuthorizationID, key.([]byte))
	// another card, another session: the id means nothing
	if err := s.ProveLink(ctx, session, c2.ID, p.AuthorizationID, good, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("the id used on another card: %v", err)
	}
	if err := s.ProveLink(ctx, randomHex(32), c.ID, p.AuthorizationID, good, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("the id used by another session: %v", err)
	}
	if err := s.ProveLink(ctx, session, c.ID, "00"+p.AuthorizationID[2:], good, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a forged id: %v", err)
	}
	// the same request in the same minute is the same check
	again, _ := startCheck(t, s, session, c.ID, "0123456789abcdef0123")
	same, _ := startCheck(t, s, session, c.ID, "0123456789abcdef0123")
	if again.AuthorizationID != same.AuthorizationID {
		t.Fatal("a repeated request must return the same check")
	}
	// it expires
	s.now = func() time.Time { return time.Now().Add(ChallengeLife + 2*time.Minute) }
	if err := s.ProveLink(ctx, session, c.ID, p.AuthorizationID, good, ""); !errors.Is(err, ErrAuthExpired) {
		t.Fatalf("an old check: %v", err)
	}
}

func TestCodeEmailsAreCountedWhenTheyAreSent(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	key, _ := linkKeys.Load(c.ID)
	type chk struct{ session, id string }
	var checks []chk
	for i := 0; i < MaxCodeSends; i++ {
		session := randomHex(32)
		p, err := startCheck(t, s, session, c.ID, randomHex(16))
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ProveLink(ctx, session, c.ID, p.AuthorizationID, linkProof(t, s, p.AuthorizationID, key.([]byte)), ""); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
		checks = append(checks, chk{session, p.AuthorizationID})
	}
	// the card's quota is used up
	session := randomHex(32)
	p, _ := startCheck(t, s, session, c.ID, randomHex(16))
	if err := s.ProveLink(ctx, session, c.ID, p.AuthorizationID, linkProof(t, s, p.AuthorizationID, key.([]byte)), ""); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("the ninth e-mail in a quarter of an hour: %v", err)
	}
	// sixteen minutes later the first sends are out of the window, but an OLD check that asks again is counted at the moment of
	// its send, not at the date it was created: the quota still holds
	later := time.Now().Add(prepWindow + time.Minute)
	s.now = func() time.Time { return later }
	for i := 0; i < MaxCodeSends; i++ {
		if err := s.ProveLink(ctx, checks[i].session, c.ID, checks[i].id, linkProof(t, s, checks[i].id, key.([]byte)), ""); err != nil {
			t.Fatalf("resend %d on an old check: %v", i, err)
		}
	}
	if err := s.ProveLink(ctx, checks[0].session, c.ID, checks[0].id, linkProof(t, s, checks[0].id, key.([]byte)), ""); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("a ninth resend through an old check: %v", err)
	}
}

func TestAnAuthorizedAttemptAndItsCheckChangeTogether(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	session := randomHex(32)
	check := emailCheck(t, s, c, session)
	p, err := s.Prepare(ctx, session, c.ID, dest1, randomHex(16), check)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := linkKeys.Load(c.ID)
	if _, _, err := s.Authorize(ctx, session, p.AuthorizationID, linkProof(t, s, p.AuthorizationID, key.([]byte))); err != nil {
		t.Fatal(err)
	}
	a, _ := s.getAuth(p.AuthorizationID)
	ck, _ := s.getAuth(check)
	if a.State != AuthAuthorized || ck.State != AuthConsumed || ck.ConsumedBy != a.ID {
		t.Fatalf("attempt %s, check %s (consumed by %q)", a.State, ck.State, ck.ConsumedBy)
	}
	// a check left unused by an older version is reconciled by the repeated call
	ck.State, ck.ConsumedBy = AuthChecked, ""
	if err := s.saveAuth(ck); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Authorize(ctx, session, p.AuthorizationID, linkProof(t, s, p.AuthorizationID, key.([]byte))); err != nil {
		t.Fatal(err)
	}
	if ck, _ = s.getAuth(check); ck.State != AuthConsumed {
		t.Fatal("the unused check was not reconciled")
	}
}

func TestCancelStopsAuthorizations(t *testing.T) {
	s, node := newService(t)
	node.balance = big.NewInt(10_000_000)
	ctx := context.Background()
	c := newCard(t, s)
	session, authID, grant := authorize(t, s, c, dest1)
	if _, err := s.Cancel(c.ID, "bad-token"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cancel with a wrong token: %v", err)
	}
	if _, err := s.Cancel(c.ID, c.SenderToken); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CosignClaim(ctx, session, authID, grant); !errors.Is(err, ErrCancelled) {
		t.Fatalf("a co-signature after cancel: %v", err)
	}
	if _, err := s.Prepare(ctx, randomHex(32), c.ID, dest1, randomHex(16), ""); !errors.Is(err, ErrCancelled) {
		t.Fatalf("a new attempt after cancel: %v", err)
	}
}

func TestFundingFollowsTheChainAndZeroNeverCancels(t *testing.T) { // C2
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateAwaitingFunds {
		t.Fatalf("state %s", p.State)
	}
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits - 1)) // one unit short
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateAwaitingFunds {
		t.Fatalf("underfunded: %s", p.State)
	}
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateReady {
		t.Fatalf("funded: %s", p.State)
	}
	// A node that answers zero (lagging, reorganised, wrong) must not cancel the card, and the card must not stay "funded".
	node.balance = big.NewInt(0)
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateAwaitingFunds {
		t.Fatalf("zero balance: %s", p.State)
	}
	if _, err := s.Prepare(ctx, randomHex(32), c.ID, dest1, randomHex(16), ""); !errors.Is(err, ErrNotFunded) {
		t.Fatalf("an attempt on a card that holds nothing right now: %v", err)
	}
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits)) // the node recovers
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateReady {
		t.Fatalf("after the node recovered: %s", p.State)
	}
}

func mustGet(t *testing.T, s *Service, id string) *Record {
	t.Helper()
	r, err := s.get(id)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestStateSurvivesRestart(t *testing.T) {
	s, _ := newService(t)
	c := newCard(t, s)
	s.mu.Lock()
	s.records[c.ID].Verified = true
	s.mu.Unlock()
	_ = s.update(mustGet(t, s, c.ID))
	again, err := New(s.dir, nil, nil)
	if err != nil || !again.Has(c.ID) {
		t.Fatalf("reload: %v", err)
	}
	if r, _ := again.get(c.ID); !r.Verified || r.Email != "friend@example.com" {
		t.Fatal("record changed across a restart")
	}
}

func TestIdSpellingsCannotAliasOneCard(t *testing.T) { // UL-01
	s, _ := newService(t)
	_, link, _ := cl.GenerateKey()
	in := Input{Asset: evm.AssetUSDTPolygon, CardID: "aabbccddeeff00112233445566778899", LinkOwner: link, Amount: "1", Protected: true, Email: "a@example.com"}
	if _, err := s.Create(in); err != nil {
		t.Fatal(err)
	}
	in.CardID, in.Email = "AABBCCDDEEFF00112233445566778899", "attacker@example.com"
	if _, err := s.Create(in); err == nil {
		t.Fatal("an upper-case spelling of the same card was accepted")
	}
	// Even another id cannot reuse the same Safe: the Safe address is unique per record.
	s.mu.Lock()
	n := len(s.records)
	s.mu.Unlock()
	if n != 1 {
		t.Fatalf("%d records", n)
	}
}

func TestRejectedClaimLeavesTheCardUntouched(t *testing.T) { // UL-02
	s, node := newService(t)
	node.balance = big.NewInt(10_000_000)
	c := newCard(t, s)
	s.mu.Lock()
	s.records[c.ID].Verified = true
	s.mu.Unlock()
	if _, err := s.claim(context.Background(), c.ID, "0x3333333333333333333333333333333333333333", nil); err == nil {
		t.Fatal("a claim without signatures was accepted")
	}
	r, _ := s.get(c.ID)
	if r.Recipient != "" || r.Op != nil {
		t.Fatalf("a rejected claim changed the card: recipient %q op %v", r.Recipient, r.Op)
	}
	for _, bad := range []string{"0x0000000000000000000000000000000000000000", c.Safe, c.Token, c.Relayer} {
		if _, err := s.Prepare(context.Background(), randomHex(32), c.ID, bad, randomHex(16), ""); err == nil {
			t.Fatalf("destination %s accepted", bad)
		}
	}
}

func TestRefundIsAFullSweepWithOneFixedDestination(t *testing.T) { // UL-04, C4
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(2_750_000) // 2.50 + 0.25 fee (0.10 gas + 6 %)
	r1, r2 := "0x4444444444444444444444444444444444444444", "0x5555555555555555555555555555555555555555"
	_, sig, units, err := s.Refund(ctx, c.ID, c.SenderToken, r1)
	if err != nil || len(sig) != 65 || units != 2_500_000 {
		t.Fatalf("refund: %v, units %d", err, units)
	}
	if r, _ := s.get(c.ID); r.Reserved == nil || !s.reserved(r) || r.Refund == nil || r.Refund.Nonce != 0 {
		t.Fatalf("refund intent not recorded: %+v", r.Refund)
	}
	// The same destination again is idempotent; another one is refused: no concurrent intents.
	if _, _, again, err := s.Refund(ctx, c.ID, c.SenderToken, r1); err != nil || again != units {
		t.Fatalf("same destination: %v", err)
	}
	if _, _, _, err := s.Refund(ctx, c.ID, c.SenderToken, r2); !errors.Is(err, ErrAddressSet) {
		t.Fatalf("second destination: %v", err)
	}
	if _, err := s.RefundClaim(ctx, c.ID, "wrong", nil); !errors.Is(err, ErrForbidden) {
		t.Fatalf("refund relay with a wrong token: %v", err)
	}
	if _, _, _, err := s.Refund(ctx, c.ID, "wrong", r1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("refund with a wrong token: %v", err)
	}
}

func TestSurplusIsRecoverableAfterThePayout(t *testing.T) { // C3
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	_, _ = s.Refresh(ctx, c.ID)
	// The payout happened (receipt found, Safe nonce is now 1) and 0.40 USDT was left on the card.
	s.mu.Lock()
	rec := s.records[c.ID]
	rec.Op = &Operation{Kind: "claim", To: "0x1111111111111111111111111111111111111111", Units: c.Units, Tx: "0xaaa", Started: s.now()}
	s.mu.Unlock()
	node.receipts["0xaaa"] = payoutReceipt(c, "0x1111111111111111111111111111111111111111", c.Units, 100)
	node.nonce, node.balance = 1, big.NewInt(650_000) // 0.40 + 0.25 fee
	p, _ := s.Refresh(ctx, c.ID)
	if p.State != StatePaid || p.TxHash != "0xaaa" || p.TxKind != "claim" {
		t.Fatalf("after the payout: %+v", p)
	}
	_, _, units, err := s.Refund(ctx, c.ID, c.SenderToken, "0x4444444444444444444444444444444444444444")
	if err != nil || units != 400_000 {
		t.Fatalf("residual refund: %v, units %d", err, units)
	}
	if r, _ := s.get(c.ID); r.Refund == nil || r.Refund.Nonce != 1 {
		t.Fatal("the refund must be made for the Safe's current nonce")
	}
	if in, _ := s.Status(ctx, c.ID); in.BalanceUnits != 650_000 || in.State != StatePaid {
		t.Fatalf("status %+v", in)
	}
}

func TestInFlightPayoutIsSettledFromTheChain(t *testing.T) { // UL-05, C1
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	_, _ = s.Refresh(ctx, c.ID) // funded
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "claim", To: "0x1111111111111111111111111111111111111111", Units: c.Units, Tx: "0xabc", Started: s.now()}
	s.mu.Unlock()
	node.balance = big.NewInt(0) // the payout moved the money, but the answer and the "paid" write were lost
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateVerifying || p.TxHash != "0xabc" {
		t.Fatalf("state %s, want verifying with the hash", p.State)
	}
	if _, err := s.Cancel(c.ID, c.SenderToken); !errors.Is(err, ErrPending) {
		t.Fatalf("cancel while a payout is in flight: %v", err)
	}
	if _, _, _, err := s.Refund(ctx, c.ID, c.SenderToken, "0x4444444444444444444444444444444444444444"); !errors.Is(err, ErrPending) {
		t.Fatalf("refund while a payout is in flight: %v", err)
	}
	// The receipt appears: the card is paid. This also works after a restart: only the journal is needed.
	node.receipts["0xabc"] = payoutReceipt(c, "0x1111111111111111111111111111111111111111", c.Units, 100)
	node.nonce = 1
	again, err := New(s.dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	again.chains = s.chains
	if p, err := again.Refresh(ctx, c.ID); err != nil || p.State != StatePaid {
		t.Fatalf("after restart: %v %+v", err, p)
	}
}

func TestRevertedOrForeignTransactionsAreTold(t *testing.T) { // C1, UL-06
	s, node := newService(t)
	ctx := context.Background()
	reverted, other := newCard(t, s), newCard(t, s)
	for id, hash := range map[string]string{reverted.ID: "0xdead", other.ID: "0xbeef"} {
		s.mu.Lock()
		s.records[id].Op = &Operation{Kind: "claim", To: "0x1111111111111111111111111111111111111111", Units: 1, Tx: hash, Started: s.now()}
		s.mu.Unlock()
	}
	node.receipts["0xdead"] = chain.Receipt{Found: true, Success: false}
	if p, _ := s.Refresh(ctx, reverted.ID); p.State == StatePaid || p.State == StateVerifying {
		t.Fatalf("a reverted payout: %s", p.State)
	}
	node.nonce = 1 // the Safe's nonce moved, our transaction is nowhere: someone else's signature was executed
	if p, _ := s.Refresh(ctx, other.ID); p.State != StateSettledOther {
		t.Fatalf("a payout we did not relay: %s", p.State)
	}
}

func TestRestoreRefusesAnotherMasterSecret(t *testing.T) { // UL-09
	s, _ := newService(t)
	newCard(t, s)
	other := make([]byte, 32)
	rand.Read(other)
	if _, err := New(s.dir, other, []Chain{s.chains[evm.AssetUSDTPolygon]}); err == nil {
		t.Fatal("restored the cards with a different master secret")
	}
}

// attemptNoSession reads an attempt without a session check (tests only: the browser knows its own attempt).
func (s *Service) attemptNoSession(authID string) (*Authorization, *Record, error) {
	a, err := s.getAuth(authID)
	if err != nil {
		return nil, nil, err
	}
	r, err := s.get(a.CardID)
	return a, r, err
}

func TestWorkingCopiesNeverShareOperations(t *testing.T) { // R2-01
	s, _ := newService(t)
	c := newCard(t, s)
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "claim", Tx: "0x1", Started: s.now()}
	s.mu.Unlock()
	r := mustGet(t, s, c.ID)
	r.Op.Done, r.Op.Tx = true, "0xchanged"
	if again := mustGet(t, s, c.ID); again.Op.Done || again.Op.Tx != "0x1" {
		t.Fatal("a working copy changed the stored operation without being written")
	}
}

func TestRefundInFlightAfterPaymentIsNotHidden(t *testing.T) { // R2-03
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance, node.nonce = big.NewInt(500_000), 1
	s.mu.Lock()
	rec := s.records[c.ID]
	rec.ClaimTx, rec.Settled = "0xpaid", 1
	rec.Op = &Operation{Kind: "refund", To: dest1, Units: 400_000, Nonce: 1, Tx: "0xrefund", Started: s.now()}
	s.mu.Unlock()
	if _, _, _, err := s.Refund(ctx, c.ID, c.SenderToken, dest1); !errors.Is(err, ErrPending) {
		t.Fatalf("a second refund while one is in flight: %v", err)
	}
	if _, err := s.RefundClaim(ctx, c.ID, c.SenderToken, nil); !errors.Is(err, ErrPending) && !errors.Is(err, ErrForbidden) {
		t.Fatalf("relay while in flight: %v", err)
	}
	if in, _ := s.Status(ctx, c.ID); in.State != StatePaid || !in.Recovering {
		t.Fatalf("status %+v: paid, but a recovery is in flight and must be said", in)
	}
}

func TestReleasedOperationIsStillRecognisedWhenItMinesLate(t *testing.T) { // R2-04
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "claim", To: dest1, Units: c.Units, Tx: "0xslow", Started: s.now().Add(-dropAfter - time.Minute)}
	s.mu.Unlock()
	p, _ := s.Refresh(ctx, c.ID) // never mined, nonce still free, past dropAfter: released, but remembered
	if p.State == StateVerifying || p.State == StatePaid {
		t.Fatalf("state %s", p.State)
	}
	if r := mustGet(t, s, c.ID); len(r.Late) != 1 || r.Late[0].Tx != "0xslow" {
		t.Fatalf("the released hash is not watched: %+v", r.Late)
	}
	node.receipts["0xslow"] = payoutReceipt(c, dest1, c.Units, 100) // it mines after all
	node.nonce, node.balance = 1, big.NewInt(0)
	if p, _ := s.Refresh(ctx, c.ID); p.State != StatePaid || p.TxHash != "0xslow" {
		t.Fatalf("a late payout was not recognised: %+v", p)
	}
}

func TestGasMoneyForASenderWithOnlyTheToken(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	relayerAddr := cl.MustAddress(c.Relayer)
	node.native[relayerAddr] = big.NewInt(1_000_000_000_000_000_000) // the relayer holds 1 POL
	sender := "0x4444444444444444444444444444444444444444"

	res, err := s.GasTopUp(ctx, c.ID, c.SenderToken, sender)
	if err != nil || res.Tx == "" || res.Already {
		t.Fatalf("top-up: %v %+v", err, res)
	}
	// 100k gas at 100 gwei, with a 50 % margin: 0.015 POL, below the cap.
	if res.Wei.Cmp(big.NewInt(15_000_000_000_000_000)) != 0 {
		t.Fatalf("gave %v wei", res.Wei)
	}
	if node.sent != 1 {
		t.Fatalf("%d transactions sent", node.sent)
	}
	if r := mustGet(t, s, c.ID); r.Drip == nil || r.Drip.Tx != res.Tx {
		t.Fatal("the transaction hash was not journaled")
	}
	// Once per card: the same answer, nothing more sent.
	if again, err := s.GasTopUp(ctx, c.ID, c.SenderToken, sender); err != nil || !again.Already || again.Tx != res.Tx || node.sent != 1 {
		t.Fatalf("second request: %v %+v sent %d", err, again, node.sent)
	}
	// The same address cannot drain the relayer through another card within a day.
	if _, err := func() (DripResult, error) { o := newCard(t, s); return s.GasTopUp(ctx, o.ID, o.SenderToken, sender) }(); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("same address again: %v", err)
	}
}

func TestGasMoneyNeedsTheSendersToken(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	node.native[cl.MustAddress(c.Relayer)] = big.NewInt(1_000_000_000_000_000_000)
	if _, err := s.GasTopUp(context.Background(), c.ID, "not-the-token", "0x4444444444444444444444444444444444444444"); !errors.Is(err, ErrForbidden) || node.sent != 0 {
		t.Fatalf("a caller without the sender's token: %v (sent %d)", err, node.sent)
	}
}

func TestGasMoneyGates(t *testing.T) {
	s, node := newService(t)
	ctx := context.Background()
	c := newCard(t, s)
	relayerAddr := cl.MustAddress(c.Relayer)
	sender := "0x5555555555555555555555555555555555555555"
	// The relayer holds less than its floor plus the drip: nothing is sent, and nothing stays recorded.
	node.native[relayerAddr] = big.NewInt(100_000_000_000_000_000)
	if _, err := s.GasTopUp(ctx, c.ID, c.SenderToken, sender); err == nil || node.sent != 0 {
		t.Fatalf("a drip below the float floor: %v (sent %d)", err, node.sent)
	}
	if r := mustGet(t, s, c.ID); r.Drip != nil {
		t.Fatal("a refused drip stayed recorded")
	}
	node.native[relayerAddr] = big.NewInt(1_000_000_000_000_000_000)
	// A sender who already has enough gas needs nothing.
	node.native[cl.MustAddress(sender)] = big.NewInt(1_000_000_000_000_000_000)
	if _, err := s.GasTopUp(ctx, c.ID, c.SenderToken, sender); !errors.Is(err, ErrGasNotNeeded) {
		t.Fatalf("a sender with gas: %v", err)
	}
	// Addresses that can never be a sender's wallet.
	for _, bad := range []string{"0x0000000000000000000000000000000000000000", c.Safe, c.Token, c.Relayer} {
		if _, err := s.GasTopUp(ctx, c.ID, c.SenderToken, bad); err == nil {
			t.Fatalf("%s accepted", bad)
		}
	}
	// Only an unfunded card needs the sender's payment.
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	delete(node.native, cl.MustAddress(sender))
	if _, err := s.GasTopUp(ctx, c.ID, c.SenderToken, "0x6666666666666666666666666666666666666666"); !errors.Is(err, ErrAuthState) {
		t.Fatalf("a funded card: %v", err)
	}
}

func TestAPayoutIsPaidOnlyWhenDeepAndVerified(t *testing.T) { // R3-01
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	_, _ = s.Refresh(ctx, c.ID)
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "claim", To: dest1, Units: c.Units, Tx: "0xdeep", Started: s.now()}
	s.mu.Unlock()
	node.balance, node.nonce = big.NewInt(0), 1
	node.receipts["0xdeep"] = payoutReceipt(c, dest1, c.Units, 990) // mined, but only 11 blocks deep (Polygon needs 64)
	node.block = 1000
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateVerifying {
		t.Fatalf("a payout that is not deep yet: %s", p.State)
	}
	node.block = 990 + 64 - 2 // one block short (the receipt's own block counts as the first confirmation)
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateVerifying {
		t.Fatalf("one block short: %s", p.State)
	}
	node.block = 990 + 64 - 1
	if p, _ := s.Refresh(ctx, c.ID); p.State != StatePaid {
		t.Fatalf("deep enough: %s", p.State)
	}
}

func TestASuccessfulReceiptWithoutTheTransfersIsNeverPaid(t *testing.T) { // R3-01, a token that returns false
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	_, _ = s.Refresh(ctx, c.ID)
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "claim", To: dest1, Units: c.Units, Tx: "0xnologs", Started: s.now()}
	s.mu.Unlock()
	node.nonce = 1
	node.receipts["0xnologs"] = chain.Receipt{Found: true, Success: true, Block: 100} // succeeded, but no Transfer events
	p, _ := s.Refresh(ctx, c.ID)
	in, _ := s.Status(ctx, c.ID)
	if p.State == StatePaid || in.Anomaly == "" {
		t.Fatalf("state %s, anomaly %q: must not be paid and must be flagged", p.State, in.Anomaly)
	}
	// the transfers of ANOTHER recipient do not count either
	node.receipts["0xnologs"] = payoutReceipt(c, "0x5555555555555555555555555555555555555555", c.Units, 100)
	if p, _ := s.Refresh(ctx, c.ID); p.State == StatePaid {
		t.Fatal("a payout to someone else was accepted as this claim")
	}
}

func TestFundingMustBeDeepToCount(t *testing.T) { // R3-01
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits)) // the latest block shows the deposit...
	node.deep = big.NewInt(0)                              // ...a deep block does not (it could still be reorganised away)
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateAwaitingFunds {
		t.Fatalf("a deposit that is not deep: %s", p.State)
	}
	if _, err := s.Prepare(ctx, randomHex(32), c.ID, dest1, randomHex(16), ""); !errors.Is(err, ErrNotFunded) {
		t.Fatalf("an attempt on a card whose funding is not deep: %v", err)
	}
	node.deep = nil // now the deep block shows it too
	if p, _ := s.Refresh(ctx, c.ID); p.State != StateReady {
		t.Fatalf("a deep deposit: %s", p.State)
	}
}

func TestGasQuotasHoldUnderConcurrency(t *testing.T) { // R3-03
	s, node := newService(t)
	ctx := context.Background()
	var cards []Created
	for i := 0; i < 6; i++ {
		cards = append(cards, newCard(t, s))
	}
	node.native[cl.MustAddress(cards[0].Relayer)] = new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000_000_000_000_000))
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i, c := range cards {
		wg.Add(1)
		go func(i int, c Created) {
			defer wg.Done()
			addr := "0x" + strings.Repeat(string(rune('a'+i)), 40)
			if _, err := s.GasTopUp(ctx, c.ID, c.SenderToken, addr); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(i, c)
	}
	wg.Wait()
	if ok != MaxDripsPerHour || node.sent != MaxDripsPerHour {
		t.Fatalf("%d drips given, %d sent, the hourly limit is %d", ok, node.sent, MaxDripsPerHour)
	}
}

func TestGasDripsPendingAreCountedInTheFloor(t *testing.T) { // R3-03
	s, node := newService(t)
	ctx := context.Background()
	a, b := newCard(t, s), newCard(t, s)
	// floor 0.2 + one drip of 0.015 + its fee: enough for ONE drip, not for two while the first is still pending
	node.native[cl.MustAddress(a.Relayer)] = big.NewInt(220_000_000_000_000_000)
	if _, err := s.GasTopUp(ctx, a.ID, a.SenderToken, "0x4444444444444444444444444444444444444444"); err != nil {
		t.Fatalf("first drip: %v", err)
	}
	// the node still shows the same balance (the first transaction is not mined): the second must be refused
	if _, err := s.GasTopUp(ctx, b.ID, b.SenderToken, "0x5555555555555555555555555555555555555555"); err == nil {
		t.Fatal("a second drip passed although the first was still pending against the same float")
	}
}

func TestGasIsOffUnlessTheChainIsConfiguredForIt(t *testing.T) { // the separate switch
	s, _ := newService(t)
	c := newCard(t, s)
	ch := s.chains[evm.AssetUSDTPolygon]
	ch.DripCap, ch.Floor = nil, nil
	s.chains[evm.AssetUSDTPolygon] = ch
	if _, err := s.GasTopUp(context.Background(), c.ID, c.SenderToken, "0x4444444444444444444444444444444444444444"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("gas money without its switch: %v", err)
	}
}

func TestNoNewCardWhenTheRelayerCannotPayForIt(t *testing.T) { // the float guard
	s, node := newService(t)
	relayerAddr := s.chains[evm.AssetUSDTPolygon].Relayer.Address()
	_, link, _ := cl.GenerateKey()
	in := Input{Asset: evm.AssetUSDTPolygon, CardID: "00112233445566778899aabbccddeeff", LinkOwner: link, Amount: "1", Protected: true, Email: "a@example.com"}
	node.native[relayerAddr] = big.NewInt(100_000_000_000_000_000) // 0.1 POL, below the 0.2 minimum
	if _, err := s.Create(in); !errors.Is(err, ErrFloatLow) {
		t.Fatalf("a card was accepted while the relayer could not pay for its claim: %v", err)
	}
	if !s.FloatStatus(context.Background())[evm.AssetUSDTPolygon] {
		t.Fatal("the health view must say the float is low")
	}
	node.native[relayerAddr] = big.NewInt(300_000_000_000_000_000)
	if _, err := s.Create(in); err != nil {
		t.Fatalf("with enough float: %v", err)
	}
}

func TestGasMoneyCanNeverCostMoreThanItsSeed(t *testing.T) { // the owner's rule: the app must never lose
	s, node := newService(t)
	ctx := context.Background()
	ch := s.chains[evm.AssetUSDTPolygon]
	ch.GasSeed = 2
	s.chains[evm.AssetUSDTPolygon] = ch
	node.native[ch.Relayer.Address()] = new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000_000_000_000_000))
	clock := time.Now()
	s.now = func() time.Time { return clock }
	give := func(i int) (Created, error) {
		clock = clock.Add(3 * time.Hour) // the hourly limit is a different protection: keep it out of the way
		c := newCard(t, s)
		_, err := s.GasTopUp(ctx, c.ID, c.SenderToken, fmt.Sprintf("0x%040x", 0xabc000+i))
		return c, err
	}
	for i := 0; i < 2; i++ {
		if _, err := give(i); err != nil {
			t.Fatalf("gift %d within the seed: %v", i, err)
		}
	}
	// Cards that are cancelled or abandoned earn nothing: the third gift is refused.
	if _, err := give(2); !errors.Is(err, ErrGasCredit) {
		t.Fatalf("a gift beyond the seed: %v", err)
	}
	if node.sent != 2 {
		t.Fatalf("%d transactions sent: nothing may be sent without a credit", node.sent)
	}
	// A completed payout earns credits: now gifts are possible again, but only that many.
	r := mustGet(t, s, mustCard(t, s).ID)
	r.Settled = 1
	if err := s.update(r); err != nil {
		t.Fatal(err)
	}
	for i := 3; i < 3+CreditsPerPayout; i++ {
		if _, err := give(i); err != nil {
			t.Fatalf("gift %d with the credits of one payout: %v", i, err)
		}
	}
	if _, err := give(20); !errors.Is(err, ErrGasCredit) {
		t.Fatalf("credits of one payout are %d gifts, not more: %v", CreditsPerPayout, err)
	}
}

// mustCard returns any existing card record (used to attach a completed payout to a card).
func mustCard(t *testing.T, s *Service) *Record {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.records {
		return r.clone()
	}
	t.Fatal("no card")
	return nil
}

func TestGasLightIsGreenOnlyWhenPhoenixWireCanPay(t *testing.T) {
	s, node := newService(t)
	ctx := context.Background()
	asset := evm.AssetUSDTPolygon
	relayerAddr := s.chains[asset].Relayer.Address()
	node.native[relayerAddr] = new(big.Int).Mul(big.NewInt(5), big.NewInt(1_000_000_000_000_000_000))
	if ok, why := s.GasStatus(ctx, asset); !ok || why != "ok" {
		t.Fatalf("with credit and float: %v %s", ok, why)
	}
	node.native[relayerAddr] = big.NewInt(100_000_000_000_000_000)
	if ok, why := s.GasStatus(ctx, asset); ok || why != "float" {
		t.Fatalf("with a low float: %v %s", ok, why)
	}
	node.native[relayerAddr] = new(big.Int).Mul(big.NewInt(5), big.NewInt(1_000_000_000_000_000_000))
	ch := s.chains[asset]
	ch.GasSeed = 1
	s.chains[asset] = ch
	c := newCard(t, s)
	if _, err := s.GasTopUp(ctx, c.ID, c.SenderToken, "0x4444444444444444444444444444444444444444"); err != nil {
		t.Fatal(err)
	}
	if ok, why := s.GasStatus(ctx, asset); ok || why != "credit" {
		t.Fatalf("with no credit left: %v %s", ok, why)
	}
	ch.DripCap, ch.Floor = nil, nil
	s.chains[asset] = ch
	if ok, why := s.GasStatus(ctx, asset); ok || why != "off" {
		t.Fatalf("with the feature off: %v %s", ok, why)
	}
}

func TestTheServiceFeeIsTheGasPlusSixPercent(t *testing.T) {
	// A chain with no price source: the flat gas fee of the chain (0.10 / 0.75) stands for the gas, and 6 % is ADDED to it.
	s, _ := newService(t)
	for _, c := range []struct {
		asset string
		units uint64
		want  uint64
	}{
		{evm.AssetUSDTPolygon, 1_000_000, 160_000},      // 1 USDT: 0.10 gas + 0.06
		{evm.AssetUSDTPolygon, 10_000_000, 700_000},     // 10 USDT: 0.10 + 0.60
		{evm.AssetUSDTPolygon, 100_000_000, 6_100_000},  // 100 USDT: 0.10 + 6
		{evm.AssetUSDTPolygon, 500_000_000, 30_100_000}, // 500 USDT: 0.10 + 30
		{evm.AssetUSDTPolygon, 1_234_567, 174_075},      // the percentage is rounded up: 74 074.02 -> 74 075
		{evm.AssetUSDTPolygon, 3_333_333, 300_000},      // 6 % of 3.333333 = 0.19999998 -> 0.200000
	} {
		fee, refund, err := s.quote(context.Background(), c.asset, s.chains[evm.AssetUSDTPolygon], c.units)
		if err != nil || fee != c.want || refund != fee {
			t.Errorf("%s %d: fee %d refund %d (%v), want %d", c.asset, c.units, fee, refund, err, c.want)
		}
	}
	if MaxFeeFor(1_000_000) != 10_000_000 || MaxFeeFor(1_000_000_000) != 100_000_000 {
		t.Fatal("the highest fee is 10 units or a tenth of the amount")
	}
}

func TestARefundKeepsTheWholeFeeForTheRelayer(t *testing.T) {
	s, node := newService(t)
	id := make([]byte, 16)
	rand.Read(id)
	key, link, _ := cl.GenerateKey()
	linkKeys.Store(hex.EncodeToString(id), key)
	c, err := s.Create(Input{Asset: evm.AssetUSDTPolygon, CardID: hex.EncodeToString(id), LinkOwner: link, Amount: "500", Protected: true, Email: "friend@example.com"})
	if err != nil || c.FeeUnits != 30_100_000 {
		t.Fatalf("a 500 card: fee %d, %v", c.FeeUnits, err)
	}
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits)) // 530.1 paid
	_, _, units, err := s.Refund(context.Background(), c.ID, c.SenderToken, "0x4444444444444444444444444444444444444444")
	if err != nil || units != 500_000_000 {
		t.Fatalf("the sender takes back the amount and the relayer keeps the whole fee: got %d, %v", units, err)
	}
	if fee, _ := s.RefundFeeUnits(c.ID); fee != 30_100_000 {
		t.Fatalf("refund fee %d", fee)
	}
}

func TestTheCardColourComesFromItsIDLikeTheNimiqCards(t *testing.T) {
	// same table and rule as PhoenixMail (first byte of the card ID modulo twelve): these must never drift apart
	for id, want := range map[string][2]string{"00" + "0123456789abcdef0123456789ab": {"#deefff", "#9fc9eb"}, "02ff": {"#dcf5e9", "#9bd8bc"}, "0b": {"#e2f4ef", "#9ed5c6"}, "0c": {"#deefff", "#9fc9eb"}, "ff": {"#ffe2ee", "#efaac7"}, "": {"#deefff", "#9fc9eb"}} {
		if got := Palette(id); got != want {
			t.Errorf("%q: %v, want %v", id, got, want)
		}
	}
}

func TestTheOwnerSeesTheRelayerAndCanTakeOutWhatIsNotNeeded(t *testing.T) {
	s, node := newService(t)
	ctx := context.Background()
	c := newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits)) // the card is funded (and the fake token balance of the relayer is the same number)
	_, _ = s.Refresh(ctx, c.ID)
	infos, err := s.RelayerInfos(ctx)
	if err != nil || len(infos) != 1 {
		t.Fatalf("relayers: %v %v", infos, err)
	}
	i := infos[0]
	if i.Chain != "polygon" || i.NativeSymbol != "POL" || i.NativeWei != "5000000000000000000" || len(i.Tokens) != 1 || i.Tokens[0].Units != "2750000" || i.PendingCards != 1 || i.LiveCards != 1 {
		t.Fatalf("relayer info: %+v", i)
	}
	// one live card keeps gas back (0.15 POL) on top of the minimum float (0.2 POL)
	if i.ReserveWei != "350000000000000000" || i.WithdrawWei != "4650000000000000000" {
		t.Fatalf("reserve %s, withdrawable %s", i.ReserveWei, i.WithdrawWei)
	}
	to := "0x4444444444444444444444444444444444444444"
	// no authorized destination: no withdrawal at all
	if _, err := s.RelayerWithdraw(ctx, "polygon", "native", to); !errors.Is(err, ErrWithdrawDisabled) {
		t.Fatalf("withdrawals must be off until the operator authorizes an address: %v", err)
	}
	s.SetWithdrawAddresses([]string{to})
	if got := s.WithdrawAddresses(); len(got) != 1 || got[0] != to {
		t.Fatalf("authorized addresses: %v", got)
	}
	if _, err := s.RelayerWithdraw(ctx, "polygon", "native", "0x5555555555555555555555555555555555555555"); !errors.Is(err, ErrWithdrawDestination) {
		t.Fatalf("an address that is not on the list: %v", err)
	}
	if _, err := s.RelayerWithdraw(ctx, "polygon", "native", "0x12"); !errors.Is(err, evm.ErrInvalidAddress) {
		t.Fatalf("a bad destination: %v", err)
	}
	if _, err := s.RelayerWithdraw(ctx, "solana", "native", to); !errors.Is(err, ErrNotFound) {
		t.Fatalf("an unknown chain: %v", err)
	}
	if _, err := s.RelayerWithdraw(ctx, "polygon", "usdt_ethereum", to); err == nil {
		t.Fatal("an asset of another chain was withdrawn")
	}
	// with a live card only the surplus goes: 5 POL - 0.35 reserve - the gas of the transfer (21000 x 100 gwei)
	res, err := s.RelayerWithdraw(ctx, "polygon", "native", to)
	if err != nil || res.Tx == "" || node.sent != 1 || res.Status != "sent" || res.Amount != "4647900000000000000" {
		t.Fatalf("native withdrawal: %+v %v (sent %d)", res, err, node.sent)
	}
	logged, err := os.ReadFile(filepath.Join(s.dir, "withdrawals.log"))
	if err != nil || !strings.Contains(string(logged), to) || !strings.Contains(string(logged), res.Tx) {
		t.Fatalf("every withdrawal is written down: %q %v", logged, err)
	}
	// less than the reserve on the relayer: nothing leaves
	node.native[s.chains[evm.AssetUSDTPolygon].Relayer.Address()] = big.NewInt(100_000_000_000_000_000) // 0.1 POL
	if _, err := s.RelayerWithdraw(ctx, "polygon", "native", to); !errors.Is(err, ErrNothingToWithdraw) {
		t.Fatalf("below the reserve: %v", err)
	}
	// with no live card left the whole balance may go
	node.native[s.chains[evm.AssetUSDTPolygon].Relayer.Address()] = big.NewInt(5_000_000_000_000_000_000)
	if _, err := s.Cancel(c.ID, c.SenderToken); err != nil {
		t.Fatal(err)
	}
	node.sent = 0
	res, err = s.RelayerWithdraw(ctx, "polygon", "native", to)
	if err != nil || node.sent != 1 || res.Amount == "" {
		t.Fatalf("an empty relayer of live cards may be swept: %+v %v", res, err)
	}
	// nothing of that token: refused with a clear error
	node.balance = big.NewInt(0)
	if _, err := s.RelayerWithdraw(ctx, "polygon", evm.AssetUSDTPolygon, to); !errors.Is(err, ErrNothingToWithdraw) {
		t.Fatalf("an empty token balance: %v", err)
	}
	// and the native coin of an empty relayer too
	node.native[s.chains[evm.AssetUSDTPolygon].Relayer.Address()] = big.NewInt(0)
	if _, err := s.RelayerWithdraw(ctx, "polygon", "native", to); !errors.Is(err, ErrNothingToWithdraw) {
		t.Fatalf("an empty relayer: %v", err)
	}
}

func TestAnExpiredGrantCannotStartANewBroadcast(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	session, authID, grant := authorize(t, s, c, dest1)
	if _, _, err := s.CosignClaim(context.Background(), session, authID, grant); err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Now().Add(GrantLife + time.Minute) }
	if _, _, err := s.ClaimAuthorized(context.Background(), session, authID, grant, nil); !errors.Is(err, ErrAuthExpired) {
		t.Fatalf("the five minutes of the grant also bind the broadcast: %v", err)
	}
}

func TestTheReserveFollowsTheMoneyNotTheDisplayedState(t *testing.T) {
	s, node := newService(t)
	ctx := context.Background()
	c := newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	_, _ = s.Refresh(ctx, c.ID)
	to := "0x4444444444444444444444444444444444444444"
	s.SetWithdrawAddresses([]string{to})
	// preparing a refund marks the card cancelled at once, long before the refund is signed and sent
	if _, _, _, err := s.Refund(ctx, c.ID, c.SenderToken, "0x5555555555555555555555555555555555555555"); err != nil {
		t.Fatal(err)
	}
	if r, _ := s.get(c.ID); s.state(r) != StateCancelled {
		t.Fatalf("state %s", s.state(r))
	}
	infos, _ := s.RelayerInfos(ctx)
	if infos[0].LiveCards != 1 || infos[0].ReserveWei != "350000000000000000" {
		t.Fatalf("a card being refunded still needs gas: %+v", infos[0])
	}
	// a token withdrawal also needs gas: it may not eat into that reserve
	rel := s.chains[evm.AssetUSDTPolygon].Relayer.Address()
	node.estimate = 65_000
	node.native[rel] = big.NewInt(200_000_000_000_000_000) // 0.2 POL, below the 0.35 reserve
	if _, err := s.RelayerWithdraw(ctx, "polygon", evm.AssetUSDTPolygon, to); !errors.Is(err, ErrReserveRequired) {
		t.Fatalf("a token withdrawal below the reserve: %v", err)
	}
	// a card that expired but still holds money is not forgotten either
	s2, node2 := newService(t)
	c2 := newCard(t, s2)
	node2.balance = big.NewInt(int64(c2.Units + c2.FeeUnits))
	_, _ = s2.Refresh(ctx, c2.ID)
	s2.now = func() time.Time { return time.Now().Add(Lifetime + time.Hour) }
	if r, _ := s2.get(c2.ID); s2.state(r) != StateExpired || !s2.needsGas(r) {
		t.Fatalf("an expired card that holds money still needs gas (state %s)", s2.state(r))
	}
}

func TestAnOldFailedOperationDoesNotExcuseAnExpiredGrant(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	session, authID, grant := authorize(t, s, c, dest1)
	if _, _, err := s.CosignClaim(context.Background(), session, authID, grant); err != nil {
		t.Fatal(err)
	}
	r, _ := s.get(c.ID)
	r.Op = &Operation{Kind: "claim", Failed: true, To: dest1} // a first try that died before any payout transaction existed
	if err := s.update(r); err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Now().Add(GrantLife + time.Minute) }
	if _, _, err := s.ClaimAuthorized(context.Background(), session, authID, grant, nil); !errors.Is(err, ErrAuthExpired) {
		t.Fatalf("an old failed operation must not replace the expiry check: %v", err)
	}
}

func TestMoneyOnACardCountsWhateverItsStateOrItsFunding(t *testing.T) {
	s, node := newService(t)
	ctx := context.Background()
	c := newCard(t, s)
	// a card meant for 2.50 (+ 0.15 fee) receives only half, then expires: not "funded", yet it holds recoverable money
	node.balance = big.NewInt(1_000_000)
	_, _ = s.Refresh(ctx, c.ID)
	s.now = func() time.Time { return time.Now().Add(Lifetime + time.Hour) }
	r, _ := s.get(c.ID)
	if s.state(r) != StateExpired || r.Funded || r.Held != 1_000_000 || !s.needsGas(r) {
		t.Fatalf("state %s funded %v held %d: a part-payment must be counted", s.state(r), r.Funded, r.Held)
	}
	// a card whose balance was never read is UNKNOWN, and an unknown counts
	r.Held, r.HeldAt = 0, time.Time{}
	if err := s.update(r); err != nil {
		t.Fatal(err)
	}
	if r, _ = s.get(c.ID); !s.needsGas(r) {
		t.Fatal("an unread balance must count")
	}
	// reading it (nothing there any more) lets it go; a node that cannot answer leaves it unknown
	node.balance = big.NewInt(0)
	s.refreshHeld(ctx, 137)
	if r, _ = s.get(c.ID); s.needsGas(r) || r.HeldAt.IsZero() {
		t.Fatalf("an expired card holding nothing, read just now, needs no gas (held %d at %v)", r.Held, r.HeldAt)
	}
}

func TestAnExpiredGrantOnlyLooksItNeverBroadcasts(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	session, authID, grant := authorize(t, s, c, dest1)
	if _, _, err := s.CosignClaim(ctx, session, authID, grant); err != nil {
		t.Fatal(err)
	}
	// a payout transaction journaled and still unknown: it looks in flight...
	r, _ := s.get(c.ID)
	r.Op = &Operation{Kind: "claim", To: dest1, Tx: "0xabc", Started: time.Now()}
	if err := s.update(r); err != nil {
		t.Fatal(err)
	}
	// ...and the refresh made by the call discovers that it was mined and reverted
	node.receipts["0xabc"] = chain.Receipt{Found: true, Success: false}
	s.now = func() time.Time { return time.Now().Add(GrantLife + time.Minute) }
	before := node.sent
	if _, state, err := s.ClaimAuthorized(ctx, session, authID, grant, nil); err != nil || state == StatePaid {
		t.Fatalf("a look at the payout: %q %v", state, err)
	}
	if node.sent != before {
		t.Fatal("an expired grant started a new broadcast")
	}
	if r, _ = s.get(c.ID); r.Op == nil || !r.Op.Failed {
		t.Fatal("the failure was discovered by the look")
	}
	// and now that the operation failed, the expired grant is refused outright
	if _, _, err := s.ClaimAuthorized(ctx, session, authID, grant, nil); !errors.Is(err, ErrAuthExpired) {
		t.Fatalf("after the failure: %v", err)
	}
}

func TestOldAttemptsArePrunedAndTheFileIsWrittenAgain(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	authorize(t, s, c, dest1)
	if len(s.auths) == 0 {
		t.Fatal("no attempt was stored")
	}
	s.now = func() time.Time { return time.Now().Add(4 * time.Hour) }
	s.pruneAuths()
	if len(s.auths) != 0 {
		t.Fatalf("%d attempts survived four hours", len(s.auths))
	}
	raw, err := os.ReadFile(filepath.Join(s.dir, "auths.json"))
	if err != nil || strings.Contains(string(raw), "grant_hash") {
		t.Fatalf("the file must be rewritten without them: %v %q", err, raw)
	}
}

func TestAFailedReadMakesAKnownZeroUnknownAgain(t *testing.T) {
	s, node := newService(t)
	ctx := context.Background()
	c := newCard(t, s)
	to := "0x4444444444444444444444444444444444444444"
	s.SetWithdrawAddresses([]string{to})
	// an expired card, read as empty: it does not count...
	node.balance = big.NewInt(0)
	s.now = func() time.Time { return time.Now().Add(Lifetime + time.Hour) }
	if err := s.refreshHeld(ctx, 137); err != nil {
		t.Fatal(err)
	}
	if r, _ := s.get(c.ID); s.needsGas(r) {
		t.Fatal("an expired card read as empty needs no gas")
	}
	// ...then a part-payment arrives and the next read fails: the old zero must not be believed
	node.balance = big.NewInt(1_000_000)
	node.failBalance = true
	if err := s.refreshHeld(ctx, 137); err != nil {
		t.Fatalf("a failed read is recorded as unknown, it is not an error: %v", err)
	}
	r, _ := s.get(c.ID)
	if !r.HeldAt.IsZero() || !s.needsGas(r) {
		t.Fatalf("the old zero (read at %v) must be forgotten: the card is unknown and counts", r.HeldAt)
	}
	// and the withdrawal keeps the reserve of that unknown card
	node.failBalance = false
	node.balance = big.NewInt(0)
	infos, _ := s.RelayerInfos(ctx)
	if infos[0].LiveCards != 0 {
		t.Fatalf("after a good read of an empty card nothing is live: %+v", infos[0])
	}
}

func TestWithdrawalStopsWhenABalanceCannotBeRecorded(t *testing.T) {
	s, node := newService(t)
	ctx := context.Background()
	c := newCard(t, s)
	_ = c
	to := "0x4444444444444444444444444444444444444444"
	s.SetWithdrawAddresses([]string{to})
	node.balance = big.NewInt(0)
	s.now = func() time.Time { return time.Now().Add(Lifetime + time.Hour) }
	// the state directory cannot be written: the new reading cannot be recorded
	if err := os.Chmod(s.dir, 0o500); err != nil {
		t.Skip("cannot make the directory read-only")
	}
	defer os.Chmod(s.dir, 0o700)
	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere")
	}
	if _, err := s.RelayerWithdraw(ctx, "polygon", "native", to); !errors.Is(err, ErrBalanceUnknown) {
		t.Fatalf("a balance that cannot be recorded must stop the withdrawal: %v", err)
	}
	if node.sent != 0 {
		t.Fatal("something was sent")
	}
}

func TestAPruneThatCannotBeWrittenIsRetried(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	authorize(t, s, c, dest1)
	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere")
	}
	s.now = func() time.Time { return time.Now().Add(4 * time.Hour) }
	if err := os.Chmod(s.dir, 0o500); err != nil {
		t.Skip("cannot make the directory read-only")
	}
	s.pruneAuths() // the memory is pruned, the file cannot be written
	os.Chmod(s.dir, 0o700)
	raw, _ := os.ReadFile(filepath.Join(s.dir, "auths.json"))
	if !strings.Contains(string(raw), "grant_hash") || !s.authsDirty {
		t.Fatalf("the old file must still be there and the retry pending (dirty %v)", s.authsDirty)
	}
	s.pruneAuths() // the next tick writes it
	raw, _ = os.ReadFile(filepath.Join(s.dir, "auths.json"))
	if strings.Contains(string(raw), "grant_hash") || s.authsDirty {
		t.Fatalf("the retry did not rewrite the file (dirty %v)", s.authsDirty)
	}
}

func claimedAtOf(t *testing.T, s *Service, id string) *time.Time {
	t.Helper()
	in, err := s.Status(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return in.ClaimedAt
}

// The claim date is the timestamp of the block that holds the confirmed payout: not the time of the look, never invented.
func TestClaimDateIsTheBlockTimeOfTheConfirmedPayout(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	mined := time.Date(2026, 10, 23, 14, 5, 9, 0, time.UTC)
	node.times = map[uint64]time.Time{100: mined, 101: mined.Add(time.Hour)}
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	_, _ = s.Refresh(ctx, c.ID)
	if claimedAtOf(t, s, c.ID) != nil {
		t.Fatal("no claim date before any claim")
	}
	// a payout that was sent but is not confirmed (no receipt yet): no date
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "claim", To: dest1, Units: c.Units, Tx: "0xclaim", Started: s.now()}
	s.mu.Unlock()
	if claimedAtOf(t, s, c.ID) != nil {
		t.Fatal("a payout that is not mined has no date")
	}
	// mined but without the transfers it must make: an anomaly is never a claim
	node.nonce = 1
	node.receipts["0xclaim"] = chain.Receipt{Found: true, Success: true, Block: 100}
	if claimedAtOf(t, s, c.ID) != nil {
		t.Fatal("an anomaly is never a claim")
	}
	s.mu.Lock()
	s.records[c.ID].Anomaly = ""
	s.mu.Unlock()
	// mined and reverted: nothing was paid
	node.receipts["0xclaim"] = chain.Receipt{Found: true, Success: false, Block: 100}
	if claimedAtOf(t, s, c.ID) != nil {
		t.Fatal("a reverted payout has no date")
	}
	// confirmed: the date is the block's, whatever the clock says now
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "claim", To: dest1, Units: c.Units, Tx: "0xclaim", Started: s.now()}
	s.records[c.ID].Anomaly = ""
	s.mu.Unlock()
	node.receipts["0xclaim"] = payoutReceipt(c, dest1, c.Units, 100)
	node.nonce, node.balance = 1, big.NewInt(0)
	at := claimedAtOf(t, s, c.ID)
	if at == nil || !at.Equal(mined) || at.Location() != time.UTC {
		t.Fatalf("claimed_at = %v, want %v (UTC)", at, mined)
	}
	// it is kept: a later look, with the block's time now unknown, changes nothing
	node.timeErr = errors.New("rpc down")
	if again := claimedAtOf(t, s, c.ID); again == nil || !again.Equal(mined) {
		t.Fatalf("the date must be stable: %v", again)
	}
}

func TestClaimDateIsNotLostWhenTheNodeCannotDateTheBlockYet(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	_, _ = s.Refresh(ctx, c.ID)
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "claim", To: dest1, Units: c.Units, Tx: "0xclaim", Started: s.now()}
	s.mu.Unlock()
	node.receipts["0xclaim"] = payoutReceipt(c, dest1, c.Units, 100)
	node.nonce, node.balance = 1, big.NewInt(0)
	node.timeErr = errors.New("rpc down")
	if in, _ := s.Status(ctx, c.ID); in.State == StatePaid || in.ClaimedAt != nil || !in.Stale {
		t.Fatalf("not settled while its date cannot be read, and said so: %+v", in)
	}
	mined := time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC)
	node.timeErr, node.times = nil, map[uint64]time.Time{100: mined}
	if in, _ := s.Status(ctx, c.ID); in.State != StatePaid || in.ClaimedAt == nil || !in.ClaimedAt.Equal(mined) {
		t.Fatalf("settled with its date once the node can say: %+v", in)
	}
}

// A refund is not a claim; a card paid before the date was kept gets it from the receipt of its payout, or stays without.
func TestRefundIsNotAClaimAndOldCardsAreNotInvented(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	mined := time.Date(2026, 10, 22, 20, 0, 0, 0, time.UTC)
	node.times = map[uint64]time.Time{100: mined}
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	_, _ = s.Refresh(ctx, c.ID)
	s.mu.Lock()
	s.records[c.ID].Op = &Operation{Kind: "refund", To: dest2, Units: c.Units, Tx: "0xrefund", Started: s.now()}
	s.mu.Unlock()
	node.nonce = 1
	node.receipts["0xrefund"] = payoutReceipt(c, dest2, c.Units, 100)
	node.balance = big.NewInt(0)
	if claimedAtOf(t, s, c.ID) != nil {
		t.Fatal("a refund must not become a claim")
	}
	// an old card: paid (claim_tx kept) with no date. The receipt of that payout dates it.
	old := newCard(t, s)
	node.balance = big.NewInt(int64(old.Units + old.FeeUnits))
	_, _ = s.Refresh(ctx, old.ID)
	s.mu.Lock()
	s.records[old.ID].ClaimTx = "0xold"
	s.records[old.ID].Op = &Operation{Kind: "claim", To: dest1, Units: old.Units, Tx: "0xold", Done: true}
	s.records[old.ID].Settled = 1
	s.mu.Unlock()
	node.nonce, node.balance = 1, big.NewInt(0)
	node.receipts["0xold"] = payoutReceipt(old, dest1, old.Units, 100)
	if at := claimedAtOf(t, s, old.ID); at == nil || !at.Equal(mined) {
		t.Fatalf("an old card is dated from its receipt: %v", at)
	}
	// an old card whose node cannot date blocks: no date, nothing invented
	older := newCard(t, s)
	node.balance = big.NewInt(int64(older.Units + older.FeeUnits))
	_, _ = s.Refresh(ctx, older.ID)
	s.mu.Lock()
	s.records[older.ID].ClaimTx = "0xolder"
	s.records[older.ID].Op = &Operation{Kind: "claim", To: dest1, Units: older.Units, Tx: "0xolder", Done: true}
	s.records[older.ID].Settled = 1
	s.mu.Unlock()
	node.receipts["0xolder"] = payoutReceipt(older, dest1, older.Units, 100)
	node.balance = big.NewInt(0)
	chainCfg := s.chains[evm.AssetUSDTPolygon]
	chainCfg.Node = plainNode{node}
	s.chains[evm.AssetUSDTPolygon] = chainCfg
	if at := claimedAtOf(t, s, older.ID); at != nil {
		t.Fatalf("no date when the node cannot say: %v", at)
	}
}

func TestDecliningARefundDoesNotCancelTheCard(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	node.balance = big.NewInt(int64(c.Units + c.FeeUnits))
	// the sender asks for the co-signature of a refund, then does not confirm it
	if _, _, _, err := s.Refund(ctx, c.ID, c.SenderToken, "0x4444444444444444444444444444444444444444"); err != nil {
		t.Fatal(err)
	}
	if in, _ := s.Status(ctx, c.ID); in.State != StateCancelled {
		t.Fatal("the card must be held back while the refund is being confirmed", in.State)
	}
	// the lease runs out: the card is his recipient's again, and still the sender's to refund
	s.now = func() time.Time { return time.Now().Add(refundLease + time.Minute) }
	if in, _ := s.Status(ctx, c.ID); in.State == StateCancelled {
		t.Fatal("declining a refund cancelled the card for good", in.State)
	}
	if _, _, _, err := s.Refund(ctx, c.ID, c.SenderToken, "0x4444444444444444444444444444444444444444"); err != nil {
		t.Fatal("the same refund can be asked again", err)
	}
	// a card the sender cancelled himself stays cancelled
	s2, _ := newService(t)
	c2 := newCard(t, s2)
	if _, err := s2.Cancel(c2.ID, c2.SenderToken); err != nil {
		t.Fatal(err)
	}
	s2.now = func() time.Time { return time.Now().Add(refundLease + time.Hour) }
	if in, _ := s2.Status(ctx, c2.ID); in.State != StateCancelled {
		t.Fatal("a card cancelled by its sender must stay cancelled", in.State)
	}
}
