// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

// Package safe builds and signs everything a USDT cashlink needs from the canonical Safe 1.4.1 contracts: the card
// address (computed before the card exists), the claim/refund transaction, its EIP-712 hash, owner signatures and the
// calldata a relayer submits. No smart contract of our own: only the audited, immutable, canonical Safe deployments
// (identical code on Ethereum and Polygon, checked in testdata/safe-1.4.1.json).
package safe

import (
	"errors"
	"math/big"
	"sort"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
)

// Canonical Safe 1.4.1 deployments (same addresses on every chain that has them).
var (
	Factory           = evm.MustAddress("0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67") // SafeProxyFactory
	Singleton         = evm.MustAddress("0x41675C099F32341bf84BFc5382aF534df5C7461a") // Safe (not L2: cheaper, events are not needed)
	MultiSendCallOnly = evm.MustAddress("0x9641d764fc13c8B624c04430C7356C1C7C8102e2")
)

// ProxyCreationCode is SafeProxyFactory.proxyCreationCode() of 1.4.1 (the constant init code of every Safe proxy).
// It is read from the factory once and checked in the tests against the real contract.
var ProxyCreationCode []byte

var (
	selSetup                = []byte{0xb6, 0x3e, 0x80, 0x0d} // setup(address[],uint256,address,bytes,address,address,uint256,address)
	selCreateProxyNonce     = []byte{0x16, 0x88, 0xf0, 0xb9} // createProxyWithNonce(address,bytes,uint256)
	selExecTransaction      = []byte{0x6a, 0x76, 0x12, 0x02} // execTransaction(address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,bytes)
	selMultiSend            = []byte{0x8d, 0x80, 0xff, 0x0a} // multiSend(bytes)
	selTransfer             = []byte{0xa9, 0x05, 0x9c, 0xbb} // transfer(address,uint256)
	domainSeparatorTypehash = evm.Keccak256([]byte("EIP712Domain(uint256 chainId,address verifyingContract)"))
	safeTxTypehash          = evm.Keccak256([]byte("SafeTx(address to,uint256 value,bytes data,uint8 operation,uint256 safeTxGas,uint256 baseGas,uint256 gasPrice,address gasToken,address refundReceiver,uint256 nonce)"))
)

var (
	ErrOwners    = errors.New("invalid_owners")
	ErrThreshold = errors.New("invalid_threshold")
)

func pad32(b []byte) []byte {
	if n := len(b) % 32; n != 0 {
		b = append(b, make([]byte, 32-n)...)
	}
	return b
}

func u256(v uint64) []byte { return evm.Word32(new(big.Int).SetUint64(v)) }

// sortedOwners returns the owners in ascending order and refuses duplicates and the zero address.
func sortedOwners(owners []evm.Address) ([]evm.Address, error) {
	if len(owners) == 0 || len(owners) > 3 {
		return nil, ErrOwners
	}
	out := append([]evm.Address(nil), owners...)
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	for i, o := range out {
		if o == (evm.Address{}) || (i > 0 && out[i-1] == o) {
			return nil, ErrOwners
		}
	}
	return out, nil
}

// SetupData is the Safe.setup(...) call that initialises a new proxy: these owners, this threshold, no module setup, no
// fallback handler, no payment. Owners are sorted so that the same card always gives the same address.
func SetupData(owners []evm.Address, threshold uint64) ([]byte, error) {
	list, err := sortedOwners(owners)
	if err != nil {
		return nil, err
	}
	if threshold == 0 || threshold > uint64(len(list)) {
		return nil, ErrThreshold
	}
	zero := make([]byte, 32)
	out := append([]byte(nil), selSetup...)
	ownersOffset := uint64(8 * 32)
	dataOffset := ownersOffset + 32 + uint64(32*len(list))
	out = append(out, u256(ownersOffset)...) // address[] _owners
	out = append(out, u256(threshold)...)    // uint256 _threshold
	out = append(out, zero...)               // address to
	out = append(out, u256(dataOffset)...)   // bytes data
	out = append(out, zero...)               // address fallbackHandler
	out = append(out, zero...)               // address paymentToken
	out = append(out, zero...)               // uint256 payment
	out = append(out, zero...)               // address payable paymentReceiver
	out = append(out, u256(uint64(len(list)))...)
	for _, o := range list {
		out = append(out, o.Word()...)
	}
	out = append(out, u256(0)...) // empty bytes data
	return out, nil
}

// SaltNonce makes the card address depend on the chain and the card id (and, through the initializer, on its owners).
func SaltNonce(chainID uint64, cardID []byte) *big.Int {
	return new(big.Int).SetBytes(evm.Keccak256([]byte("PhoenixCashlink v1"), u256(chainID), cardID))
}

// ProxyAddress computes, without any chain access, where SafeProxyFactory.createProxyWithNonce(Singleton, initializer, nonce)
// will deploy the proxy (CREATE2). USDT can be sent there before it exists.
func ProxyAddress(initializer []byte, saltNonce *big.Int) evm.Address {
	salt := evm.Keccak256(evm.Keccak256(initializer), evm.Word32(saltNonce))
	initCode := append(append([]byte(nil), ProxyCreationCode...), Singleton.Word()...)
	h := evm.Keccak256([]byte{0xff}, Factory[:], salt, evm.Keccak256(initCode))
	var a evm.Address
	copy(a[:], h[12:])
	return a
}

// CreateProxyData is the factory call a relayer sends to deploy the card (before the first claim or refund).
func CreateProxyData(initializer []byte, saltNonce *big.Int) []byte {
	out := append([]byte(nil), selCreateProxyNonce...)
	out = append(out, Singleton.Word()...)
	out = append(out, u256(3*32)...)
	out = append(out, evm.Word32(saltNonce)...)
	out = append(out, u256(uint64(len(initializer)))...)
	return append(out, pad32(append([]byte(nil), initializer...))...)
}

// Operation of a Safe transaction.
const (
	Call         = 0
	DelegateCall = 1
)

// Tx is a Safe transaction. Gas fields stay zero: the relayer is paid in USDT by the batch itself, not by Safe's own refund.
type Tx struct {
	To        evm.Address
	Value     *big.Int
	Data      []byte
	Operation uint8
	Nonce     uint64
}

// Hash is the EIP-712 hash owners sign (Safe.getTransactionHash).
func (t Tx) Hash(chainID uint64, safe evm.Address) []byte {
	domain := evm.Keccak256(domainSeparatorTypehash, u256(chainID), safe.Word())
	value := t.Value
	if value == nil {
		value = new(big.Int)
	}
	zero := make([]byte, 32)
	body := evm.Keccak256(safeTxTypehash, t.To.Word(), evm.Word32(value), evm.Keccak256(t.Data), u256(uint64(t.Operation)),
		zero, zero, zero, zero, zero, u256(t.Nonce)) // safeTxGas, baseGas, gasPrice, gasToken, refundReceiver = 0
	return evm.Keccak256([]byte{0x19, 0x01}, domain, body)
}

// Sign signs the transaction hash with an owner key.
func (t Tx) Sign(chainID uint64, safe evm.Address, key []byte) (evm.Address, evm.Signature, error) {
	owner, err := evm.AddressFromKey(key)
	if err != nil {
		return owner, evm.Signature{}, err
	}
	sig, err := evm.SignDigest(key, t.Hash(chainID, safe))
	return owner, sig, err
}

// Signed is one owner's signature.
type Signed struct {
	Owner evm.Address
	Sig   evm.Signature
}

// Signatures packs owner signatures sorted by owner address, as Safe requires.
func Signatures(list ...Signed) []byte {
	l := append([]Signed(nil), list...)
	sort.Slice(l, func(i, j int) bool { return l[i].Owner.Less(l[j].Owner) })
	var out []byte
	for _, s := range l {
		out = append(out, s.Sig.Bytes()...)
	}
	return out
}

// ExecData is the Safe.execTransaction call a relayer sends once the proxy exists.
func (t Tx) ExecData(signatures []byte) []byte {
	value := t.Value
	if value == nil {
		value = new(big.Int)
	}
	zero := make([]byte, 32)
	out := append([]byte(nil), selExecTransaction...)
	dataOffset := uint64(10 * 32)
	sigOffset := dataOffset + 32 + uint64(len(pad32(append([]byte(nil), t.Data...))))
	out = append(out, t.To.Word()...)
	out = append(out, evm.Word32(value)...)
	out = append(out, u256(dataOffset)...)
	out = append(out, u256(uint64(t.Operation))...)
	out = append(out, zero...) // safeTxGas
	out = append(out, zero...) // baseGas
	out = append(out, zero...) // gasPrice
	out = append(out, zero...) // gasToken
	out = append(out, zero...) // refundReceiver
	out = append(out, u256(sigOffset)...)
	out = append(out, u256(uint64(len(t.Data)))...)
	out = append(out, pad32(append([]byte(nil), t.Data...))...)
	out = append(out, u256(uint64(len(signatures)))...)
	return append(out, pad32(append([]byte(nil), signatures...))...)
}

// TransferData is ERC-20 transfer(to, amount) calldata.
func TransferData(to evm.Address, amount *big.Int) []byte {
	out := append([]byte(nil), selTransfer...)
	out = append(out, to.Word()...)
	return append(out, evm.Word32(amount)...)
}

// BatchCall is one call inside a MultiSend batch: the target, the native value and the calldata.
type BatchCall struct {
	To    evm.Address
	Value *big.Int
	Data  []byte
}

// MultiSendData packs calls for MultiSendCallOnly.multiSend(bytes). Each call is encoded as operation(1 byte, always 0 =
// CALL) to(20) value(32) dataLength(32) data; the whole is the single `bytes` argument of the function.
func MultiSendData(calls ...BatchCall) []byte {
	var packed []byte
	for _, c := range calls {
		v := c.Value
		if v == nil {
			v = new(big.Int)
		}
		packed = append(packed, 0) // CALL only
		packed = append(packed, c.To[:]...)
		packed = append(packed, evm.Word32(v)...)
		packed = append(packed, u256(uint64(len(c.Data)))...)
		packed = append(packed, c.Data...)
	}
	out := append([]byte(nil), selMultiSend...)
	out = append(out, u256(32)...)
	out = append(out, u256(uint64(len(packed)))...)
	return append(out, pad32(packed)...)
}

// Payout builds the claim (or refund): the card pays `toRecipient` USDT to `recipient` and `fee` USDT to `relayer`, in one
// batch executed by the Safe (a delegatecall to MultiSendCallOnly, whose inner calls are plain CALLs). Whatever signs it
// signs exactly this destination and this fee: a relayer can only execute it as signed.
func Payout(usdt, recipient, relayer evm.Address, toRecipient, fee *big.Int, nonce uint64) Tx {
	return Tx{
		To:        MultiSendCallOnly,
		Value:     new(big.Int),
		Operation: DelegateCall,
		Nonce:     nonce,
		Data: MultiSendData(
			BatchCall{To: usdt, Data: TransferData(recipient, toRecipient)},
			BatchCall{To: usdt, Data: TransferData(relayer, fee)},
		),
	}
}

// Card is everything that fixes a card's address: its id and its owners (the link key, and for a protected card the
// co-signer key) with the threshold.
type Card struct {
	ID        []byte // 16-32 random bytes chosen by the sender's browser
	Owners    []evm.Address
	Threshold uint64
}

// Initializer is the Safe.setup call of this card.
func (c Card) Initializer() ([]byte, error) { return SetupData(c.Owners, c.Threshold) }

// Address is where this card lives on a chain (computed offline).
func (c Card) Address(chainID uint64) (evm.Address, error) {
	if len(c.ID) < 16 || len(c.ID) > 32 {
		return evm.Address{}, errors.New("invalid_card_id")
	}
	init, err := c.Initializer()
	if err != nil {
		return evm.Address{}, err
	}
	return ProxyAddress(init, SaltNonce(chainID, c.ID)), nil
}

// DeployData is the factory call that creates the card.
func (c Card) DeployData(chainID uint64) ([]byte, error) {
	init, err := c.Initializer()
	if err != nil {
		return nil, err
	}
	return CreateProxyData(init, SaltNonce(chainID, c.ID)), nil
}

// NonceCall is Safe.nonce() calldata.
var NonceCall = []byte{0xaf, 0xfe, 0xd0, 0xe0}
