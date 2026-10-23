# PhoenixWire

**Rapide. Simple. Sécuritaire.** — Powered by Phoenix Mail.

Email payment cards without a password. The sender confirms an email address, funds a protected card once, and the recipient
claims it after confirming their own email. Wire runs its UI, gateway and imported SecureClaim engine in one Go process.

## Current interface

- `/` hosts **PhoenixMail's payment window**, copied into `internal/web/static/pm/`.
- `wire-pay.js` handles sender verification, the window's options, card monitoring and invitation delivery;
  `wire-pay.css` and `index.html` provide Wire's frame and visual overrides.
- The window offers **USDC Polygon, USDT Polygon and USDT Ethereum**, subject to enabled server assets. All cards are protected.
- NIM remains in `/classic.html`; `/?claim=…` also opens that earlier interface for NIM claims.
- Sender confirmation lasts **24 hours on this device**. A different device or an expired/cleared session requires a new code.
- Explicit navigation returns to the PhoenixWire presentation or PhoenixMail; the payment window has a return button.
- Five UI languages (FR/EN/ES/DE/ZH), light/dark appearance, pastel payment cards, favicon and mobile shortcut icons.

Do not edit the payment-window copies casually: [PAYMENT-WINDOW.md](docs/PAYMENT-WINDOW.md) documents the synchronization
boundary and existing marked changes. Wire's own host should carry appearance and navigation changes.

## Start locally

Requires Go 1.22+. No npm installation is needed to run the application.

```sh
./run-local.sh
```

Open http://127.0.0.1:8790. Without payment configuration the current sender page reports payments unavailable;
`/classic.html` retains its local design preview. With payments enabled, Wire uses private configuration from
`~/.config/phoenixwire/local.env` (0600), following `.env.example`. USDC/USDT require their additional chain switches,
relay funds and RPC configuration. See [local testing](docs/LOCAL-TESTING.md).

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

## Repository

- `cmd/phoenixwire`: entry point.
- `internal/wire`: sender sessions, payment gateways, encrypted state and invitations.
- `internal/web`: current host, copied payment window and classic NIM interface.
- `internal/cards`: mirrored email presentation and Wire adapters.
- `engine/secureclaim`: integrated engine with its own Go module and retained upstream provenance.
- [Architecture](docs/ARCHITECTURE.md), [design](docs/DESIGN.md), [hosting](docs/DEPLOYMENT.md),
  [engine import](docs/IMPORT-USDT-CASHLINK-2026-10-23.md).

Original code in this repository is MIT; third-party terms remain intact. This does not change other repositories' licenses.
Work on `main`; commit subjects end with `[skip ci]`. Do not modify workflows or deploy as part of UI changes.
