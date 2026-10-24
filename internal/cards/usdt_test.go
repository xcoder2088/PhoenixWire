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
		for _, must := range []string{c.ClaimURL, "data:image/png;base64,", "USDC · Polygon", "2.5", "Phoenix Wire", "&lt;b&gt;", `<meta name="color-scheme" content="light only">`} { // the card keeps its colours in a mail app's dark mode
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

func TestNoticesSayTheEndOfACardInFiveLanguages(t *testing.T) {
	for _, lang := range []string{"fr", "en", "es", "de", "zh"} {
		for _, kind := range []string{"reminder", "refunded", "expired"} {
			for _, auto := range []bool{true, false} {
				subject, plain, body, err := NoticeEmail(Notice{Kind: kind, Amount: "2.5", Symbol: "USDC", Recipient: "friend@example.test", URL: "https://wire.example.test/", Auto: auto}, lang)
				if err != nil || subject == "" || !strings.Contains(plain, "friend@example.test") || !strings.Contains(plain, "https://wire.example.test/") {
					t.Fatal(lang, kind, err, subject)
				}
				for _, bad := range []string{"Mainnet", "OTP", "Hub", "%!"} {
					if strings.Contains(subject+plain+body, bad) {
						t.Fatalf("%s %s says %q", lang, kind, bad)
					}
				}
			}
		}
	}
	// the recipient's address is shown, never run as HTML
	_, _, body, err := NoticeEmail(Notice{Kind: "expired", Amount: "1", Symbol: "NIM", Recipient: "a<script>@example.test", URL: "https://wire.example.test/"}, "en")
	if err != nil || strings.Contains(body, "<script>") {
		t.Fatal("an address was not escaped", err)
	}
	if _, _, _, err := NoticeEmail(Notice{Kind: "other", Amount: "1", Symbol: "NIM", Recipient: "a@example.test", URL: "https://wire.example.test/"}, "en"); err == nil {
		t.Fatal("an unknown kind was accepted")
	}
	if _, _, _, err := NoticeEmail(Notice{Kind: "expired", Amount: "1", Symbol: "NIM", Recipient: "a@example.test", URL: "javascript:alert(1)"}, "en"); err == nil {
		t.Fatal("a link that is not a web address was accepted")
	}
}
