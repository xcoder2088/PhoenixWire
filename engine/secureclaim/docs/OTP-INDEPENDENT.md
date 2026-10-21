# Independent NimiqSecureClaim email OTP

## Why this module exists

PhoenixMail's existing account OTP routes verify a PhoenixMail **user login**, not ownership of the recipient address specified by a pending Nimiq claim. Calling those routes directly would wrongly couple access to funds to PhoenixMail accounts.

NimiqSecureClaim therefore owns its OTP logic under internal/otp. It reuses the established **concept** (cryptographic six-digit code, 10-minute expiry, limited retries, MailerSend delivery), without importing PhoenixMail account data, sessions or tokens.

## Current implementation

- A cryptographically random six-digit code is bound by keyed HMAC-SHA256 to the exact claim ID and its server-side recipient email. It is never stored as plain text.
- A process-unique key is generated at startup; it is not persisted. OTP state is stored in memory only.
- Each claim permits a new code no more than once per minute, at most five requests in fifteen minutes and at most five verification guesses per period. Previous codes are revoked on resend.
- A code becomes unusable when it expires or when it is used successfully.
- The claim recipient email is selected by the server. The browser cannot supply or change the recipient on the OTP request.
- The development service has **no payout endpoint**; successful OTP does not provide a Nimiq signing key or authorization to move funds.

## Test locally without sending email

Run from the repository root:

    go test ./...
    NIMIQ_CLAIM_DEV=1 go run ./cmd/server

Open http://127.0.0.1:8099. Click "Send verification code", then copy the temporary code from the server terminal. Only the explicit development console sender logs a code. No SMTP/MailerSend credentials are required for this demonstration.

## Test actual MailerSend delivery in the development-only service

Configure the existing PhoenixMail MailerSend variables in the server's private environment file, never in Git:

    MAILERSEND_API_KEY
    MAILERSEND_FROM

SecureClaim also accepts the dedicated aliases `NIMIQ_CLAIM_MAILERSEND_API_KEY` and `NIMIQ_CLAIM_MAILERSEND_FROM`; those take precedence if set. `NIMIQ_CLAIM_DEMO_EMAIL` is optional and is only used to show the unfunded local demo claim. Then run:

    NIMIQ_CLAIM_DEV=1 go run ./cmd/server

The server binds to 127.0.0.1 only. Funded OTP claims send a six-digit code to the recipient email stored on that claim. It refuses partial or malformed MailerSend configuration. The UI distinguishes actual email delivery from a console-only code, and PhoenixMail keeps funded OTP disabled in console-only mode.

**Do not expose the development service to the internet or enable real NIM transfers.** Its demo claim is not backed by money, request state is not durable, and it has not been through a security audit.

## Requirements before production

1. Store claim and OTP state in transactional shared durable storage, with atomic attempt throttles, consumed-state transitions and an expiry cleanup.
2. Tie creation of a claim to an authenticated sender, a specific authorized recipient email, an immutable amount/network, and verified payment funding.
3. Remove ConsoleSender from deployed builds; configure MailerSend with secrets, delivery-rate monitoring and anti-abuse protection.
4. Keep email OTP separate from Nimiq Hub wallet ownership verification and the one-time **actual fund release** protocol. Validate custody/refund requirements before implementing wallet claims.
5. Ensure outgoing messages, diagnostic logs, tracing and client HTML cannot reveal OTPs, claim secrets or private keys.
6. Do not reuse PhoenixMail session or /api/auth/verify routes.
7. Replace reliance on a single public RPC node (`rpc.nimiqwatch.com`) for `ConfirmedSpendable` with a trusted own node or an independently verified quorum before production. Current funding checks trust the selected node's answers; they do not independently verify consensus.
8. Review transaction-history pagination if Cashlink addresses are ever reused: `getTransactionsByAddress` currently scans only the latest 50 transactions. Addresses are single-use today.
