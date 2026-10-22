package cards

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"regexp"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 160, 160))); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestProtectedInvitationParityAndNoBearerSecret(t *testing.T) {
	for _, lang := range []string{"fr", "en", "es", "de", "zh"} {
		c := ProtectedCashlinkCard{ClaimID: strings.Repeat("a", 48), ClaimURL: "https://wire.example.test/?claim=" + strings.Repeat("a", 48), Amount: "1", Network: "mainnet", Message: "<script>hello</script>"}
		subject, plain, h, err := Invitation("sender@example.test", c, lang, fixture(t))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(subject, "Phoenix Wire") || !strings.Contains(plain, c.ClaimURL) || !strings.Contains(h, "&lt;script&gt;") || !strings.Contains(h, "margin:18px auto 0") || !strings.Contains(h, `alt="Phoenix Wire"`) {
			t.Fatal("card parity")
		}
		visible := regexp.MustCompile(`data:image/png;base64,[^"]+`).ReplaceAllString(h, "")
		if strings.Contains(visible, "cashlink/#") || strings.Contains(visible, "cid:") || strings.Contains(visible, "OTP") || strings.Contains(visible, "Mainnet") || strings.Contains(visible, "<script>") {
			t.Fatal("unsafe email")
		}
		if strings.Index(h, "pm-footer-wrap") < strings.Index(h, c.ClaimURL) || strings.Count(h, "pm-footer-wrap") != 1 {
			t.Fatal("footer must follow card exactly once")
		}
		if !strings.Contains(h, PaymentCardColorsFor(c.ClaimID, true)[0]) {
			t.Fatal("wrong pastel family")
		}
	}
}
func TestArtworkRejectsNonRasterAndWrongDimensions(t *testing.T) {
	for _, data := range []string{"", base64.StdEncoding.EncodeToString([]byte(`<svg onload="bad()"/>`)), strings.Repeat("A", 90001)} {
		if _, err := CleanNimicon(data); err == nil {
			t.Fatal("invalid artwork accepted")
		}
	}
	if _, err := CleanNimicon(fixture(t)); err != nil {
		t.Fatal(err)
	}
}
