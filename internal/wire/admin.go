package wire

import (
	"net/http"
	"strings"
)

// The administrator's panel: who may send with Phoenix Wire, changed in the app and not in a file. An administrator is a verified sender whose
// address is named in PHOENIXWIRE_ADMINS; nobody else (and no hand-over session) reaches these routes.
//
//	GET  /api/admin/access   {open, allowed, blocked, registered, admins}
//	POST /api/admin/access   {"open": true|false}  open or close sending to everyone
//	POST /api/admin/allow    {"email"}             let one address send (and lift a block on it)
//	POST /api/admin/block    {"email"}             refuse one address: its sessions end now, it cannot sign in again until it is allowed
//
// Closing never touches a card that was already sent: recipients open their cards without signing in.
const adminListMax = 500

func (a *API) accessView() map[string]any {
	var admins []string
	for _, e := range strings.Split(envList("PHOENIXWIRE_ADMINS"), ",") {
		if e = strings.TrimSpace(e); e != "" {
			admins = append(admins, e)
		}
	}
	nonNil := func(l []string) []string {
		if l == nil {
			return []string{}
		}
		return l
	}
	return map[string]any{"open": a.accessOpen(), "locked_open": envList("PHOENIXWIRE_SENDERS") == "open", // the configuration forces it open: closing in the panel changes nothing until it is lifted
		"allowed": nonNil(a.db.state.Access.Allowed), "blocked": nonNil(a.db.state.Access.Blocked), "registered": a.registeredSenders(), "admins": nonNil(admins)}
}

func removeFrom(list []string, address string) []string {
	out := list[:0:0]
	for _, e := range list {
		if e != address {
			out = append(out, e)
		}
	}
	return out
}

// adminServe is called under a.mu for a verified sender.
func (a *API) adminServe(w http.ResponseWriter, r *http.Request, s session) {
	if s.Scope != "" || !a.isAdmin(s.Email) {
		failure(w, 403, "admin_required")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/")
	if path == "access" && r.Method == "GET" {
		respond(w, 200, a.accessView())
		return
	}
	if r.Method != "POST" {
		failure(w, 404, "not_found")
		return
	}
	switch path {
	case "access":
		var in struct {
			Open *bool `json:"open"`
		}
		if decode(w, r, &in) != nil || in.Open == nil {
			failure(w, 400, "invalid_request")
			return
		}
		a.db.state.Access.Open = *in.Open
	case "allow", "block":
		var in struct {
			Email string `json:"email"`
		}
		if decode(w, r, &in) != nil {
			failure(w, 400, "invalid_email")
			return
		}
		address, err := email(in.Email)
		if err != nil {
			failure(w, 400, "invalid_email")
			return
		}
		acc := &a.db.state.Access
		if path == "allow" {
			if len(acc.Allowed) >= adminListMax {
				failure(w, 429, "list_full")
				return
			}
			acc.Blocked = removeFrom(acc.Blocked, address)
			if !listHas(acc.Allowed, address) {
				acc.Allowed = append(acc.Allowed, address)
			}
		} else {
			if a.isAdmin(address) {
				failure(w, 409, "cannot_block_admin") // an administrator cannot lock himself, or another one, out
				return
			}
			if len(acc.Blocked) >= adminListMax {
				failure(w, 429, "list_full")
				return
			}
			acc.Allowed = removeFrom(acc.Allowed, address)
			if !listHas(acc.Blocked, address) {
				acc.Blocked = append(acc.Blocked, address)
			}
			for key, sess := range a.db.state.Sessions { // the block takes effect now, not at the end of their session
				if strings.ToLower(sess.Email) == address {
					delete(a.db.state.Sessions, key)
				}
			}
		}
	default:
		failure(w, 404, "not_found")
		return
	}
	if a.db.save() != nil {
		failure(w, 503, "state_unavailable")
		return
	}
	respond(w, 200, a.accessView())
}
