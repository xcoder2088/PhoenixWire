package usdtlink

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"sync"
	"time"

	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
)

// The fee a new card pays follows the price of gas, so that the relayer keeps a margin whatever the network costs.
//
//	fee = the largest of: the flat fee of the chain (DefaultFee), FeeBps of the amount, and FeeMargin times what the gas of one card costs NOW
//
// "What the gas costs now" is the node's own fee estimate (the highest fee per gas a transaction would be sent with) times the gas one card
// is assumed to use, in the chain's coin, converted to token units by the price of that coin. The price is read from the chain itself
// (a Chainlink USD feed, one read-only call), remembered for priceTTL, and, when the feed cannot be read, the last good price is kept for
// priceStale. When the feed fails, CoinGecko's public price is asked next; past that, a deliberately HIGH fallback price is used, so an outage makes fees higher, never lower. The flat fee and the
// percentage are floors: a quiet network never lowers a fee below what it already was.
//
// A refund pays the relayer the gas of a refund with the same margin, never more than the card's fee, fixed when the card is created
// (the sender's page checks it) and kept in the record.
var (
	// FeeMargin is how many times the estimated gas cost the fee must cover (default 3).
	FeeMargin = 3.0
	// CardGas is the gas assumed for one card's whole life (deployment of its Safe and the payout), RefundGas for a refund.
	CardGas   uint64 = 450_000
	RefundGas uint64 = 200_000
)

const (
	priceTTL   = 5 * time.Minute
	priceStale = 6 * time.Hour
	feedMaxAge = 26 * time.Hour // a feed that has not been updated for longer is not trusted (they update at least daily)
)

// ErrFeeTooHigh: the fee needed to cover the gas of a card would exceed what a card of this amount may carry. New cards wait for gas to fall.
var ErrFeeTooHigh = errors.New("gas_too_high")

type nativePrice struct {
	usd  float64
	at   time.Time // when it was read
	live bool      // read from the feed (not the fallback)
}

type pricer struct {
	mu     sync.Mutex
	prices map[string]nativePrice // by asset
}

// nativeUSD is the price of the chain's coin in dollars (token units are dollars: USDC and USDT).
func (s *Service) nativeUSD(ctx context.Context, asset string, c Chain) (float64, bool) {
	s.price.mu.Lock()
	if s.price.prices == nil {
		s.price.prices = map[string]nativePrice{}
	}
	p, have := s.price.prices[asset]
	s.price.mu.Unlock()
	now := s.now()
	if have && p.live && now.Sub(p.at) < priceTTL {
		return p.usd, true
	}
	if c.PriceFeed != "" {
		if usd, err := readFeed(ctx, c); err == nil {
			s.price.mu.Lock()
			s.price.prices[asset] = nativePrice{usd: usd, at: now, live: true}
			s.price.mu.Unlock()
			return usd, true
		} else {
			log.Printf("USDT cashlink %s: price feed unavailable (%v)", c.Network.Chain, err)
		}
	}
	if c.CoinGeckoID != "" {
		if usd, err := readCoinGecko(ctx, c.CoinGeckoID); err == nil {
			s.price.mu.Lock()
			s.price.prices[asset] = nativePrice{usd: usd, at: now, live: true}
			s.price.mu.Unlock()
			return usd, true
		} else {
			log.Printf("USDT cashlink %s: CoinGecko price unavailable (%v)", c.Network.Chain, err)
		}
	}
	if have && p.live && now.Sub(p.at) < priceStale {
		return p.usd, true // the last good price, a little old
	}
	if c.FallbackUSD > 0 {
		return c.FallbackUSD, false
	}
	return 0, false
}

// geckoURL is CoinGecko's public price endpoint (no key); tests replace it.
var geckoURL = "https://api.coingecko.com/api/v3/simple/price"

// readCoinGecko asks CoinGecko for the dollar price of one coin by its CoinGecko id. The answer is small, bounded and checked.
func readCoinGecko(ctx context.Context, id string) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q := url.Values{"ids": {id}, "vs_currencies": {"usd"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, geckoURL+"?"+q.Encode(), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, errors.New("coingecko_status")
	}
	var out map[string]struct {
		USD float64 `json:"usd"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&out) != nil {
		return 0, errors.New("coingecko_answer")
	}
	usd := out[id].USD
	if usd <= 0 || usd > 1e7 {
		return 0, errors.New("coingecko_answer")
	}
	return usd, nil
}

// readFeed reads latestRoundData() of a Chainlink USD feed: (roundId, answer, startedAt, updatedAt, answeredInRound), answer with `decimals` digits.
func readFeed(ctx context.Context, c Chain) (float64, error) {
	feed, err := cl.ParseAddress(c.PriceFeed)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := c.Node.Call(ctx, cl.Address{}, feed, []byte{0xfe, 0xaf, 0x96, 0x8c}) // latestRoundData()
	if err != nil {
		return 0, err
	}
	if len(out) < 160 {
		return 0, errors.New("feed_answer_short")
	}
	answer := new(big.Int).SetBytes(out[32:64])
	if out[32]&0x80 != 0 || answer.Sign() <= 0 { // negative or zero
		return 0, errors.New("feed_answer_invalid")
	}
	updated := new(big.Int).SetBytes(out[96:128])
	if !updated.IsInt64() || time.Since(time.Unix(updated.Int64(), 0)) > feedMaxAge {
		return 0, errors.New("feed_stale")
	}
	dec, err := c.Node.Call(ctx, cl.Address{}, feed, []byte{0x31, 0x3c, 0xe5, 0x67}) // decimals()
	if err != nil || len(dec) < 32 {
		return 0, errors.New("feed_decimals")
	}
	d := new(big.Int).SetBytes(dec[:32])
	if !d.IsInt64() || d.Int64() < 0 || d.Int64() > 18 {
		return 0, errors.New("feed_decimals")
	}
	f := new(big.Float).Quo(new(big.Float).SetInt(answer), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), d, nil)))
	usd, _ := f.Float64()
	if usd <= 0 || usd > 1e7 {
		return 0, errors.New("feed_answer_invalid")
	}
	return usd, nil
}

// gasCostUnits is what `gas` gas costs now, in token units (6 decimals), rounded up. ok is false when it cannot be priced.
func (s *Service) gasCostUnits(ctx context.Context, asset string, c Chain, gas uint64) (uint64, bool) {
	if c.Node == nil {
		return 0, false
	}
	usd, _ := s.nativeUSD(ctx, asset, c)
	if usd <= 0 {
		return 0, false
	}
	fctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, maxFee, err := c.Node.Fees(fctx)
	if err != nil || maxFee == nil || maxFee.Sign() <= 0 {
		return 0, false
	}
	wei := new(big.Int).Mul(maxFee, new(big.Int).SetUint64(gas))
	// wei / 1e18 coins * usd dollars * 1e6 token units
	f := new(big.Float).SetInt(wei)
	f.Mul(f, big.NewFloat(usd))
	f.Quo(f, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)))
	units, _ := f.Uint64()
	if units == ^uint64(0) {
		return 0, false
	}
	return units + 1, true
}

func margined(units uint64) uint64 {
	return uint64(float64(units)*FeeMargin) + 1
}

// quote is the fee and the refund fee a new card of this many token units carries.
func (s *Service) quote(ctx context.Context, asset string, c Chain, units uint64) (fee, refund uint64, err error) {
	fee = FeeFor(asset, units)
	flat := DefaultFee[asset]
	refund = flat
	if cost, ok := s.gasCostUnits(ctx, asset, c, CardGas); ok {
		if dyn := margined(cost); dyn > fee {
			fee = dyn
		}
	}
	if cost, ok := s.gasCostUnits(ctx, asset, c, RefundGas); ok {
		if dyn := margined(cost); dyn > refund {
			refund = dyn
		}
	}
	if fee > MaxFeeFor(units) {
		return 0, 0, ErrFeeTooHigh
	}
	if refund > fee {
		refund = fee
	}
	return fee, refund, nil
}
