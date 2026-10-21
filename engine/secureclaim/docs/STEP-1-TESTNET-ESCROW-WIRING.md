# Step 1 — Testnet funding and encrypted vault wiring (development branch)

## What is implemented

1. **Independent Cashlink seed/address derivation** in `internal/nimiq/address.go`, using Ed25519 and BLAKE2b-256, with a deterministic test vector. BLAKE2b implementation is vendored from PhoenixMail with its original BSD license.
2. **Testnet funding verifier** in `internal/nimiq/testnet_rpc.go`. It queries a configured JSON-RPC node and refuses admission unless the node identifies as `TestAlbatross`, the Cashlink address has sufficient current balance, and at least one successful confirmed funding transaction contains the Cashlink marker. The RPC is a trusted external data source; this is **not an independent full-node consensus verification**.
3. **Internal admission controller** in `internal/escrow/admission.go` checks claim network, amount, 32-byte seed/address correspondence, confirmed funding, and single-write encrypted vault publication. Its inputs must come from an authenticated internal caller, never an unauthenticated browser.
4. **Internal wiring** in `internal/escrow/wiring.go` connects the independent verifier and deriver to the AES-256-GCM vault. No seed ingestion, payout or refund HTTP endpoint exists.
5. **Opt-in persistent draft metadata** in `internal/claim/persistent.go`, with private 0700 directory, 0600 JSON file, atomic replacement, reload validation, and failure rollback on create/email-verification. **OTP challenge state is still in memory**.
6. **OTP dispatch response metric** `dispatch_response_ms` measures MailerSend acceptance latency, not recipient inbox delivery.

## Important: the correct pre-funding sequence

SecureClaim **must not** first create a classic Cashlink URL in the browser and then try to hide its seed afterward. A classic Cashlink URL is already a bearer secret.

The new `internal/escrow/provision.go` implements the safe internal creation order:

1. The trusted server creates a 32-byte Cashlink seed for an immutable Testnet claim.
2. It seals the seed immediately in the private vault **before any funds are sent**.
3. It derives and returns **only the public Nimiq funding address**.
4. The sender funds that address with their Nimiq wallet.
5. `FundingConfirmed` checks the current chain through the Testnet verifier, without exposing the seed.
6. Later, a separate durable OTP-authorized settlement flow will spend from that address exactly once.

The internal provisioner is idempotent across retries and can reconstruct the same address from the vault after a restart. The existing `Admission.Admit` is a separate legacy/import-style guard and must not be mistaken for the secure creation flow. **Neither path is exposed through HTTP yet.**

## Opt-in local development configuration (never publish actual secrets)

The server is still gated behind `NIMIQ_CLAIM_DEV=1` and bound to loopback. All of the following are required together to initialize escrow admission:

- `NIMIQ_CLAIM_STATE_DIR`: absolute private directory for claim metadata
- `NIMIQ_CLAIM_TESTNET_RPC_URL`: explicitly trusted Testnet Nimiq Albatross RPC URL (HTTPS; loopback HTTP allowed)
- `NIMIQ_CLAIM_VAULT_DIR`: separate absolute private ciphertext-only directory
- `NIMIQ_CLAIM_MASTER_HEX`: 64-character hex representation of a randomly generated 32-byte master key, supplied outside Git
- `NIMIQ_CLAIM_KEY_ID`: key-rotation label

These variables **only initialize internal components**. No user can fund or redeem SecureClaim through this prototype. The service will refuse startup if the escrow configuration is partial or if the durable claim directory is missing.

## Explicit security blockers before Step 2 / any real-money testing

- Verify exact Nimiq JSON-RPC method and response shapes with a real Testnet node. Unit tests use mocked RPC responses; the service has not been connected to a real node in this branch.
- Add authenticated sender-only creation/funding handoff, ensuring Cashlink seed NEVER appears in email, public URL, QR, browser storage, logs, analytics, or a public API response. The legacy PhoenixMail Cashlink flow currently returns the seed-bearing Cashlink link to the browser; **do not reuse it as SecureClaim**.
- Design a transactional **funding/admission state machine** that is recoverable across process crashes and restarts. Current admission writes ciphertext, but does not mark a claim funded or authorize a payout. A vault entry alone is not a funded state.
- Move OTP challenge/verification state to durable shared storage with atomic limits; OTP proof must authorize only the designated recipient and a bound destination wallet.
- Verify settlement, transaction broadcast, idempotency, fees, expiry and refund semantics. Payout must never be possible twice, even across two processes or after a crash.
- Claim snapshot storage is currently single-process and local-filesystem only, not suitable for multi-instance production. Keep the master key in a real secret manager or other protected service configuration, not in source control or backups alongside ciphertext.
- Run Go tests, race tests and an end-to-end Testnet scenario before deploying. No live blockchain transfer has been performed or claimed.

## Branch and review

This code is in the NimiqSecureClaim encrypted-vault development PR. PhoenixMail's existing classic Cashlinks remain unchanged. **Do not deploy or enable mainnet based on this code.**
