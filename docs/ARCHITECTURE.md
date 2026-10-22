# Phoenix Wire integration

One Go process runs the embedded UI, Wire gateway and imported SecureClaim engine.
The legacy engine CLI delegates to the same exported service package.

## Passwordless sender

Email OTP verifies the sender without registration or password. The engine's durable
OTP manager is reused with separate keys/state for sender and recipient codes.
Per-email counters prevent cookie changes from resetting resend/guess limits.
Per-IP request limits are persisted. Verification rotates an HttpOnly cookie.
Verified sessions last 24 hours; pending sessions 15 minutes. Mutations require the
configured same Origin; the gateway enables no CORS.

## Payment order

1. Persist an immutable recipient/amount/idempotency intent for the verified sender.
2. Call the existing CreateOnce and Provisioner: generate/seal the secret server-side.
3. Persist funding_started before returning the public wallet checkout request.
4. Official Hub checkout uses extraData to encode the Cashlink marker that the
   existing verifier checks in the transaction's recipientData.
5. Confirm funding through the existing RPC verifier.
6. Persist funded/delivery_pending, send the opaque claim URL, persist sent.
7. Gate public actions on confirmed durable state; delegate recipient OTP
   request/release to the existing handler.

Public routing never exposes issuer creation, sealing, legacy demo or vault access.
Gateway snapshots use AES-GCM, atomic publication and fsync. The engine keeps its
existing durable claims, encrypted OTP counters and vault. Private directories and
process locks prevent shared writers; persistence failure disables further actions.

## Recovery

Idempotency binds the verified sender to one immutable payment. Started checkout
retries check the same address without another checkout. Sender re-verification
restores owner history, including the original Cashlink recovery link. Recovery is
manual and only possible while funds remain unspent. Email resends reuse the same
claim with persisted cooldown/count limits.

## Boundaries

Unconfigured startup remains a card preview. Only NIM sending is implemented.
USDT Ethereum / USDT0 Polygon require their own escrow, token, gas and recovery
design. No production deployment or repository visibility change is included.
See LOCAL-TESTING.md for constraints and manual validation.
