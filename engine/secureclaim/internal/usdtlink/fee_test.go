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

func TestFeeFollowsThePriceOfGas(t *testing.T) {
	// 450 000 gas at 100 gwei is 0.045 POL; at $2 that is 9 cents, three times is 27 cents: above the flat fee and the percentage.
	s := feeService(t, &feedNode{usd: 2, maxFee: gwei100()})
	fee, refund, err := quoteFor(t, s, "2.50")
	if err != nil || fee < 270_000 || fee > 270_010 {
		t.Fatal("fee should be about 0.27", fee, err)
	}
	// A refund pays the gas of a refund with the same margin (200 000 gas: 4 cents, 12 cents), never more than the card's fee.
	if refund < 120_000 || refund > 120_010 || refund > fee {
		t.Fatal("refund fee", refund)
	}
}

func TestQuietGasNeverLowersTheFeeBelowItsFloors(t *testing.T) {
	// $0.09: the gas of a card costs a fraction of a cent. The flat fee (0.10) and 6 % of the amount (0.15) stay.
	s := feeService(t, &feedNode{usd: 0.09, maxFee: gwei100()})
	fee, refund, err := quoteFor(t, s, "2.50")
	if err != nil || fee != 150_000 || refund != DefaultFee[evm.AssetUSDTPolygon] {
		t.Fatal("quiet gas", fee, refund, err)
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
	if err != nil || fee < 270_000 || fee > 270_010 {
		t.Fatal("CoinGecko price not used", fee, err)
	}
	// Neither source answers: a HIGH fallback price (0.50 here) is used, so the fee is higher than at the real price, never lower.
	geckoURL = "http://127.0.0.1:1/none"
	s2 := feeService(t, &feedNode{usd: 0, maxFee: gwei100()})
	c2 := s2.chains[evm.AssetUSDTPolygon]
	c2.CoinGeckoID = "polygon-ecosystem-token"
	s2.chains[evm.AssetUSDTPolygon] = c2
	fee, _, err = quoteFor(t, s2, "2.50")
	if err != nil || fee != 150_000 { // 0.045 POL * $0.50 * 3 = 6.75 cents: the floors still hold
		t.Fatal("fallback", fee, err)
	}
}

func TestAStaleFeedIsNotTrusted(t *testing.T) {
	node := &feedNode{usd: 2, age: 30 * time.Hour, maxFee: gwei100()}
	s := feeService(t, node)
	fee, _, err := quoteFor(t, s, "2.50") // falls back to $0.50: below the percentage
	if err != nil || fee != 150_000 {
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
	if rec.RefundFeeUnits < 120_000 || rec.RefundFeeUnits > rec.FeeUnits || s.refundFee(rec) != rec.RefundFeeUnits {
		t.Fatal("refund fee not kept with the card", rec.RefundFeeUnits, rec.FeeUnits)
	}
	// Gas changes later: the card's fees do not.
	node.maxFee = new(big.Int).Mul(gwei100(), big.NewInt(10))
	if s.refundFee(rec) != rec.RefundFeeUnits {
		t.Fatal("a card's refund fee moved with the price of gas")
	}
}
