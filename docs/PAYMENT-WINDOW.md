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

## Nimiq Pay per-card handoff (client)

The wallet chooser creates nothing on opening. Only clicking “I already have Nimiq Pay” prepares the protected NIM card
using the verified sender's existing `POST /api/payments` idempotent request (or reuses its ready card). No funding is started.
Then `POST /api/handoff {"payment_id": "..."}` returns a 43-character opaque token, single-use, valid for five minutes
(`expires_in: 300`). No draft fields are sent to the handoff route.

The mini-app address is only `https://wire.phoenixmail.ca/#h=<token>`, wrapped in
`nimiqpay://miniapp?url=<encoded URL>`. No email, amount, message, payment ID, recovery key or cookie is put in the link.
The host removes the fragment synchronously before API requests, posts the credential once to redeem, and never puts it in
local/session storage or the console. Retired `#wire-nim=` fragments and their saved personal-data drafts are discarded.
Foreground events and reload never retry a consumed token. Failed/stalled requests show translated feedback, not a blank page.

After redeem the host trusts the actual `/api/session` response, not a client-side verified flag. A scoped session
(`verified:false, handoff:{payment_id}`) sees only the prepared card's read-only summary, payment button, tracking and
sign-out. No creation form, full My cards dashboard, stablecoin API lists or recovery button is shown or requested.
Reload reconstructs this same view from the scope and `GET /api/payments`; storage cannot switch the payment being shown.
Closing the window returns to a limited card panel, not a signed-in sender dashboard.

A same-sender `already_signed_in:true` response keeps the existing full sender session and opens this prepared card's summary.
A different sender's `409 session_conflict` never logs them out or replaces them automatically: the page asks them to sign out
explicitly and reopen from Safari. A `409 payment_not_fundable` or `410 claim_expired` displays the appropriate warning and
blocks another funding action. Since those errors do not return the payment, no exact private on-chain state is invented.
A `404 invalid_or_expired` asks the user to reopen the link from Safari; normal email verification remains available.

Restoring a prepared card never creates a second card and never opens a wallet automatically. Explicit payment first refreshes
the payment and calls confirm. For a still-ready card the current server returns `funding_not_started` (it does not read the
chain for a ready card); only that result permits a fresh start. The service's atomic start rejects a concurrent start from
Safari or the app. A rejected/racing start refreshes and tracks the same card, never requests a second wallet payment.
The native provider's exact funding text/UTF-8 mark, canonical escrow and luna amount are unchanged. Native refusal does not
fall back automatically; an uncertain broadcast is checked before offering Hub.

The browser pre-opens a blank target on the explicit tap to retain the iOS gesture across the requests and removes its opener
before navigating to the opaque URL. If blocked, a direct retry link is offered. Closing the chooser prevents late navigation;
creation failures keep Hub and store choices available. Five-language tests simulate the route contract and isolated contexts,
not a physical iPhone/Android mini-app launch or real funds. Those device checks remain necessary after owner deployment.

Wire's host also removes the copied title's decorative arrow, styles the close control like PhoenixMail's lock (44 px square,
9 px corners, theme-accent border and 20 px outlined icon), and supplies the
existing `PhoenixViewport.center` hook for keyboard/viewport resizing. CSS keeps narrow forms within the viewport and
uses at least 16 px text in editable controls to avoid Safari focus zoom, without disabling manual pinch zoom. No `pm/`
file, Go code or CSP is changed for these adaptations.

## Focused checkout and sender session feedback

The host initially shows a currency picker inside the copied window. Selecting a currency opens that checkout and hides
the asset controls; the embedded NIM flow also keeps its asset row hidden. To choose another currency, close the window
and open Payments again. The host hides the copied Form/Card preview tabs. The module still changes its view when a card
is created, so the funding address, QR code and payment status remain available without a preview-tab action.

The sender-code button displays sending while its request is pending, then a translated sent label with a green outline
only after the server acknowledges the request. Feedback is beside the button as well as in the page status. The button
blocks duplicate requests and permits resending after 30 seconds; this is UI feedback, not a replacement for server quotas.
Failed or timed-out requests remain retryable and never show a successful-send state. Editing the email clears old feedback.

The home page has a PhoenixMail-style lock button: a 44 px square with 9 px rounded corners and a 20 px outlined padlock.
Its outline and icon follow the theme accent. Sign out is provided as an accessible label and tooltip in all five languages.
It uses the existing logout route, closes
the payment window, stops tracking and hides private history after server success. A failed logout remains visibly signed
in. Card recovery data stays in local storage; signing out never cancels a card or discards its recovery keys.

## Live themes

The header restores PhoenixFire, Aurora and Ocean from the original Wire palettes, each in light and dark. The selection
is saved in `wire-palette`, shared with the classic page for fixed palettes, and old `wire-accent` preferences are respected.
Without a saved colour preference, the new page uses Live shuffle: it starts with PhoenixFire, then selects a different
palette at random every 20 seconds. A fixed palette remains available at any time. Light/dark is an independent choice.

Live updates change CSS palette variables only. They never recreate the window, replace inputs, move focus, initiate a
payment, or change a payment card's own pastel colours. Background, border and foreground colour transitions last two
seconds. Transitions are disabled for reduced-motion preferences. The timer pauses in a hidden tab and resumes with a
fresh interval on return; missed intervals are not replayed. The selector and live-mode label are translated in five languages.

The light variants now use visibly tinted backgrounds and stronger field borders, accents and secondary text. Dark colours
retain the original palettes. Official bundled currency logos appear in the currency picker, selected checkout heading and
history rows, at 24 px, with text labels retained and decorative icons hidden from screen readers.

## Card history and dates

Payment statuses belong to My cards and the existing delivery banners, not the sender session panel. History refreshes no
longer copy the latest card state into `wire-status`; that area remains available for sender verification and service errors.

History shows server-provided `created` dates. It is also ready to show optional `sent_at` and `claimed_at` fields, formatted
in the selected language and the browser's local time zone. USDC/USDT dates are accepted from `/api/usdt/cards` and its
cashlink status response; NIM dates come from `/api/payments`. A NIM `claimed_at` must mean an actual confirmed claim,
not release of the claim link. Missing, invalid and zero dates are omitted. Polling time and local creation estimates are
never substituted for these dates. The current Go responses do not yet expose sent/claimed dates: see
`HANDOFF-DATES-CARTES-2026-10-23.md` for the server work left to implement.

## Hand-over of one payment to the Nimiq Pay app (server)

The app is another browser (its own cookies). What crosses over is not the sender's session but a **right to fund and follow ONE payment**
(`internal/wire/handoff.go`):

1. In Safari the verified sender prepares the NIM payment as usual (`POST /api/payments`: amount, recipient and funding address are fixed by the
   server) and asks `POST /api/handoff {"payment_id"}`. Only a payment of that sender that is prepared and not started (`stage: ready`, within its
   day) can be handed over. Answer: `{token, expires_in: 300}`.
2. The token: 43 random characters, only its hash kept (memory, never logged), **one use**, five minutes, tied to that payment and to the Safari session
   it came from (signing out there cancels it). A lost answer loses the right: the sender asks for another.
3. In the app, `POST /api/handoff/redeem {"token"}` (no session) answers `{payment, scoped: true, expires_at}` and sets a **scoped** session cookie.
   It is not a signed-in sender (`/api/session` says `verified:false` and `handoff:{payment_id}`). It can only: `GET /api/payments` (that payment),
   `POST /api/payments/<id>/{art,start,confirm,resend}` for that id, and sign out. Everything else answers `403 handoff_scope`: no other payment,
   no creation, no recovery key, no USDC / USDT card, no hand-over of its own. It lasts two hours at most, never beyond the payment's own day or the
   Safari session. It survives a restart of the service.
4. A browser already signed in as someone else is never replaced (`409 session_conflict`, the right is not used up); the same sender already
   signed in is left as it is (`200 {already_signed_in: true}`, no cookie change). A payment whose funding has started meanwhile: `409 payment_not_fundable`.
   Every other failure of the token: the same `404 invalid_or_expired`.
5. No double payment: `start` can be called once for a payment from either browser (`409 funding_already_started` after); the client checks
   `confirm` before any new request to the wallet.
