package wire

import (
	"os"
	"sort"
	"strings"
)

// Who may SEND with Phoenix Wire. The site is public and so are the cards: anyone can open a card they received, and recipients never sign in.
// Signing in as a SENDER (the e-mail code that opens the payment window) is by invitation: the administrator opens or closes it in the app
// (admin.go) and keeps a list of who may and who may not.
//
// May send, in this order:
//   - an administrator (PHOENIXWIRE_ADMINS), always;
//   - nobody on the administrator's blocked list;
//   - everyone, when the administrator opened access (or PHOENIXWIRE_SENDERS=open);
//   - the senders already registered (who created a card before), the addresses the administrator allowed, and the ones named in
//     PHOENIXWIRE_SENDERS.
//
// A visitor who may not is refused before any e-mail is sent.
type accessPolicy struct {
	Open    bool     `json:",omitempty"` // everyone may send (set in the app)
	Allowed []string `json:",omitempty"` // lower-case addresses the administrator allowed
	Blocked []string `json:",omitempty"` // lower-case addresses the administrator refused: they may not sign in even if they were registered
}

func listHas(list []string, address string) bool {
	for _, e := range list {
		if strings.TrimSpace(e) == address {
			return true
		}
	}
	return false
}

func envList(name string) string { return strings.ToLower(strings.TrimSpace(os.Getenv(name))) }

// isAdmin: an administrator is an address named in PHOENIXWIRE_ADMINS (comma-separated).
func (a *API) isAdmin(address string) bool {
	return listHas(strings.Split(envList("PHOENIXWIRE_ADMINS"), ","), strings.ToLower(address))
}

func (a *API) accessOpen() bool {
	return envList("PHOENIXWIRE_SENDERS") == "open" || a.db.state.Access.Open
}

func (a *API) senderAllowed(address string) bool {
	if a.isAdmin(address) {
		return true
	}
	if listHas(a.db.state.Access.Blocked, address) {
		return false
	}
	if a.accessOpen() || listHas(strings.Split(envList("PHOENIXWIRE_SENDERS"), ","), address) || listHas(a.db.state.Access.Allowed, address) {
		return true
	}
	return listHas(a.registeredSenders(), address)
}

// registeredSenders are the senders Wire already knows: of its NIM payments, of its USDC / USDT cards, and of verified sessions. Nothing new is
// stored for this: an existing sender keeps working and the limit needs no list to keep.
func (a *API) registeredSenders() []string {
	seen := map[string]bool{}
	for _, saved := range a.db.state.Payments {
		seen[strings.ToLower(saved.Payment.Sender)] = true
	}
	for _, c := range a.db.state.USDT {
		seen[strings.ToLower(c.Sender)] = true
	}
	for _, s := range a.db.state.Sessions {
		if s.Verified {
			seen[strings.ToLower(s.Email)] = true
		}
	}
	delete(seen, "")
	out := make([]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

// InviteOnly says whether sending is limited (what the public configuration tells the page, so a visitor is not asked for an e-mail that will
// be refused). It reads the state, so it takes the lock.
func (a *API) InviteOnly() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return !a.accessOpen()
}
