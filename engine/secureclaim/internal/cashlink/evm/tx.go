// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

package evm

import (
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// Tx is an EIP-1559 (type 2) transaction. Both Ethereum and Polygon accept it.
type Tx struct {
	ChainID     uint64
	Nonce       uint64
	MaxPriority *big.Int
	MaxFee      *big.Int
	Gas         uint64
	To          string // 0x address
	Value       *big.Int
	Data        []byte
}

// fields returns the nine RLP-encoded fields common to the signed payload and the signed transaction
// (chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data, accessList=[]).
func (t Tx) fields() ([][]byte, error) {
	to, err := hex.DecodeString(t.To[2:])
	if len(t.To) != 42 || err != nil {
		return nil, ErrInvalidAddress
	}
	if t.MaxFee == nil || t.MaxPriority == nil || t.MaxFee.Cmp(t.MaxPriority) < 0 {
		return nil, errors.New("invalid_fee")
	}
	return [][]byte{rlpUint(t.ChainID), rlpUint(t.Nonce), rlpInt(t.MaxPriority), rlpInt(t.MaxFee), rlpUint(t.Gas),
		rlpBytes(to), rlpInt(t.Value), rlpBytes(t.Data), rlpList()}, nil
}

// Sign returns the raw transaction bytes (0x02 || rlp) and its hash.
func (t Tx) Sign(private []byte) (raw []byte, hash string, err error) {
	fields, err := t.fields()
	if err != nil {
		return nil, "", err
	}
	digest := Keccak256([]byte{0x02}, rlpList(fields...))
	key := secp256k1.PrivKeyFromBytes(private)
	defer key.Zero()
	compact := ecdsa.SignCompact(key, digest, false) // [27+recid, r(32), s(32)], low-S, RFC 6979
	parity := uint64(compact[0] - 27)
	r := new(big.Int).SetBytes(compact[1:33])
	s := new(big.Int).SetBytes(compact[33:65])
	signed := append(append([][]byte{}, fields...), rlpUint(parity), rlpInt(r), rlpInt(s))
	raw = append([]byte{0x02}, rlpList(signed...)...)
	return raw, "0x" + hex.EncodeToString(Keccak256(raw)), nil
}

// TransferData encodes ERC-20 transfer(to, amount).
func TransferData(to string, amount *big.Int) ([]byte, error) {
	addr, err := hex.DecodeString(to[2:])
	if len(to) != 42 || err != nil {
		return nil, ErrInvalidAddress
	}
	data := []byte{0xa9, 0x05, 0x9c, 0xbb}
	data = append(data, make([]byte, 12)...)
	data = append(data, addr...)
	word := amount.FillBytes(make([]byte, 32))
	return append(data, word...), nil
}

// BalanceOfData encodes ERC-20 balanceOf(owner).
func BalanceOfData(owner string) ([]byte, error) {
	addr, err := hex.DecodeString(owner[2:])
	if len(owner) != 42 || err != nil {
		return nil, ErrInvalidAddress
	}
	data := []byte{0x70, 0xa0, 0x82, 0x31}
	data = append(data, make([]byte, 12)...)
	return append(data, addr...), nil
}
