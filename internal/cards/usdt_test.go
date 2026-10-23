package cards

import (
	"strings"
	"testing"
)

func usdtFixture(id string) USDTCard {
	return USDTCard{ID: id, Chain: "polygon", Symbol: "USDC", Amount: "2.5", Message: "test <b>", ClaimURL: "https://wire.example.test/usdt.html?lang=fr#c=AAAAAAAAAAAAAAAAAAAAAA.BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"}
}

func TestUSDTInvitationInFiveLanguages(t *testing.T) {
	c := usdtFixture("0a" + strings.Repeat("0", 30))
	for _, lang := range []string{"fr", "en", "es", "de", "zh", "xx"} {
		subject, plain, body, err := USDTInvitation("ada@example.test", c, lang)
		if err != nil {
			t.Fatal(lang, err)
		}
		for _, must := range []string{c.ClaimURL, "data:image/png;base64,", "USDC · Polygon", "2.5", "Phoenix Wire", "&lt;b&gt;"} {
			if !strings.Contains(body, must) {
				t.Fatalf("%s: missing %q", lang, must)
			}
		}
		if strings.Contains(body, "cid:") || strings.Contains(body, "USDT") || strings.Contains(body, "<b>") {
			t.Fatalf("%s: a mail attachment reference, a wrong symbol or unescaped text", lang)
		}
		if !strings.Contains(plain, c.ClaimURL) || subject == "" {
			t.Fatalf("%s: plain text or subject", lang)
		}
	}
}

func TestUSDTCardWearsTheColourOfItsID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 12; i++ {
		id := string([]byte{"0123456789abcdef"[i>>4], "0123456789abcdef"[i&15]}) + strings.Repeat("0", 30)
		body, err := USDTCardHTML(usdtFixture(id), "en")
		if err != nil {
			t.Fatal(err)
		}
		seen[PaymentCardColorsFor(id, true)[0]] = true
		if !strings.Contains(body, PaymentCardColorsFor(id, true)[0]) {
			t.Fatalf("card %s does not wear its colour", id)
		}
	}
	if len(seen) != 12 {
		t.Fatalf("twelve ids, %d colours", len(seen))
	}
}

func TestUSDTCardRefusesWhatItShouldNot(t *testing.T) {
	good := usdtFixture("0a" + strings.Repeat("0", 30))
	for name, mutate := range map[string]func(*USDTCard){
		"short id":     func(c *USDTCard) { c.ID = "0a" },
		"odd chain":    func(c *USDTCard) { c.Chain = "solana" },
		"odd symbol":   func(c *USDTCard) { c.Symbol = "DOGE" },
		"odd amount":   func(c *USDTCard) { c.Amount = "1e9" },
		"another path": func(c *USDTCard) { c.ClaimURL = strings.Replace(c.ClaimURL, "/usdt.html", "/x", 1) },
		"no fragment":  func(c *USDTCard) { c.ClaimURL = "https://wire.example.test/usdt.html" },
		"long message": func(c *USDTCard) { c.Message = strings.Repeat("x", 141) },
		"line break":   func(c *USDTCard) { c.Message = "a\nb" },
	} {
		c := good
		mutate(&c)
		if _, err := USDTCardHTML(c, "en"); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
