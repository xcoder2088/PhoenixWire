#!/usr/bin/env bash
# Mirror the shared files of NimiqSecureClaim `main` into engine/secureclaim/ and run the tests.
# engine/secureclaim is a MIRROR of NimiqSecureClaim, not a fork: fix things there first, then run this script.
set -euo pipefail
here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
src="${NIMIQ_SECURECLAIM_REPO:-$(dirname -- "$here")/NimiqSecureClaim}"
[[ -f "$src/go.mod" ]] || { echo "NimiqSecureClaim checkout not found: $src (set NIMIQ_SECURECLAIM_REPO)" >&2; exit 1; }
branch="$(git -C "$src" branch --show-current)"
[[ "$branch" == "main" ]] || { echo "NimiqSecureClaim must be on main (is on '$branch')" >&2; exit 1; }
git -C "$src" diff --quiet && git -C "$src" diff --cached --quiet || { echo "NimiqSecureClaim has uncommitted changes" >&2; exit 1; }
files=(internal/otp/mailer.go internal/otp/mailer_test.go internal/otp/challenge.go internal/otp/challenge_test.go
       internal/otp/mail_language.go internal/otp/mail_language_test.go
       internal/otp/assets/phoenixmail-email-icon.png web/index.html web/tests/claim.test.cjs)
for f in "${files[@]}"; do mkdir -p "$here/engine/secureclaim/$(dirname "$f")"; cp "$src/$f" "$here/engine/secureclaim/$f"; done
echo "Mirrored NimiqSecureClaim $(git -C "$src" rev-parse --short HEAD) into engine/secureclaim"
echo "Record the commit in docs/SECURECLAIM-IMPORT.md (Sync log), then commit with [skip ci]."
(cd "$here/engine/secureclaim" && go vet ./... && go test ./... -count=1)
(cd "$here" && go vet ./... && go test ./... -count=1)
echo "Go tests passed. JS: node --test tests/*.test.mjs engine/secureclaim/web/tests/*.test.cjs (needs jsdom)."
