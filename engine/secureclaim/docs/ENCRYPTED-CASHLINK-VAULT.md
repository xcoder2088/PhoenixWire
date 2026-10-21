# Encrypted Cashlink secret vault — development milestone

## What is implemented

The `internal/vault` package provides authenticated **AES-256-GCM** encryption for a Nimiq Cashlink **spending secret**, not for a public recipient address or regular payment-request QR.

- 32-byte randomly generated master key, supplied by trusted backend configuration only. The master key is **never** written into the vault directory, claim record, email, browser or URL.
- HMAC-SHA-256 derives a separate wrapping key for each opaque 48-character claim ID. AES-GCM authenticates the claim ID, envelope format version and key identifier as associated data.
- Unique cryptographically random 96-bit nonce per encryption; never derived from the OTP or a predictable claim ID.
- Ciphertext-only JSON envelope containing `version`, `key_id`, `nonce` and `ciphertext`.
- Dedicated vault directory with `0700` permissions, `0600` files, atomic create-once publication using a same-filesystem hardlink (fails closed when hardlinks are unsupported). No overwriting a previous secret under the same claim ID.
- Wrong master key, wrong claim ID, incorrect key version, corrupted ciphertext, expired wrapping format and attempted overwrites fail closed.
- Unseal is a **Go internal method only**. No HTTP route gives callers plaintext or allows a Cashlink to be claimed, funded or marked as funded.

These are **standalone building blocks**, not yet wired into the SecureClaim server. The only currently exposed API remains **unfunded Testnet draft creation and email OTP verification**.

## Why separate the encryption key?

Someone stealing only `*.sealed.json` cannot normally redeem the NIM without the separately stored master key. However, a compromise of the running service (or its key material) may permit decryption. The encrypted file is **not** noncustodial and does not itself enforce OTP: the settlement service will have to perform authorization and one-time on-chain spending before calling the internal unseal method.

Never use a six-digit OTP as the AES key. An OTP only authorizes a server-verified step.

For production, obtain the master key from a dedicated secret manager or platform-protected secret file / KMS. Generate `32` random bytes using an approved CSPRNG. Avoid putting it in a build artifact, checked-in .env, command-line arguments, analytics, chat or log. Rotation needs a registry keyed by `key_id`, a carefully controlled rewrapping process, and backups *encrypted separately* from key material.

## Intended secure claim lifecycle

1. PhoenixMail creates a recipient-bound *draft* through its authenticated server-to-server API, not from a browser-held issuer key.
2. The sender generates and wallet-funds a conventional **Nimiq Cashlink**, holding the signing seed on the sender's device at this point.
3. A future authenticated server-to-server escrow admission operation securely accepts the Cashlink secret, binds the funding address and immutable claim, encrypts the seed into the vault, then independently checks Nimiq Testnet blockchain funding. **Not implemented yet.**
4. The outgoing email contains only the claim URL with an opaque ID. Never include the actual Cashlink spending QR/link, key or private seed.
5. Recipient receives the OTP at the immutable email address, enters it on the landing page, and supplies a receiving public wallet address.
6. A future settlement controller checks the verified OTP state, funding finality, expiry/refund rights and atomic one-time claim status; it unseals and uses the Cashlink spending seed **internally** to transfer to the authorized wallet. Never return the raw Cashlink link/seed to the browser. **Not implemented yet.**
7. After the actual on-chain transfer is confirmed, settle the claim exactly once and schedule vault-record destruction.

## What must be completed before real NIM are involved

- Durable transactional claim database and funding/claim state machine with one-time locks and idempotency.
- Sender authentication and a defensible custody/trust policy.
- Verified Nimiq Cashlink generation/address derivation and chain funding checks with trusted RPC/node.
- Secure seed admission tied to the original sender, exact claim ID, blockchain network and amount.
- Restricted audited settlement worker and destination-wallet binding; explicit refund/cancel behavior.
- Master key provisioning, backup, rotation and disaster recovery; no key stored next to ciphertext.
- Crash/restart, double-claim, malformed seed, concurrent settlements, rate limiting and red-team tests.
- Separate legal/custody and security review **before mainnet**.

## Developer checks

```bash
go test ./internal/vault -count=1
go test -race ./internal/vault
go vet ./internal/vault
```

The test suite verifies round trips, nonce uniqueness, confidentiality of stored file bytes, private permissions, create-once behavior, claim-ID binding, tampering and concurrency. No real Cashlink should be submitted to these tests.

This repository does **not** expose a public payment service or secret upload endpoint. That is deliberate.
