// Package nimiq contains the independent Nimiq Mainnet address derivation
// used by SecureClaim. This is a minimal, server-only protocol implementation.
package nimiq

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq/blake2b"
)

const alphabet = "0123456789ABCDEFGHJKLMNPQRSTUVXY"

type AddressDeriver struct{}

// CashlinkAddress derives the Nimiq address from an Ed25519 Cashlink seed.
// This must match the wallet-derived address BEFORE a secret is escrowed.
func (AddressDeriver) CashlinkAddress(seed []byte) (string, error) {
	if len(seed) != ed25519.SeedSize {
		return "", errors.New("invalid_cashlink_seed")
	}
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	sum := blake2b.Sum256(pub)
	bits := new(big.Int).SetBytes(sum[:20])
	mask := big.NewInt(31)
	b := make([]byte, 32)
	for i := 31; i >= 0; i-- {
		d := new(big.Int).And(bits, mask)
		b[i] = alphabet[d.Int64()]
		bits.Rsh(bits, 5)
	}
	body := string(b)
	digits := strings.Builder{}
	for _, c := range body + "NQ00" {
		if c >= '0' && c <= '9' {
			digits.WriteRune(c)
		} else if c >= 'A' && c <= 'Z' {
			digits.WriteString(strconv.Itoa(int(c) - 55))
		} else {
			return "", errors.New("invalid_nimiq_alphabet")
		}
	}
	n, ok := new(big.Int).SetString(digits.String(), 10)
	if !ok {
		return "", errors.New("invalid_nimiq_checksum")
	}
	check := 98 - int(new(big.Int).Mod(n, big.NewInt(97)).Int64())
	return fmt.Sprintf("NQ%02d%s", check, body), nil
}
