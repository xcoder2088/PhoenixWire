# PhoenixWire hosting handoff

Hosting configuration and owner-run checklist. This document does not certify the state of a deployed instance.
The current sender page hosts PhoenixMail’s payment window; see PAYMENT-WINDOW.md.
Use **127.0.0.1:8790** for Wire. The laptop's `phoenixmail-web` already uses
8788; neither that service nor the claim service on 8099 should be changed.

## Existing Cloudflare tunnel

Public Wire origin: `https://wire.phoenixmail.ca`. Set `PHOENIXWIRE_PUBLIC_URL`
to that exact origin when hosting. Insert `deploy/cloudflared-ingress.example.yml`
before the existing tunnel's catch-all, preserving all existing ingress entries.

The owner's DNS record is **CNAME**, name **wire**, target
**<existing-tunnel-UUID>.cfargotunnel.com**, proxied, TTL Auto. Obtain the UUID
from the existing `cloudflared-phoenixmail` configuration's `tunnel:` field;
it is not a credential. The UUID is unavailable in this checkout, so the exact
target must be read on the laptop; do not substitute a guessed identifier.
Cloudflare's equivalent command, with that existing UUID, is:

```sh
cloudflared tunnel route dns "$TUNNEL_ID" wire.phoenixmail.ca
```

## Independent service and backup

`deploy/phoenixwire.service` is a reviewable systemd unit, not an installer.
It expects a dedicated `phoenixwire` user, binary `/opt/phoenixwire/phoenixwire`
and a real private environment file `/etc/phoenixwire/local.env` (0600).
Use actual MailerSend configuration; no fabricated credentials or console delivery.

Wire alone owns `/etc/phoenixwire/master.key` and `/var/lib/phoenixwire/`.
Never use `/etc/nimiqsecureclaim/*` or `/var/lib/nimiqsecureclaim/*` for Wire.
Startup rejects those locations, including symlink aliases and new child paths.
The original Wire master key is generated once; existing state blocks regeneration.
An upgrade must retain that key and the complete engine/gateway state together.

Before any funded live test, make an owner-controlled encrypted backup containing
the original master key, private environment file and complete state. Give any
temporary local backup mode 0600, move it off the laptop, then remove the temporary
copy. Do not print keys into a terminal or put backups in Git. An environment-file
backup alone is insufficient: Wire's master key is a separate binary file.

## Owner-run acceptance — NIM and USDC / USDT separately

Automated tests do not demonstrate real delivery or receipt. After the owner has
reviewed and enabled hosting, record each step without logging codes or secrets:

1. Sender receives and enters their sign-in email code.
2. Select NIM in the common `/` payment window and use an owner-approved small amount. The same window offers configured
   USDC Polygon, USDT Polygon and USDT Ethereum; run each asset’s acceptance separately. Keep classic.html and NIM claim
   links in the regression checks.
3. Funding is confirmed, then the recipient receives the branded invitation.
4. Check the card and footer. NIM uses an opaque invitation URL; USDC/USDT use a protected `#c=…` link. The sender submits
   that full link to Wire for validation and email delivery without persistence; recipient claim calls use proofs/signatures.
5. Recipient requests a code; wrong/incomplete codes reveal no wallet link.
6. Correct code turns the same button into “Receive my NIM in Nimiq”.
7. Recipient accepts the payment; confirm actual wallet receipt.
8. On iPhone, verify redirect/return restores the original payment and never
   starts a second checkout. Check sender history after restarting Wire.

Wire persists sessions: restart preserves valid sessions, codes, counters and
pending payments. PhoenixMail's in-memory web sessions behave differently;
restarting PhoenixMail can log its users out. Restart only the intended service.

For USDC/USDT, also check quote/hash agreement, confirmation depth, interrupted deployment/payout, sender refunds, remaining
relay gas and recovery data on the sender’s device. A successful NIM check is not acceptance of either EVM chain.
The page explains the 24-hour sender session and includes tab/home-screen icons; verify these on desktop and phone.

The live check needs access to the serving laptop, real configured mail and the
owner's wallet approval. None is assumed available to this coding environment.
