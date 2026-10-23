package service

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"

	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
)

func TestPayLinkAndClaimLinkChecks(t *testing.T) {
	_, owner, err := cl.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if link, err := USDTPayLink("usdc_polygon", owner, 2_650_000); err != nil || !strings.HasPrefix(link, "ethereum:0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359@137/transfer?address=") || !strings.HasSuffix(link, "&uint256=2650000") {
		t.Fatalf("pay link: %q %v", link, err)
	}
	if _, err := USDTPayLink("usdc_polygon", strings.ToLower(owner), 1); err == nil {
		t.Fatal("an address that is not checksummed was accepted")
	}
	if _, err := USDTPayLink("doge", owner, 1); err == nil {
		t.Fatal("an unknown asset was accepted")
	}
	if _, err := USDTPayLink("usdc_polygon", owner, 0); err == nil {
		t.Fatal("a zero amount was accepted")
	}
	// a claim link: the card id, the key whose address is the card's link owner
	id, key := make([]byte, 16), make([]byte, 32)
	rand.Read(id)
	priv, linkOwner, err := cl.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	copy(key, priv)
	cardID := hex.EncodeToString(id)
	good := "https://wire.example.test/usdt.html?lang=fr#c=" + b64u.EncodeToString(id) + "." + b64u.EncodeToString(key)
	if err := CheckClaimLink(good, "https://wire.example.test", cardID, linkOwner); err != nil {
		t.Fatalf("the card's own link was refused: %v", err)
	}
	for name, bad := range map[string]string{
		"another origin":    strings.Replace(good, "wire.example.test", "evil.example", 1),
		"another path":      strings.Replace(good, "/usdt.html", "/x.html", 1),
		"another card":      strings.Replace(good, "#c="+b64u.EncodeToString(id), "#c="+b64u.EncodeToString(make([]byte, 16)), 1),
		"a foreign query":   strings.Replace(good, "?lang=fr", "?next=//evil", 1),
		"no key":            "https://wire.example.test/usdt.html#c=" + b64u.EncodeToString(id),
		"a key of nobody's": "https://wire.example.test/usdt.html#c=" + b64u.EncodeToString(id) + "." + b64u.EncodeToString(make([]byte, 32)),
	} {
		if CheckClaimLink(bad, "https://wire.example.test", cardID, linkOwner) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if CheckClaimLink(good, "https://wire.example.test", cardID, "0x"+strings.Repeat("11", 20)) == nil {
		t.Fatal("a link whose key is not the card's link owner was accepted")
	}
}
