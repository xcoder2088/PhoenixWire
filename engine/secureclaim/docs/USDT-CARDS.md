# USDT cards (Ethereum and Polygon)

**No money, no key, no gas here.** The service only remembers a card and watches the chain. Decided with the owner on 2026-10-22: PhoenixMail never advances a fee and never charges one; the sender pays the network fee of his own transfer, exactly as in any wallet.

## Flow
1. **Create** (issuer only, `X-Claim-Service-Key`): `POST /api/v1/evm/cards` `{asset, amount, protected, email?, message?, sender_address?, sender_email?, sender_lang?}` with `asset` = `usdt_ethereum` | `usdt_polygon`. Answer: the card ID and a **sender token** (shown once, kept by the sender's device; only its SHA-256 is stored).
2. **Recipient** opens `claim.phoenixmail.ca/?claim=<id>`. Protected card: `?action=request` mails a 6-digit code (20 s resend, 8 requests / 5 guesses per 15 min), `?action=release {code}` verifies it. Then `?action=address {to}` gives **his own address**. It is final (the sender's QR is built from it). The page shows the address back for a second confirmation before sending it.
3. **Sender is told** (email, in his language) and sees the recipient's address in `?action=status` (issuer). He makes ONE USDT transfer straight to it from his wallet; the wallet adds the fee.
4. **Payment seen**: from the block the address was given, the service reads the USDT `Transfer` logs to the recipient (and from `sender_address` when it was given). A candidate ≥ amount counts only after the **receipt is confirmed on two independent nodes** (status success, a Transfer log to that address ≥ amount) and the block is deep enough (Ethereum 12, Polygon 64). Card state → `paid`, with the transaction.
5. **Cancel** (issuer + sender token): `?action=cancel` while unpaid.

## Rules
- Amount 0.01 – 1,000 USDT. Overpayment is fine.
- A week to give an address, then a week to pay (`expired`). Money already sent goes to the recipient's own address: nothing is ever stuck.
- Open (non-protected) card: whoever holds the link chooses the destination — same trade-off as a Nimiq cashlink. Protected: only after the emailed code.
- The sender's own address is refused as destination.
- Public view never shows email, address or token. The issuer sees the recipient's address once known.
- `NIMIQ_CLAIM_EVM=off` disables it. RPC lists: `NIMIQ_CLAIM_EVM_RPC_ETHEREUM` / `_POLYGON` (comma-separated https URLs). Only a few public nodes serve `eth_getLogs` (publicnode on both chains, checked 2026-10-22 from the production laptop); receipts work on many, which is why the log query uses one node and the confirmation uses two. Some home-network providers silently block a few node hostnames.
- Notification email: `NIMIQ_CLAIM_APP_URL` (default https://mail.phoenixmail.ca).

## Code
`internal/evm` (Keccak-256, EIP-55, networks, JSON-RPC: `Transfers`, `Confirm`), `internal/evmcard` (state machine, persistence in `<state>/evm/cards.json`, watcher), `internal/evmtest` (fake node), `cmd/server/evm_http.go`, `internal/otp/notice.go` (the sender's email), `web/index.html` (claim page). Go standard library only.
