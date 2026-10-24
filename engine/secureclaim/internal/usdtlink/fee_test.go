package usdtlink

import (
	"context"
	"encoding/hex"
	"errors"
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
	// 450 000 gas at 100 gwei is 0.045 POL; at $2 that is 9 cents, 1.5 times is 13.5 cents; 6 % of 2.50 is 15 cents: 28.5 cents.
	s := feeService(t, &feedNode{usd: 2, maxFee: gwei100()})
	fee, refund, err := quoteFor(t, s, "2.50")
	if err != nil || fee < 285_000 || fee > 285_010 {
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
	if err != nil || quiet < 156_000 || quiet > 156_100 {
		t.Fatal("quiet gas", quiet, err)
	}
	// Gas ten times dearer: only the gas part moves; the percentage is the same 15 cents.
	s2 := feeService(t, &feedNode{usd: 0.9, maxFee: gwei100()})
	busy, _, err := quoteFor(t, s2, "2.50")
	if err != nil || busy-quiet < 54_000 || busy-quiet > 55_000 {
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
	if err != nil || fee < 285_000 || fee > 285_010 {
		t.Fatal("CoinGecko price not used", fee, err)
	}
	// Neither source answers: a HIGH fallback price (0.50 here) is used, so the fee is higher than at the real price, never lower.
	geckoURL = "http://127.0.0.1:1/none"
	s2 := feeService(t, &feedNode{usd: 0, maxFee: gwei100()})
	c2 := s2.chains[evm.AssetUSDTPolygon]
	c2.CoinGeckoID = "polygon-ecosystem-token"
	s2.chains[evm.AssetUSDTPolygon] = c2
	fee, _, err = quoteFor(t, s2, "2.50")
	if err != nil || fee < 183_000 || fee > 184_000 { // 0.045 POL * $0.50 * 1.5 = 3.4 cents of gas + 15 cents
		t.Fatal("fallback", fee, err)
	}
}

func TestAStaleFeedIsNotTrusted(t *testing.T) {
	node := &feedNode{usd: 2, age: 30 * time.Hour, maxFee: gwei100()}
	s := feeService(t, node)
	fee, _, err := quoteFor(t, s, "2.50") // falls back to $0.50, not the $2 of the stale feed
	if err != nil || fee < 183_000 || fee > 184_000 {
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
