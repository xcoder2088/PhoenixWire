# Included client libraries

- internal/web/static/hub.js: official @nimiq/hub-api 1.14.0 bundle, reused unchanged
  from PhoenixMail; adjacent Apache-2.0 LICENSE-HUB-API preserved. API reference:
  https://www.nimiq.dev/hub/api-reference
- internal/web/static/iqons.js: existing official @nimiq/iqons 1.5.2 artwork/adapter
  from the imported engine, with adjacent LICENSE-IQONS. See the engine's
  web/NIMIQ-NOTICES.md for full upstream attribution/packaging notes.
- Go Authors portable Blake2b and its BSD license remain under the engine.
- JSDOM is used only for development tests; it is not part of the Go binary.

The MIT grant for original code does not replace these licenses or branding rights.

Card branding: the Nimiq wallet's USDT mark is reproduced from
https://github.com/nimiq/wallet/blob/master/src/components/swap/animation/usdt.svg
for identification of the token. Nimiq and Tether branding remain their respective
owners' property; the software MIT license does not grant trademark rights.
The Phoenix emblem and official Nimiq mark reuse the existing project assets.

## Added with the USDC / USDT cashlink import (2026-10-23)

- `github.com/decred/dcrd/dcrec/secp256k1/v4` v4.4.1 — ISC License (a Go module dependency; it is not copied into this repository).
- `engine/secureclaim/web/jsqr.js` — jsQR 1.4.0 by Cosmo Wolfe, Apache-2.0, license kept in `engine/secureclaim/web/LICENSE-JSQR`.
  It reads a wallet address from the camera on the claim page and is loaded only when the recipient asks for it.
- USDC and USDT marks in `engine/secureclaim/web/usdc-coin.png` and `usdt-coin.png` identify the tokens; they belong to Circle and Tether.

- internal/qrcode: skip2/go-qrcode, BSD-3-Clause (internal/qrcode/LICENSE), copied from PhoenixMail's vendored copy with only the module import paths changed. It draws the QR of a payment link on the server.
- internal/web/static/pm/nimiq-hub.js: the same official @nimiq/hub-api bundle as hub.js (see LICENSE-HUB-API); pm/nimiq-iqons.js: Iqons (see LICENSE-IQONS); pm/LICENSE-NIMIQ-MODULE: the Nimiq module notice that goes with them.
