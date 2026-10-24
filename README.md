# PhoenixWire

**Fast. Simple. Secure.** — Powered by Phoenix Mail.

Email payment cards without a password. The sender confirms an email address, funds a protected card once, and the recipient
claims it after confirming their own email. Wire runs its UI, gateway and imported SecureClaim engine in one Go process.

## Current interface

- `/` hosts **PhoenixMail's payment window**, copied into `internal/web/static/pm/`.
- `wire-pay.js` handles sender verification, the window's options, card monitoring and invitation delivery;
  `wire-pay.css` and `index.html` provide Wire's frame and visual overrides. `flow.js` exports the existing NIM flow so it can mount in that same host.
- The common payment window offers **NIM (Nimiq), USDC Polygon, USDT Polygon and USDT Ethereum**, subject to enabled server assets. All cards are protected.
- NIM now uses its existing protected flow in the common window after the same sender confirmation; `classic.html` remains a compatibility page and `/?claim=…` continues serving NIM recipients.
- Sender confirmation lasts **24 hours on this device**. A different device or an expired/cleared session requires a new code.
- Explicit navigation returns to the PhoenixWire presentation or PhoenixMail; the payment window has a return button.
- Five UI languages (FR/EN/ES/DE/ZH), working light/dark appearance, neutral buttons with theme-colored outlines, pastel payment cards, favicon and mobile shortcut icons.

Do not edit the payment-window copies casually: [PAYMENT-WINDOW.md](docs/PAYMENT-WINDOW.md) documents the synchronization
boundary and existing marked changes. Wire's own host should carry appearance and navigation changes.

## Start locally

Needs Go (the manifests say 1.22; build releases with the latest patch release of the current Go, the standard library receives security fixes).
No npm installation is needed to run the application.

```sh
./run-local.sh
```

Open http://127.0.0.1:8790. Without payment configuration the current sender page reports payments unavailable;
`/classic.html` retains its local design preview. With payments enabled, Wire uses private configuration from
`~/.config/phoenixwire/local.env` (0600), following `.env.example`. USDC/USDT require their additional chain switches,
relay funds and RPC configuration. See [local testing](docs/LOCAL-TESTING.md).

## Who may send, and how long a card lives

- **Sending is by invitation.** Anyone can open a card they received (recipients never sign in); signing in as a *sender* is limited to
  senders already registered, addresses an administrator allowed, and administrators. `PHOENIXWIRE_ADMINS` names the administrators; their
  panel (opened by `"admin": true` in `GET /api/session`) opens or closes sending, allows and blocks addresses, and keeps the list in the
  sealed state. `PHOENIXWIRE_SENDERS=open` forces access open (the panel then says so); a list of addresses adds allowed senders.
- **Every card lives 24 hours** (NIM, USDC, USDT). An hour before the end the sender gets one reminder. At the end a NIM card that nobody
  opened is **refunded automatically** to the address that funded it, but only when that origin is certain (one funder, whole history read);
  a USDC/USDT card cannot be moved without the key kept in the sender's browser, so its owner takes the money back himself (the page does it
  on opening when it knows the destination of that sender). A refund keeps the relayer's fee. See [FIN-DES-CARTES.md](docs/FIN-DES-CARTES.md).
- **Fees** (USDC/USDT): the card's own gas at today's price (read from the chain's price feed, then CoinGecko, then a high fallback) times a
  margin, **plus 6 %** of the amount, always both. New cards wait when gas would eat them. NIM has no service fee. Each settled payout writes
  one line in the margin ledger (fee earned against gas paid). The relayer's float and the free-gas light follow the price of gas too.
- **Receiving NIM**: the recipient chooses *Nimiq Pay* (the server pays the card's coins to the address the app shares, without asking the
  code again thanks to a one-use hand-over token) or *Nimiq Wallet* (the usual link). See [PAIEMENT-DIRECT-NIM.md](docs/PAIEMENT-DIRECT-NIM.md).

## Secrets and invitations

The browser generates and retains the USDC/USDT link key. Creation sends its public owner address; recipient claims and sender
refunds send proofs/signatures rather than the private key. **Invitation delivery is the exception:** the sender submits the
complete claim link to Wire, which validates it and sends it through the mail provider without persisting the link. Network
retries may repeat that submission. Do not describe this as a key that can never reach the operator.

Local card storage is device-specific; keep recovery material available while a card may hold funds. Wire has separate keys and
state from PhoenixMail/SecureClaim. No source synchronization should copy operational keys or state.

## Validate

```sh
node --test tests/*.test.mjs engine/secureclaim/web/tests/*.cjs
go vet ./...
go test ./...
(cd engine/secureclaim && go vet ./... && go test ./...)
```

Install the pinned development dependencies with `npm ci --ignore-scripts` for browser tests. Tests use local RPC/mail fixtures;
they do not establish that a deployed instance has delivered mail or settled real funds. Deployment and live acceptance are
owner-controlled; this repository's documentation is not a statement that a particular installation was verified.

## Operating it

- [DEPLOIEMENT-SIMPLE.md](docs/DEPLOIEMENT-SIMPLE.md): deploy, check, roll back, change a setting, back up (French, copy-paste commands).
  `scripts/deploy-laptop.sh` runs the tests, builds, installs, and checks that **every file the pages name is served** (a file missing from the
  server's list breaks the page: `internal/web/server.go` lists what is public, and a test fails if a shipped file is not on it).
- [DEPLOY-LAPTOP.md](docs/DEPLOY-LAPTOP.md): what lives where on the laptop, relayer addresses, moving funds from PhoenixMail's relayer.
- [AUDIT-2026-10-24.md](docs/AUDIT-2026-10-24.md): the independent review and what was done about each finding.

## Repository

- `cmd/phoenixwire`: entry point.
- `internal/wire`: sender sessions, payment gateways, encrypted state and invitations.
- `internal/web`: current host, copied payment window and classic NIM interface.
- `internal/cards`: mirrored email presentation and Wire adapters.
- `engine/secureclaim`: integrated engine with its own Go module and retained upstream provenance.
- [Architecture](docs/ARCHITECTURE.md), [design](docs/DESIGN.md), [hosting](docs/DEPLOYMENT.md),
  [engine import](docs/IMPORT-USDT-CASHLINK-2026-10-23.md). The other dated files in `docs/` (reviews, plans, hand-offs) are history, not
  current instructions.

Original code in this repository is MIT; third-party terms remain intact. This does not change other repositories' licenses.
Work on `main`; commit subjects end with `[skip ci]`. Do not modify workflows or deploy as part of UI changes.
