# Phoenix Wire hosting handoff

Prepared files only: no remote installation or DNS change was performed.
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

## Live acceptance — pending, exactly 1 NIM

Automated tests do not demonstrate real delivery or receipt. After the owner has
reviewed and enabled hosting, record each step without logging codes or secrets:

1. Sender receives and enters their sign-in email code.
2. Sender creates a **1 NIM** protected card and approves one wallet funding.
3. Funding is confirmed, then the recipient receives the branded invitation.
4. The invitation has no redeemable QR or bearer link. Its card and footer render.
5. Recipient requests a code; wrong/incomplete codes reveal no wallet link.
6. Correct code turns the same button into “Receive my NIM in Nimiq”.
7. Recipient accepts the payment; confirm actual wallet receipt.
8. On iPhone, verify redirect/return restores the original payment and never
   starts a second checkout. Check sender history after restarting Wire.

Wire persists sessions: restart preserves valid sessions, codes, counters and
pending payments. PhoenixMail's in-memory web sessions behave differently;
restarting PhoenixMail can log its users out. Restart only the intended service.

The live check needs access to the serving laptop, real configured mail and the
owner's wallet approval. None is assumed available to this coding environment.
