// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

package evm

// A minimal RLP encoder: just what an EIP-1559 transaction needs (byte strings, unsigned integers, lists). Decoding is not
// needed anywhere (the tests decode with go-ethereum). RLP: https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/

import "math/big"

// rlpLen is the length prefix of a payload of n bytes: `short` is 0x80 for strings and 0xc0 for lists.
func rlpLen(n int, short byte) []byte {
	if n <= 55 {
		return []byte{short + byte(n)}
	}
	var b []byte
	for v := n; v > 0; v >>= 8 {
		b = append([]byte{byte(v)}, b...)
	}
	return append([]byte{short + 55 + byte(len(b))}, b...)
}

// rlpBytes encodes a byte string (a single byte below 0x80 is its own encoding).
func rlpBytes(b []byte) []byte {
	if len(b) == 1 && b[0] < 0x80 {
		return []byte{b[0]}
	}
	return append(rlpLen(len(b), 0x80), b...)
}

// rlpList encodes a list whose items are already RLP-encoded.
func rlpList(items ...[]byte) []byte {
	var body []byte
	for _, it := range items {
		body = append(body, it...)
	}
	return append(rlpLen(len(body), 0xc0), body...)
}

// rlpInt encodes a non-negative integer: big-endian, no leading zeros, zero is the empty string.
func rlpInt(v *big.Int) []byte {
	if v == nil || v.Sign() == 0 {
		return rlpBytes(nil)
	}
	return rlpBytes(v.Bytes())
}

// rlpUint is rlpInt for a uint64.
func rlpUint(v uint64) []byte { return rlpInt(new(big.Int).SetUint64(v)) }
