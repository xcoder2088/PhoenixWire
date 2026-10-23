package nimiq

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"testing"
)

func TestSignBasicLayoutAndSignature(t *testing.T) {
	seed := bytes.Repeat([]byte{7}, 32)
	other, _ := (AddressDeriver{}).CashlinkAddress(bytes.Repeat([]byte{9}, 32))
	raw, err := SignBasic(seed, other, 123456, 0, 777, NetworkMainAlbatross)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 139 || raw[0] != 0 || raw[1] != 0 {
		t.Fatal("wrong layout", len(raw))
	}
	pub := ed25519.PublicKey(raw[2:34])
	to, _ := AddressBytes(other)
	if !bytes.Equal(raw[34:54], to[:]) || binary.BigEndian.Uint64(raw[54:62]) != 123456 || binary.BigEndian.Uint64(raw[62:70]) != 0 ||
		binary.BigEndian.Uint32(raw[70:74]) != 777 || raw[74] != NetworkMainAlbatross {
		t.Fatal("fields are not where the node reads them")
	}
	sender := SenderOf(pub)
	if !ed25519.Verify(pub, content(sender, to, 123456, 0, 777, NetworkMainAlbatross), raw[75:]) {
		t.Fatal("signature does not cover the content")
	}
	if !bytes.HasSuffix(content(sender, to, 1, 0, 1, 24), []byte{24, 0, 0}) {
		t.Fatal("content must end with network id, flags and an empty sender data")
	}
}

func TestSignBasicRefusals(t *testing.T) {
	seed := bytes.Repeat([]byte{7}, 32)
	self, _ := (AddressDeriver{}).CashlinkAddress(seed)
	other, _ := (AddressDeriver{}).CashlinkAddress(bytes.Repeat([]byte{9}, 32))
	for name, call := range map[string]func() error{
		"to itself":  func() error { _, err := SignBasic(seed, self, 1, 0, 5, 24); return err },
		"to burn":    func() error { _, err := SignBasic(seed, BurnAddress, 1, 0, 5, 24); return err },
		"zero value": func() error { _, err := SignBasic(seed, other, 0, 0, 5, 24); return err },
		"no height":  func() error { _, err := SignBasic(seed, other, 1, 0, 0, 24); return err },
		"short seed": func() error { _, err := SignBasic(seed[:16], other, 1, 0, 5, 24); return err },
		"bad address": func() error {
			_, err := SignBasic(seed, "NQ00 0000 0000 0000 0000 0000 0000 0000 0000", 1, 0, 5, 24)
			return err
		},
	} {
		if call() == nil {
			t.Fatal("accepted:", name)
		}
	}
}
