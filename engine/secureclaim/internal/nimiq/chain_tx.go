package nimiq

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// Height is the node's current block number.
func (v *MainnetVerifier) Height(ctx context.Context) (uint32, error) {
	if v == nil || ctx == nil {
		return 0, ErrRPC
	}
	raw, err := v.call(ctx, "getBlockNumber", []any{})
	if err != nil {
		return 0, err
	}
	height, err := uintFromJSON(raw)
	if err != nil || height < 3 || height > 1<<32-1 {
		return 0, ErrRPC
	}
	return uint32(height), nil
}

// Balance is what an address holds, in luna.
func (v *MainnetVerifier) Balance(ctx context.Context, address string) (uint64, error) {
	if v == nil || ctx == nil || !ValidAddress(address) {
		return 0, ErrRPC
	}
	raw, err := v.call(ctx, "getAccountByAddress", []any{address})
	if err != nil {
		return 0, err
	}
	var account struct {
		Balance json.RawMessage `json:"balance"`
	}
	if err := json.Unmarshal(raw, &account); err != nil {
		return 0, ErrRPC
	}
	return uintFromJSON(account.Balance)
}

// Broadcast hands a signed raw transaction (hex) to the node and returns the hash the node computed for it. A node that already knows the
// transaction may refuse it again: callers treat an error as "not accepted now", never as proof the transaction is gone.
func (v *MainnetVerifier) Broadcast(ctx context.Context, rawHex string) (string, error) {
	if v == nil || ctx == nil || rawHex == "" {
		return "", ErrRPC
	}
	raw, err := v.call(ctx, "sendRawTransaction", []any{rawHex})
	if err != nil {
		return "", err
	}
	var hash string
	if json.Unmarshal(raw, &hash) != nil || len(hash) != 64 {
		return "", ErrRPC
	}
	return strings.ToLower(hash), nil
}

// TxState looks a transaction up by hash. found is false when the node does not know it (not yet in a block, or never seen); the node's
// answer for an unknown hash is an RPC error, so callers check Height first to tell "unknown" from "node down".
func (v *MainnetVerifier) TxState(ctx context.Context, hash string) (found bool, confirmations int64, executed bool) {
	if v == nil || ctx == nil || len(hash) != 64 {
		return false, 0, false
	}
	raw, err := v.call(ctx, "getTransactionByHash", []any{hash})
	if err != nil {
		return false, 0, false
	}
	var tx struct {
		Confirmations   int64 `json:"confirmations"`
		ExecutionResult bool  `json:"executionResult"`
	}
	if json.Unmarshal(raw, &tx) != nil {
		return false, 0, false
	}
	return true, tx.Confirmations, tx.ExecutionResult
}

// RawHash asks the node for the hash of a signed raw transaction without sending it: the node parses and names it, nothing moves.
func (v *MainnetVerifier) RawHash(ctx context.Context, rawHex string) (string, error) {
	if v == nil || ctx == nil || rawHex == "" {
		return "", ErrRPC
	}
	raw, err := v.call(ctx, "getRawTransactionInfo", []any{rawHex})
	if err != nil {
		return "", err
	}
	var info struct {
		Hash string `json:"hash"`
	}
	if json.Unmarshal(raw, &info) != nil || len(info.Hash) != 64 {
		return "", ErrRPC
	}
	return strings.ToLower(info.Hash), nil
}

// ErrAmbiguousFunding: the money on a Cashlink did not come from one sender (or its history is too long to be read whole). Nothing is paid back
// automatically then: whoever sent it recovers it by hand.
var ErrAmbiguousFunding = errors.New("funding_ambiguous")

// fundingWindow is how many of the address's latest transactions the node is asked for; a full window means there may be more and is ambiguous.
const fundingWindow = 50

// FundingSender is the address a Cashlink was funded from, and only when that is certain: every confirmed transaction that carries the funding
// mark comes from ONE address, and the node's whole answer was read (a full window of 50 may hide older ones). Anyone can send a small amount
// with the mark to a card's public address, so "the oldest" or "the biggest" sender is not an answer: two different senders, or a history
// too long to read, is ErrAmbiguousFunding. "" with a nil error: no funding shown yet.
func (v *MainnetVerifier) FundingSender(ctx context.Context, address string) (string, error) {
	if v == nil || ctx == nil || !ValidAddress(address) {
		return "", ErrRPC
	}
	raw, err := v.call(ctx, "getTransactionsByAddress", []any{address, fundingWindow, nil})
	if err != nil {
		return "", err
	}
	var txs []struct {
		From            string `json:"from"`
		To              string `json:"to"`
		RecipientData   string `json:"recipientData"`
		Confirmations   int64  `json:"confirmations"`
		ExecutionResult bool   `json:"executionResult"`
	}
	if json.Unmarshal(raw, &txs) != nil {
		return "", ErrRPC
	}
	if len(txs) >= fundingWindow {
		return "", ErrAmbiguousFunding
	}
	want := normalized(address)
	sender := ""
	for _, tx := range txs {
		if normalized(tx.To) != want || !fundingMarker(tx.RecipientData) {
			continue
		}
		if !tx.ExecutionResult || tx.Confirmations < 1 {
			return "", ErrAmbiguousFunding // a funding that is not settled yet: not the moment to choose
		}
		if !ValidAddress(tx.From) || normalized(tx.From) == want {
			return "", ErrAmbiguousFunding
		}
		if sender != "" && normalized(sender) != normalized(tx.From) {
			return "", ErrAmbiguousFunding
		}
		sender = tx.From
	}
	return sender, nil
}
