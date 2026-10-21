# SecureClaim source import

Imported at the owner's explicit request on 2026-10-21.

- Source: `xcoder2088/NimiqSecureClaim`
- Branch: `work/otp-existing-cashlink-reliability`
- Commit: `a344dcbc466ac2c6dbda776ff2849f9c3e75350c`
- Destination: `engine/secureclaim/`
- Scope: all 54 tracked source, test, documentation, branding and deployment-template
  files, byte-for-byte. Excluded: the two `.github/workflows/*.yml` files.
- No `.git` directory, local environment, vault, encrypted state, payment data,
  credential or original repository history is imported.

The imported module keeps its existing Go module path and internal imports to make
this initial copy verifiable without changing behavior. It builds independently from
its directory. A module path migration can be a separate reviewed change.

## Licensing

The owner explicitly requested MIT licensing for this copy. `engine/secureclaim/LICENSE`
contains the owner's MIT grant for original code, matching the PhoenixWire root grant.
The Go Authors BSD license in `internal/nimiq/blake2b/LICENSE` and Nimiq IQons license
and notices under `web/` are retained unchanged. MIT does not replace these terms,
grant trademark rights or relicense the rest of PhoenixMail.

## Important behavior preserved

This is a source import, not a payment integration or production approval. The source
branch already contains Mainnet-specific support. Nothing in this import enables it
in the Phoenix Wire UI, starts the service, funds a wallet or copies live state.

Historical Testnet notes and an old deployment template are preserved as source
history/context, not active configuration. There is no Testnet selector in Phoenix Wire.
The upstream README's current integration section takes precedence over its clearly
labeled historical notes. Do not run deployment templates without review.

The OTP-before-vault-decrypt audit fix is included. Remaining known constraints include
trust in one public Mainnet RPC, a 50-transaction scan window, sender recovery rather
than automatic refunds, and lack of independently validated real email/wallet E2E.
Keep these visible when designing the future shared API.

## Follow-up

Validate the existing engine before connecting Wire's sender form. Establish one
canonical update path so PhoenixMail and Wire do not accumulate diverging security
fixes. No changes have been made to the original repository's code, visibility, license,
or access settings. Making any repository private later cannot revoke MIT rights
already granted for copies distributed under that license.
