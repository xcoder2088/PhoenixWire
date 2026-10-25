package wire

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// The administrator's panel: who may send with Phoenix Wire, changed in the app and not in a file. An administrator is a verified sender whose
// address is named in PHOENIXWIRE_ADMINS; nobody else (and no hand-over session) reaches these routes.
//
//	GET  /api/admin/access   {open, locked_open, allowed, blocked, registered_count, admins}
//	POST /api/admin/access   {"open": true|false}  open or close sending to everyone
//	POST /api/admin/allow    {"email"}             invite one address (and lift a block on it)
//	POST /api/admin/revoke   {"email"}             take an address off the invited list (a sender who has already sent cards stays able to sign in, to recover his money: block him to cut him off)
//	POST /api/admin/remove   {"email"}             forget an address altogether: off the invited list AND off the blocked list
//	POST /api/admin/block    {"email"}             refuse one address: its sessions end now, it cannot sign in again until it is allowed
//
//	POST /api/admin/maintenance {"on": true|false}  close the site to every sender but the administrators (a message tells the visitors), or open it again
//	GET  /api/admin/relayers   what the gas wallets of the USDC / USDT cards hold (the coin that pays the gas, the fees earned), what the payouts earned and cost, and the addresses a withdrawal may go to
//	POST /api/admin/relayers/withdraw {"chain","what","to"}  take out the surplus of one wallet (it always keeps the minimum the service needs), to an address of the operator's list
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
	// Only what the administrator entered by hand is listed. The senders who signed in on their own are counted, not listed: with hundreds of them the
	// list would be unusable, and one is found by typing his address to block him.
	return map[string]any{"open": a.accessOpen(), "locked_open": envList("PHOENIXWIRE_SENDERS") == "open", // the configuration forces it open: closing in the panel changes nothing until it is lifted
		"allowed": nonNil(a.db.state.Access.Allowed), "blocked": nonNil(a.db.state.Access.Blocked), "registered_count": len(a.registeredSenders()), "registered": []string{}, "admins": nonNil(admins), "maintenance": a.db.state.Access.Maintenance} // "registered" stays an empty list only so that a page that still reads it does not break
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
	if path == "relayers" && r.Method == "GET" {
		a.relayersView(w, r)
		return
	}
	if path == "relayers/withdraw" && r.Method == "POST" {
		a.relayerWithdraw(w, r)
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
	case "maintenance":
		var in struct {
			On *bool `json:"on"`
		}
		if decode(w, r, &in) != nil || in.On == nil {
			failure(w, 400, "invalid_request")
			return
		}
		a.db.state.Access.Maintenance = *in.On
		log.Printf("maintenance: %v", *in.On)
	case "allow", "block", "revoke", "remove":
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
		switch path {
		case "revoke": // off the invited list; a block stays what it is
			acc.Allowed = removeFrom(acc.Allowed, address)
			if !listHas(a.registeredSenders(), address) { // he never sent a card: his session ends now. Someone who has cards keeps his, to recover his money.
				for key, sess := range a.db.state.Sessions {
					if strings.ToLower(sess.Email) == address {
						delete(a.db.state.Sessions, key)
					}
				}
			}
		case "remove": // forgotten: neither invited nor blocked
			acc.Allowed = removeFrom(acc.Allowed, address)
			acc.Blocked = removeFrom(acc.Blocked, address)
		}
		if path == "revoke" || path == "remove" {
			break
		}
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

var (
	chainNameRE = regexp.MustCompile(`^[a-z]{3,20}$`)
	assetNameRE = regexp.MustCompile(`^(native|[a-z0-9_]{3,40})$`)
	evmAddrRE   = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
)

// relayersView is what the owner sees of the gas wallets and of what the cards earned. The wallets' own routes are the engine's (issuer key, in-process).
func (a *API) relayersView(w http.ResponseWriter, r *http.Request) {
	engine := a.USDT()
	if engine == nil {
		failure(w, 404, "usdt_unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()
	held, status, code := engineAnswer(engine.IssuerCall(ctx, "GET", "/api/v1/usdt/issuer/relayers", nil))
	if code != "" {
		failure(w, status, code)
		return
	}
	earned, status, code := engineAnswer(engine.IssuerCall(ctx, "GET", "/api/v1/usdt/issuer/ledger", nil))
	if code != "" {
		failure(w, status, code)
		return
	}
	var last []json.RawMessage
	_ = json.Unmarshal(earned["last"], &last)
	if len(last) > 10 {
		last = last[len(last)-10:]
	}
	respond(w, 200, map[string]any{"relayers": held["relayers"], "withdraw_to": held["withdraw_to"], "totals": earned["totals"], "last": last})
}

// relayerWithdraw takes the surplus out of one wallet. The engine decides what stays (the minimum float and the gas of the live cards) and refuses any
// address that is not on the operator's list (NIMIQ_CLAIM_USDTLINK_WITHDRAW_TO): a stolen session cannot send the money anywhere else.
func (a *API) relayerWithdraw(w http.ResponseWriter, r *http.Request) {
	engine := a.USDT()
	if engine == nil {
		failure(w, 404, "usdt_unavailable")
		return
	}
	var in struct {
		Chain string `json:"chain"`
		What  string `json:"what"`
		To    string `json:"to"`
	}
	if decode(w, r, &in) != nil || !chainNameRE.MatchString(in.Chain) || !assetNameRE.MatchString(in.What) || !evmAddrRE.MatchString(in.To) {
		failure(w, 400, "invalid_request")
		return
	}
	body, _ := json.Marshal(map[string]string{"what": in.What, "to": in.To})
	ctx, cancel := context.WithTimeout(r.Context(), 55*time.Second)
	defer cancel()
	out, status, code := engineAnswer(engine.IssuerCall(ctx, "POST", "/api/v1/usdt/issuer/relayers/"+in.Chain+"/withdraw", body))
	if code != "" {
		failure(w, status, code)
		return
	}
	log.Printf("administrator withdrawal: %s %s", in.Chain, in.What)
	respond(w, 200, out)
}
