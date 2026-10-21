# Foundation and integration boundaries

## Decisions

- Phoenix Wire is payment-only, not another email client. No IMAP/Gmail OAuth.
- Web-first UI, reusable later by a desktop wrapper.
- Shared PhoenixMail identity; permanent “Powered by Phoenix Mail” footer.
- Preview runs independently on localhost:8788; PhoenixMail remains on 8787.
- SecureClaim now lives in `engine/secureclaim`, imported at the owner's request.
  Preserve a single engine within this repository and expose a versioned server-side
  API to Phoenix Wire and, later, PhoenixMail. Do not maintain diverging payment implementations.
- Keep the standalone app MIT without changing PhoenixMail's existing licensing.

## Intended flow (NOT implemented)

1. Verify sender identity; typed email alone is not proof of ownership.
2. Validate recipient, asset/network, amount in integer base units and expiry server-side.
3. Persist an idempotent intent and recovery record before opening the funding wallet.
4. Confirm funding for that exact intent, amount and network independently of the browser.
5. Email an opaque claim link through server-side MailerSend integration.
6. Request and verify recipient OTP server-side before opening encrypted spending material.
7. Release via the existing Nimiq claim engine; provide mobile wallet handoff and recovery.

Never send a spendable QR, seed, or unprotected link before successful verification.
No fallback from protected to unprotected payments. No duplicate funding on retries.
Expiration, attempt/resend limits, encryption, persistence, restore, fund recovery and
concurrency tests are requirements, not optional follow-up polish.

## Future modules

- Payment service: validates intent state transitions and idempotency.
- Nimiq adapter: the imported SecureClaim module's APIs, validated before UI connection.
- Delivery adapter: PhoenixMail MailerSend configuration server-side only. No fake sends.
- EVM adapter: separate design/review for USDT Ethereum and USDT0 Polygon;
  ordinary token transfer to a public QR address is NOT an OTP-protected claim.
  Determine escrow/recovery, token contract/network, approvals, gas and key custody first.
- Recipient claim UI: separate from sender preview; no secret-bearing content pre-OTP.

All capabilities remain disabled in /api/config, and POST /api/payments returns 503.
The preview does not load service credentials, seed material or financial databases.
It makes no third-party requests. Only display preferences use localStorage.
The imported service remains independent: the preview never launches it. See
SECURECLAIM-IMPORT.md for exact provenance, preserved behavior and review requirements.

## Release gates

No live integration until existing PhoenixMail/SecureClaim OTP review is complete.
Require a threat model, durable state/recovery drills, independent review, real delivery
validation and explicit authorization before financial deployment. No automatic mainnet
activation. No assumption that a themed preview means a supported payment rail.

## Work conventions

Use work branches, [skip ci] commits, no PR/main merge, no workflows or deployment
without owner approval. Do not log/commit vault keys, issuer keys, API keys or emails.
