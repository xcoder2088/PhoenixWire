# USDT cashlink service (`internal/usdtlink`)

> Wire integration: `/` now hosts PhoenixMail's payment window via `wire-pay.js`; classic.html remains for NIM. Recipient API calls use link proofs/signatures without the private key. Wire's separate invitation route transiently receives the complete claim URL to validate and e-mail it without persistence (a retry can resubmit it). Thus a local signing key is not a guarantee that the operator never sees the invitation's link key. See `docs/PAYMENT-WINDOW.md` at the repository root.

Registers cards, follows them on the chain, gives the **co-signature** of protected cards, and has the **relayer** pay claims,
refunds and gas money. The protocol (Safe wallet per card, link key + co-signer, relayer) is the PhoenixCashlink repository; this
repository vendors it under `internal/cashlink` (`scripts/vendor.sh` there; never edit the mirror).

## Turn it on
`NIMIQ_CLAIM_USDTLINK_CHAINS=polygon-usdc,polygon,ethereum` (any subset; `polygon-usdc` = USDC on Polygon, `polygon` = USDT on Polygon, `ethereum` = USDT on Ethereum; default: off) and a state directory. On first start it creates, mode 0600, in
`<state>/usdtlink/`: `master.key` (co-signer: **back it up**, losing it blocks every protected card), `auth.key`, and
`relayer-<chain>.key`; it logs each relayer's PUBLIC address: send it a little POL / ETH (about 1 POL). The service refuses to start
on a mismatching master secret. `NIMIQ_CLAIM_ALLOWED_ORIGINS` (default `https://claim.phoenixmail.ca`) lists the page origins that
may write. Test networks are opt-in (`NIMIQ_CLAIM_TESTNET=1`), for a separate instance. Both Polygon assets share one relayer key and one nonce lock.

## Routes (`/api/v1/usdt/…`)
Recipient (cookie session `__Host-usdtlink`, allowed Origin on every write): `GET cards/{id}`, `POST sessions`,
`POST cards/{id}/email-checks`, `POST cards/{id}/email-checks/{cid}/code`, `POST email-checks/{cid}/verify`, `POST cards/{id}/authorizations`, `POST authorizations/{aid}/{grant|signature|claims}`.
Issuer (header `X-Claim-Service-Key`, PhoenixMail only): `POST issuer/cards`, `…/{id}/{status|cancel|gas|refund|refund-claim}`.

## Gas money for token-only senders (off by default)
Only with `NIMIQ_CLAIM_USDTLINK_GAS=on` (and `PHOENIXMAIL_USDT_GAS=1` in PhoenixMail). It can never cost the operator more than its **seed**: each confirmed payout (claim or refund) earns 10 gifts, each gift spends one, and the service starts with `NIMIQ_CLAIM_USDTLINK_GAS_SEED` (default 5). No credit left: no gift, whatever happens to cancelled or abandoned cards and whatever the exchange rate does. On top: once per card, once per address a day, 2 an hour and 10 a day overall, never below the relayer's float floor.

## Authorization of a claim (what stands between a link and a co-signature)
The e-mail code comes first, like any sign-in. **E-mail check** (per card and browser session, no destination; STATELESS until its proof is valid: its id is a MAC-protected derivation, nothing is stored for a stranger's request): the browser proves it
holds the link key with an EIP-712 signature (action 4, never sending the key) → only then the code is sent, to the card's own
recipient address → the right code marks the check done for ten minutes. **Claim attempt**: the recipient then chooses where to
receive; Prepare (destination + exact Safe transaction hash, referring to the live check of the same session) → the browser signs
the claim proof (action 1) → the service reserves the destination, **consumes the check** and issues a secret **grant** bound to the
session, the destination and that one transaction hash (5 minutes) → the co-signature is given only against the grant, after
re-reading funding and the Safe nonce on the chain. The check authorizes nothing by itself (one attempt, one session, ten minutes);
a global "verified" flag authorizes nothing.

## Truth sources and journals
A payout's transaction hash is written to disk **before** it is broadcast. State comes from the chain in this order: the receipt of
the journaled transaction, the Safe's nonce, then the balance; a zero balance never cancels a card. Refunds are full sweeps made for
the Safe's current nonce, so a surplus or a late deposit stays recoverable; one refund address per intent.

## Limits worth knowing
A co-signature already handed out stays valid on chain until the Safe nonce is used (a cancel stops new ones only; the first
transaction mined wins). Confirmation depth (the specification's 64 / 12 blocks) is not enforced yet. See the the maintainer reviews in the
PhoenixWire repository (`docs/REVIEW-USDTLINK-*`, `docs/PLAN-TESTS-*`) for the open findings.

## Tests
`go test ./...`, `node --test web/tests/usdt-page.test.cjs`.
