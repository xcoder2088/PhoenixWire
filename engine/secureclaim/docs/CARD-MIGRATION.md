# Apply the approved design to all PhoenixMail-generated cards

Visual reference: the user-approved working NimiqSecureClaim card, reduced desktop size at commit b3dd7e2, with approved PhoenixMail transparent emblem subsequently integrated.

**Current state:** The web claim preview uses the approved design; PhoenixMail outbound emails have not yet been migrated. This document is a migration specification, not a claim that they have changed.

## Separate responsibilities

- **NimiqSecureClaim** owns the shared, versioned visual design contract and secure claim flow. It may publish static assets and email-safe server-rendered templates.
- **PhoenixMail** owns composer UX, message delivery, generated email MIME, request creation, its five UI languages and current wallet controls. The web client may fetch public versioned presentation assets, but email HTML must contain its final styles inline, since email recipients cannot depend on third-party CSS, scripts or network calls.
- Do not transfer legacy Cashlink secret-bearing links or funded cashlink JSON to this new preview renderer. Secure claim emails will need a new safe, opaque claim URL after the backend is audited.

## Uniform design across card variants

1. **Send NIM / Payment to receive:** use approved pastel/glass design. Once secure claiming is implemented, include only an opaque claim URL with no spending secret/claim QR inside the email.
2. **Request NIM / Payment request:** same branding, padding, rounding and rotating pastel palettes. Keep beneficiary public address and payment QR/link visible. Payment request QR is safe to display.
3. **Claim verification web page:** use the approved responsive design, six-digit OTP and a **server-verified sender address Nimicon** when available. Current PhoenixMail emblem at center is only the approved visual placeholder.
4. **Future confirmation/receipt cards:** same visual vocabulary, explicit pending/confirmed status backed by blockchain verification; do not imply a payment was confirmed merely because a link was clicked.

## Compatibility and quality checks

- Shared twelve pastel palettes; stable selection by card ID so refresh does not reshuffle an existing sent card.
- On the web, glass/gradient effects may use CSS. In email, use a simple table-based, inline-styled fallback that looks polished in Gmail, Outlook and clients that do not support gradients or backdrop-filter.
- Maintain native PhoenixMail logo and proper official Nimiq identicons. Never invent recipient wallets, sender funding addresses or claim status.
- Preserve five-language translations, HTML sanitization, links with safe URL schemes, accessible contrast, mobile text sizing and dark/light email compatibility.
- Add screenshot regression tests comparing all cards to the approved reference and functional tests verifying unchanged network/address/amount/QR and payment validation.
- Migrate the shared design to PhoenixMail on a separate branch after tests; never silently break existing cashlink redemption.
