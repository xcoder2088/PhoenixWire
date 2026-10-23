package evm

import (
	"encoding/hex"
	"testing"
)

func TestKeccakVectors(t *testing.T) {
	for in, want := range map[string]string{
		"":    "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
		"abc": "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45",
	} {
		if got := hex.EncodeToString(Keccak256([]byte(in))); got != want {
			t.Fatalf("keccak(%q)=%s", in, got)
		}
	}
	long := make([]byte, 1000)
	if hex.EncodeToString(Keccak256(long[:136])) == hex.EncodeToString(Keccak256(long[:137])) {
		t.Fatal("block boundary")
	}
}

func TestAddressVectors(t *testing.T) {
	// EIP-55 reference vectors.
	for _, a := range []string{
		"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed", "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
		"0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB", "0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb",
	} {
		if !ValidAddress(a) || Checksum(a) != a {
			t.Fatalf("EIP-55 vector rejected: %s", a)
		}
	}
	bad := "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAeD" // one letter's case flipped
	if ValidAddress(bad) {
		t.Fatal("a wrong checksum must be refused")
	}
	for _, a := range []string{"", "0x", "5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed", "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAe", "0x0000000000000000000000000000000000000000", "0xZZAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"} {
		if ValidAddress(a) {
			t.Fatalf("accepted %q", a)
		}
	}
	if !ValidAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed") {
		t.Fatal("all-lowercase must be accepted")
	}
	n, err := NormalizeAddress("  0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed ")
	if err != nil || n != "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed" {
		t.Fatal(n, err)
	}
}

func TestParseAmount(t *testing.T) {
	good := map[string]uint64{"1": 1_000_000, "0.01": 10_000, "12.5": 12_500_000, "1000": 1_000_000_000, "0.123456": 123_456, "7.000001": 7_000_001}
	for in, want := range good {
		units, canon, err := ParseAmount(in)
		if err != nil || units != want {
			t.Fatalf("%s -> %d %v", in, units, err)
		}
		if in == "12.5" && canon != "12.5" {
			t.Fatal(canon)
		}
	}
	if _, canon, _ := ParseAmount("3.500000"); canon != "3.5" {
		t.Fatal("canonical", canon)
	}
	for _, in := range []string{"", "0", "0.009", "1000.01", "1.1234567", "-1", "1e3", "01", ".5", "5.", "1,5", "abc", " ", "99999999"} {
		if _, _, err := ParseAmount(in); err == nil {
			t.Fatalf("accepted %q", in)
		}
	}
}

func TestLookup(t *testing.T) {
	for _, a := range Assets() {
		n, err := Lookup(a)
		if err != nil || !ValidAddress(n.Token) || Checksum(n.Token) != n.Token || len(n.RPCs) == 0 || n.Decimals != 6 {
			t.Fatalf("network %s", a)
		}
	}
	if _, err := Lookup("dai_polygon"); err == nil {
		t.Fatal("unknown asset accepted")
	}
	t.Setenv("NIMIQ_CLAIM_EVM_RPC_POLYGON", "http://evil.test, https://good.test")
	n, _ := Lookup(AssetUSDTPolygon)
	if len(n.RPCs) != 1 || n.RPCs[0] != "https://good.test" {
		t.Fatalf("RPC override must keep https only: %v", n.RPCs)
	}
}
