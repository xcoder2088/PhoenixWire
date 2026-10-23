# Import of the USDC / USDT cashlink engine (2026-10-23)

Imported at the owner's explicit request, ahead of the competition submission: Phoenix Wire is MIT-licensed, the sources below
were written by the same owner and are licensed here under the MIT License by the copyright holder. The originals stay under
their own terms in their own repositories (PhoenixMail and PhoenixCashlink: AGPL-3.0-only; NimiqSecureClaim: private, no
license selected). This import does not change those licenses and grants no trademark rights.

## Sources and commits

| Source repository | Commit | What was taken |
|---|---|---|
| NimiqSecureClaim | `4eb23e309ceec7be9c37ba0e33c458a32638f97b` | everything added or changed since the first import (`a344dcb`): `internal/usdtlink`, `internal/evm`, `internal/evmcard`, `internal/evmtest`, `internal/cashlink`, the HTTP handlers (as `service/usdtlink_http.go`, `service/evm_http.go`), `internal/otp` and `internal/vault` updates, `web/usdt.html`, `web/cashlink.js`, `web/jsqr.js`, the USDC/USDT coin images and tests |
| PhoenixCashlink | `924d4918609aab083715aaad958552f58c27ae87` | the protocol packages mirrored in `engine/secureclaim/internal/cashlink` (evm, safe, chain, relayer, cosigner, proof) and `web/cashlink.js` |
| PhoenixMail | `6d2b3b69a95f6f7e34d397f16569e9451105ff54` | reference for the card palettes (twelve families), the USDC/USDT card texts and the withdrawal/relayer ideas; the card renderer mirror (`internal/cards/*_mirror.go`) is refreshed separately by `scripts/sync-phoenixmail-cards.sh` |

Merge method: a three-way merge per file against the first import (`a344dcb`), with Wire's own changes kept. New files were
copied; `cmd/server/*.go` became `service/*.go` (`package service`). Conflicts were resolved by hand in `internal/otp/mailer.go`,
`internal/otp/mailer_test.go` and `web/index.html` (the upstream versions were taken, they already contain Wire's changes plus
the e-mail language and card-colour additions). Wire's own `internal/otp/mail_language.go` was kept except for the card-colour
helpers added at the end.

## What this import is and is NOT

- It makes the whole engine **available and tested** in this repository: `go test ./...` in `engine/secureclaim` and at the root,
  and the JavaScript tests, all pass.
- It does **not** switch anything on. Wire still starts as a card preview unless `PHOENIXWIRE_ENABLE_PAYMENTS=1`, and the USDC /
  USDT cashlink routes stay off unless their own environment switches are set (see `docs/USDT-CASHLINK.md`, `docs/USDT-CARDS.md`).
  No key, state, vault, log or credential was copied.
- Wiring the USDC / USDT cashlink into Wire's own gateway and interface is the next piece of work.

## Licensing notes

- Files that carried `SPDX-License-Identifier: AGPL-3.0-only` now carry `SPDX-License-Identifier: MIT` and a note that they were
  originally AGPL-3.0-only (the copyright holder relicensed this copy). 14 files.
- Third-party code keeps its own license: `github.com/decred/dcrd/dcrec/secp256k1/v4` (ISC, a Go dependency), jsQR 1.4.0
  (Apache-2.0, `web/LICENSE-JSQR`), `@nimiq/iqons` (see `web/LICENSE-IQONS`), the Go Authors' Blake2b (BSD) and the go-qrcode
  encoder where present. See `docs/THIRD-PARTY-NOTICES.md`.
- Brand names and marks (Nimiq, Tether, Circle/USDC, PhoenixMail, Phoenix Wire) remain their owners'; MIT does not license them.

## Keeping the mirror current

`scripts/sync-phoenixcashlink.sh` re-runs PhoenixCashlink's `scripts/vendor.sh` into `engine/secureclaim/internal/cashlink` and
re-applies the MIT headers. Run it after any protocol change in PhoenixCashlink, then the tests, then commit with `[skip ci]`.
