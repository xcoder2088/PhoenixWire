package usdtlink

import "strconv"

// palettes are the twelve pastel families of the PhoenixMail cards (background, accent), the same table as the Nimiq cards of
// PhoenixMail (internal/nimiq/payment_request.go) and as web/usdt.html. The colour of a card comes from its ID alone, so the
// sender's card, the card in the e-mail, the e-mail with the code and the claim page all wear the same one.
var palettes = [12][2]string{
	{"#deefff", "#9fc9eb"}, // sky
	{"#ece3ff", "#c7b1ee"}, // lavender
	{"#dcf5e9", "#9bd8bc"}, // mint
	{"#ffe2ee", "#efaac7"}, // blush
	{"#ffe7d8", "#eeb999"}, // peach
	{"#fff4cc", "#e8d181"}, // butter
	{"#e6ebff", "#b0bdf4"}, // periwinkle
	{"#dcf6fa", "#90d7e5"}, // aqua
	{"#f4e2fa", "#d9afea"}, // orchid
	{"#ffe2dc", "#edaea1"}, // coral
	{"#ecf6d8", "#bed89a"}, // pistachio
	{"#e2f4ef", "#9ed5c6"}, // seafoam
}

// Palette is the (background, accent) of a card: the first byte of its ID modulo twelve.
func Palette(cardID string) [2]string {
	if len(cardID) < 2 {
		return palettes[0]
	}
	n, _ := strconv.ParseUint(cardID[:2], 16, 8)
	return palettes[int(n)%len(palettes)]
}
