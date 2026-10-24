# Contributing

Phoenix Wire is MIT-licensed. By contributing you agree your contribution is licensed under the same terms.

## Before you open a pull request

```sh
npm ci --ignore-scripts
node --test tests/*.test.mjs engine/secureclaim/web/tests/*.cjs
go vet ./... && go test ./...
(cd engine/secureclaim && go vet ./... && go test ./...)
```

All of it must pass. Tests use local fixtures only: never put a real key, address, e-mail or credential in a test, a screenshot or a document.

## Rules that keep the project safe

- **Money paths need tests that attack them**, not only tests that show them working (a stranger paying a card's public address, a retry after a
  crash, two senders, a full disk).
- **A new file the pages load must be added to the server's list** (`internal/web/server.go`); a test fails if a shipped file is not served.
- **Mirrored code**: `internal/web/static/pm/` and parts of `internal/cards/` are copies of PhoenixMail files. Change Wire's own host,
  styles and overrides instead; see [docs/PAYMENT-WINDOW.md](docs/PAYMENT-WINDOW.md) for the boundary.
- **Texts** are in five languages (FR, EN, ES, DE, ZH). Do not show internal words ("Mainnet", "OTP", "Hub") to users.
- **Do not change workflows or deploy** as part of an interface change; deployment is the owner's.
- Keep the vulnerability reports private ([SECURITY.md](SECURITY.md)).
