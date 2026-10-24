package wire

import (
	"os"
	"strings"
)

// Who may SEND with Phoenix Wire. The site is public and so are the cards: anyone can open a card they received, and recipients never sign in.
// Signing in as a SENDER (the e-mail code that opens the payment window) is limited to the senders who are already registered here: anyone who
// has created a card before, plus the addresses named in PHOENIXWIRE_SENDERS. A visitor who is not on that list is refused before any e-mail is
// sent. PHOENIXWIRE_SENDERS=open lifts the limit (everyone may send).
//
// The registered senders are read from what Wire already keeps: the senders of its NIM payments and of its USDC / USDT cards, and the sessions that
// were verified. Nothing new is stored, so an existing sender keeps working and the limit needs no list to maintain.
func (a *API) senderAllowed(address string) bool {
	extra := strings.ToLower(strings.TrimSpace(os.Getenv("PHOENIXWIRE_SENDERS")))
	if extra == "open" {
		return true
	}
	for _, e := range strings.Split(extra, ",") {
		if strings.TrimSpace(e) == address {
			return true
		}
	}
	for _, saved := range a.db.state.Payments {
		if strings.ToLower(saved.Payment.Sender) == address {
			return true
		}
	}
	for _, c := range a.db.state.USDT {
		if strings.ToLower(c.Sender) == address {
			return true
		}
	}
	for _, s := range a.db.state.Sessions {
		if s.Verified && strings.ToLower(s.Email) == address {
			return true
		}
	}
	return false
}

// InviteOnly says whether sending is limited to the senders already registered (what the public configuration tells the page, so a visitor is
// not asked for an e-mail that will be refused).
func (a *API) InviteOnly() bool {
	return strings.ToLower(strings.TrimSpace(os.Getenv("PHOENIXWIRE_SENDERS"))) != "open"
}
