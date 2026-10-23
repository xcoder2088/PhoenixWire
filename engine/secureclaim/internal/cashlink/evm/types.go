// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

package evm

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// Address is a 20-byte Ethereum address.
type Address [20]byte

// ParseAddress accepts 0x + 40 hex (EIP-55 checked when mixed case).
func ParseAddress(s string) (Address, error) {
	var a Address
	s = strings.TrimSpace(s)
	if !ValidAddress(s) {
		return a, ErrInvalidAddress
	}
	b, _ := hex.DecodeString(s[2:])
	copy(a[:], b)
	return a, nil
}

// MustAddress is for constants.
func MustAddress(s string) Address {
	a, err := ParseAddress(s)
	if err != nil {
		panic("invalid address constant " + s)
	}
	return a
}

// Hex is the EIP-55 form.
func (a Address) Hex() string { return Checksum("0x" + hex.EncodeToString(a[:])) }

// Word is the address left-padded to 32 bytes (ABI encoding).
func (a Address) Word() []byte { return append(make([]byte, 12), a[:]...) }

// Less orders addresses as numbers (Safe wants signatures sorted by signer address).
func (a Address) Less(b Address) bool { return bytes.Compare(a[:], b[:]) < 0 }

// Signature is an ECDSA signature over a 32-byte digest, V being 27 or 28.
type Signature struct {
	R, S [32]byte
	V    byte
}

// Bytes is r || s || v (65 bytes), the layout Safe expects for an owner's signature.
func (s Signature) Bytes() []byte {
	out := make([]byte, 0, 65)
	out = append(out, s.R[:]...)
	out = append(out, s.S[:]...)
	return append(out, s.V)
}

// SignDigest signs a 32-byte digest with a private key (deterministic RFC 6979, low-S).
func SignDigest(private, digest []byte) (Signature, error) {
	var sig Signature
	if len(private) != 32 || len(digest) != 32 {
		return sig, errors.New("invalid_key_or_digest")
	}
	key := secp256k1.PrivKeyFromBytes(private)
	defer key.Zero()
	compact := ecdsa.SignCompact(key, digest, false)
	sig.V = compact[0] // 27 + recovery id
	copy(sig.R[:], compact[1:33])
	copy(sig.S[:], compact[33:65])
	return sig, nil
}

// Recover returns the address that signed digest.
func Recover(digest []byte, sig Signature) (Address, error) {
	var a Address
	compact := append([]byte{sig.V}, append(sig.R[:], sig.S[:]...)...)
	pub, _, err := ecdsa.RecoverCompact(compact, digest)
	if err != nil {
		return a, err
	}
	copy(a[:], Keccak256(pub.SerializeUncompressed()[1:])[12:])
	return a, nil
}

// AddressFromKey derives the address of a private key.
func AddressFromKey(private []byte) (Address, error) {
	var a Address
	s, err := AddressOf(private)
	if err != nil {
		return a, err
	}
	return ParseAddress(s)
}

// Word32 left-pads a big integer to 32 bytes.
func Word32(v *big.Int) []byte { return v.FillBytes(make([]byte, 32)) }
