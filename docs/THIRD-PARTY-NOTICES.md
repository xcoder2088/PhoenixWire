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

## Token/network composites (2026-10-24)

`internal/web/static/brand/usdt-polygon.png`, `usdt-ethereum.png` and
`usdc-polygon.png` are transparent 192×192 composites. Token artwork comes from
the repository's existing `brand/usdt-coin.png` and `brand/usdc-coin.png`, retaining
the existing Tether/Circle identification notices below. Network marks are reused
from Trust Wallet's assets collection, not newly drawn logos:

- Polygon: https://github.com/trustwallet/assets/blob/master/blockchains/polygon/info/logo.png
  (download SHA-256 `ba495a374776ab6cfaecc15b3e3d0f9b04743cd0309f99575b8599fc0112e030`).
- Ethereum: https://github.com/trustwallet/assets/blob/master/blockchains/ethereum/info/logo.png
  (download SHA-256 `c253d7c5d4203783e31bde1d33e143f6a9ce1defac24ab5e06948847696b80a1`).
- Collection license: https://github.com/trustwallet/assets/blob/master/LICENSE
- Brand references: https://polygon.technology/brand-guidelines and https://ethereum.org/assets/

The network marks' original background discs were removed; their geometry was
kept and rendered white (Polygon) / light grey facets (Ethereum) on the requested
solid purple/blue-grey discs. Token diameter is 184px on a 192px canvas; the badge
has a 72px outside diameter including a 2px white rim. Composition was rendered at
4× resolution and reduced for antialiased edges. The supplied reference image was
used for placement, not as a replacement for the existing token artwork.
Brand/trademark rights remain with Polygon, Ethereum, Tether and Circle; this
software license does not grant trademark rights or imply endorsement.

Upstream network-asset license, retained here:

```text
MIT License

Copyright (c) 2019-2023 Trust Wallet

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

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


## Files copied from PhoenixMail and PhoenixCashlink (owner's own code)

`internal/web/static/pm/` (PhoenixMail's payment window), parts of `internal/cards/` (PhoenixMail's card e-mails) and the protocol packages
under `engine/secureclaim/internal/cashlink/` (PhoenixCashlink) were written by the same copyright holder as this repository. The originals
are published or kept under the licenses of their own repositories (PhoenixMail and PhoenixCashlink: AGPL-3.0-only). **The copies in this
repository are licensed under the MIT License by the copyright holder**, which does not change the license of the originals. A few copied files
still carry an old header that names the original license; they are marked in the file with the note "licensed here under the MIT License by
the copyright holder". The third-party parts they contain (the Nimiq Hub API bundle, Iqons, jsQR, the Go Authors' Blake2b, go-qrcode) keep their
own licenses, listed above and in the `LICENSE-*` files next to them.

Brand names, logos and marks (Phoenix Wire, PhoenixMail, Nimiq, Tether, Circle/USDC, Polygon, Ethereum) are not licensed by the MIT License.
