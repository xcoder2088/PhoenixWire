package nimiq

import (
	"context"
	"encoding/json"
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

// FundingSender is the address a Cashlink was funded from: the sender of its oldest confirmed funding transaction (the one that carries the
// funding mark). "" with a nil error when the node does not show one yet. The answer is the node's, like every chain fact here.
func (v *MainnetVerifier) FundingSender(ctx context.Context, address string) (string, error) {
	if v == nil || ctx == nil || !ValidAddress(address) {
		return "", ErrRPC
	}
	raw, err := v.call(ctx, "getTransactionsByAddress", []any{address, 50, nil})
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
	want := normalized(address)
	for i := len(txs) - 1; i >= 0; i-- { // the node lists the newest first: the oldest funding is the last match
		tx := txs[i]
		if normalized(tx.To) == want && tx.ExecutionResult && tx.Confirmations >= 1 && fundingMarker(tx.RecipientData) && ValidAddress(tx.From) && normalized(tx.From) != want {
			return tx.From, nil
		}
	}
	return "", nil
}
