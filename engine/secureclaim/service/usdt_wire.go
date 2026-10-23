package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
)

// Helpers for the Wire gateway: they only compute or check, they hold no key and no state.

// USDTPayLink is the EIP-681 link (and QR payload) of the ONE payment a sender makes to a card's address.
func USDTPayLink(asset, safe string, units uint64) (string, error) {
	n, err := evm.Lookup(asset)
	if err != nil || units == 0 {
		return "", errors.New("invalid_pay_link")
	}
	addr, err := cl.ParseAddress(safe)
	if err != nil || addr.Hex() != safe {
		return "", errors.New("invalid_pay_link")
	}
	return fmt.Sprintf("ethereum:%s@%d/transfer?address=%s&uint256=%d", n.Token, n.ChainID, safe, units), nil
}

// USDTAssetKnown says whether an asset id is one the engine serves (the chains the operator enabled are checked by the engine itself).
func USDTAssetKnown(asset string) bool { _, err := evm.Lookup(asset); return err == nil }

var b64u = base64.RawURLEncoding

// CheckClaimLink checks that a claim URL really is this card's: on Wire's own origin, the claim page, only an allowed language in the
// query, and a fragment "#c=<card id>.<link key>" whose card id is this card and whose key belongs to the card's link owner. The key
// is used transiently to compute that address and is not kept. Wire receives this full link at invitation delivery,
// then passes it to its e-mail provider; this is distinct from recipient claim and sender refund calls.
func CheckClaimLink(rawURL, origin, cardID, linkOwner string) error {
	u, err := url.Parse(rawURL)
	o, err2 := url.Parse(origin)
	if err != nil || err2 != nil || u.Scheme != o.Scheme || u.Host != o.Host || u.User != nil || u.Path != "/usdt.html" {
		return errors.New("invalid_claim_link")
	}
	for k, v := range u.Query() {
		if k != "lang" || len(v) != 1 || len(v[0]) != 2 {
			return errors.New("invalid_claim_link")
		}
	}
	frag := u.Fragment
	if !strings.HasPrefix(frag, "c=") {
		return errors.New("invalid_claim_link")
	}
	parts := strings.Split(strings.TrimPrefix(frag, "c="), ".")
	if len(parts) != 2 {
		return errors.New("invalid_claim_link")
	}
	id, err1 := b64u.DecodeString(parts[0])
	key, err2 := b64u.DecodeString(parts[1])
	if err1 != nil || err2 != nil || len(id) != 16 || len(key) != 32 || hex.EncodeToString(id) != cardID {
		return errors.New("invalid_claim_link")
	}
	owner, err := cl.AddressFromKey(key)
	clear(key)
	if err != nil || !strings.EqualFold(owner.Hex(), linkOwner) {
		return errors.New("invalid_claim_link")
	}
	return nil
}

// NewCardLink makes what the sender's browser makes with cashlink.js: a card id, a link key and the key's address. For tools and tests;
// real senders generate and retain it in their browser. The full link is also submitted for invitation delivery, without persistence.
func NewCardLink() (id, key []byte, owner string, err error) {
	id = make([]byte, 16)
	if _, err = rand.Read(id); err != nil {
		return nil, nil, "", err
	}
	priv, addr, err := cl.GenerateKey()
	if err != nil {
		return nil, nil, "", err
	}
	return id, priv, addr, nil
}

// ClaimFragment is the "#c=<card id>.<link key>" fragment of a claim link.
func ClaimFragment(id, key []byte) string {
	return "#c=" + b64u.EncodeToString(id) + "." + b64u.EncodeToString(key)
}
