#!/usr/bin/env bash
# Re-mirror the PhoenixCashlink protocol packages into the embedded engine and re-apply the MIT headers.
# The copyright holder relicensed this copy under the MIT License (see docs/IMPORT-USDT-CASHLINK-2026-10-23.md).
set -euo pipefail
here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
src="${PHOENIXCASHLINK_REPO:-$(dirname -- "$here")/PhoenixCashlink}"
[[ "$(git -C "$src" branch --show-current)" == main ]] || { echo 'PhoenixCashlink must be on main' >&2; exit 1; }
git -C "$src" diff --quiet && git -C "$src" diff --cached --quiet || { echo 'PhoenixCashlink has uncommitted changes' >&2; exit 1; }
engine="$here/engine/secureclaim"
"$src/scripts/vendor.sh" "$engine/internal/cashlink" github.com/xcoder2088/NimiqSecureClaim/internal/cashlink "$engine/web"
python3 - "$engine" <<'PY'
import subprocess,sys
root=sys.argv[1]
for p in subprocess.run(["grep","-rl","SPDX-License-Identifier: AGPL-3.0-only",root],capture_output=True,text=True).stdout.split():
    s=open(p,encoding="utf-8").read()
    if p.endswith(".go"):
        s=s.replace("// SPDX-License-Identifier: AGPL-3.0-only","// SPDX-License-Identifier: MIT\n// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.",1)
    else:
        s=s.replace("SPDX-License-Identifier: AGPL-3.0-only","SPDX-License-Identifier: MIT (originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder)",1)
    open(p,"w",encoding="utf-8").write(s)
PY
(cd "$engine" && gofmt -l internal/cashlink && go vet ./... && go test ./... -count=1)
echo 'Mirror refreshed with MIT headers; review, then commit with [skip ci].'
