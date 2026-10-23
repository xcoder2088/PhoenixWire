# PhoenixWire architecture

One Go process runs `internal/web`, the `internal/wire` gateway and the imported `engine/secureclaim/service` package.
The imported engine keeps its own module path. Wire owns separate keys and private state; it does not use another service's vault.

## Pages and synchronization boundaries

| Route / files | Role |
|---|---|
| `/`, `index.html`, `wire-pay.js`, `wire-pay.css` | Current sender host: one sender verification for NIM/USDC/USDT, window, monitoring and invitations |
| `/pm/*` | PhoenixMail payment-window mirror; see PAYMENT-WINDOW.md before changing it |
| `/classic.html` | Compatibility sender interface and earlier design preview |
| `/?claim=…` | Earlier interface in NIM recipient mode |
| `/usdt.html#c=…` | Engine's USDC/USDT recipient page |

`internal/wire/usdt_compat.go` adapts the PhoenixMail-shaped `/api/usdt/cashlink/*` calls to Wire's gateway.
The gateway reaches issuer operations in-process. Public `/api/v1/usdt/issuer*` routes are refused.
Normal app pages retain self-hosted scripts/styles under the CSP in `internal/web/server.go`; the embedded recipient page has
its separately generated script hashes and camera permission in `internal/web/claim.go`.

## Sender identity

A code confirms the sender without registration or a password. Durable per-email and per-IP limits protect requests and guesses.
Verification rotates the HttpOnly session cookie. Confirmed sessions last 24 hours; pending sessions last 15 minutes.
The UI explains the 24-hour period before verification and beside the signed-in email. This duration is unchanged.
Mutations require the configured same Origin. Gateway snapshots use AES-GCM, atomic publication and fsync.

## USDC / USDT flow

1. The browser generates a random card ID and private link key, then submits the public link-owner address, recipient and amount.
2. The engine derives its co-signer owner and computes a two-owner Safe. The browser independently checks the address before funding.
3. The sender funds the card's address with amount plus fees using the wallet QR/link. The gateway generates the funding QR.
4. `wire-pay.js` watches the card. Once funded, it submits the complete claim URL for invitation delivery.
5. Wire checks the origin, card and key-derived public owner, then sends the invitation through its email provider without persisting the link.
6. The recipient page proves possession of the link key, verifies the email code, confirms a destination and checks the exact payment hash.
7. The service co-signs the authorized payment; the relay pays gas. Confirmed chain state determines the outcome.
8. Sender refunds use the gateway's ownership check, a service co-signature and a local browser signature over the prepared refund.

The browser keeps card/recovery data locally, including `wire-usdt-cards`; the copied payment window also maintains its own
sender history. Browser signing does not send the private key to claim/refund routes. Invitation delivery transiently exposes
the full link to Wire and the mail provider, so “the link key never leaves the browser” is not an accurate end-to-end guarantee.
One logical invitation is sent on funding; uncertain responses/retries can resubmit its link. Nothing in this revision changes that behavior.
All cards in this flow are protected. Assets are USDC Polygon, USDT Polygon and USDT Ethereum, according to enabled configuration.
Sender free-gas support remains unavailable in the Wire host (`gas:false`).

## NIM flow shared with the current window

`flow.js` exports `mountNimFlow(root, {embedded:true})`. Wire mounts that same existing protected flow in its NIM template
from `index.html`, selected alongside the three stablecoin options. `classic.html` still mounts it in its original context;
`/?claim=…` retains the recipient path. The host uses the same 24-hour sender cookie for every currency. No server security
route or funding algorithm is replaced.

The gateway persists an immutable sender/recipient/amount intent, then creates and seals a NIM Cashlink using the existing engine.
Official Hub checkout funds that address. A started checkout is persisted before opening the wallet and is not automatically restarted.
Funding confirmation precedes invitation delivery. Recipient codes gate release of the original Hub-compatible Cashlink link.
NIM uses an opaque invitation URL and encrypted server-side seed storage; it differs from the browser-held USDC/USDT link model.
Re-verifying the sender restores NIM owner history; recovery still depends on unspent funds.

## Presentation and operational boundaries

Email cards are rendered by `internal/cards`, using PhoenixMail presentation mirrors and Wire adapters. NIM artwork uses the
official Iqons renderer; validated PNG art is decorative and never establishes funding or ownership. Current sender cards use
the copied window's `o.brand` option. Historic design notes do not override the implemented window.

Unconfigured startup does not enable payments. Follow LOCAL-TESTING.md and DEPLOYMENT.md for configuration and owner-run checks.
Historical review documents under docs/ describe their pinned commits, not a current deployment certification.

## Theme and host controls

The saved preference remains `dark`/`light`, shared with the classic page. The current host maps `light` to PhoenixMail’s
actual `light-paper` selector; this fixes the earlier toggle that only set an unused `light` value. `wire-pay.css` applies
neutral button fills and accent outlines, including selected currency controls and the protection tile. Mirrored pm/ styles
and code remain unchanged.
