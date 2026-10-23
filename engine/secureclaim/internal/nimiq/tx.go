package nimiq

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"math/big"
	"strconv"
	"strings"

	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq/blake2b"
)

// A basic Nimiq transaction, signed by the key of a Cashlink, paying all of what the Cashlink holds to one address.
//
// The format is the node's "basic" one (core-rs-albatross, transaction serialization, format byte 0): the sender is not written, it is the
// address of the public key that signed; no data, no flags. What is signed is the transaction's content:
//
//	data length u16 (0) | sender 20 | sender type u8 (0) | recipient 20 | recipient type u8 (0) | value u64 | fee u64 |
//	validity start height u32 | network id u8 | flags u8 (0) | sender data length (0)
//
// all numbers big-endian. The wire form is:
//
//	format u8 (0) | signature type u8 (0: Ed25519, no flags) | public key 32 | recipient 20 | value u64 | fee u64 |
//	validity start height u32 | network id u8 | signature 64
//
// (139 bytes). Nothing here talks to the network: building a transaction moves nothing.

// NetworkMainAlbatross is the network id of Nimiq's main network.
const NetworkMainAlbatross = 24

var errBadAddress = errors.New("invalid_nimiq_address")

// BurnAddress is where every coin sent is destroyed: never a place to receive money.
const BurnAddress = "NQ07 0000 0000 0000 0000 0000 0000 0000 0000"

// ValidAddress checks the format of a friendly address (36 characters once the spaces are removed, the Nimiq alphabet) and its IBAN-style
// checksum: moved four characters to the end and read as a number (A=10…), it is 1 modulo 97.
func ValidAddress(address string) bool {
	s := strings.ToUpper(strings.ReplaceAll(address, " ", ""))
	if len(s) != 36 || !strings.HasPrefix(s, "NQ") {
		return false
	}
	for _, c := range s[4:] {
		if !strings.ContainsRune(alphabet, c) {
			return false
		}
	}
	digits := strings.Builder{}
	for _, c := range s[4:] + s[:4] {
		if c >= '0' && c <= '9' {
			digits.WriteRune(c)
		} else if c >= 'A' && c <= 'Z' {
			digits.WriteString(strconv.Itoa(int(c) - 55))
		} else {
			return false
		}
	}
	n, ok := new(big.Int).SetString(digits.String(), 10)
	return ok && new(big.Int).Mod(n, big.NewInt(97)).Int64() == 1
}

// AddressBytes returns the 20 bytes of a friendly Nimiq address ("NQxx XXXX …"): 32 base-32 characters after the check digits, with the
// IBAN-style checksum verified.
func AddressBytes(address string) ([20]byte, error) {
	var out [20]byte
	s := strings.ToUpper(strings.ReplaceAll(address, " ", ""))
	if len(s) != 36 || !strings.HasPrefix(s, "NQ") || !ValidAddress(address) {
		return out, errBadAddress
	}
	var bits uint32
	var have uint
	n := 0
	for _, c := range s[4:] {
		i := strings.IndexRune(alphabet, c)
		if i < 0 {
			return out, errBadAddress
		}
		bits = bits<<5 | uint32(i)
		have += 5
		for have >= 8 {
			if n >= 20 {
				return out, errBadAddress
			}
			out[n] = byte(bits >> (have - 8))
			n++
			have -= 8
			bits &= 1<<have - 1
		}
	}
	if n != 20 {
		return out, errBadAddress
	}
	return out, nil
}

// SenderOf is the address bytes of a public key: the first 20 bytes of its Blake2b-256 hash.
func SenderOf(pub ed25519.PublicKey) [20]byte {
	sum := blake2b.Sum256(pub)
	var out [20]byte
	copy(out[:], sum[:20])
	return out
}

func content(sender, recipient [20]byte, value, fee uint64, validity uint32, network byte) []byte {
	b := make([]byte, 0, 2+20+1+20+1+8+8+4+1+1+1)
	b = append(b, 0, 0)         // data length (u16): none
	b = append(b, sender[:]...) // sender
	b = append(b, 0)            // sender type: basic account
	b = append(b, recipient[:]...)
	b = append(b, 0) // recipient type: basic account
	b = binary.BigEndian.AppendUint64(b, value)
	b = binary.BigEndian.AppendUint64(b, fee)
	b = binary.BigEndian.AppendUint32(b, validity)
	b = append(b, network)
	b = append(b, 0) // flags: none
	b = append(b, 0) // sender data: empty (length, postcard)
	return b
}

// SignBasic builds and signs the transaction that pays `value` luna (fee `fee`) from the account of the Cashlink seed to `to`, valid from
// block `validity` on `network`. The recipient may not be the sender, nor the burn address, and a value of zero is refused.
func SignBasic(seed []byte, to string, value, fee uint64, validity uint32, network byte) ([]byte, error) {
	if len(seed) != ed25519.SeedSize || value == 0 || validity == 0 {
		return nil, errors.New("invalid_transaction")
	}
	recipient, err := AddressBytes(to)
	if err != nil {
		return nil, err
	}
	burn, _ := AddressBytes(BurnAddress)
	key := ed25519.NewKeyFromSeed(seed)
	defer clear(key)
	pub := key.Public().(ed25519.PublicKey)
	sender := SenderOf(pub)
	if recipient == sender || recipient == burn {
		return nil, errBadAddress
	}
	signature := ed25519.Sign(key, content(sender, recipient, value, fee, validity, network))
	raw := make([]byte, 0, 139)
	raw = append(raw, 0) // format: basic
	raw = append(raw, 0) // signature type: Ed25519, no flags
	raw = append(raw, pub...)
	raw = append(raw, recipient[:]...)
	raw = binary.BigEndian.AppendUint64(raw, value)
	raw = binary.BigEndian.AppendUint64(raw, fee)
	raw = binary.BigEndian.AppendUint32(raw, validity)
	raw = append(raw, network)
	raw = append(raw, signature...)
	return raw, nil
}
