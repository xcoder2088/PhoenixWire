package evmcard

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evmtest"
)

const (
	recipient = "0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb"
	sender    = "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"
	stranger  = "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359"
)

type rig struct {
	svc   *Service
	chain *evmtest.Chain
	now   time.Time
	dir   string
	net   evm.Network
}

func newRig(t *testing.T, asset string) *rig {
	t.Helper()
	n, _ := evm.Lookup(asset)
	chain := evmtest.NewChain(t, n.ChainID)
	server := chain.Server()
	t.Cleanup(server.Close)
	dir := t.TempDir() + "/evm"
	svc, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := &rig{svc: svc, chain: chain, now: time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC), dir: dir, net: n}
	svc.now = func() time.Time { return r.now }
	svc.UseClients(func(net evm.Network) *evm.Client {
		net.RPCs = []string{server.URL, server.URL}
		c := evm.NewClient(net)
		c.AllowHTTPForTests()
		return c
	})
	return r
}

func TestFullFlowOpenCard(t *testing.T) {
	for _, asset := range []string{evm.AssetUSDTPolygon, evm.AssetUSDTEthereum} {
		r := newRig(t, asset)
		ctx := context.Background()
		c, err := r.svc.Create(Input{Asset: asset, Amount: "2.5", Message: "merci", SenderAddress: sender, SenderEmail: "me@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		if c.State != StateAwaiting || c.Units != 2_500_000 || c.Amount != "2.5" || c.SenderToken == "" || c.Protected || c.Recipient != "" || c.HasRecipient {
			t.Fatalf("created: %+v", c)
		}
		// Nothing can be paid yet: nobody knows where.
		if st, _ := r.svc.Status(ctx, c.ID); st.State != StateAwaiting || st.Recipient != "" {
			t.Fatal("no address yet")
		}
		var told []Record
		r.svc.OnReady = func(rec Record) { told = append(told, rec) }
		pub, err := r.svc.SetRecipient(ctx, c.ID, strings.ToLower(recipient))
		if err != nil || pub.State != StateReady || !pub.HasRecipient {
			t.Fatalf("set recipient: %+v %v", pub, err)
		}
		time.Sleep(20 * time.Millisecond)
		if len(told) != 1 || told[0].SenderEmail != "me@example.com" || told[0].Recipient != recipient {
			t.Fatalf("the sender must be told once: %+v", told)
		}
		st, _ := r.svc.Status(ctx, c.ID)
		if st.Recipient != recipient || st.Units != 2_500_000 || st.Token != r.net.Token || st.ChainID != r.net.ChainID {
			t.Fatalf("issuer view: %+v", st)
		}
		if pubView, _ := r.svc.Public(c.ID); toJSON(pubView) != "" && strings.Contains(toJSON(pubView), recipient) {
			t.Fatal("the public view must not show the address")
		}
		// The address is final.
		if _, err := r.svc.SetRecipient(ctx, c.ID, stranger); !errors.Is(err, ErrAddressSet) {
			t.Fatalf("second address: %v", err)
		}
		// Paid from the wrong wallet, too little, to someone else: none counts.
		head := r.chain.Head()
		r.chain.Transfer(head+1, stranger, recipient, 2_500_000)
		r.chain.Transfer(head+2, sender, recipient, 2_499_999)
		r.chain.Transfer(head+3, sender, stranger, 2_500_000)
		r.chain.Mine(r.net.Confirmations + 5)
		if st, _ := r.svc.Status(ctx, c.ID); st.State != StateReady {
			t.Fatalf("an unrelated transfer must not count: %+v", st)
		}
		// The sender's transfer, but not yet deep enough.
		hash := r.chain.Transfer(r.chain.Head()+1, sender, recipient, 2_500_000)
		r.chain.Mine(r.net.Confirmations - 2)
		if st, _ := r.svc.Status(ctx, c.ID); st.State != StateReady {
			t.Fatal("not deep enough yet")
		}
		r.chain.Mine(3)
		st, err = r.svc.Status(ctx, c.ID)
		if err != nil || st.State != StatePaid || st.TxHash != hash || !strings.HasSuffix(st.TxURL, hash) {
			t.Fatalf("paid: %+v %v", st, err)
		}
		if pub, _ := r.svc.Refresh(ctx, c.ID); pub.State != StatePaid || pub.TxHash != hash {
			t.Fatal("the recipient's page sees it too")
		}
		if _, err := r.svc.Cancel(c.ID, c.SenderToken); !errors.Is(err, ErrForbidden) {
			t.Fatalf("a paid card cannot be cancelled: %v", err)
		}
	}
}

func TestAPaymentOnlyOneNodeReportsNeverCounts(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	ctx := context.Background()
	c, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "1"})
	r.svc.SetRecipient(ctx, c.ID, recipient)
	r.chain.PhantomTransfer(r.chain.Head()+1, sender, recipient, 1_000_000)
	r.chain.Mine(100)
	for i := 0; i < 8; i++ {
		if st, _ := r.svc.Status(ctx, c.ID); st.State != StateReady {
			t.Fatalf("a transfer the receipts do not confirm was taken for the payment: %+v", st)
		}
	}
	// ... and a real payment that comes later is still found once the phantom has been given up on.
	hash := r.chain.Transfer(r.chain.Head()+1, sender, recipient, 1_000_000)
	r.chain.Mine(100)
	if st, _ := r.svc.Status(ctx, c.ID); st.State != StatePaid || st.TxHash != hash {
		t.Fatalf("real payment after the phantom: %+v", st)
	}
}

func TestOverpaymentCountsAndAnyWalletWhenNoSenderAddress(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	ctx := context.Background()
	c, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "1"})
	r.svc.SetRecipient(ctx, c.ID, recipient)
	r.chain.Transfer(r.chain.Head()+1, stranger, recipient, 1_200_000) // from an exchange, say
	r.chain.Mine(100)
	if st, _ := r.svc.Status(ctx, c.ID); st.State != StatePaid {
		t.Fatalf("%+v", st)
	}
}

func TestOnlyNewBlocksAreScannedAndOldPaymentsDoNotCount(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	ctx := context.Background()
	// A transfer to the recipient BEFORE he gave his address (say, from somebody else) never counts.
	r.chain.Transfer(900, stranger, recipient, 5_000_000)
	c, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "1"})
	r.svc.SetRecipient(ctx, c.ID, recipient)
	r.chain.Mine(200)
	if st, _ := r.svc.Status(ctx, c.ID); st.State != StateReady {
		t.Fatal("an older transfer was taken for the payment")
	}
	before := r.chain.Calls["eth_getLogs"]
	r.chain.Mine(10)
	r.svc.Status(ctx, c.ID)
	if r.chain.Calls["eth_getLogs"] == before {
		t.Fatal("it should look again")
	}
	rec, _ := r.svc.get(c.ID)
	if rec.Scanned == 0 || rec.Scanned > r.chain.Head() {
		t.Fatalf("scan cursor: %d", rec.Scanned)
	}
}

func TestProtectedCardNeedsTheEmailCodeBeforeTheAddress(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	ctx := context.Background()
	c, err := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "5", Protected: true, Email: "friend@example.com"})
	if err != nil || !c.Protected {
		t.Fatal(c, err)
	}
	if _, err := r.svc.SetRecipient(ctx, c.ID, recipient); !errors.Is(err, ErrNotVerified) {
		t.Fatalf("protected card without the code: %v", err)
	}
	if e, err := r.svc.Recipient(c.ID); err != nil || e != "friend@example.com" {
		t.Fatal(e, err)
	}
	if err := r.svc.MarkVerified(c.ID); err != nil {
		t.Fatal(err)
	}
	if p, _ := r.svc.Public(c.ID); !p.Verified {
		t.Fatal("verified flag")
	}
	if _, err := r.svc.SetRecipient(ctx, c.ID, recipient); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCardHasNoVerification(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	c, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "1", Email: "ignored@example.com"})
	if err := r.svc.MarkVerified(c.ID); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := r.svc.Recipient(c.ID); !errors.Is(err, ErrForbidden) {
		t.Fatal("an open card has no recipient email")
	}
}

func TestExpiryCancelAndBadAddresses(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	ctx := context.Background()
	c, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "1", SenderAddress: sender})
	for _, bad := range []string{"", "0x123", "0x0000000000000000000000000000000000000000", "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAeD", sender, r.net.Token} {
		if _, err := r.svc.SetRecipient(ctx, c.ID, bad); err == nil {
			t.Fatalf("address %q accepted", bad)
		}
	}
	if _, err := r.svc.Cancel(c.ID, "wrong"); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	c2, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "1"})
	if p, err := r.svc.Cancel(c2.ID, c2.SenderToken); err != nil || p.State != StateCancelled {
		t.Fatal(p, err)
	}
	if _, err := r.svc.SetRecipient(ctx, c2.ID, recipient); !errors.Is(err, ErrCancelled) {
		t.Fatalf("cancelled card: %v", err)
	}
	r.now = r.now.Add(8 * 24 * time.Hour)
	if p, _ := r.svc.Public(c.ID); p.State != StateExpired {
		t.Fatalf("a week without an address: %s", p.State)
	}
	if _, err := r.svc.SetRecipient(ctx, c.ID, recipient); !errors.Is(err, ErrExpired) {
		t.Fatal(err)
	}
	// A ready card that nobody pays also expires, and is then no longer watched.
	r.now = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	c3, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "1"})
	r.svc.SetRecipient(ctx, c3.ID, recipient)
	r.now = r.now.Add(8 * 24 * time.Hour)
	before := r.chain.Calls["eth_getLogs"]
	r.chain.Transfer(r.chain.Head()+1, sender, recipient, 1_000_000)
	r.chain.Mine(100)
	if st, _ := r.svc.Status(ctx, c3.ID); st.State != StateExpired || r.chain.Calls["eth_getLogs"] != before {
		t.Fatal("an expired card is not watched")
	}
}

func TestInvalidInputs(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	for _, in := range []Input{
		{Asset: "dai_polygon", Amount: "1"}, {Asset: evm.AssetUSDTPolygon, Amount: "0"}, {Asset: evm.AssetUSDTPolygon, Amount: "1001"},
		{Asset: evm.AssetUSDTPolygon, Amount: "1", Protected: true}, {Asset: evm.AssetUSDTPolygon, Amount: "1", Protected: true, Email: "nope"},
		{Asset: evm.AssetUSDTPolygon, Amount: "1", Message: strings.Repeat("é", 141)}, {Asset: evm.AssetUSDTPolygon, Amount: "1", Message: "a\nb"},
		{Asset: evm.AssetUSDTPolygon, Amount: "1", SenderAddress: "0x123"}, {Asset: evm.AssetUSDTPolygon, Amount: "1", SenderEmail: "x"},
	} {
		if _, err := r.svc.Create(in); err == nil {
			t.Fatalf("accepted %+v", in)
		}
	}
	if _, err := r.svc.SetRecipient(context.Background(), strings.Repeat("a", 48), recipient); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestStateSurvivesRestartAndNeverLeaksEmails(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	ctx := context.Background()
	c, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "4", Protected: true, Email: "a@b.co", SenderEmail: "s@b.co", SenderAddress: sender})
	r.svc.MarkVerified(c.ID)
	r.svc.SetRecipient(ctx, c.ID, recipient)
	p, _ := r.svc.Public(c.ID)
	raw := toJSON(p)
	for _, secret := range []string{"a@b.co", "s@b.co", c.SenderToken, recipient, sender} {
		if strings.Contains(raw, secret) {
			t.Fatalf("public view leaks %q", secret)
		}
	}
	again, err := New(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Has(c.ID) {
		t.Fatal("lost")
	}
	if rec, _ := again.get(c.ID); rec.State != StateReady || rec.Recipient != recipient || rec.Email != "a@b.co" || !rec.Verified {
		t.Fatalf("record changed: %+v", rec)
	}
}

func TestBackgroundLoopDetectsWithoutAPageOpen(t *testing.T) {
	r := newRig(t, evm.AssetUSDTPolygon)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, _ := r.svc.Create(Input{Asset: evm.AssetUSDTPolygon, Amount: "1"})
	r.svc.SetRecipient(ctx, c.ID, recipient)
	r.chain.Transfer(r.chain.Head()+1, sender, recipient, 1_000_000)
	r.chain.Mine(100)
	go r.svc.Run(ctx, 20*time.Millisecond)
	for i := 0; i < 100; i++ {
		time.Sleep(20 * time.Millisecond)
		if p, _ := r.svc.Public(c.ID); p.State == StatePaid {
			return
		}
	}
	t.Fatal("the loop did not notice the payment")
}

func toJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
