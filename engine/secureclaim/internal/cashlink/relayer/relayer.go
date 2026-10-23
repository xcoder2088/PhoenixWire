// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

// Package relayer submits claim and refund transactions of USDT cashlinks and pays their gas, in exchange for the fee that
// the card itself pays it in USDT. Anyone can run one; it does not need to be trusted. It never executes data that came
// from a client: it receives three fields (recipient, amount, fee) and the owners' signatures, rebuilds the one batch the
// protocol allows, and refuses anything that does not verify. A relayer can only refuse or delay, never redirect money.
package relayer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/chain"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/safe"
)

var (
	ErrCard         = errors.New("card_mismatch")
	ErrAmount       = errors.New("invalid_amount")
	ErrFeeTooLow    = errors.New("fee_too_low")
	ErrNotFunded    = errors.New("card_not_funded")
	ErrSignatures   = errors.New("invalid_signatures")
	ErrRejected     = errors.New("transaction_would_revert")
	ErrRelayerFunds = errors.New("relayer_has_no_gas")
	ErrDestination  = errors.New("invalid_destination")
	ErrJournal      = errors.New("journal_failed")
	ErrFloorReached = errors.New("relayer_float_floor")
)

// Config of one relayer on one chain.
type Config struct {
	Chain chain.Chain
	Key   []byte      // the relayer's own key: pays gas in POL/ETH, receives the fee in USDT
	Token evm.Address // the USDT contract
	// MinFee is the smallest fee (USDT units) this relayer accepts. It is the operator's policy: it should cover the
	// gas of a deployment and a claim with a margin.
	MinFee *big.Int
	// MaxAmount caps one card (USDT units). Zero means 1,000 USDT.
	MaxAmount *big.Int
	// GasMargin is applied to estimates (percent, default 130).
	GasMargin uint64
	// MinExecGas is the least gas limit of a payout execution (default DefaultMinExecGas).
	MinExecGas uint64
	// CodeWaitTries and CodeWaitStep bound the wait for a freshly deployed card to be visible (default 15 x 1 s).
	CodeWaitTries int
	CodeWaitStep  time.Duration
}

// Relayer is one running relayer. Create it with New; Submit is the entry point; Handler (http.go) puts it on HTTP.
type Relayer struct {
	cfg  Config
	addr evm.Address
	mu   *sync.Mutex // the relayer's own nonce must not be used by two requests at once: shared by every Relayer of the same chain and key
}

// New checks the configuration (a valid key, a chain, a fee floor) and fills in the defaults: a ceiling of 1,000 USDT per
// card and a 30 % gas margin.
func New(cfg Config) (*Relayer, error) {
	addr, err := evm.AddressFromKey(cfg.Key)
	if err != nil || cfg.Chain == nil || cfg.MinFee == nil {
		return nil, errors.New("invalid_relayer_config")
	}
	if cfg.MaxAmount == nil {
		cfg.MaxAmount = big.NewInt(1_000_000_000)
	}
	if cfg.CodeWaitTries == 0 {
		cfg.CodeWaitTries = 15
	}
	if cfg.CodeWaitStep == 0 {
		cfg.CodeWaitStep = time.Second
	}
	if cfg.MinExecGas == 0 {
		cfg.MinExecGas = DefaultMinExecGas
	}
	if cfg.GasMargin == 0 {
		cfg.GasMargin = 130
	}
	return &Relayer{cfg: cfg, addr: addr, mu: nonceLock(cfg.Chain.ChainID(), addr)}, nil
}

var nonceLocks sync.Map

// nonceLock gives ONE mutex per (chain, relayer address): two Relayer objects that use the same key on the same chain (for
// instance one per token) must not allocate the same transaction nonce at the same time.
func nonceLock(chainID uint64, a evm.Address) *sync.Mutex {
	m, _ := nonceLocks.LoadOrStore(fmt.Sprintf("%d/%x", chainID, a[:]), &sync.Mutex{})
	return m.(*sync.Mutex)
}

// Address is where this relayer wants its fee.
func (r *Relayer) Address() evm.Address { return r.addr }

// Request is all a client sends. Nothing else: no calldata, no target, no gas.
type Request struct {
	Card       safe.Card
	Recipient  evm.Address
	Amount     *big.Int // USDT units paid to the recipient
	Fee        *big.Int // USDT units paid to this relayer
	Signatures []byte   // owners' signatures over the batch, sorted by owner address, 65 bytes each
	// Journal, when set, is called with the hash of each transaction (kind "deploy" or "exec") AFTER it is signed and BEFORE it
	// is broadcast. The caller writes it somewhere durable; if it returns an error nothing is sent. This is what lets a caller
	// that crashes, or loses the answer, find out later from the chain what became of the transaction.
	Journal func(kind, hash string) error
}

// Result of a submission.
type Result struct {
	Safe     evm.Address
	DeployTx string // empty when the card already existed
	ExecTx   string
}

func tokenBalance(ctx context.Context, c chain.Chain, token, owner evm.Address) (*big.Int, error) {
	data := append([]byte{0x70, 0xa0, 0x82, 0x31}, owner.Word()...)
	out, err := c.Call(ctx, evm.Address{}, token, data)
	if err != nil || len(out) < 32 {
		return nil, chain.ErrUnavailable
	}
	return new(big.Int).SetBytes(out[:32]), nil
}

// Check verifies a request completely and sends nothing. It returns the card address and the Safe transaction.
func (r *Relayer) Check(ctx context.Context, req Request) (evm.Address, safe.Tx, bool, error) {
	var zero safe.Tx
	cid := r.cfg.Chain.ChainID()
	card, err := req.Card.Address(cid)
	if err != nil {
		return card, zero, false, ErrCard
	}
	if req.Amount == nil || req.Fee == nil || req.Amount.Sign() <= 0 || req.Fee.Sign() < 0 || req.Amount.Cmp(r.cfg.MaxAmount) > 0 {
		return card, zero, false, ErrAmount
	}
	if req.Fee.Cmp(r.cfg.MinFee) < 0 {
		return card, zero, false, ErrFeeTooLow
	}
	if req.Recipient == (evm.Address{}) || req.Recipient == card || req.Recipient == r.cfg.Token || req.Recipient == r.addr {
		return card, zero, false, ErrDestination
	}
	if uint64(len(req.Signatures)) != req.Card.Threshold*65 {
		return card, zero, false, ErrSignatures
	}
	// Does the card exist? Its nonce is read from the contract, never taken from the client.
	code, err := r.cfg.Chain.Code(ctx, card)
	if err != nil {
		return card, zero, false, err
	}
	deployed := len(code) > 0
	var nonce uint64
	if deployed {
		out, err := r.cfg.Chain.Call(ctx, evm.Address{}, card, safe.NonceCall)
		if err != nil || len(out) < 32 {
			return card, zero, false, chain.ErrUnavailable
		}
		n := new(big.Int).SetBytes(out[:32])
		if !n.IsUint64() {
			return card, zero, false, ErrCard
		}
		nonce = n.Uint64()
	}
	tx := safe.Payout(r.cfg.Token, req.Recipient, r.addr, req.Amount, req.Fee, nonce)
	// The signatures must be those of `threshold` distinct owners of this card over exactly this transaction.
	hash := tx.Hash(cid, card)
	isOwner := map[evm.Address]bool{}
	for _, o := range req.Card.Owners {
		isOwner[o] = true
	}
	seen := map[evm.Address]bool{}
	var previous evm.Address
	for i := uint64(0); i < req.Card.Threshold; i++ {
		var sig evm.Signature
		chunk := req.Signatures[i*65 : (i+1)*65]
		copy(sig.R[:], chunk[:32])
		copy(sig.S[:], chunk[32:64])
		sig.V = chunk[64]
		if sig.V != 27 && sig.V != 28 {
			return card, zero, false, ErrSignatures
		}
		signer, err := evm.Recover(hash, sig)
		if err != nil || !isOwner[signer] || seen[signer] || (i > 0 && !previous.Less(signer)) {
			return card, zero, false, ErrSignatures
		}
		seen[signer], previous = true, signer
	}
	// The money must be there.
	balance, err := tokenBalance(ctx, r.cfg.Chain, r.cfg.Token, card)
	if err != nil {
		return card, zero, false, err
	}
	if balance.Cmp(new(big.Int).Add(req.Amount, req.Fee)) < 0 {
		return card, zero, false, ErrNotFunded
	}
	return card, tx, deployed, nil
}

func (r *Relayer) sendTx(ctx context.Context, to evm.Address, data []byte, gas uint64, kind string, floor *big.Int, journal func(kind, hash string) error) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	nonce, err := r.cfg.Chain.PendingNonce(ctx, r.addr)
	if err != nil {
		return "", err
	}
	tip, maxFee, err := r.cfg.Chain.Fees(ctx)
	if err != nil {
		return "", err
	}
	raw, hash, err := evm.Tx{ChainID: r.cfg.Chain.ChainID(), Nonce: nonce, MaxPriority: tip, MaxFee: maxFee, Gas: gas, To: to.Hex(), Value: new(big.Int), Data: data}.Sign(r.cfg.Key)
	if err != nil {
		return "", err
	}
	need := new(big.Int).Mul(maxFee, new(big.Int).SetUint64(gas))
	have, err := r.cfg.Chain.NativeBalance(ctx, r.addr)
	if err != nil {
		return "", err
	}
	if have.Cmp(need) < 0 {
		return "", ErrRelayerFunds
	}
	if floor != nil && have.Cmp(new(big.Int).Add(need, floor)) < 0 {
		return "", ErrFloorReached // the worst-case cost of THIS transaction, at the price read now, would eat into the reserve
	}
	if journal != nil {
		if err := journal(kind, hash); err != nil {
			return "", ErrJournal // nothing was sent
		}
	}
	if err := r.cfg.Chain.Send(ctx, raw); err != nil {
		return hash, err // the hash is returned: the transaction may exist even though the answer was lost
	}
	return hash, nil
}

func (r *Relayer) wait(ctx context.Context, hash string) (chain.Receipt, error) {
	for i := 0; i < 90; i++ {
		rc, err := r.cfg.Chain.Receipt(ctx, hash)
		if err == nil && rc.Found {
			return rc, nil
		}
		select {
		case <-ctx.Done():
			return chain.Receipt{}, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return chain.Receipt{}, chain.ErrUnavailable
}

// DefaultMinExecGas is the least gas limit given to a Safe payout execution. Measured on Polygon: a payout needs about 157k and
// fails (GS013) below that; unused gas is not charged.
const DefaultMinExecGas = 300000

// ErrNotDeployedYet: the deployment was mined but no node shows the card's code yet. Nothing was paid out; a retry finds the card
// deployed and goes straight to the payout.
var ErrNotDeployedYet = errors.New("deployment_not_visible")

// waitForCode waits (a few seconds at most) until the node answers that the card now has code: the deployment was seen mined
// by one node, the next call may reach another. It says so when the code never shows up, instead of paying out blindly.
func (r *Relayer) waitForCode(ctx context.Context, card evm.Address) error {
	for i := 0; i < r.cfg.CodeWaitTries; i++ {
		if code, err := r.cfg.Chain.Code(ctx, card); err == nil && len(code) > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.cfg.CodeWaitStep):
		}
	}
	return ErrNotDeployedYet
}

func (r *Relayer) gas(ctx context.Context, to evm.Address, data []byte) (uint64, error) {
	g, err := r.cfg.Chain.EstimateGas(ctx, r.addr, to, data)
	if err != nil {
		return 0, ErrRejected
	}
	return g * r.cfg.GasMargin / 100, nil
}

// Submit verifies the request, deploys the card if it does not exist yet, and executes the payout. It returns when both
// transactions are mined and successful.
func (r *Relayer) Submit(ctx context.Context, req Request) (Result, error) {
	card, tx, deployed, err := r.Check(ctx, req)
	if err != nil {
		return Result{Safe: card}, err
	}
	res := Result{Safe: card}
	if !deployed {
		data, err := req.Card.DeployData(r.cfg.Chain.ChainID())
		if err != nil {
			return res, ErrCard
		}
		gas, err := r.gas(ctx, safe.Factory, data)
		if err != nil {
			return res, err
		}
		if res.DeployTx, err = r.sendTx(ctx, safe.Factory, data, gas, "deploy", nil, req.Journal); err != nil {
			return res, err
		}
		rc, err := r.wait(ctx, res.DeployTx)
		if err != nil || !rc.Success {
			return res, ErrRejected
		}
		if err := r.waitForCode(ctx, card); err != nil {
			return res, err
		}
	}
	exec := tx.ExecData(req.Signatures)
	gas, err := r.gas(ctx, card, exec) // fails here if the payout would revert: nothing is sent
	if err != nil {
		return res, err
	}
	// A call to an address where nothing is deployed "succeeds" for ~50k gas. The node that estimates may not have seen the
	// deployment yet (public nodes answer from different blocks): its estimate is then far too low and the payout would run
	// out of gas and revert, as happened on the first real claim. A Safe execution that moves tokens never costs this little.
	if gas < r.cfg.MinExecGas {
		gas = r.cfg.MinExecGas
	}
	if res.ExecTx, err = r.sendTx(ctx, card, exec, gas, "exec", nil, req.Journal); err != nil {
		return res, err
	}
	rc, err := r.wait(ctx, res.ExecTx)
	if err != nil || !rc.Success {
		return res, ErrRejected
	}
	return res, nil
}

// Chain gives access to the node client (the service reads balances and fees through it).
func (r *Relayer) Chain() chain.Chain { return r.cfg.Chain }

// SendNative sends `wei` of the chain's native coin (gas money) from the relayer to `to`: how a sender who holds only the token
// is given the few cents of POL/ETH his one transfer needs. It refuses to leave the relayer with less than `floor` (the float
// that keeps claims and refunds payable). journal is called with the transaction hash AFTER signing and BEFORE broadcasting,
// like Request.Journal: if it fails nothing is sent; on a lost answer the hash is still returned.
func (r *Relayer) SendNative(ctx context.Context, to evm.Address, wei, floor *big.Int, journal func(hash string) error) (string, error) {
	if wei == nil || wei.Sign() <= 0 || to == (evm.Address{}) {
		return "", ErrAmount
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	nonce, err := r.cfg.Chain.PendingNonce(ctx, r.addr)
	if err != nil {
		return "", err
	}
	tip, maxFee, err := r.cfg.Chain.Fees(ctx)
	if err != nil {
		return "", err
	}
	const gas = 21000
	raw, hash, err := evm.Tx{ChainID: r.cfg.Chain.ChainID(), Nonce: nonce, MaxPriority: tip, MaxFee: maxFee, Gas: gas, To: to.Hex(), Value: new(big.Int).Set(wei)}.Sign(r.cfg.Key)
	if err != nil {
		return "", err
	}
	need := new(big.Int).Add(wei, new(big.Int).Mul(maxFee, big.NewInt(gas)))
	if floor != nil {
		need.Add(need, floor)
	}
	have, err := r.cfg.Chain.NativeBalance(ctx, r.addr)
	if err != nil {
		return "", err
	}
	if have.Cmp(need) < 0 {
		return "", ErrFloorReached
	}
	if journal != nil {
		if err := journal(hash); err != nil {
			return "", ErrJournal
		}
	}
	if err := r.cfg.Chain.Send(ctx, raw); err != nil {
		return hash, err
	}
	return hash, nil
}

// SweepNative sends the relayer's whole native balance (less the gas of this very transfer) to `to`: how the owner empties the
// relayer. It holds the same nonce lock as every other send of this key.
func (r *Relayer) SweepNative(ctx context.Context, to evm.Address, journal func(hash string) error) (string, *big.Int, error) {
	if to == (evm.Address{}) || to == r.addr {
		return "", nil, ErrDestination
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	nonce, err := r.cfg.Chain.PendingNonce(ctx, r.addr)
	if err != nil {
		return "", nil, err
	}
	tip, maxFee, err := r.cfg.Chain.Fees(ctx)
	if err != nil {
		return "", nil, err
	}
	have, err := r.cfg.Chain.NativeBalance(ctx, r.addr)
	if err != nil {
		return "", nil, err
	}
	const gas = 21000
	wei := new(big.Int).Sub(have, new(big.Int).Mul(maxFee, big.NewInt(gas)))
	if wei.Sign() <= 0 {
		return "", nil, ErrAmount
	}
	raw, hash, err := evm.Tx{ChainID: r.cfg.Chain.ChainID(), Nonce: nonce, MaxPriority: tip, MaxFee: maxFee, Gas: gas, To: to.Hex(), Value: new(big.Int).Set(wei)}.Sign(r.cfg.Key)
	if err != nil {
		return "", nil, err
	}
	if journal != nil {
		if err := journal(hash); err != nil {
			return "", nil, ErrJournal
		}
	}
	if err := r.cfg.Chain.Send(ctx, raw); err != nil {
		return hash, wei, err
	}
	return hash, wei, nil
}

// SendToken sends `units` of `token` from the relayer's own address to `to`: how the owner takes out the fees the relayer earned.
// The gas is paid in the native coin by the relayer itself, and the transaction is refused (ErrFloorReached) when its worst-case cost,
// at the fees read under the send lock, would leave the relayer below `floor` (the reserve the live cards need). journal is called with
// the hash AFTER signing and BEFORE broadcasting.
func (r *Relayer) SendToken(ctx context.Context, token, to evm.Address, units *big.Int, floor *big.Int, journal func(hash string) error) (string, error) {
	if units == nil || units.Sign() <= 0 || to == (evm.Address{}) || to == r.addr || token == (evm.Address{}) {
		return "", ErrAmount
	}
	data := safe.TransferData(to, units)
	gas, err := r.gas(ctx, token, data) // fails when the relayer does not hold that much
	if err != nil {
		return "", err
	}
	return r.sendTx(ctx, token, data, gas, "withdraw", floor, func(_, hash string) error {
		if journal == nil {
			return nil
		}
		return journal(hash)
	})
}

// ErrPayoutLogs: the transaction succeeded but did not emit the transfers a payout must emit.
var ErrPayoutLogs = errors.New("payout_transfers_not_found")

var transferTopic = evm.Keccak256([]byte("Transfer(address,address,uint256)"))

// VerifyPayout checks a SUCCESSFUL receipt against what the payout was supposed to do: the token contract emitted a
// Transfer(card -> recipient, amount) and, when fee > 0, a Transfer(card -> relayer, fee). A token that returns false without
// reverting, or a transaction that did something else, succeeds on chain yet fails this check: such a card is never shown as paid.
func VerifyPayout(rc chain.Receipt, token, card, recipient, relayer evm.Address, amount, fee *big.Int) error {
	has := func(to evm.Address, want *big.Int) bool {
		for _, l := range rc.Logs {
			if l.Address != token || len(l.Topics) != 3 || !bytesEqual(l.Topics[0], transferTopic) || len(l.Data) != 32 {
				continue
			}
			if bytesEqual(l.Topics[1], card.Word()) && bytesEqual(l.Topics[2], to.Word()) && new(big.Int).SetBytes(l.Data).Cmp(want) == 0 {
				return true
			}
		}
		return false
	}
	if !rc.Found || !rc.Success || !has(recipient, amount) || (fee != nil && fee.Sign() > 0 && !has(relayer, fee)) {
		return ErrPayoutLogs
	}
	return nil
}

func bytesEqual(a, b []byte) bool { return string(a) == string(b) }
