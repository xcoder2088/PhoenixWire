// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

// Package proof is how a browser proves, without ever sending the link key, that it holds a card's link key AND which single
// action it wants to authorize. It signs an EIP-712 typed message (LinkAuthorization) with the link key. This is NOT a Safe
// transaction: its domain and type differ from SafeTx, so the signature can never be replayed as a spend of the card.
//
// The server builds the same message itself from its own records (it never accepts a digest from the client), recovers the
// signer and compares it with the card's registered link owner.
package proof

import (
	"errors"
	"math/big"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
)

// Actions a proof can authorize.
const (
	ActionClaim  uint8 = 1 // the recipient asks for his payout
	ActionRefund uint8 = 2 // the sender asks to take the money back
	ActionCancel uint8 = 3 // the sender asks that the recipient no longer be co-signed (no Safe transaction is signed)
	ActionEmail  uint8 = 4 // the recipient asks for the e-mail code, BEFORE choosing where to receive (no destination, no transaction)
)

var (
	ErrAction    = errors.New("invalid_action")
	ErrSignature = errors.New("invalid_link_proof")

	domainTypehash = evm.Keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash       = evm.Keccak256([]byte("PhoenixCashlinkAuthorization"))
	versionHash    = evm.Keccak256([]byte("1"))
	proofTypehash  = evm.Keccak256([]byte("LinkAuthorization(bytes16 cardId,bytes16 authorizationId,bytes32 sessionBinding,uint8 action,bytes32 emailBinding,address destination,bytes32 safeTxHash,bytes32 challenge,uint64 deadline)"))

	secp256k1N, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
)

// LinkAuthorization is the message. Every field is chosen by the SERVER except the signature itself: the card, the
// authorization attempt, the opaque session binding, the e-mail binding, the destination the user confirmed, the exact Safe
// transaction hash, a random challenge and the deadline. For ActionCancel and ActionEmail, Destination and SafeTxHash are zero.
type LinkAuthorization struct {
	CardID          [16]byte
	AuthorizationID [16]byte
	SessionBinding  [32]byte
	Action          uint8
	EmailBinding    [32]byte
	Destination     evm.Address
	SafeTxHash      [32]byte
	Challenge       [32]byte
	Deadline        uint64 // Unix seconds, UTC
}

func right32(b []byte) []byte { return append(append([]byte{}, b...), make([]byte, 32-len(b))...) } // bytesN is right-padded

func (a LinkAuthorization) valid() error {
	switch a.Action {
	case ActionClaim, ActionRefund:
	case ActionCancel, ActionEmail:
		var zero [32]byte
		if a.Destination != (evm.Address{}) || a.SafeTxHash != zero {
			return ErrAction // these sign no destination and no transaction
		}
	default:
		return ErrAction
	}
	return nil
}

// Digest is the EIP-712 hash to sign: the domain binds the chain and the card's Safe address.
func (a LinkAuthorization) Digest(chainID uint64, safe evm.Address) ([]byte, error) {
	if err := a.valid(); err != nil {
		return nil, err
	}
	domain := evm.Keccak256(domainTypehash, nameHash, versionHash, evm.Word32(new(big.Int).SetUint64(chainID)), safe.Word())
	body := evm.Keccak256(proofTypehash, right32(a.CardID[:]), right32(a.AuthorizationID[:]), a.SessionBinding[:],
		evm.Word32(new(big.Int).SetUint64(uint64(a.Action))), a.EmailBinding[:], a.Destination.Word(), a.SafeTxHash[:], a.Challenge[:],
		evm.Word32(new(big.Int).SetUint64(a.Deadline)))
	return evm.Keccak256([]byte{0x19, 0x01}, domain, body), nil
}

// Sign makes the proof with the link key (what the browser does; Go is used by tests and tools).
func (a LinkAuthorization) Sign(chainID uint64, safe evm.Address, key []byte) (evm.Signature, error) {
	d, err := a.Digest(chainID, safe)
	if err != nil {
		return evm.Signature{}, err
	}
	return evm.SignDigest(key, d)
}

// Signer recovers who signed. It accepts only canonical signatures (v 27 or 28, low s), so one proof has one encoding.
func (a LinkAuthorization) Signer(chainID uint64, safe evm.Address, sig []byte) (evm.Address, error) {
	var zero evm.Address
	if len(sig) != 65 || (sig[64] != 27 && sig[64] != 28) {
		return zero, ErrSignature
	}
	var s evm.Signature
	copy(s.R[:], sig[:32])
	copy(s.S[:], sig[32:64])
	s.V = sig[64]
	if new(big.Int).SetBytes(s.S[:]).Cmp(new(big.Int).Rsh(secp256k1N, 1)) > 0 {
		return zero, ErrSignature
	}
	d, err := a.Digest(chainID, safe)
	if err != nil {
		return zero, err
	}
	who, err := evm.Recover(d, s)
	if err != nil {
		return zero, ErrSignature
	}
	return who, nil
}
