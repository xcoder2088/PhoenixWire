package usdtlink

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/chain"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
)

const testFeed = "0xAB594600376Ec9fD91F8e885dADF0CE036862dE0"

// feedNode is the test node with a price feed on it (what the chain says the coin costs) and a gas price the test chooses.
type feedNode struct {
	*fakeNode
	usd      float64 // the feed's answer, in dollars (8 decimals); 0: no answer
	age      time.Duration
	maxFee   *big.Int
	feedHits atomic.Int32
}

func (f *feedNode) Fees(context.Context) (*big.Int, *big.Int, error) {
	return big.NewInt(1_000_000_000), f.maxFee, nil
}
func (f *feedNode) Call(ctx context.Context, from, to cl.Address, data []byte) ([]byte, error) {
	if to != cl.MustAddress(testFeed) {
		return f.fakeNode.Call(ctx, from, to, data)
	}
	f.feedHits.Add(1)
	if f.usd <= 0 {
		return nil, errors.New("feed unavailable")
	}
	if hex.EncodeToString(data) == "313ce567" { // decimals()
		return cl.Word32(big.NewInt(8)), nil
	}
	word := func(v int64) []byte { return cl.Word32(big.NewInt(v)) }
	out := append(word(1), cl.Word32(big.NewInt(int64(f.usd*1e8)))...)
	updated := time.Now().Add(-f.age).Unix()
	out = append(out, word(updated)...)
	out = append(out, word(updated)...)
	return append(out, word(1)...), nil
}

func feeService(t *testing.T, node *feedNode) *Service {
	t.Helper()
	s, base := newService(t)
	node.fakeNode = base
	c := s.chains[evm.AssetUSDTPolygon]
	c.Node = node
	c.PriceFeed = testFeed
	c.FallbackUSD = 0.50
	s.chains[evm.AssetUSDTPolygon] = c
	return s
}

func quoteFor(t *testing.T, s *Service, amount string) (uint64, uint64, error) {
	t.Helper()
	units, _, err := evm.ParseAmount(amount)
	if err != nil {
		t.Fatal(err)
	}
	return s.quote(context.Background(), evm.AssetUSDTPolygon, s.chains[evm.AssetUSDTPolygon], units)
}

func gwei100() *big.Int { return big.NewInt(100_000_000_000) }

func TestFeeIsTheGasPlusThePercentage(t *testing.T) {
	// 550 000 gas (the card, plus the gift of gas money to the sender) at 100 gwei is 0.055 POL; at $2 that is 11 cents, 1.5 times is 16.5 cents; 6 % of 2.50 is 15 cents: 31.5 cents.
	s := feeService(t, &feedNode{usd: 2, maxFee: gwei100()})
	fee, refund, err := quoteFor(t, s, "2.50")
	if err != nil || fee < 315_000 || fee > 315_010 {
		t.Fatal("fee should be about 0.285", fee, err)
	}
	// A refund keeps the whole fee for the relayer.
	if refund != fee {
		t.Fatal("refund fee", refund, fee)
	}
}

func TestThePercentageStaysWhateverGasCosts(t *testing.T) {
	// $0.09: the gas of a card costs under a cent; the percentage (15 cents) is still added on top of it.
	s := feeService(t, &feedNode{usd: 0.09, maxFee: gwei100()})
	quiet, _, err := quoteFor(t, s, "2.50")
	if err != nil || quiet < 157_000 || quiet > 157_900 {
		t.Fatal("quiet gas", quiet, err)
	}
	// Gas ten times dearer: only the gas part moves; the percentage is the same 15 cents.
	s2 := feeService(t, &feedNode{usd: 0.9, maxFee: gwei100()})
	busy, _, err := quoteFor(t, s2, "2.50")
	if err != nil || busy-quiet < 66_000 || busy-quiet > 68_000 {
		t.Fatal("busy gas", busy, quiet, err)
	}
}

func TestFeeWhenTheFeedIsDownUsesCoinGeckoThenAHighFallback(t *testing.T) {
	gecko := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ids") != "polygon-ecosystem-token" {
			http.Error(w, "no", 404)
			return
		}
		w.Write([]byte(`{"polygon-ecosystem-token":{"usd":2}}`))
	}))
	defer gecko.Close()
	old := geckoURL
	geckoURL = gecko.URL
	defer func() { geckoURL = old }()
	s := feeService(t, &feedNode{usd: 0, maxFee: gwei100()})
	c := s.chains[evm.AssetUSDTPolygon]
	c.CoinGeckoID = "polygon-ecosystem-token"
	s.chains[evm.AssetUSDTPolygon] = c
	fee, _, err := quoteFor(t, s, "2.50")
	if err != nil || fee < 315_000 || fee > 315_010 {
		t.Fatal("CoinGecko price not used", fee, err)
	}
	// Neither source answers: a HIGH fallback price (0.50 here) is used, so the fee is higher than at the real price, never lower.
	geckoURL = "http://127.0.0.1:1/none"
	s2 := feeService(t, &feedNode{usd: 0, maxFee: gwei100()})
	c2 := s2.chains[evm.AssetUSDTPolygon]
	c2.CoinGeckoID = "polygon-ecosystem-token"
	s2.chains[evm.AssetUSDTPolygon] = c2
	fee, _, err = quoteFor(t, s2, "2.50")
	if err != nil || fee < 191_000 || fee > 192_000 { // 0.045 POL * $0.50 * 1.5 = 3.4 cents of gas + 15 cents
		t.Fatal("fallback", fee, err)
	}
}

func TestAStaleFeedIsNotTrusted(t *testing.T) {
	node := &feedNode{usd: 2, age: 30 * time.Hour, maxFee: gwei100()}
	s := feeService(t, node)
	fee, _, err := quoteFor(t, s, "2.50") // falls back to $0.50, not the $2 of the stale feed
	if err != nil || fee < 191_000 || fee > 192_000 {
		t.Fatal("a feed older than a day was trusted", fee, err)
	}
}

func TestPriceIsRememberedForAFewMinutes(t *testing.T) {
	node := &feedNode{usd: 2, maxFee: gwei100()}
	s := feeService(t, node)
	quoteFor(t, s, "2.50")
	quoteFor(t, s, "5")
	quoteFor(t, s, "9")
	if node.feedHits.Load() > 2 { // latestRoundData and decimals, once
		t.Fatal("the feed was read for every card", node.feedHits.Load())
	}
}

func TestNewCardsWaitWhenGasWouldEatTheCard(t *testing.T) {
	// A coin at $1000 and 100 gwei: the gas of a card is $45, far above what a 2.50 card may carry.
	s := feeService(t, &feedNode{usd: 1000, maxFee: gwei100()})
	if _, _, err := quoteFor(t, s, "2.50"); !errors.Is(err, ErrFeeTooHigh) {
		t.Fatal("a card was taken at a loss", err)
	}
}

func TestACardKeepsItsRefundFee(t *testing.T) {
	node := &feedNode{usd: 2, maxFee: gwei100()}
	s := feeService(t, node)
	c := newCard(t, s)
	rec := s.records[c.Issuer.ID]
	if rec.RefundFeeUnits != rec.FeeUnits || s.refundFee(rec) != rec.RefundFeeUnits {
		t.Fatal("refund fee not kept with the card", rec.RefundFeeUnits, rec.FeeUnits)
	}
	// Gas changes later: the card's fees do not.
	node.maxFee = new(big.Int).Mul(gwei100(), big.NewInt(10))
	if s.refundFee(rec) != rec.RefundFeeUnits {
		t.Fatal("a card's refund fee moved with the price of gas")
	}
}

func TestTheFloorFollowsThePriceOfGas(t *testing.T) {
	ten := func(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1_000_000_000_000_000)) } // n / 1000 coin
	node := &feedNode{usd: 2, maxFee: big.NewInt(160_000_000)}                                                  // 0.16 gwei, like Ethereum today
	s := feeService(t, node)
	c := s.chains[evm.AssetUSDTPolygon]
	c.MinFloat, c.FloatMin = ten(10), ten(1) // at most 0.01, at least 0.001
	s.chains[evm.AssetUSDTPolygon] = c
	// 10 cards x 450 000 gas x 0.16 gwei = 0.00072: under the least, so the least (0.001) is the floor.
	if got := s.minFloat(context.Background(), c); got.Cmp(ten(1)) != 0 {
		t.Fatal("quiet gas: the floor is the least", got)
	}
	// Gas at 1 gwei: 0.0045 coin for ten cards: the floor follows it.
	node.maxFee = big.NewInt(1_000_000_000)
	if got := s.minFloat(context.Background(), c); got.Cmp(ten(4)) < 0 || got.Cmp(ten(5)) > 0 {
		t.Fatal("the floor should be about 0.0045", got)
	}
	// Gas at 50 gwei: 0.225 for ten cards: never more than the prudent maximum.
	node.maxFee = big.NewInt(50_000_000_000)
	if got := s.minFloat(context.Background(), c); got.Cmp(ten(10)) != 0 {
		t.Fatal("the floor never goes above the cap", got)
	}
	// An explicit floor (no least): exactly that, whatever gas costs.
	c.FloatMin = nil
	if got := s.minFloat(context.Background(), c); got.Cmp(ten(10)) != 0 {
		t.Fatal("explicit floor", got)
	}
}

func TestTheFreeGasLightFollowsThePriceOfGasToo(t *testing.T) {
	coin := func(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1_000_000_000_000_000)) } // n / 1000 coin
	node := &feedNode{usd: 2, maxFee: big.NewInt(160_000_000)}                                                   // 0.16 gwei
	s := feeService(t, node)
	c := s.chains[evm.AssetUSDTPolygon]
	c.MinFloat, c.Floor, c.FloatMin, c.DripCap = coin(10), coin(10), coin(1), coin(5) // keep at most 0.01, give at most 0.005
	s.chains[evm.AssetUSDTPolygon] = c
	node.native[c.Relayer.Address()] = coin(3) // 0.003 coin on the relayer: far under the old 0.015, plenty at this price
	if ok, why := s.GasStatus(context.Background(), evm.AssetUSDTPolygon); !ok {
		t.Fatal("the light should be green at a quiet price:", why)
	}
	node.maxFee = big.NewInt(50_000_000_000) // gas at 50 gwei: the float needed rises to its cap and one gift costs much more
	if ok, why := s.GasStatus(context.Background(), evm.AssetUSDTPolygon); ok || why != "float" {
		t.Fatal("the light should turn red when gas is dear and the relayer is low", ok, why)
	}
}

func TestTheLedgerSaysWhatAPayoutEarnedAndCost(t *testing.T) {
	node := &feedNode{usd: 2, maxFee: gwei100()}
	s := feeService(t, node)
	c := newCard(t, s)
	rec := s.records[c.Issuer.ID]
	rec.Drip = &Drip{To: "0x1111111111111111111111111111111111111111", Wei: "10000000000000000", At: time.Now()} // a gift of 0.01 coin
	op := &Operation{Kind: "claim", Units: rec.Units, Tx: "0xaaa"}
	// the payout used 100 000 gas at 50 gwei: 0.005 coin; at $2 that is 1 cent. The gift is 0.01 coin: 2 cents.
	rc := chain.Receipt{Found: true, Success: true, GasUsed: 100_000, GasPrice: big.NewInt(50_000_000_000)}
	s.recordCost(context.Background(), rec, op, rc)
	if op.GasWei != "5000000000000000" {
		t.Fatal("gas paid", op.GasWei)
	}
	sums, rows := s.Ledger(10)
	if len(sums) != 1 || len(rows) != 1 || sums[0].Payouts != 1 {
		t.Fatal("ledger", sums, rows)
	}
	row := rows[0]
	if row.Earned != rec.FeeUnits || row.CostUnit < 29_000 || row.CostUnit > 31_000 || row.Net != int64(row.Earned)-int64(row.CostUnit) || row.Card != rec.ID[:8] {
		t.Fatalf("row %+v", row)
	}
	if sums[0].Net <= 0 {
		t.Fatal("a 2.50 card at this price must leave a margin", sums[0])
	}
}

func TestAnEndedEmptyCardLosesItsPersonalData(t *testing.T) {
	s, node := newService(t)
	c := newCard(t, s)
	ctx := context.Background()
	rec := s.records[c.Issuer.ID]
	rec.SenderEmail, rec.Message = "sender@example.com", "hello"
	// a live card is not forgotten
	if err := s.Forget(ctx, rec.ID); !errors.Is(err, ErrNotFinal) {
		t.Fatal("a live card was forgotten", err)
	}
	// its time is over but it still holds money: kept, so that its sender can recover it
	node.balance = big.NewInt(int64(rec.Units + rec.FeeUnits))
	s.now = func() time.Time { return time.Now().Add(Lifetime + time.Hour) }
	if err := s.Forget(ctx, rec.ID); !errors.Is(err, ErrNotFinal) {
		t.Fatal("a card that holds money was forgotten", err)
	}
	if rec.Email == "" {
		t.Fatal("the recipient's e-mail was erased from a card that holds money")
	}
	// over and empty: the addresses and the message go, the card's id and amounts stay, and asking again changes nothing
	node.balance = big.NewInt(0)
	if err := s.Forget(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if rec.Email != "" || rec.SenderEmail != "" || rec.Message != "" || rec.Units == 0 {
		t.Fatalf("personal data kept or card erased: %+v", rec)
	}
	if err := s.Forget(ctx, rec.ID); err != nil {
		t.Fatal("forgetting twice must be harmless", err)
	}
}
