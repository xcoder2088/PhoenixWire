# Local integration and review

## Startup and private state

Run `./run-local.sh` in one terminal. Wire serves its UI, gateway and integrated engine on `127.0.0.1:8790`.
Never run two processes over the same private state. The original PhoenixMail services are separate.

The optional environment file is `~/.config/phoenixwire/local.env`, owner-only 0600. Follow `.env.example` for MailerSend,
the exact public origin, trusted Nimiq RPC and `PHOENIXWIRE_ENABLE_PAYMENTS=1`. The USDC/USDT engine additionally needs
its selected chains/RPCs/relay configuration; see `engine/secureclaim/docs/USDT-CASHLINK.md` and `deploy/enable-usdt.sh`.
Read the enable script before running it; UI changes do not authorize funding, deployment or transactions.
Without integration, `/` reports payments unavailable; `/classic.html` still provides the local design preview.

The master key is generated once at `~/.config/phoenixwire/master.key` (0600). Private state is under
`~/.local/share/phoenixwire/state/`, in independent engine/gateway directories with separate derived keys.
Back up the original key and all state together, encrypted and owner-controlled. Never delete them while payments may hold funds.
Existing state blocks regeneration; wrong keys and concurrent writers fail closed. Do not import PhoenixMail/SecureClaim live state.

## Automated checks

```sh
npm ci --ignore-scripts
node --test tests/*.test.mjs engine/secureclaim/web/tests/*.cjs
go vet ./...
go test ./...
(cd engine/secureclaim && go vet ./... && go test ./...)
```

The engine is a nested Go module, so root `go test ./...` alone does not run its full suite. RPC/mail fixtures do not spend money
or prove a deployed service works. Record any unavailable tools and failed checks rather than treating them as passed.

## Current page: NIM, USDC and USDT

1. Open `/`. Confirm the PhoenixWire title, tab icon, mobile touch icon and links to both presentation sites.
2. Before signing in, check the 24-hour explanation below the email field in FR/EN/ES/DE/ZH.
3. Request and verify the sender email. Check the 24-hour confirmation next to the email. Reload within the same session:
   no new sender code should be needed. Explicit logout, expired sessions and other devices require confirmation again.
4. Open Payments: check NIM, USDC Polygon, USDT Polygon and USDT Ethereum according to enabled assets, with protection required.
   Switch to NIM without another sender-code request; validate its original Hub checkout and saved redirect/return.
   Toggle light/dark before opening and inside the window; check neutral button fills and accent outlines.
   Exit using the return button; reopen and switch coins, then verify the return button still works.
5. With local fixtures, create a card and check the locally verified Safe address, total including fees, funding QR and preview.
6. Funding confirmation triggers the invitation. Wire receives the complete link for this mail, validates it and does not persist it.
   Creation, recipient claim and sender refund requests carry public data/proofs/signatures rather than the private link key.
7. Recipient flow: prove link ownership, enter the email code, confirm an address, verify the exact quote/hash and follow payment
   confirmation. A sent transaction is not yet a confirmed payment.
8. Test sender status/refund using the same locally retained card data; verify the stated fee and recovery amount before signing.
   Check interruption/reload, mail failure, pending chain state and stale views without creating another funded card.

Any live check requires a separately approved small amount and wallet confirmation by the owner. None is implied by these instructions.
Keep local recovery data; the browser-held USDC/USDT link is not restored solely by logging in on a different device.

## Classic NIM regression checks

NIM is now available in the common Payments window. Also check `/classic.html` for compatibility and `/?claim=…` for its recipient path. Sender verification precedes an immutable payment intent.
Official Hub funds the prepared address, including mobile redirect/return. Once checkout starts, reload follows the same address
and does not automatically start a second checkout. After confirmed funding, the invitation uses the existing opaque claim URL.
Recipient code verification releases the original Hub-compatible Cashlink link. Sender recovery only works while funds are unspent.
NIM claim lifetime is 24 hours; USDC/USDT cards use the engine's 14-day lifetime. Sender session duration is a separate 24-hour limit.

## Limits and remote checks

A localhost link works only on the serving machine. Phone testing needs a correctly configured HTTPS origin routed to Wire.
Do not change DNS, tunnel configuration or deploy as part of UI work. Do not trust forwarded IP headers without reviewing the proxy.
The embedded durable engine currently requires Linux filesystem locks. NIM still trusts the configured RPC with its 50-transaction
scan window. EVM Safe funding/refunds and relay monitoring are implemented; real-chain outcomes and operational reserves must
be checked separately. See DEPLOYMENT.md for the owner-run hosting checklist.
