# Secure Cashlink — development lifecycle (Testnet-only, unfunded)

**Confirmed product direction:** NimiqSecureClaim is an OTP-gated escrow **wrapper around existing funded Nimiq Cashlinks**, not a replacement payment protocol. The eventual funded Cashlink bearer secret stays out of the email/QR and must remain encrypted server-side, only spendable after server-authorized OTP, wallet destination and on-chain confirmation checks. Simply revealing a Cashlink link after OTP remains a transferable bearer-secret and is weaker security. The current dev-only API does not accept, store or redeem actual Cashlinks.


## Stage completed

The service supports independent claim drafts **without moving or holding any NIM**:

1. An authenticated issuer (future PhoenixMail server integration) creates a claim for one immutable destination email, amount and Testnet network.
2. The claim receives a cryptographically random, 48-character opaque ID.
3. A public read-only endpoint returns the amount, Testnet, 12-palette index, current state and expiry **but not the recipient email**.
4. The recipient requests an OTP. It is delivered to the server-held destination email and is bound to the claim.
5. After successful OTP entry, the claim changes from \`draft_unfunded\` to \`email_verified_unfunded\`.
6. There is **no endpoint to confirm funding, sign a transaction, reveal a cashlink secret or redeem NIM**.

These drafts expire after 24 hours and are stored in memory in this development build. They will be lost on restart.

## Run locally, dev-only

\`\`\`bash
go test ./...
go vet ./...
NIMIQ_CLAIM_DEV=1 go run ./cmd/server
\`\`\`

By default the server binds to \`127.0.0.1:8099\` and serves the **approved existing card design**, using a demo claim. A simulated OTP appears in the **local terminal**, never a real email, unless MailerSend is configured.

## Create a new unfunded draft (local developer test)

Creation is disabled unless a secret environment variable of at least 32 characters is configured. Use a fresh random value:

\`\`\`bash
export NIMIQ_CLAIM_DEV_CREATE_KEY="$(openssl rand -hex 32)"
NIMIQ_CLAIM_DEV=1 go run ./cmd/server
\`\`\`

In a *different* terminal, using the same locally configured secret:

\`\`\`bash
curl -sS http://127.0.0.1:8099/api/v1/claims \\
  -H "Content-Type: application/json" \\
  -H "X-Claim-Service-Key: $NIMIQ_CLAIM_DEV_CREATE_KEY" \\
  -d '{"email":"recipient@example.com","amount":"5","network":"testnet"}'
\`\`\`

The JSON response includes a \`claim_path\`, e.g. \`/?claim=<opaque-id>\`, which selects the corresponding draft in the real claim UI. No Nimiq wallet address is needed for draft creation.

The create key **must not be placed in any frontend HTML, URL query parameter, browser storage, source code, chat screenshot, or repository**. Only a trusted server (not a browser) may act as issuer.

## Development routes

| Route | Access | Result |
|---|---|---|
| \`POST /api/v1/claims\` | Trusted local issuer key in \`X-Claim-Service-Key\` | Creates an unfunded Testnet draft |
| \`GET /api/v1/claims/{id}\` | Claim-ID holder | Public amount / network / palette / expiry / state |
| \`POST /api/v1/claims/{id}?action=request\` | Claim-ID holder, rate limited | Sends OTP only to registered recipient |
| \`POST /api/v1/claims/{id}?action=verify\` | Claim-ID holder with six-digit OTP | Marks email verified, **still unfunded** |
| \`POST ...?action=funded\`, \`redeem\`, \`wallet\` | Nobody | **Not implemented; 404** |

Use \`{"code":"123456"}\` as the JSON body for verify. The code is never in the public status endpoint.

## iPhone-friendly preview in ChatGPT

A phone **cannot reach another computer's** \`127.0.0.1\`: that address points at the phone itself. This repository does not expose a public preview server. Until a temporary authenticated HTTPS staging service is available, iPhone testing can use the **single-file offline interactive demo** provided as a chat attachment. It reproduces the card's visual system and simulates OTP locally; it does **not** call the Go service, send email, connect a wallet, or move money.

A real server-backed iPhone test will require deploying this Go server behind properly authenticated HTTPS on a staging domain, **after** implementing durable claim storage, anti-abuse and securing issuer access. Do not publish the current loopback development server directly.

## Before adding real NIM

There is no such thing as making an already public classic Cashlink safe merely by gating a web page with OTP. Classic Cashlinks embed a private spending seed; their QR/link remains a bearer secret.

Before funding/redeeming real claims, separately specify and audit:

- An explicit custody model, Nimiq Hub wallet integration, signing and transfer lifecycle.
- A funding transaction registered only from a trusted Nimiq blockchain confirmation; never from browser input.
- Atomic one-time claim authorization and spending, recipient-wallet ownership, expiry and sender refund/recovery.
- Durable transactional storage of encrypted secrets (if any), replay protection, server hardening, TLS, restricted service-to-service credentials and independent security review.
- Provider abuse prevention, OTP send throttles, privacy and legal/custody implications.

All of this is **future work**. A UI showing an amount does not mean a transfer has occurred.
