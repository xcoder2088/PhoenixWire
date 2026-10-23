# The payment window is PhoenixMail's

Wire's page (`internal/web/static/index.html`) opens **PhoenixMail's own payment window**, mirrored from PhoenixMail
`6d2b3b6` into `internal/web/static/pm/`: `nimiq-payments.js`, `usdt-payments.js`, `usdt-cashlink.js`, `nimiq-hub.js`,
`phoenix-nimiq.js`, `nimiq-iqons.js` and the styles `app.css`, `nimiq-payments.css`, `theme-light.css`.
Those files are the owner's own code (MIT in this repository), mirrored, not rewritten.

What is Wire's (and only that):

- `wire-pay.js`: who the sender is (e-mail code), the window's host options, and what happens after a card is made:
  the browser keeps the claim link, watches the card, and once it is funded asks Wire to e-mail the recipient
  (`/api/usdt/card/<id>/invite`, the claim link is checked against the card and never stored).
- `internal/wire/usdt_compat.go`: the routes the window calls, in PhoenixMail's shape (`/api/usdt/status`,
  `/api/usdt/cashlink/{create,status,cancel,refund,refund-claim,gas-status,gas}`), each answered by the gateway's
  own USDC / USDT card logic behind the sender's session. The QR of the payment link is drawn on the server.

Four small, marked changes to the mirrored integration (to re-apply after a new sync):

1. `pm/nimiq-payments.js` `cardShell`: the card's emblem and name come from an `o.brand` option (PhoenixMail's by default).
2. `pm/usdt-cashlink.js` and `pm/usdt-payments.js` `sw()`: the texts say `window.PhoenixBrandName` instead of "PhoenixMail".
3. `wire-pay.css` hides "Request payment". NIM is now visible: the host binds its currency button to Wire’s existing protected NIM flow, without modifying the copied files.
4. `pm/usdt-cashlink.js` and `pm/usdt-payments.js` merge the selected language's `window.PhoenixWordOverrides` over their default words. Wire sets only its host-specific copy in `wire-pay.js`. The funding panel in `pm/usdt-payments.js` formats `fundedMsg` with `{email}` from `entry.protectedRecipient || entry.to`. With no overrides, the original words and behavior are unchanged. Wire says it sends the card, uses Done, and does not describe an email composer.

NIM now shares this entry point and sender verification. `flow.js` exports its existing flow for the NIM template in
`index.html`; `classic.html` remains compatible and `/?claim=…` still opens it for NIM recipients. Sender free gas remains
unavailable (`/api/usdt/status` says `gas:false`).

## Wire navigation, identity and session copy

Wire adds an explicit return control to each payment-window instance from `wire-pay.js`. It invokes the copied window's own
close button so its listeners and observers are cleaned up. Asset switches recreate the window; a host observer enhances each
replacement. Beyond the marked hooks, the mirrored files remain unchanged. The page also links directly to the PhoenixWire presentation
(`https://phoenixmail.ca/#phoenixwire`) and PhoenixMail (`https://phoenixmail.ca/#phoenixmail`). Classic NIM has a link back to `/`.

The tab and mobile shortcut name is Phoenix Wire, preserving the latest owner-side update. Icons derive from the existing emblem; normal static routing serves them with
the existing CSP. Before sender verification and beside the signed-in address, five-language text explains the unchanged 24-hour
session. `tests/wire-pay.test.mjs` covers those labels and navigation with the copied window and fake API.

## Exact invitation boundary

The browser generates and retains the link key. It sends public owner data at card creation and proofs/signatures to claim and
refund routes. When inviting the recipient, it submits the **full claim URL** to Wire, which validates it and uses it in the mail
without persisting it. A retry may submit it again. The recipient browser does not submit its fragment to claim routes.
Do not extend that last property into a claim that Wire or its mail provider can never see the key. The transient invitation
handoff and the service co-signer are separate parts of the trust model; no runtime behavior changed in this clarification.

## NIM and theme follow-up

All four assets sit behind the same sender session. The host replaces the visible payment window when switching between
NIM and a stablecoin, just as the copied window replaces itself between stablecoins. NIM reuses `mountNimFlow` from `flow.js`
with a scoped root; it does not call PhoenixMail’s legacy unprotected NIM creation path. Its immutable intent, official Hub
checkout, artwork, history/recovery and mobile return handling are preserved. A saved Hub return reopens NIM automatically.

`wire-pay.css` no longer hides NIM. `wire-pay.js` binds that button to the embedded NIM flow and removes the separate Send NIM
exit link. Light mode maps to `light-paper`, the selector present in pm/; dark mode remains `dark`. Buttons have neutral fills
and theme-accent outlines. These are host changes, not additional modifications under pm/.

## NIM funded from Nimiq Pay

Inside the Nimiq Pay app Wire funds a NIM card with `window.nimiq.sendBasicTransactionWithData`, which takes the data as text and sends it as UTF-8. The
funding mark of a cashlink (`00 82 80 92 87`) therefore reaches the chain as `00 c2 82 c2 80 c2 92 c2 87`. The funding check
(`engine/secureclaim/internal/nimiq/mainnet_rpc.go`, `fundingMarker`) accepts exactly these two forms and nothing else: the amount, the address
and the two confirmations are checked as before, and a transfer with no data or any other data is still refused. Confirm on a real device after
any change to either side.

## My cards on Wire's home page

`wire-pay.js` renders USDC/USDT history from `wire-usdt-cards`, enriches amount/asset/recipient/invitation status using the existing
authenticated `/api/usdt/cards` list, and polls `/api/usdt/cashlink/status` even after the payment window closes or an invitation
has been sent. It resumes on foreground, marks failed reads as stale, and keeps the last known state. The host never displays
the claim fragment or refund key in this dashboard and never evicts a fragment to limit the history.

An invitation is labelled sent only after an `invited:true` response or the server's persisted invitation flag. A lost response
is reconciled with that flag before retrying; failed attempts are spaced by at least a minute. The sent banner persists on this
device until explicitly dismissed, independently of the copied window's automatic close. This confirms acceptance by the mail
service, not delivery to an inbox. History is hidden on logout and polling stops. Records belonging to another sender are not shown.

NIM rows use the existing `/api/payments` states and the NIM window's status callback. They do not infer that a NIM card was claimed:
that information is not provided by this list. No new server route, CSP rule or payment operation is added by the dashboard.
