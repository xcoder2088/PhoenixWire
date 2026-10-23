// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

package evm

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// ErrInvalidAddress is returned for anything that is not a 0x + 40 hex address (or has a wrong EIP-55 checksum).
var ErrInvalidAddress = errors.New("invalid_evm_address")

// isHex reports whether s is made only of hexadecimal digits.
func isHex(s string) bool {
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// Checksum returns the EIP-55 mixed-case form of a 20-byte hex address.
func Checksum(address string) string {
	lower := strings.ToLower(strings.TrimPrefix(address, "0x"))
	hash := hex.EncodeToString(Keccak256([]byte(lower)))
	out := make([]byte, 40)
	for i := 0; i < 40; i++ {
		c := lower[i]
		if c >= 'a' && c <= 'f' && hash[i] >= '8' {
			c -= 32
		}
		out[i] = c
	}
	return "0x" + string(out)
}

// ValidAddress accepts 0x + 40 hex. Mixed case must carry a correct EIP-55 checksum (catches typos);
// all-lowercase or all-uppercase is accepted without a checksum. The zero address is refused.
func ValidAddress(address string) bool {
	if len(address) != 42 || !strings.HasPrefix(address, "0x") || !isHex(address[2:]) {
		return false
	}
	body := address[2:]
	if body == strings.Repeat("0", 40) {
		return false
	}
	if body != strings.ToLower(body) && body != strings.ToUpper(body) && Checksum(address) != address {
		return false
	}
	return true
}

// NormalizeAddress validates and returns the checksummed address.
func NormalizeAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	if !ValidAddress(address) {
		return "", ErrInvalidAddress
	}
	return Checksum(address), nil
}

// AddressOf derives the checksummed address of a 32-byte private key.
func AddressOf(private []byte) (string, error) {
	if len(private) != 32 {
		return "", errors.New("invalid_private_key")
	}
	key := secp256k1.PrivKeyFromBytes(private)
	defer key.Zero()
	pub := key.PubKey().SerializeUncompressed()
	return Checksum("0x" + hex.EncodeToString(Keccak256(pub[1:])[12:])), nil
}

// GenerateKey returns a fresh random private key (never zero, below the curve order) and its address.
func GenerateKey() (private []byte, address string, err error) {
	for i := 0; i < 8; i++ {
		b := make([]byte, 32)
		if _, err = rand.Read(b); err != nil {
			return nil, "", err
		}
		var scalar secp256k1.ModNScalar
		if overflow := scalar.SetByteSlice(b); overflow || scalar.IsZero() {
			clear(b)
			continue
		}
		address, err = AddressOf(b)
		if err != nil {
			return nil, "", err
		}
		return b, address, nil
	}
	return nil, "", errors.New("key_generation_failed")
}
