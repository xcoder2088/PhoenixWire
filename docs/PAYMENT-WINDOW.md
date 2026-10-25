# The payment window is PhoenixMail's

## Legal-copy clarification and public contact (cache 0.4.40)

The owner-confirmed public contact is `phoenixmails@proton.me`, with the same
address in the visible label and mailto target, in terms and privacy for all five
languages and in the static English fallbacks. Privacy now directs requests to
that contact instead of saying the address is still missing. No response deadline
is added. The operator, law, minimum age and prohibited-use decisions are retained.

The fee explanation clarifies that the existing network-cost margin helps absorb
network-cost changes. Refunds retain the complete relay fee: estimated network
cost with its margin, plus the existing 6% service fee. The text names network
nodes for network costs, Chainlink for prices and CoinGecko as the fallback. No
amount, payment, refund or provider-selection logic is changed. LEGAL-FACTS.md
records the owner's contact and margin clarification.

All references to the changed legal.js module use 0.4.40. No new static file,
Go change, mirrored-file change, push or deployment is included.

Validation executed: `npm ci --ignore-scripts`, then
`node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs`.
282 tests executed and passed, none failed or skipped.

## Relay and maintenance presentation review (cache 0.4.39)

The server implementation from 8555bb2 is retained. The relay section still
loads only when its native details control opens, and uses the server-provided
destinations and withdrawal decisions. No new static file or Go change is needed.

Gas balance, reserve, withdrawable balance, pending cards and token fees now use
label/value groups. Earnings have separate payout, fee, cost and margin values
for each asset. The mobile layout uses one column; wider layouts use two. Long
labels and transaction identifiers wrap, figures use tabular numerals, controls
are at least 40px high, and focus outlines follow the active theme accent.
Each destination selector has an explicit label, even with one allowed address.
Positive amounts below display precision show a less-than value, not zero.
Formatting still uses integer arithmetic and never changes the withdrawal body.
Statuses are announced politely; rerendering a focused control preserves a
keyboard focus target. Explorer links use only the supported explorer prefixes.

The maintenance switch has a larger labeled hit area. The maintenance banner
stays in document flow and uses the theme's panel and accent colours. While
maintenance is active it exposes the administrator sign-in form even when
invitation-only mode is also active. Administrator authorization is still decided
by the server. Already authenticated administrators retain the existing panel
state rather than a redundant visitor banner.

Classic maintenance support is worthwhile because classic.html remains a sender
and recipient entry point. Its actual authentication handlers are in flow.js;
app.js handles presentation. The classic sender page now reads config.maintenance
and translates maintenance errors, leaving its email editable and its code box
closed after a rejected request. Recipient access remains unchanged. Embedded
NIM windows notify their parent page of maintenance without changing pm/.

Mock Audio tests cover a newly sent NIM card and each stablecoin, one play only,
autoplay rejection, cards at least 30 minutes old, future dates and cards first
discovered on another browser. The existing recording, volume and server wait
countdown are retained. Changed module/style references use 0.4.39; the unchanged
audio and translation assets keep their own versions. These are automated DOM
and CSS contract checks, not a physical iPhone/Safari or real-payment validation.
The legal-copy edits were kept separate and were not included in this review.

Validation executed: `npm ci --ignore-scripts`, then
`node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs`.
281 tests executed and passed, none failed or skipped. No deployment or push.

## Administration: invitations and blocks (cache 0.4.35)

The access view displays only the total `registered_count`. It never lists,
retains or derives a count from the deprecated `registered` array. Missing or
empty `registered` does not affect the panel. If the count itself is unavailable,
the counter displays a dash rather than guessing from addresses.

The invited (`allowed`) list has Revoke (`POST /api/admin/revoke`) and Block
(`/api/admin/block`) actions. Its invitation form calls `/api/admin/allow`.
The blocked list has Remove (`/api/admin/remove`) and Invite (`/api/admin/allow`)
actions, plus a separate block form for an address absent from either list.
All four routes receive `{email}`. Each successful response replaces the view,
including counts. Blocking and removing require confirmation; inviting and
revoking do not. Opening access to everyone still requires its existing
confirmation, and `locked_open` still disables the access switch.

The five-language explanation distinguishes revocation from blocking: revoking
removes the invitation; a sender who has already sent cards retains refund
access. Blocking cuts off access even for an existing sender. Remove forgets the
address from both manual lists. The server remains the authority for sessions
and permissions. Administrator block buttons are disabled, and server errors
remain short, translated messages.

A case-insensitive substring search filters both lists locally without an API
call or moving the search-field focus. It preserves total counts, the query and
unfinished form values across mutations; a successful submitted form is cleared.
Each list is keyboard-focusable and scrolls within a maximum height of 20rem.
Addresses and confirmation messages use text, never HTML. The panel remains
restricted to an exact `session.admin === true` authorization.

No new static file, Go change, mirrored file change or deployment is included.
Existing stylesheet and module references are versioned to 0.4.35 in the main
and classic entry points. Regression tests cover all four actions, confirmation
acceptance/refusal, counters without registered addresses, local search,
hundreds of inert addresses, list bounds, translations, errors and busy-state
request overlap protection. No physical device validation is claimed.

Validation executed: `npm ci --ignore-scripts`, followed by
`node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs`.
268 tests executed and passed, none failed or skipped, including 12 administrator
panel tests.

## Public terms and privacy: owner details filled in (cache 0.4.34)

The five owner fields are no longer placeholders: operator `operator` = Xcoder, `contact` = phoenixmails@proton.me, `law` = Québec (Canada) and the
federal laws of Canada that apply there, `minimum-age` = 18 years, `prohibited-uses` = any illegal use (fraud, money laundering, payments for goods or
services that are themselves illegal; paying for lawful goods or services agreed between users is allowed). The values live in `legal.js`
(`copy.values`, five languages); the static English fallbacks carry the same. The "last updated" line is `copy.notice`. Change a value in both places.

## Public terms and privacy (cache 0.4.32)

The only factual source is `docs/LEGAL-FACTS.md`. The public pages retain its
current gaps rather than inventing retention periods, guarantees or operator
decisions. Finished cards are not automatically deleted today; removal of email
addresses and messages from Wire's state is available on request. The request
contact remains visibly unresolved. No response deadline is promised. Chain
records and provider logs are outside that removal. The pages disclose NIM card
private-key storage, the temporary visibility of a complete stablecoin claim link
when mailing it, provider visibility, the margin ledger and the exact fee rule.

**Four new static files must be declared by the server before deployment:**

- `internal/web/static/terms.html` → `/terms.html`
- `internal/web/static/privacy.html` → `/privacy.html`
- `internal/web/static/legal.css` → `/legal.css`
- `internal/web/static/legal.js` → `/legal.js`

No Go, mirrored file, third-party script, inline script/style or deployment is
included. The pages use self-hosted assets only. English content is also present
as static HTML if scripts are unavailable. Their self-hosted module renders the
full FR/EN/ES/DE/ZH content with text nodes, including the data table. It uses the
same `wire-language` and `wire-theme` preferences as Wire; automatic language
selection and blocked storage are supported. Reading a page calls no API.

The main window, classic page, NIM recipient entry pages and stablecoin recipient
page link both documents in their footers. The local card templates also link
them beside the code-required explanation. Links carry only an allowlisted
language, never card parameters or the fragment. Shared links follow the current
page language. The logo and fixed PhoenixMail footer identity are retained.

### Complete register of unresolved public fields

Every field below displays `[À CONFIRMER PAR LE PROPRIÉTAIRE]` in all five
languages. Field labels and explanations are translated; the literal marker
remains the same so it can be searched reliably. These are unresolved decisions,
not finished legal terms. No contact address, operator, jurisdiction or age is
invented.

| Field id | Owner decision | Terms | Privacy |
|---|---|---|---|
| `operator` | Operator name | yes | yes |
| `contact` | Contact address for requests | yes | yes |
| `law` | Governing law | yes | no |
| `minimum-age` | Minimum age | yes | no |
| `prohibited-uses` | Prohibited uses | yes | no |

`pendingFields` in `legal.js` is the executable register. Tests compare it to
every rendered marker and this list, check the English fallback, all five
languages, shared links, storage denial, theme preferences and absence of
forbidden visible wording. No legal review or physical device validation is
claimed. The owner must resolve the highlighted fields before treating these
pages as final public policies.

Validation executed: `npm ci --ignore-scripts`, then
`node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs`.
261 tests executed, 261 passed, none failed or skipped (including 15 legal-page
tests). The existing footer colour check now targets the brand word directly
because the attribution prefix is also localized.

## Initial navigation appearance (cache 0.4.31)

The main entry page declares a dark browser colour scheme before its external
stylesheets. Its body starts hidden using the standard HTML attribute; the
existing `.wire-body` display rule reveals it as soon as Wire's stylesheet is
applied. This avoids exposing raw text and a white canvas during initial loading.
The page does not wait for scripts, session requests or browser storage to become
visible. The selected light theme still uses its existing light colour scheme.
When the main module runs, it clears the startup attribute so visibility checks
and keyboard focus traps continue to recognise all visible controls.

No inline script/style, content-security-policy change, mirrored file or new
static file is required. The CSS cache version is bumped to 0.4.31. A DOM
regression checks the state before/after CSS, light appearance and operation
without scripts. This is not a physical browser or device validation.

Validation: `npm ci --ignore-scripts`, followed by
`node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs`:
246 tests executed and passed, none failed or skipped.

## Token and network marks (cache 0.4.30)

Three new **192×192 transparent RGBA PNG files** distinguish the token and its
network. The original USDT/USDC artwork is unchanged; a network badge sits at
the lower right, approximately 38% of the token diameter, with a 2px white rim.
Polygon uses its existing connected-hexagon mark in white on `#8247E5`;
Ethereum uses its existing diamond facets in white/grey on `#627EEA`.
The finished files are checked visually and by decoding their PNG pixels.

**Server registration required before deploying** (no Go change is included):

- `internal/web/static/brand/usdt-polygon.png` → `/brand/usdt-polygon.png`
- `internal/web/static/brand/usdt-ethereum.png` → `/brand/usdt-ethereum.png`
- `internal/web/static/brand/usdc-polygon.png` → `/brand/usdc-polygon.png`

These are the only new files under `internal/web/static/`. Serve them from Wire's
static allowlist with an image/png content type; the sender and recipient pages
use these same absolute paths. No separate engine copy, new script route or 96px
variant is needed. Existing token-only image routes remain available for older
pages and server-generated emails. Email rendering and assets in Go are untouched.

The existing `model.js` exports `assetMark(asset, language)` for the exact asset
path and FR/EN/ES/DE/ZH alternative text, e.g. `USDT on Polygon` versus `USDT on
Ethereum`. NIM keeps its existing mark. The shared live-card renderer, classic
preview/funding card, currency picker, window heading and My cards rows use this
mapping. Wire's existing window observer replaces only the mirrored page's
displayed `.usdt-coin` source/alternative text, including its real funding/sent
card; no mirrored code or emailed HTML is modified. Language changes repaint
the marks. Network names remain written on the cards as well.

The recipient HTML (`engine/secureclaim/web/usdt.html`, plus the older EVM
recipient renderer in `index.html`) uses the same paths and five-language naming
convention. It shows a coin only for one of the three known token/network pairs,
after the card is known. Existing verification, signing and transfer logic is
unchanged. Plain inline scripts remain plain scripts so the existing generated
content-security-policy hashes continue to apply.

Regression checks cover three distinct images and paths, dimensions, alpha,
badge colours/rim, alternative text in all five languages, every live preview,
the picker, history, native funding/sent card and recipient pages. No deployment,
real payment or physical iPhone validation is performed. Artwork provenance and
upstream license are recorded in `THIRD-PARTY-NOTICES.md`.

Validation: `npm ci --ignore-scripts`, then
`node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs`:
245 tests executed, 245 passed, none failed or skipped.

## Local stablecoin card preview (Wire override, cache 0.4.29)

`wire-pay.js` enhances the mirrored USDC/USDT window without changing `pm/`.
Once a stablecoin has been selected, the normal form has a live card alongside
it (below it at mobile widths). The currency picker itself stays uncluttered.
The override observes replacement windows and listens to bubbling input/change
events, including recipient, decimal amount and message changes. It creates no
payment, calls no API, opens no wallet and stores no payment key while rendering.

NIM and stablecoins share `renderCardPreview` in the existing `model.js`, the
existing NIM card template, and `paintCard`'s twelve pastel palettes. The stablecoin
clone uses scoped data attributes instead of duplicate IDs. Its public colour seed
stays fixed for that window; it is unrelated to payment or link keys. The preview
shows the Phoenix Wire logo, the localized protected badge and heading, the selected
USDC/USDT symbol and Polygon/Ethereum network, the coin pictogram, email-code
explanation, inert message and small recipient email. All text exists in FR/EN/ES/DE/ZH.

Valid stablecoin preview amounts follow the mirrored form's decimal syntax and
range: 0.01 through 1,000, up to six fractional digits, no exponent or sign.
Empty, incomplete or invalid values show `0` with the selected symbol. The short
localized 24-hour sentence is shown once on the local card; the same lifetime
note is moved from the form rather than duplicated. User text is assigned through
`textContent`, and repeated observer passes do not rebuild unchanged text nodes.

When the mirrored module creates its server-backed card, its normal replacement
of the preview panel removes the local card. Wire detects the real card, removes
the draft layout and leaves the real funding/confirmation controls in charge.
The existing full lifetime/refund explanation is then restored to the form once.
No local draft can overwrite a server card. Changing language repaints the draft
without resetting fields or changing its colour.

Tests cover every stablecoin asset in all five languages, each input's immediate
update, invalid amounts, inert HTML, no API/wallet calls or local payment record,
one 24-hour sentence, language changes, mobile CSS visibility, all twelve shared
palettes, and replacement by the server card. Cache versions are bumped for changed
styles, modules and their loaders/imports. No new static file is added, and no Go,
mirrored code, deployment or real payment is involved. Responsive checks here are
automated DOM/CSS checks, not a physical iPhone validation.

Validation executed: `npm ci --ignore-scripts`, followed by
`node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs`:
241 tests passed, no failures or skipped tests.

## Sender-specific refund memory and configuration-locked access (server c8b3405)

Both sender interfaces now store the last successful refund destination under
`wire-refund-address:<lowercase chain>:<verified session email>`, with a trimmed,
lowercase email. On module load, legacy global Polygon/Ethereum entries (including
the classic interface's capitalized chain names) are deleted without migration.
Signing out clears the departing sender's remembered destinations on both chains;
changing sender also clears the previous sender's memory. Card link keys are kept.

An automatic refund requires a destination saved for this verified sender on this
browser and a successful authenticated ownership/status check for the same card.
The card's recorded sender must match the current identity. A funding address or
another sender's saved destination never supplies an automatic destination. Without
the sender's own saved address, the normal manual address field stays empty. The
classic flow receives the verified email directly from the sender session, records
ownership on newly created cards, reconciles older local entries with the server's
owned-card list, and invalidates stale work when the session changes. Neither path
saves a late refund result into the next sender's memory.

Administration reads `locked_open === true` from every access response. It disables
the invitation-only switch, keeps the server's effective `open` state, and adds an
accessible explanation below the switch in FR/EN/ES/DE/ZH. It also ignores a
programmatically dispatched change while locked. Allow/block actions remain usable;
responses that omit `locked_open` or set it to false preserve the normal switch.

Regression tests cover sender A's successful remembered refund and sign-out,
sender B with preserved local link keys and an expired card, legacy value deletion,
an empty manual field, foreign/unverified cards, same-sender automatic refunds,
and the locked/unlocked administrator view in all five languages. Module loaders
and imports use cache version `0.4.28`. No new static file is introduced.

Validation executed for these changes: `npm ci --ignore-scripts`, then
`node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs`:
224 tests passed, no failures or skipped tests. No Go or mirrored file changed;
no deployment or real payment was performed.

## Administrator access (server c0f0cad and later)

The shared Wire-only `admin.js` panel is mounted only for a verified session with
`admin === true`. It is removed on sign-out and re-reads the session after email
verification. A scoped recipient/payment handoff does not gain administration.
The panel reads `/api/admin/access`, displays the server's `open` switch and
registered/allowed/blocked lists, and applies each POST response as the new view.
The switch has a fixed five-language ‘Invitation-only access’ label: checked means
`open === false`; unchecked means `open === true`. It posts `{open: !checked}`.
A separate sentence below explains whether only registered/allowed senders can
send (existing cards remain accessible), or anyone can sign up and send.
Only opening access to everyone asks ‘Open Phoenix Wire to everyone?’; closing
access does not ask for confirmation.
Blocking/removing an address still requires a confirmation.
Administrators listed by the server cannot be blocked through the panel.
`admin_required`, `cannot_block_admin`, `list_full`, and `invalid_email` have
friendly FR/EN/ES/DE/ZH messages; API authorization remains server-enforced.

For signed-out visitors, `config.invite_only === true` shows the invitation
notice and an “I already have access” control before the email form. Opening
that control only unfolds and focuses the form: it sends no request. Open-mode
configuration shows the normal form directly. Classic and payment-window
entry points share this behavior. Public card reception remains available.

## Card lifetime: 24 hours (server b176bd3)

Read `docs/FIN-DES-CARTES.md` for the server contract. All currencies have a
24-hour lifetime measured from creation. Before payment, both Wire sender
interfaces show the five-language lifetime/refund notice by the amount. NIM
completion and the stablecoin window/notification show the local expiry date;
completion dates use creation plus 24 hours, never the time the UI saw payment.
Recipient pages show `expires_at` below the amount. `claim_unavailable`,
`claim_expired` and `card_expired` produce the friendly unavailable-card message.

`refunding` and `refunded` NIM records have translated labels and no recovery,
retry or cancellation actions. They participate in the three latest displayed
records even after 24 hours. Pending refunds continue polling; a foreground
refresh also discovers server refunds after the original funding view expired.
Expired stablecoin cards with a remaining balance stay visible beyond 24 hours.

Wire's dashboard signs an expired stablecoin refund using its local card entry,
with the existing checked cashlink refund implementation. It needs both the
browser key and an explicit valid refund address: a recorded sender-wallet
address, or the last successful refund address stored locally for that chain.
The card's own funding destination, a recipient address, and arbitrary wallet
accounts are never used as refund destinations. If no such address is available,
one address field and a clear recovery button are shown. Missing local keys
produce a browser-recovery explanation. No address is guessed.

Automatic recovery is attempted once per card per signed-in page session, only
after a fresh non-stale expired status with funds. In-flight requests are guarded,
accepted refunds are remembered across reloads, and failures remain visible with
manual retry. Manual recovery retains the existing confirmation about retained
relayer fees; that confirmation text is unchanged. The classic stablecoin flow
also remembers successful refund addresses and can reuse them for expired cards.
Neither private card keys nor verification codes are stored in the new address
memory or sent as part of a refund request. No mirrored `pm/` file is changed.

The PhoenixMail presentation FAQ has the 24-hour explanation in five languages,
selectable within that FAQ; the main FR/EN page choice also updates it. No new
screenshots are published. Tests cover the sender notices, local deadlines,
recipient errors, refund stages, old funded stablecoins, explicit and automatic
refunds, missing browser keys, repeated foreground events and the FAQ languages.

## Sender access by invitation

As of server `f25eb40`, `POST /api/sender/request` returns HTTP 403 with
`code: "invite_only"` for an address that cannot send yet. No email is sent.
Wire's classic sender screen and payment-window entry show the same friendly
invitation notice in FR, EN, ES, DE and ZH, from `flow-text.js`. The notice
explicitly says that received cards can still be opened. The public pages and
recipient claim flow remain available.

After this refusal, the sender address remains editable, the code-entry box is
hidden, and no resend countdown starts. Another address can be tried immediately;
the payment-window cooldown starts only after a successful request. Neither
interface advertises new-sender registration. All handling lives in Wire's own
files; `internal/web/static/pm/` is unchanged.

Regression coverage in `tests/wire-pay.test.mjs` exercises all five languages in
both sender interfaces, immediate retries, timer cleanup and a successful request
after changing the email address.

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

Five small, marked changes to the mirrored integration (to re-apply after a new sync):

1. `pm/nimiq-payments.js` `cardShell`: the card's emblem and name come from an `o.brand` option (PhoenixMail's by default).
2. `pm/usdt-cashlink.js` and `pm/usdt-payments.js` `sw()`: the texts say `window.PhoenixBrandName` instead of "PhoenixMail".
3. `wire-pay.css` hides "Request payment". NIM is now visible: the host binds its currency button to Wire’s existing protected NIM flow, without modifying the copied files.
4. `pm/usdt-cashlink.js` and `pm/usdt-payments.js` merge the selected language's `window.PhoenixWordOverrides` over their default words. Wire sets only its host-specific copy in `wire-pay.js`. The funding panel in `pm/usdt-payments.js` formats `fundedMsg` with `{email}` from `entry.protectedRecipient || entry.to`. With no overrides, the original words and behavior are unchanged. Wire says it sends the card, uses Done, and does not describe an email composer.

5. `pm/usdt-cashlink.js` first line: `SPDX-License-Identifier: MIT (originally AGPL-3.0-only in PhoenixMail; licensed here under the MIT License by the copyright holder)` instead of `AGPL-3.0-only`. This repository is MIT; see THIRD-PARTY-NOTICES.md.

NIM now shares this entry point and sender verification. `flow.js` exports its existing flow for the NIM template in
`index.html`; `classic.html` remains compatible and `/?claim=…` still opens it for NIM recipients. Sender free gas remains
unavailable (`/api/usdt/status` says `gas:false`).

## Wire navigation, identity and session copy

The two top PhoenixWire/PhoenixMail navigation buttons have been removed from
both sender pages. PhoenixMail is linked once in the footer through the whole
“Powered by PhoenixMail” pill: a fixed orange outline, dark surface, white
Phoenix and orange Mail, slightly larger text. These brand colours do not follow
Wire's palette rotation. The payment-window return controls are retained.

Wire adds an explicit return control to each payment-window instance from `wire-pay.js`. It invokes the copied window's own
close button so its listeners and observers are cleaned up. Asset switches recreate the window; a host observer enhances each
replacement. Beyond the marked hooks, the mirrored files remain unchanged. The
footer attribution links to `https://phoenixmail.ca`; the redundant top navigation
is removed from both the main and classic sender pages.

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

NIM rows use `/api/payments` and its `claim_state`; mail delivery alone never establishes receipt.
The sender history and cancellation contract is documented below.

## Nimiq Pay per-card handoff (client)

The wallet chooser creates nothing on opening. Only clicking “Pay with Nimiq Pay” prepares the protected NIM card
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

The light variants use visibly tinted backgrounds and stronger field borders, accents and secondary text. PhoenixFire dark now uses
neutral charcoal surfaces rather than brown-purple panels, while retaining its orange outlines and icons. Its buttons are neutral charcoal
in dark mode and white in light mode. Aurora and Ocean are unchanged. Official bundled currency logos appear in the currency picker, selected checkout heading and
history rows, at 24 px, with text labels retained and decorative icons hidden from screen readers.

## Card history and dates

Payment statuses belong to My cards and the existing delivery banners, not the sender session panel. History refreshes no
longer copy the latest card state into `wire-status`; that area remains available for sender verification and service errors.

History shows server-provided `created` dates. It is also ready to show optional `sent_at` and `claimed_at` fields, formatted
in the selected language and the browser's local time zone. USDC/USDT dates are accepted from `/api/usdt/cards` and its
cashlink status response; NIM dates come from `/api/payments`. A NIM `claimed_at` must mean an actual confirmed claim,
not release of the claim link. Missing, invalid and zero dates are omitted. Polling time and local creation estimates are
never substituted for these dates. The current server exposes stablecoin `claimed_at` and NIM `claim_state`; receipt labels use those fields as described below.

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
   `POST /api/payments/<id>/{art,start,checkout,confirm,resend}` for that id, and sign out. Everything else answers `403 handoff_scope`: no other payment,
   no creation, no recovery key, no USDC / USDT card, no hand-over of its own. It lasts two hours at most, never beyond the payment's own day or the
   Safari session. It survives a restart of the service.
4. A browser already signed in as someone else is never replaced (`409 session_conflict`, the right is not used up); the same sender already
   signed in is left as it is (`200 {already_signed_in: true}`, no cookie change). A payment whose funding has started meanwhile: `409 payment_not_fundable`.
   Every other failure of the token: the same `404 invalid_or_expired`.
5. No double payment: `start` can be called once for a payment from either browser (`409 funding_already_started` after); the client checks
   `confirm` before any new request to the wallet.

### Asking the wallet again (`POST /api/payments/<id>/checkout`)

For a payment whose funding started (`start`) but never showed up on the chain (the wallet refused, failed, or the app was closed). It answers the same
`{payment, checkout}` as `start`, without starting anything again, and only when: the funding was started at least **3 minutes** ago (a first payment would have
shown by then; `409 retry_too_soon` before), the chain does not already hold the amount (`409 already_funded`: follow the card with `confirm`) and the card is within
its day (`410`). A payment that was never started answers `409 funding_not_started` (use `start`). A scoped session may call it for its own payment. Unfunded cards can now be cancelled through the server cancellation route described below; late funds remain recoverable.


### Native request failure and explicit retry (client)

`flow.js` displays a top-of-card failure state, card expiry from `created + 24h`, an explicit Close action, and a five-language
Close action and a separate explicit Cancel card action for verified senders with an unpaid card. “Not yet paid” means
the last server check has not confirmed funding, not a guarantee that the wallet has debited nothing: a pending transaction
may still arrive. A failed chain read displays uncertainty, not an unpaid claim.

Only the primitive type/code/message of a native failure are sampled. A bounded, escaped diagnostic removes known card fields,
emails, URLs, Nimiq addresses, long key-like tokens and message numbers before display as selectable text and logging with
`console.warn`. The raw error object, checkout, page URL and card metadata are never logged. A Copy button copies only that
redacted string; diagnostics are not persisted in browser storage. The exact native funding text remains unchanged.

Retry first confirms the SAME card. Funded/sent cards resume tracking without another wallet request. Otherwise it requests
`checkout`; `retry_too_soon` disables retry and counts down from the server's `started_at + 3 minutes`, with no automatic
wallet opening at zero. Missing timestamps are not replaced by invented dates. `already_funded` follows with confirm,
`funding_not_started` uses start, and `410` disables payment and explains expiry. A chain/API read error fails closed.
The Hub fallback uses the same confirm/checkout gate; it cannot bypass the three-minute delay. One flow-wide busy guard
prevents overlapping wallet requests. An interrupted scoped reload restores the same card and checking/retry controls,
never the raw error or an automatic payment.

Safari's My cards keeps ready cards as waiting for payment, and labels `funding_started` as awaiting payment confirmation:
that state does not itself establish that money was received. Confirmed funding and sent-email states remain separate.

The chooser's main actions are now “Pay with Nimiq Pay” then “Nimiq Wallet”, with “Get Nimiq Pay” below, in all five languages.
They do not detect installation or change the transfer protocol. The selector's Cancel closes only the wallet chooser, not a card.


### Pending NIM cards must not occupy the next form

A non-terminal payment is no longer treated as a permanent editor lock. Full verified senders always have a New card
action, including while a card is waiting for funding. It folds the displayed payment, clears recipient/amount/message,
and creates a fresh request ID. It makes no API request and does not cancel, delete, refund or alter the old card.
Only folded payment IDs are persisted in localStorage `wire-nim-folded`; the fresh-form preference and request ID stay
in tab session storage. Card records and recovery data are never evicted. A folded card remains in history/My cards and
continues to be checked; late funding can send it normally without replacing or clearing the next form. A form generation
guard also prevents a late create/start/wallet response from reclaiming an editor that the sender has reset.

An incoming `funding_started` payment without local approval is checked and given a visible status/retry path, not a
read-only dead end. The status includes server `started_at` in the browser's local HH:MM, the remaining card lifetime,
and a warning that a payment already approved may still arrive. Retry is gated by the existing three-minute checkout
protocol. Progress is in normal document flow, never sticky or overlaid on the card. A single native wallet request still
blocks another wallet request, not the ability to fold a card or edit a fresh form; the reason is visible while waiting.
Scoped app sessions never get New card or recovery: they retain only their card's summary, retry, tracking and Close.

Recovery previously shared the payment-wide `task()` busy guard (which could silently drop a tap), and wrote failures
to `#feedback`, hidden in some layouts. It now has independent per-row loading/error/ready state, a 30-second request
timeout, and an adjacent live status. Rows are stable across background refreshes so an in-flight result remains visible.
A successful response shows a validated Hub recovery link with an explicit “Open recovery” action; a failure leaves a
retryable button on that row. Recovery now requires an address, confirmed funding and `claim_state: unclaimed`, as detailed below; expiry alone never enables recovery. Its link is not put in localStorage or logs. These changes are client-only.

The shared flow remains usable in classic NIM mode; its script/style version references are refreshed without changing
claim routes, the native funding marker, server authorization, CSP or mirrored payment files. Tests cover 375 px CSS/DOM,
fold persistence, a distinct next request/card, late funding, concurrent wallet/recovery actions, recovery timeouts and
scoped restrictions. They do not replace an actual iPhone check after owner deployment.

## Completion and unpaid-card presentation

The sent NIM view hides every other direct child of the host and every success-panel action except its message and Done.
This also covers a prepared-card wrapper and New card; Done closes the complete payment window and returns to home history.
CSS keeps this panel in normal flow and within a 375 px viewport. An unpaid card keeps one detailed status and a short
closing hint instead of repeating the wallet/expiry warning. Retry and its countdown precede technical details.

The 18:23 iPhone recording exposed a CSS sizing error: the row-oriented progress rule gave every descendant paragraph
a 240 px flex basis, including paragraphs in the vertical diagnostic panel. That made large empty bands above the card.
Only the direct progress message now receives that horizontal flex basis. Diagnostics use one content-sized grid column,
10 px gaps, zero-margin details/buttons and no paragraph flex growth. The 375 px computed-style regression test checks
these constraints; it is not a substitute for an iPhone rendering check after deployment. No payment logic changes.

## Receiving NIM: exactly two choices (supersedes 26aa944)

The recipient page (`classic.html`, `/?claim=…`) has one six-digit code field, Resend code immediately below it, then
exactly two buttons: **Receive in Nimiq Pay** and **Receive in Nimiq Wallet**, translated into all five languages.
Both are disabled until six digits are entered. There is no address input, account selector, other-wallet route or extra
release button. Enter in the form does not silently choose a wallet. The technical wallet implementation name is not UI copy.

The first API request is still POST `payout_status` with `{}`. Existing pending/confirmed/failed payouts replace the code
controls and resume the existing three-second tracker without requesting a new code. Unknown status fails closed with Retry.
On an ordinary browser's first load, after confirming that no payout exists and the card has not expired, the existing
public request route sends the code. In Nimiq Pay the page does **not** automatically request another code: the recipient
normally uses its existing code entry. With the recipient handoff described below, it continues without asking for the
code again. Resend remains explicit and rate limited.

Outside the app, clicking Receive in Nimiq Pay sends the entered code by POST to the public card-scoped `/handoff` route.
Only then does it open `nimiqpay://miniapp?url=` with the encoded target
`https://wire.phoenixmail.ca/?claim=<id>&h=<token>`. The opaque 43-character right lasts five minutes and is single-use.
It is not a sender session handoff. The code itself is never in the URL. The browser retains it only in page memory for
an explicit retry if app opening fails, and clears the visible code field. Creating the right does not call payout/release.

On arrival, the page removes `h` immediately with `history.replaceState`, before network requests, storage or display.
It redeems the right exactly once through POST `/api/claims/<id>/handoff/redeem`. A valid six-digit response is kept only
in page memory, never echoed into the input or UI. After checking the existing payout and card, the app automatically
calls `listAccounts()` and `payout {code,to}` using the first account. This continues the explicit choice made in Safari;
it does not request another email or another code entry. An existing payout is simply followed, not submitted again.
Invalid/expired/used or malformed redemption falls back quietly to normal code entry. A payout rejection with status 403
clears the transferred code and displays the empty input for re-entry. Reload/foreground cannot redeem the erased token
again; pending payout state remains recoverable through the existing status route.

With `window.nimiq`, an explicit click calls `listAccounts()` and uses **only the first** address, without an account selector.
After that it submits `payout {code,to}`. Refusal, empty/invalid first account and legacy error envelopes leave both choices
usable and preserve the code. No funding transaction or funding marker changes. Provider reference:
https://nimiq.dev/mini-apps/api-reference/nimiq-provider#listaccounts.

Nimiq Wallet uses the existing `release {code}` operation and validated cashlink URL. The same button opens that URL once
released; no additional lower button is rendered. After release the code controls and Nimiq Pay choice are hidden.
A payout already started locks this route. Pending retry uses only the server-returned destination without another code;
response loss, failures, foreground resumption, expiry and terminal success retain the existing handling. The code and
recipient payout data are never written to browser storage or logs. No Go, CSP or pm/ changes are needed.

Tests simulate both browser contexts, exact button labels and placement, disabled states, token-only app links, creation
only on click, early URL cleanup, redemption followed by payout, quiet fallback, 403 re-entry, no secret storage/logging,
first-account receipt, release, errors, lost replies, status recovery and 375 px
DOM/CSS. These are simulated checks, not a real Nimiq Pay/iPhone execution or real payment.


## Sender history and unpaid-card cancellation (23 October 2026)

This section supersedes earlier history/recovery and no-cancellation descriptions. All changes are in the Wire host;
no Go, CSP, workflows or mirrored `pm/` files change.

`model.js` shares the selection and action rules between the home dashboard and the classic/embedded NIM history.
The home dashboard merges NIM and USDC/USDT before selecting the three newest transactions created within the last
24 hours (creation descending, exact 24-hour boundary excluded). Older NIM cards are additionally retained only with
confirmed funds and `claim_state: unclaimed`. Expired unpaid cards and ordinary cancelled cards are hidden. Filtering
never deletes browser-held claim fragments or recovery records. Missing/invalid creation dates are not replaced by
observation timestamps. Classic/embedded NIM history applies the same selection to its NIM records.

NIM `claim_state` takes priority over mail delivery: `claimed` means Received, `claiming` means Sending, and funded
`unclaimed` means Sent, not yet received. Only the last case has Recover funds/Open recovery. Unknown chain state has
no recovery action. Unfunded recent cards have Payment not received, Retry payment and Cancel card. For stablecoins,
a valid `claimed_at` means Received even if an earlier local state still says ready.

Both host views poll `/api/payments` every six seconds while a recent card lacks a final claimed state, including
cards already sent by email. Older recoverable balances are also rechecked. Polling pauses in hidden tabs, resumes on
foreground, retries failed reads and is cleaned up on close/logout. It never opens a wallet. Funding confirmation and
mail tracking retain their existing separate guards; a history refresh is not a new payment request.

Cancel card appears beside Retry payment in the home/NIM history and in the compact payment failure panel for full
verified senders. Scoped app sessions keep their existing funding-only permissions and do not get cancellation or
recovery. A single translated confirmation precedes `POST /api/payments/<id>/cancel` with `{}`. Success removes the
row, clears its current editor/payment controls and keeps its record available for later balance checks. A card whose
stage is cancelled but whose `claim_state` is unclaimed is treated as late-funded: it remains visible with recovery.

`409 already_funded` is positive feedback (The payment has arrived), followed by `POST .../confirm {}` on the same
card and normal tracking. A temporarily unavailable confirmation preserves that positive result and permits later
read/confirm tracking; it never cancels or requests wallet payment again. `409 cannot_cancel` refreshes the record and
shows a short adjacent message. `503 chain_unavailable` keeps cancellation retryable. Loading guards prevent duplicate
cancel requests. Recovery links are validated and kept only in page memory, never logged or stored.

The unpaid panel has short status lines, Close, Retry, Cancel card and collapsed Technical details. Buttons inherit
the theme accent. All new labels and prompts exist in FR/EN/ES/DE/ZH. Tests cover selection, boundaries, old balances,
state changes, polling, receipt dates, successful/refused/racing cancellation, scoped permissions and 375 px DOM/CSS.
These are automated simulations, not a physical iPhone/Safari/Nimiq Pay validation or real payments. No deployment.
## Vérification du 24 octobre 2026 : aperçu avant paiement

L'aperçu NIM se met à jour localement pendant la saisie d'un montant valide, du destinataire et du message. Il ne crée pas de carte serveur et n'ouvre aucun portefeuille. Le Nimicon réel apparaît après la préparation de la carte. Le texte des 24 heures n'est ajouté qu'une fois ; l'enrichissement stablecoin ignore la fenêtre NIM. Les modules modifiés utilisent la version de cache `0.4.27`.

Revue de sécurité et limites de validation : voir `AUDIT-2026-10-24.md`. Les défauts Go recensés dans ce rapport restent à corriger ; le correctif d'aperçu n'est pas une validation générale des remboursements.




