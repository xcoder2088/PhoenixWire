package wire

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/service"
)

func listPayments(t *testing.T, f *fixture) []map[string]any {
	t.Helper()
	w := f.call("GET", "/api/payments", nil)
	var out struct{ Payments []map[string]any }
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	return out.Payments
}

// sent_at is the first invitation the mail service accepted: not a failed attempt, not a resend, and it survives a restart.
func TestNimSentAtIsTheFirstSuccessfulInvitation(t *testing.T) {
	f := newFixture(t)
	f.verifySender("sender@example.test")
	p := f.create()
	good(f.call("POST", "/api/payments/"+p.ID+"/start", map[string]string{}), t)
	f.rpcMu.Lock()
	f.funded = true
	f.rpcMu.Unlock()
	t1 := time.Date(2026, 10, 23, 14, 30, 0, 0, time.UTC)
	f.api.now = func() time.Time { return t1 }
	f.mail.fail = true
	if w := f.call("POST", "/api/payments/"+p.ID+"/confirm", map[string]string{}); w.Code != 502 {
		t.Fatal("delivery failure hidden")
	}
	if got := listPayments(t, f); len(got) != 1 || got[0]["sent_at"] != nil {
		t.Fatalf("a failed delivery gives no sent date: %v", got)
	}
	f.mail.fail = false
	f.api.now = func() time.Time { return t1.Add(2 * time.Minute) }
	good(f.call("POST", "/api/payments/"+p.ID+"/resend", map[string]string{}), t)
	got := listPayments(t, f)
	if got[0]["sent_at"] != "2026-10-23T14:32:00Z" || got[0]["created"] == nil {
		t.Fatalf("the date of the first success, in UTC: %v", got[0])
	}
	f.api.now = func() time.Time { return t1.Add(time.Hour) }
	good(f.call("POST", "/api/payments/"+p.ID+"/resend", map[string]string{}), t)
	if listPayments(t, f)[0]["sent_at"] != "2026-10-23T14:32:00Z" {
		t.Fatal("a resend must not replace the first date")
	}
	f.restart()
	if listPayments(t, f)[0]["sent_at"] != "2026-10-23T14:32:00Z" {
		t.Fatal("the date is the same after a restart")
	}
	// another sender does not see it, and an old payment (no date was kept) shows none: nothing is made up
	f.verifySender("other@example.test")
	if got := listPayments(t, f); len(got) != 0 {
		t.Fatalf("another sender sees: %v", got)
	}
	if strings.Contains(string(mustJSON(t, payment{ID: "x"})), "sent_at") {
		t.Fatal("a payment without a date has no sent_at field")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// claimEngine is the real engine, but it says what the chain says about a card: its state, the kind of its payout and the date the
// engine took from the block (the engine's own date logic is tested in the engine).
type claimEngine struct {
	*service.Engine
	state, kind, at string
}

func (c *claimEngine) IssuerCall(ctx context.Context, method, path string, body []byte) (int, []byte, error) {
	code, out, err := c.Engine.IssuerCall(ctx, method, path, body)
	if strings.HasSuffix(path, "/status") && c.state != "" {
		var m map[string]any
		_ = json.Unmarshal(out, &m)
		m["state"], m["tx_kind"] = c.state, c.kind
		if c.at != "" {
			m["claimed_at"] = c.at
		}
		out, _ = json.Marshal(m)
	}
	return code, out, err
}

func TestUSDCSentAndClaimedDates(t *testing.T) {
	node := &evmNode{balance: word(0), block: 5000}
	rpc := node.serve()
	defer rpc.Close()
	t.Setenv("NIMIQ_CLAIM_USDTLINK_CHAINS", "polygon-usdc")
	t.Setenv("NIMIQ_CLAIM_USDTLINK_COINGECKO", "off") // tests never ask the internet for a price
	t.Setenv("NIMIQ_CLAIM_EVM_RPC_POLYGON", rpc.URL)
	t.Setenv("NIMIQ_CLAIM_ALLOWED_ORIGINS", "https://wire.example.test")
	f := newFixture(t)
	f.verifySender("sender@example.test")
	id, key, owner, err := service.NewCardLink()
	if err != nil {
		t.Fatal(err)
	}
	cardID := hex.EncodeToString(id)
	if w := f.call("POST", "/api/usdt/cashlink/create", map[string]string{"asset": "usdc_polygon", "cardId": cardID, "linkOwner": owner, "amount": "2.5", "recipient": "friend@example.test", "lang": "en"}); w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	engine := &claimEngine{Engine: f.engine}
	f.api.engine = engine
	card := func() map[string]any {
		t.Helper()
		w := f.call("GET", "/api/usdt/cards", nil)
		var out struct{ Cards []map[string]any }
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || len(out.Cards) != 1 {
			t.Fatalf("cards: %d %s", w.Code, w.Body)
		}
		return out.Cards[0]
	}
	status := func() map[string]any {
		t.Helper()
		w := f.call("POST", "/api/usdt/cashlink/status", map[string]string{"id": cardID})
		var out map[string]any
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatalf("status: %d %s", w.Code, w.Body)
		}
		return out
	}
	if c := card(); c["sent_at"] != nil || c["claimed_at"] != nil || !strings.HasSuffix(c["created"].(string), "Z") {
		t.Fatalf("a new card: created in UTC, no sent or claimed date: %v", c)
	}
	// sent_at: the first invitation the mail service accepted
	engine.state = "ready"
	claim := "https://wire.example.test/usdt.html?lang=en" + service.ClaimFragment(id, key)
	t1 := time.Date(2026, 10, 23, 15, 40, 0, 0, time.UTC)
	f.api.now = func() time.Time { return t1 }
	f.mail.fail = true
	if w := f.call("POST", "/api/usdt/card/"+cardID+"/invite", map[string]string{"claim_url": claim}); w.Code != 502 {
		t.Fatalf("a failed delivery: %d", w.Code)
	}
	if c := card(); c["sent_at"] != nil || c["invited"] != false {
		t.Fatalf("a failed delivery gives no sent date: %v", c)
	}
	f.mail.fail = false
	f.api.now = func() time.Time { return t1.Add(time.Minute) }
	good(f.call("POST", "/api/usdt/card/"+cardID+"/invite", map[string]string{"claim_url": claim}), t)
	if c := card(); c["sent_at"] != "2026-10-23T15:41:00Z" {
		t.Fatalf("the first success: %v", c)
	}
	f.api.now = func() time.Time { return t1.Add(time.Hour) }
	good(f.call("POST", "/api/usdt/card/"+cardID+"/invite", map[string]string{"claim_url": claim}), t)
	if c := card(); c["sent_at"] != "2026-10-23T15:41:00Z" || status()["sent_at"] != "2026-10-23T15:41:00Z" {
		t.Fatalf("a resend keeps the first date: %v", c)
	}
	// claimed_at: only for a card paid by a CLAIM, from the engine's block date, taken once
	f.api.now = func() time.Time { return time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC) }
	engine.state, engine.kind, engine.at = "paid", "refund", "2026-10-23T16:00:00Z"
	if status()["claimed_at"] != nil || card()["claimed_at"] != nil {
		t.Fatal("a refund must not become a claim")
	}
	engine.state, engine.kind = "ready", "claim"
	if status()["claimed_at"] != nil {
		t.Fatal("no claim date before the card is paid")
	}
	engine.state = "paid"
	for _, bad := range []string{"", "yesterday", "1999-01-01T00:00:00Z", "2031-01-01T00:00:00Z", "2026-10-23 16:00:00"} {
		engine.at = bad
		if status()["claimed_at"] != nil || card()["claimed_at"] != nil {
			t.Fatalf("%q is not a block date", bad)
		}
	}
	engine.at = "2026-10-23T16:05:09Z"
	if status()["claimed_at"] != "2026-10-23T16:05:09Z" || card()["claimed_at"] != "2026-10-23T16:05:09Z" {
		t.Fatal("the block date of the confirmed payout")
	}
	engine.at = "2026-10-23T18:00:00Z" // a later look, another answer: the first date stays
	if status()["claimed_at"] != "2026-10-23T16:05:09Z" {
		t.Fatal("a later look never moves the claim date")
	}
	f.restart()
	f.api.engine = &claimEngine{Engine: f.engine}
	if c := card(); c["claimed_at"] != "2026-10-23T16:05:09Z" || c["sent_at"] != "2026-10-23T15:41:00Z" {
		t.Fatalf("the same dates after a restart: %v", c)
	}
	// someone else's card: nothing, not even that it exists
	f.verifySender("other@example.test")
	if w := f.call("POST", "/api/usdt/cashlink/status", map[string]string{"id": cardID}); w.Code != 404 || strings.Contains(w.Body.String(), "2026") {
		t.Fatalf("another sender: %d %s", w.Code, w.Body)
	}
}
