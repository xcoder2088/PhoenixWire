# Phoenix Wire

**Rapide. Simple. Sécuritaire.** — Powered by Phoenix Mail.

Lightweight email payment cards with no account or password. Verify the sender's
email, approve a Nimiq wallet payment, and email a protected card. The existing
SecureClaim engine is embedded in the same Go process.

## Start locally

Requires Go 1.22+. No npm installation is needed to run the application.

```sh
./run-local.sh
```

Open http://127.0.0.1:8790. One terminal; Ctrl+C stops the application.
Without payment configuration this starts the independent card preview.
PhoenixMail's existing ports remain available, including the hosted service on 8788.

To enable the integration, create an owner-only environment file at
`~/.config/phoenixwire/local.env` (0600), following `.env.example`.
Use the real existing MailerSend credentials and a trusted Nimiq RPC.
Then run the same command. See [local testing](docs/LOCAL-TESTING.md).

## Included

- Selected phoenix/wire logo, approved slogan, permanent PhoenixMail footer.
- Five automatic/manual UI languages; light/dark themes, accents and pastel cards.
- Passwordless sender email verification; durable counters and HttpOnly sessions.
- Server-side Cashlink generation and encryption before wallet funding.
- Official Hub checkout to the prepared public address, including mobile redirect.
- Durable idempotent intents: a started checkout is never automatically restarted.
- Funding confirmation through the existing SecureClaim verifier.
- Real MailerSend invitations containing only an opaque protected claim link.
- Existing recipient email verification, official Nimicon and a wallet link after verification.
- PhoenixMail's mirrored pastel renderer, centered linked footer and untracked email delivery.
- Sender history/recovery; email retry reuses the existing funded payment.

Only NIM can be sent after explicit local configuration. USDT Ethereum / USDT0
Polygon remain visual previews. Real email/wallet acceptance with **1 NIM** still
needs validation; browser and automated checks use fixtures. No production service
has been deployed. See [hosting and acceptance](docs/DEPLOYMENT.md).

## Validate

```sh
go test ./... -race -count=1
go vet ./...
(cd engine/secureclaim && go test ./... -race -count=1 && go vet ./...)
npm ci --ignore-scripts
node --test tests/*.test.mjs engine/secureclaim/web/tests/*.test.cjs
```

Integration tests run the real embedded engine with explicit local RPC/mail fixtures.
They do not send real email or spend money. Tests cover code gating/replay, restarts,
sender identity, immutable recipients, duplicate checkout blocking, email failure,
ownership and encrypted state. JSDOM tests exercise the actual flow UI.

## Repository

- `cmd/phoenixwire`: single application entry point.
- `internal/wire`: passwordless sessions, durable intents, mail and claim gateway.
- `internal/web`: embedded responsive UI.
- `engine/secureclaim`: MIT engine; reusable service package plus legacy CLI.
- [Architecture](docs/ARCHITECTURE.md), [import provenance](docs/SECURECLAIM-IMPORT.md).

MIT for original code; third-party licenses are retained. PhoenixMail's licensing is
unchanged. Development on `main`; commits use `[skip ci]`, without workflows or PRs.
