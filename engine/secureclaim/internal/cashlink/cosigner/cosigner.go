// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

// Package cosigner is the second key of a protected card. Its key is derived per card from a master secret, so the server
// stores nothing per card; and it signs only a payout of a card it is an owner of, never anything else. WHEN to sign (after
// the recipient proved his email) is the caller's decision: this package is only the key and the safety checks.
package cosigner

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/safe"
)

var (
	ErrNotOwner = errors.New("cosigner_is_not_an_owner_of_this_card")
	ErrMaster   = errors.New("master_secret_too_short")
)

// Signer derives and uses the co-signer keys. It holds only the master secret.
type Signer struct{ master []byte }

// New takes the master secret (at least 32 random bytes, generated once and never replaced: protected cards depend on it).
func New(master []byte) (*Signer, error) {
	if len(master) < 32 {
		return nil, ErrMaster
	}
	return &Signer{master: append([]byte(nil), master...)}, nil
}

var curveOrder, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)

// Key derives the card's co-signer private key. Callers must clear it after use.
func (s *Signer) Key(chainID uint64, cardID []byte) []byte {
	for counter := byte(0); ; counter++ {
		mac := hmac.New(sha256.New, s.master)
		mac.Write([]byte("PhoenixCashlink v1 cosigner"))
		mac.Write(evm.Word32(new(big.Int).SetUint64(chainID)))
		mac.Write(cardID)
		mac.Write([]byte{counter})
		k := mac.Sum(nil)
		if v := new(big.Int).SetBytes(k); v.Sign() > 0 && v.Cmp(curveOrder) < 0 {
			return k
		}
	}
}

// Address is what the sender's browser needs to put in the card's owners; it is public.
func (s *Signer) Address(chainID uint64, cardID []byte) (evm.Address, error) {
	k := s.Key(chainID, cardID)
	defer clear(k)
	return evm.AddressFromKey(k)
}

// SignPayout signs one payout of one card: only if this signer is one of the card's owners, and only for the batch the
// protocol allows (the relayer rebuilds the same one). It returns the signature and its owner.
func (s *Signer) SignPayout(chainID uint64, card safe.Card, token, recipient, relayer evm.Address, amount, fee *big.Int, nonce uint64) (safe.Signed, error) {
	key := s.Key(chainID, card.ID)
	defer clear(key)
	me, err := evm.AddressFromKey(key)
	if err != nil {
		return safe.Signed{}, err
	}
	owns := false
	for _, o := range card.Owners {
		if o == me {
			owns = true
		}
	}
	if !owns {
		return safe.Signed{}, ErrNotOwner
	}
	addr, err := card.Address(chainID)
	if err != nil {
		return safe.Signed{}, err
	}
	tx := safe.Payout(token, recipient, relayer, amount, fee, nonce)
	owner, sig, err := tx.Sign(chainID, addr, key)
	return safe.Signed{Owner: owner, Sig: sig}, err
}
