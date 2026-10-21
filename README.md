# Phoenix Wire

Lightweight payment-card application in the PhoenixMail ecosystem.

**Current state: local UI foundation only. No funds, email, wallet connection,
OTP verification, or claim handling. Nothing is sent when previewing a card.**

The complete tracked SecureClaim source and tests are included under
[`engine/secureclaim`](engine/secureclaim), except GitHub workflows. This is a
separate Go module, not yet connected to the preview. Its original runtime behavior
is preserved; it is not started automatically and must not be run with funded state
or production credentials as part of a UI preview.

## Run locally

Requires Go 1.22+ (1.22.0 or newer). No npm install, credentials, or sibling service required.

```sh
go run ./cmd/phoenixwire
```

Open http://127.0.0.1:8788. One terminal; Ctrl+C stops the application.
PhoenixMail keeps its own port 8787 and is not modified.
The UI assets are embedded in the Go binary. A desktop wrapper is not implemented yet.

Optional override: `PHOENIXWIRE_ADDR=127.0.0.1:8788`. The default binds only to loopback.
Do not expose this foundation as a production financial service.

## Included

- Selected phoenix + luminous wire logo concept.
- Light/dark appearance, three accent choices, pastel payment-card previews.
- French, English, Spanish, German, and Chinese UI.
- Sender/recipient email, currency, amount, optional note; all form content stays in memory.
- NIM, USDT Ethereum, USDT0 Polygon **visual previews only**.
- Permanent footer: **Powered by Phoenix Mail**.
- Read-only health/config endpoints; payment creation returns HTTP 503.
- Standard-library Go server and dependency-free JS tests.

## Validate

```sh
go test ./... -race -count=1
go vet ./...
node --test tests/*.test.mjs
(cd engine/secureclaim && go test ./... -race -count=1 && go vet ./...)
node --test engine/secureclaim/web/tests/*.test.cjs
```

See [architecture and next steps](docs/ARCHITECTURE.md) before implementing payments.
No workflows or automatic deployments are included.

## License

MIT; the existing [LICENSE](LICENSE) is preserved. Imported original SecureClaim code
has its own MIT license; third-party notices remain in place. PhoenixMail itself is
not relicensed. See [import provenance](docs/SECURECLAIM-IMPORT.md) and [brand notes](docs/BRAND.md).
