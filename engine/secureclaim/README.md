# Current local Mainnet integration

The work branch now accepts Mainnet claims only. PhoenixMail retains the original Cashlink funding engine; this service encrypts and holds that original secret and releases it only after email OTP verification. No second payment engine or production deployment is introduced.

Start both repositories from PhoenixMail with `./run-local.sh` (ports 8787 and 8099). Real MailerSend credentials are requested locally if missing and are never printed. The launcher stores private stable keys and uses a dedicated Mainnet state directory. Do not reuse older network-state directories or delete their ciphertext/keys.

Funded OTP requires real email delivery configuration, encrypted persistent challenges/counters, private claim persistence and vault, an issuer key, and a Mainnet node identity check. The old `NIMIQ_CLAIM_DEV=1` startup requirement is removed. Unconfigured console mode cannot seal or release funded Cashlinks. All challenge/release APIs enforce this server-side.

The RPC variable is `NIMIQ_CLAIM_MAINNET_RPC_URL`. Existing `MAILERSEND_API_KEY` / `MAILERSEND_FROM` are reused; claim-specific overrides remain supported. Codes expire after 10 minutes with five guesses; claims expire after 24 hours. Retrying protection reuses the same immutable claim and secret. A new code can reopen the same claim after a lost response while it is valid, without funding again. Back up the original master configuration together with the private state; replacing master keys cannot recover old ciphertext.

The sender's original local recovery Cashlink remains in PhoenixMail My sends. A protected recipient receives only the opaque claim URL until OTP succeeds. The sender can still reclaim unspent funds. The public RPC is an external trust/availability dependency; configure a trusted Mainnet node when appropriate.

Automated verification uses fixtures, not real funded payments or actual MailerSend delivery. A real local email-and-wallet check still needs the real credentials and a user-approved payment.

> **USDT cards (Ethereum / Polygon)**: see `docs/USDT-CARDS.md`.

---

## Historical design notes

The following older notes describe earlier development stages and do not define the current runtime network or readiness behavior.

# NimiqSecureClaim

Independent service for secure Nimiq payment claims initiated by PhoenixMail and other clients.

> **Status: development-only existing-Cashlink OTP wrapper, Testnet only.** When explicitly configured, this branch can validate an already-funded Testnet Cashlink, encrypt its seed in a local vault and release the original bearer Cashlink link once after email OTP. **It does not broadcast payouts, bind a receiving wallet, automatically refund, or support Mainnet.** It has not passed full end-to-end live testing; do not use it with real-value NIM.

## Intended flow

1. PhoenixMail creates a pending claim tied to the intended recipient email address and a specific Nimiq network.
2. The recipient receives a pastel payment card containing only an opaque claim URL. **No cashlink seed, private key, or redeemable QR is included in the email or its attachments.**
3. The recipient requests a six-digit one-time code sent to the originally designated email address.
4. After verifying the code, the service authorizes a time-limited, single-use claim operation.
5. The recipient selects a wallet through the supported official Nimiq Hub integration; the transfer/claim outcome is verified on-chain.
6. Unclaimed funds can be safely reclaimed or refunded according to an explicit policy.

## Security design requirements

- Decide and document custody and threat model **before** implementing funding: a server that holds a funded cashlink seed can spend those funds, even if the seed is encrypted at rest.
- Do not rely on client-side gating or hiding QR images. Existing PhoenixMail cashlinks embed a redeemable seed in the link; they cannot be retroactively protected by an OTP screen.
- OTP: cryptographically random six-digit codes, short TTL, hashed server-side, one-time consumption, limited attempts and resend throttling. Do not reveal whether a recipient email exists.
- Bind the claim to its recipient, network, amount, claim ID and wallet-authorization step. Enforce atomic one-time state transitions, replay protection and audit-safe logs.
- Never log cashlink secrets, OTPs, private keys or full bearer tokens; avoid storing sensitive tokens in browser storage.
- Use TLS, restricted API origins, authenticated service-to-service calls and protected configuration/secrets. No production funds before independent review and live testnet tests.
- Provide explicit expiry, cancellation, sender recovery, outage recovery and idempotency semantics.
- Verify real Nimiq Hub behavior and any custody/legal implications before choosing an implementation.

## Separation from PhoenixMail

PhoenixMail remains responsible for its email composer, existing multicolor pastel cards, Nimiq identicons, localization and rendering. This repository owns the independent claim API and secure backend. Payment requests (which contain only a public receiving address) remain unchanged. Legacy cashlinks remain unchanged and must not be presented as OTP-protected.

## Existing-Cashlink Testnet OTP path (development only)

PhoenixMail continues to create/fund the original Nimiq Cashlink using the sender's wallet. PhoenixMail privately sends the original 32-byte seed to this service after confirming the Testnet funding transaction. This service independently verifies the address/confirmed balance, seals the secret with AES-256-GCM, and returns an opaque claim link. No Cashlink seed or redeemable bearer QR appears in the outgoing OTP card.

- `GET /api/v1/health`: non-secret capability report (`issuer_ready`, `vault_ready`, `otp_delivery`, `protected_cashlink_ready`).
- `POST /api/v1/claims`: trusted issuer creates a draft bound to recipient email and exact Testnet amount.
- `POST /api/v1/claims/{id}?action=seal`: issuer-only admission of the already funded original Cashlink, chain verification, encrypted local vault storage.
- `POST /api/v1/claims/{id}?action=request`: OTP dispatched to the fixed recipient; console-only mode is strictly for local development.
- `POST /api/v1/claims/{id}?action=release`: consumes the six-digit OTP and, once, reveals the original Testnet Cashlink URL. **This reveals a bearer spending capability to the verified browser and is NOT equivalent to a wallet-bound on-chain payout.**

The OTP challenges themselves are still held only in memory. Sender recovery is manual through the original Cashlink retained on the sender's device. Repeated admission and release, crash recovery, timeout, refund and custody/legal risks require security review before production.

Run the paired PhoenixMail branch `work/secureclaim-otp-usable` as documented in its `docs/SECURECLAIM-OTP-TESTNET-SETUP.md`.

## Historical draft API (demo-only and unfunded)

- `POST /api/v1/claims` — issuer-authenticated draft creation; the issuer supplies an immutable recipient email, Testnet network and fixed amount.
- `GET /api/v1/claims/{id}` — public read-only draft state, with no email or secret in the response.
- `POST /api/v1/claims/{id}?action=request` — send a claim-specific OTP to its server-held recipient.
- `POST /api/v1/claims/{id}?action=verify` — consume code and mark **email verified, unfunded**.

See [historical draft API and iPhone testing limitations](docs/DEVELOPMENT-CLAIMS.md). Funded OTP claims use the new `release` action; the older `verify` action applies only to the unfunded `demo` claim. These endpoints are available in the *dev-only loopback server*. They are **not** an internet-hosted payment service.

## Planned production API (not implemented)

- `POST /v1/claims` — authenticated claim creation
- `POST /v1/claims/{id}/otp/request` — request recipient email OTP
- `POST /v1/claims/{id}/otp/verify` — verify OTP, establish restricted claim session
- `POST /v1/claims/{id}/redeem` — wallet-bound redeem authorization
- `GET /v1/claims/{id}/status` — safe status view
- `POST /v1/claims/{id}/cancel` — authorized sender cancellation

All endpoints and payloads are subject to security review; do not treat this draft as a deployed service.

## Licensing

No license has been selected for this private repository. Do not assume PhoenixMail's license or a competition submission license applies to this service.
