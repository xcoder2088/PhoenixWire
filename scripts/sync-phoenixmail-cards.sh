#!/usr/bin/env bash
# Vendor only the approved presentation layer; keep application adapters separate.
set -euo pipefail
here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
src="${PHOENIXMAIL_REPO:-$(dirname -- "$here")/PhoenixMail}"
[[ "$(git -C "$src" branch --show-current)" == main ]] || { echo 'PhoenixMail must be on main' >&2; exit 1; }
git -C "$src" diff --quiet && git -C "$src" diff --cached --quiet || { echo 'PhoenixMail has uncommitted changes' >&2; exit 1; }
python3 "$here/scripts/vendor-phoenixmail.py" "$src" "$here" "$(git -C "$src" rev-parse HEAD)"
python3 "$here/scripts/prepare-email-assets.py"
gofmt -w "$here/internal/cards/"*_mirror.go
(cd "$here" && go test ./... -race -count=1 && go vet ./...)
(cd "$here" && node --test tests/*.test.mjs engine/secureclaim/web/tests/*.test.cjs)
echo 'Presentation mirrored; review the adapter, stamp static versions, commit with [skip ci].'
