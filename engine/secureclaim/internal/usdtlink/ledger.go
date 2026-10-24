package usdtlink

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/chain"
)

// The ledger: for every payout the relayer settles (a claim or a refund), what it EARNED (the card's fee) and what it COST (the gas it paid for
// the deployment and the payout, as the chain reports it, plus the gift of gas money it gave the sender), in token units at the price of the
// coin when the payout settled. One line per payout in <state>/usdtlink/ledger.jsonl (no key, no address, no e-mail: only the first
// characters of the card id) and one line in the log. It answers one question: does the fee leave a margin?
//
// Costs the ledger does not count: the gas of the gift transaction itself (21 000 gas, a few hundredths of a cent) and the gas of any
// transaction that failed. Earned is what the card's fee promised; a payout whose settlement was not seen is not in the ledger.
type LedgerRow struct {
	Time     time.Time `json:"time"`
	Chain    string    `json:"chain"`
	Asset    string    `json:"asset"`
	Card     string    `json:"card"` // first 8 characters of the card id
	Kind     string    `json:"kind"` // "claim" | "refund"
	Amount   uint64    `json:"amount_units"`
	Earned   uint64    `json:"earned_units"`
	GasWei   string    `json:"gas_wei"`
	GiftWei  string    `json:"gift_wei"`
	PriceUSD float64   `json:"price_usd"`
	CostUnit uint64    `json:"cost_units"`
	Net      int64     `json:"net_units"`
}

// LedgerSummary is the sum of the ledger, per asset.
type LedgerSummary struct {
	Asset      string `json:"asset"`
	Payouts    int    `json:"payouts"`
	Earned     uint64 `json:"earned_units"`
	Cost       uint64 `json:"cost_units"`
	Net        int64  `json:"net_units"`
	NetText    string `json:"net"`
	EarnedText string `json:"earned"`
	CostText   string `json:"cost"`
}

var ledgerMu sync.Mutex

func unitsText(u int64) string {
	sign := ""
	if u < 0 {
		sign, u = "-", -u
	}
	return fmt.Sprintf("%s%d.%06d", sign, u/1_000_000, u%1_000_000)
}

// paidGas is what the relayer paid in gas for an operation: its execution (the receipt in hand) and, when it deployed the card, the deployment.
func (s *Service) paidGas(ctx context.Context, c Chain, op *Operation, rc chain.Receipt) *big.Int {
	total := new(big.Int)
	add := func(r chain.Receipt) {
		if r.Found && r.GasPrice != nil {
			total.Add(total, new(big.Int).Mul(new(big.Int).SetUint64(r.GasUsed), r.GasPrice))
		}
	}
	add(rc)
	if op.Deploy != "" {
		if d, err := c.Node.Receipt(ctx, op.Deploy); err == nil {
			add(d)
		}
	}
	return total
}

// recordCost is called when a payout of ours settles.
func (s *Service) recordCost(ctx context.Context, r *Record, op *Operation, rc chain.Receipt) {
	c := s.chains[r.Asset]
	if c.Node == nil {
		return
	}
	gas := s.paidGas(ctx, c, op, rc)
	op.GasWei = gas.String()
	gift := new(big.Int)
	if r.Drip != nil {
		gift.SetString(r.Drip.Wei, 10)
	}
	usd, _ := s.nativeUSD(ctx, r.Asset, c)
	earned := r.FeeUnits
	if op.Kind == "refund" {
		earned = s.refundFee(r)
	}
	cost := weiToUnits(new(big.Int).Add(gas, gift), usd)
	row := LedgerRow{Time: s.now().UTC(), Chain: c.Network.Chain, Asset: r.Asset, Card: r.ID[:8], Kind: op.Kind, Amount: op.Units, Earned: earned,
		GasWei: gas.String(), GiftWei: gift.String(), PriceUSD: usd, CostUnit: cost, Net: int64(earned) - int64(cost)}
	if err := appendLedger(filepath.Join(s.dir, "ledger.jsonl"), row); err != nil {
		log.Printf("USDT cashlink ledger: not written: %v", err)
		return
	}
	log.Printf("USDT cashlink ledger: %s %s card %s earned %s cost %s net %s (gas price paid %s gwei)", row.Chain, row.Kind, row.Card, unitsText(int64(earned)), unitsText(int64(cost)), unitsText(row.Net), gweiText(rc.GasPrice))
}

func gweiText(p *big.Int) string {
	if p == nil {
		return "?"
	}
	return new(big.Float).Quo(new(big.Float).SetInt(p), big.NewFloat(1e9)).Text('f', 3)
}

// weiToUnits converts a native amount to token units (6 decimals) at `usd` dollars per coin, rounded up.
func weiToUnits(wei *big.Int, usd float64) uint64 {
	if usd <= 0 || wei.Sign() <= 0 {
		return 0
	}
	f := new(big.Float).SetInt(wei)
	f.Mul(f, big.NewFloat(usd))
	f.Quo(f, big.NewFloat(1e12))
	u, _ := f.Uint64()
	return u + 1
}

func appendLedger(path string, row LedgerRow) error {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(row)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// Ledger returns the sums per asset and the last `n` rows.
func (s *Service) Ledger(n int) ([]LedgerSummary, []LedgerRow) {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	f, err := os.Open(filepath.Join(s.dir, "ledger.jsonl"))
	if err != nil {
		return []LedgerSummary{}, []LedgerRow{}
	}
	defer f.Close()
	sums := map[string]*LedgerSummary{}
	var rows []LedgerRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var row LedgerRow
		if json.Unmarshal(sc.Bytes(), &row) != nil {
			continue
		}
		t := sums[row.Asset]
		if t == nil {
			t = &LedgerSummary{Asset: row.Asset}
			sums[row.Asset] = t
		}
		t.Payouts++
		t.Earned += row.Earned
		t.Cost += row.CostUnit
		t.Net += row.Net
		rows = append(rows, row)
	}
	if n >= 0 && len(rows) > n {
		rows = rows[len(rows)-n:]
	}
	out := []LedgerSummary{}
	for _, t := range sums {
		t.NetText, t.EarnedText, t.CostText = unitsText(t.Net), unitsText(int64(t.Earned)), unitsText(int64(t.Cost))
		out = append(out, *t)
	}
	return out, rows
}
