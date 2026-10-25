# What Phoenix Wire keeps, and who else sees it (facts for the terms and the privacy page)

This is a factual inventory of the software as it is deployed, written to feed the public terms and privacy pages. It is **not legal advice**: have the
pages reviewed before relying on them. Every sentence here can be checked in the code (paths in brackets). If the software changes, change this file.

## What Phoenix Wire is, in one paragraph

A sender confirms an e-mail address with a six-digit code, funds a card (NIM, USDC on Polygon, USDT on Polygon or Ethereum), and Wire e-mails the recipient a
link. The recipient confirms their own e-mail with another code, then receives the money at an address they choose (or opens the usual Nimiq link for
NIM). Wire does not hold anyone's wallet: a card is its own address on the chain. A card lives 24 hours; at the end an unopened NIM card is paid back
automatically to the address that funded it, and the sender of a USDC/USDT card takes the money back himself. A refund keeps the relayer's fee.

## Data Wire keeps (one encrypted file, AES-256-GCM, owner-only permissions) [`internal/wire/store.go`]

| Data | Why | How long |
|---|---|---|
| Sender e-mail address; session (random token, only its hash is kept) | to let the sender use the window | a confirmation code lives 15 minutes; a signed-in session 24 hours; sessions are deleted when they expire or the sender is blocked |
| Each NIM payment: sender and recipient e-mail, amount, optional message (160 characters), language, the card's public address, stage, dates, a small decorative image (Nimicon) | to run the card, show "Mes cartes", send the e-mail, refund | **deleted 60 days after the card ends** (24 hours after it was made, plus 60 days), if the card holds nothing; unpaid drafts are deleted after 2 days, cancelled cards after 14 days |
| Each USDC/USDT card: sender and recipient e-mail, amount, message, public card address, a refund token held by Wire, dates | same | **the e-mail addresses and the message are erased 60 days after the card ends**, if the card holds nothing (its id, amounts and public address stay: they are on the chain) |
| Rate-limit counters: a **hash** of the visitor's IP address with a secret, and a count | to stop abuse | about 1 hour |
| Administrator lists (allowed, blocked addresses) | to run invitation-only access | until an administrator removes them |
| Which reminders and notices were sent for a card | to send each once | with the card |

Separately, in the engine's own files (also private and encrypted where they hold secrets) [`engine/secureclaim`]:
- the **private key of each NIM card**, sealed with the master key (needed to pay the recipient or to refund);
- the co-signing and relayer keys for USDC/USDT, derived from the master key;
- one journal file per NIM payout or refund (destination address, signed transaction, hash; no key);
- the recipient's e-mail address inside the NIM claim, **erased with the card's data after the same 60 days** (and the USDC/USDT card's addresses and message, as above);
- hashed one-time codes with their expiry;
- the **margin ledger**: one line per settled USDC/USDT payout (first 8 characters of the card id, amounts, gas paid). No e-mail, no address.

Wire does **not** keep: passwords (there are none), the USDC/USDT link key (it stays in the sender's browser and in the recipient's link; Wire sees the
claim link once, to e-mail it, and does not store it), IP addresses in clear text, advertising identifiers, or any analytics. Mail tracking (opens,
clicks) is switched off in every message [`internal/wire/mail.go`].

## Who else sees what

| Party | What it sees | Why |
|---|---|---|
| **MailerSend** (e-mail delivery) | sender and recipient e-mail addresses, the message, the card amount, the link, the codes | to deliver the e-mails |
| **Cloudflare** (public tunnel in front of the site) | the visitors' IP addresses and requests, as for any site behind Cloudflare | to reach the server; Wire reads the visitor address it forwards (`CF-Connecting-IP`) only to rate-limit, as a hash |
| **Blockchain nodes** (public RPC endpoints, or an own node) | the card and relayer addresses Wire asks about | to read balances, funding, gas price |
| **The blockchains themselves** | every payment, public and permanent | by nature of NIM, Polygon and Ethereum |
| **Price sources** (Chainlink feed on chain; CoinGecko only as a fallback) | the coin identifier only | to price the gas in the fee |
| **Nimiq Hub / Nimiq Pay / the recipient's wallet** | what the recipient does with them | to receive NIM; their own terms apply |

## What a user can ask, and the honest limits

- **Delete my data:** Wire erases a card's e-mail addresses and message by itself 60 days after the card ends, if it holds nothing; it can do it sooner on request
  for a card that is over. A card that still holds money keeps its data (its sender may need to recover it). It cannot remove anything from
  the blockchain, or from MailerSend's delivery logs (their retention applies), or from Cloudflare.
- **A card's money is on-chain:** nobody, including Wire, can reverse a payment that was claimed.
- **Refunds:** an unopened NIM card goes back to its funding address; a USDC/USDT card is taken back by its sender with his browser's key. The relayer keeps
  its fee (the gas of the card plus 6 % of the amount) in every refund. If the money came from an exchange or a custodial wallet, the refund goes to that
  address.
- **Availability and risk:** the software is provided as it is, with no warranty (MIT). It moves real money; it has had a code review, not a security
  certification ([AUDIT-2026-10-24.md](AUDIT-2026-10-24.md)). Prices and gas come from third-party data.
- **Fees:** USDC/USDT: the gas of the card at the moment it is created (with a margin) plus 6 % of the amount; NIM: none beyond the network. The sender sees one
  amount to pay. The margin helps absorb changes in network costs. A refund retains the full relay fee, including that margin and the 6 % service fee.
- **Access:** sending is by invitation unless an administrator opens it; opening or closing it never changes a card already sent.

## What stays, whatever the retention

- The **card's key** (NIM) stays sealed in the vault, and the journal of a payout or refund (a destination address, a signed transaction, a hash; no e-mail):
  a payment someone sends to a card's address later can still be recovered, and a payout can still be explained.
- The **margin ledger** lines (no e-mail, no address) and the chain's own records.
- Backups: an encrypted backup of the state keeps what it held on the day it was made, until that backup is deleted.

## Gaps to decide before promising anything in the pages

1. **Contact for requests.** The owner has selected the public contact address `phoenixmails@proton.me`.
2. **Who is the operator / which law.** The pages must name the operator and the governing law; that is the owner's decision.
3. **Minors, and prohibited uses.** Decide whether to state an age limit and what may not be sent.
