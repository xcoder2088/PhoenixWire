package evm

import (
	"encoding/hex"
	"errors"
	"strings"
)

var ErrInvalidAddress = errors.New("invalid_evm_address")

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
