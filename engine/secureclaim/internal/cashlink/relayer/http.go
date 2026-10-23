// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

package relayer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/chain"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/safe"
)

// Handler exposes the relayer over HTTP (JSON):
//
//	GET  /v1/info   -> {chainId, relayer, token, minFee, maxAmount}
//	POST /v1/relay  -> {cardId, owners[], threshold, recipient, amount, fee, signatures} => {safe, deployTx?, execTx}
//
// Amounts are decimal strings of USDT units. Errors are {"error": code}. The server rate-limits per client address.
func Handler(r *Relayer) http.Handler {
	mux := http.NewServeMux()
	limiter := newLimiter(10, time.Minute)
	mux.HandleFunc("/v1/info", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			reply(w, 405, map[string]string{"error": "method_not_allowed"})
			return
		}
		reply(w, 200, map[string]any{"chainId": r.cfg.Chain.ChainID(), "relayer": r.addr.Hex(), "token": r.cfg.Token.Hex(), "minFee": r.cfg.MinFee.String(), "maxAmount": r.cfg.MaxAmount.String()})
	})
	mux.HandleFunc("/v1/relay", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			reply(w, 405, map[string]string{"error": "method_not_allowed"})
			return
		}
		if !limiter.allow(clientAddress(req)) {
			reply(w, 429, map[string]string{"error": "too_many_requests"})
			return
		}
		req.Body = http.MaxBytesReader(w, req.Body, 4096)
		var in struct {
			CardID     string   `json:"cardId"`
			Owners     []string `json:"owners"`
			Threshold  uint64   `json:"threshold"`
			Recipient  string   `json:"recipient"`
			Amount     string   `json:"amount"`
			Fee        string   `json:"fee"`
			Signatures string   `json:"signatures"`
		}
		dec := json.NewDecoder(req.Body)
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil {
			reply(w, 400, map[string]string{"error": "invalid_json"})
			return
		}
		parsed, err := parse(in.CardID, in.Owners, in.Threshold, in.Recipient, in.Amount, in.Fee, in.Signatures)
		if err != nil {
			reply(w, 400, map[string]string{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 4*time.Minute)
		defer cancel()
		res, err := r.Submit(ctx, parsed)
		if err != nil {
			out := map[string]string{"error": errorCode(err)}
			if res.DeployTx != "" { // hashes already known stay known: the client must not lose track of them
				out["deployTx"] = res.DeployTx
			}
			if res.ExecTx != "" {
				out["execTx"] = res.ExecTx
			}
			reply(w, statusOf(err), out)
			return
		}
		out := map[string]string{"safe": res.Safe.Hex(), "execTx": res.ExecTx}
		if res.DeployTx != "" {
			out["deployTx"] = res.DeployTx
		}
		reply(w, 200, out)
	})
	return mux
}

func parse(cardID string, owners []string, threshold uint64, recipient, amount, fee, signatures string) (Request, error) {
	bad := errors.New("invalid_request")
	id, err := hex.DecodeString(strings.TrimPrefix(cardID, "0x"))
	if err != nil || len(owners) == 0 || len(owners) > 3 {
		return Request{}, bad
	}
	var list []evm.Address
	for _, o := range owners {
		a, err := evm.ParseAddress(o)
		if err != nil {
			return Request{}, bad
		}
		list = append(list, a)
	}
	to, err := evm.ParseAddress(recipient)
	if err != nil {
		return Request{}, bad
	}
	a, ok1 := new(big.Int).SetString(amount, 10)
	f, ok2 := new(big.Int).SetString(fee, 10)
	sigs, err := hex.DecodeString(strings.TrimPrefix(signatures, "0x"))
	if !ok1 || !ok2 || err != nil || a.BitLen() > 128 || f.BitLen() > 128 || len(sigs) > 3*65 {
		return Request{}, bad
	}
	return Request{Card: safe.Card{ID: id, Owners: list, Threshold: threshold}, Recipient: to, Amount: a, Fee: f, Signatures: sigs}, nil
}

func errorCode(err error) string {
	for _, known := range []error{ErrCard, ErrAmount, ErrFeeTooLow, ErrNotFunded, ErrSignatures, ErrRejected, ErrRelayerFunds, ErrDestination, chain.ErrUnavailable} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return "relay_failed"
}

func statusOf(err error) int {
	switch {
	case errors.Is(err, ErrNotFunded):
		return 409
	case errors.Is(err, ErrRelayerFunds), errors.Is(err, chain.ErrUnavailable):
		return 503
	case errors.Is(err, ErrRejected):
		return 502
	}
	return 400
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func clientAddress(r *http.Request) string {
	// behind a reverse proxy the operator sets X-Forwarded-For; otherwise the socket address counts
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		return strings.TrimSpace(strings.Split(f, ",")[0])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}

type limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string][]time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{max: max, window: window, hits: map[string][]time.Time{}}
}

func (l *limiter) allow(who string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	kept := l.hits[who][:0]
	for _, t := range l.hits[who] {
		if now.Sub(t) < l.window {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.hits[who] = kept
		return false
	}
	l.hits[who] = append(kept, now)
	if len(l.hits) > 10000 {
		l.hits = map[string][]time.Time{who: l.hits[who]}
	}
	return true
}
