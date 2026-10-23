# Third-party components

Original nimiq-go and email authentication integration code is MIT licensed.

- `internal/blake2b`: Go Authors, golang.org/x/crypto v0.31.0, BSD license in `internal/blake2b/LICENSE`. The portable implementation is used on all architectures; the platform build restriction was removed from `blake2b_ref.go`.
- `assets/nimiq-hub.js`: official @nimiq/hub-api 1.14.0 standalone distribution, Apache-2.0; see `assets/LICENSE-HUB-API`. Source: https://github.com/nimiq/hub/tree/master/client. Download: https://cdn.jsdelivr.net/npm/@nimiq/hub-api@1.14.0/dist/standalone/HubApi.standalone.umd.js.
- `assets/nimiq-logo.svg`: official Nimiq wallet icon, unchanged from https://github.com/nimiq/wallet/blob/master/public/img/icon-gradient-on-blue.svg. Nimiq branding remains its owner's property; the MIT grant for our original code does not relicense that branding or grant trademark rights.

PhoenixMail's own license is separate and unchanged by using this module.

- `nimiq-iqons.js`: official `@nimiq/iqons` 1.5.2 bundled artwork and algorithm, matching Nimiq Wallet. ES module exports adapted to a scoped classic script; address adapter and PNG export are original MIT code. Upstream repository MIT notice in `LICENSE-IQONS`; npm package metadata labels this release ISC. No remote image lookup is used. Source: https://github.com/nimiq/identicons and https://www.npmjs.com/package/@nimiq/iqons/v/1.5.2. An Iqon is a visual aid, not proof of ownership or authenticity.

## jsQR 1.4.0 (web/jsqr.js)
QR code decoder used by the claim page to read a wallet address from the camera. Apache License 2.0, Copyright Cosmo Wolfe. Source: https://github.com/cozmo/jsQR. Loaded only when the recipient presses "Scan with the camera" and the browser has no native BarcodeDetector.
