# Local integration and review

## Startup

Run ./run-local.sh in one terminal. It runs the UI, gateway and SecureClaim engine
in one process on 127.0.0.1:8788. The legacy engine CLI is not needed alongside Wire.
Never launch two processes over the same state.

The optional environment file is ~/.config/phoenixwire/local.env, owner-only 0600.
Follow .env.example and supply the real MailerSend credentials, a trusted Mainnet
RPC and the exact Wire public origin. PHOENIXWIRE_ENABLE_PAYMENTS=1 explicitly
enables the integration. Missing configuration stops startup; credentials are not logged.

The master key is generated once at ~/.config/phoenixwire/master.key (0600).
State lives under ~/.local/share/phoenixwire/state/, in private engine/gateway
directories with separate derived keys. Back up the original key together with
all state in an owner-controlled encrypted backup. Never delete either while a
payment may be funded. Existing state blocks key regeneration. Wrong keys and
concurrent writers fail closed. Previous PhoenixMail/SecureClaim state is not moved.

## Manual validation

1. Verify real sender-code delivery and enter the code.
2. Enter recipient, NIM amount (maximum 1,000) and optional note.
3. Approve the exact funding address/value in the official Nimiq Hub.
4. After confirmation, verify receipt of the pastel card email.
5. Open its opaque link: no claimable QR or wallet bearer link should appear.
6. Request the recipient code. Invalid/incomplete codes must not expose a wallet link.
7. Enter a valid code and use Receive in my wallet; verify wallet receipt.
8. Check sender recovery with an unclaimed payment and confirm it cannot be spent twice.

Use an explicitly approved small amount for the first live check. Automated tests
use local fixtures and do not prove that real email or funds have been exercised.

## Interruptions

Once checkout starts, refresh checks the same address without opening a second
checkout, including after a cancelled, blocked or ambiguous wallet result. This
build deliberately requires the existing pending record to be resolved manually
rather than offer another automatic funding attempt.

Email failure retains the funded record. Explicit resend uses the same opaque
claim, with a one-minute cooldown and maximum five delivery attempts. Re-verifying
the sender email restores history on another device. Recovery opens the original
Cashlink; it is not an automatic refund and only works while funds are unspent.

OTP codes expire after 10 minutes with five guesses and resend limits. Recipient
claims expire after 24 hours; the authenticated sender recovery path remains separate.

## Remote testing and remaining constraints

A localhost link works only on the serving computer. Phone testing outside the
network requires a reviewed HTTPS public origin routed to this server. No tunnel,
DNS change or deployment is performed here.

The existing verifier trusts one configured RPC and scans 50 recent transactions.
Use a trusted-node / independent-verification plan before production. Automatic
refunds, EVM escrow and production-scale abuse protection are not implemented.
Do not trust forwarded IP headers without a reviewed reverse-proxy configuration.
The embedded durable engine currently requires Linux filesystem locks.

The UI supports five languages. Delivery templates currently use English. Full
recipient-language email templates and live desktop/mobile browser QA remain.
