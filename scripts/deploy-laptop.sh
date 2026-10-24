#!/usr/bin/env bash
# Deploys Phoenix Wire to the laptop that serves wire.phoenixmail.ca. Run it from your computer, in this repository:
#
#   ./scripts/deploy-laptop.sh               tests, build, install, restart, check (rolls back by itself if the service does not start)
#   ./scripts/deploy-laptop.sh --dry-run     tests and build only, nothing is sent
#   ./scripts/deploy-laptop.sh --skip-tests  deploy without running the tests
#   ./scripts/deploy-laptop.sh --rollback    put the previous binary back
#
# Needs: Go, ssh access to the laptop (key), and the laptop's sudo rights for cp / install / systemctl (already in place).
# It never prints or touches a secret, a key, the master key or the state: it only replaces the program /opt/phoenixwire/phoenixwire.
set -euo pipefail

LAPTOP="${WIRE_LAPTOP:-user@host}"
SITE="${WIRE_SITE:-https://wire.phoenixmail.ca}"
here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$here"

mode=deploy
skip_tests=0
for arg in "$@"; do
  case "$arg" in
    --dry-run) mode=dry ;;
    --skip-tests) skip_tests=1 ;;
    --rollback) mode=rollback ;;
    -h|--help) sed -n 2,12p "${BASH_SOURCE[0]}"; exit 0 ;;
    *) echo "Unknown option: $arg (see --help)" >&2; exit 2 ;;
  esac
done

check_site() {
  echo "--- checking $SITE"
  local bad=0
  # the pages themselves, then every file they name (a file the server does not serve breaks the page for everyone)
  for path in / /classic.html /usdt.html; do
    code="$(curl -s -o /dev/null -m 15 -w '%{http_code}' "$SITE$path" || true)"
    echo "$path: HTTP $code"
    [ "$code" = 200 ] || bad=1
  done
  for page in / /classic.html; do
    for file in $(curl -s -m 15 "$SITE$page" | grep -oE '(src|href)="/?[A-Za-z0-9_./-]+\.(js|css|png|webp|svg|ico|webmanifest)' | sed -E 's/^(src|href)="//; s#^([^/])#/\1#' | sort -u); do
      code="$(curl -s -o /dev/null -m 15 -w '%{http_code}' "$SITE$file" || true)"
      if [ "$code" != 200 ]; then echo "MISSING $file: HTTP $code (named by $page)"; bad=1; fi
    done
  done
  echo "$(curl -s -m 15 "$SITE/api/config" || true)"
  return $bad
}

rollback() {
  echo "--- putting the previous program back on $LAPTOP"
  ssh "$LAPTOP" 'set -e; test -f /opt/phoenixwire/phoenixwire.prev; sudo install -o root -g root -m 755 /opt/phoenixwire/phoenixwire.prev /opt/phoenixwire/phoenixwire; sudo systemctl restart phoenixwire; sleep 7; echo "phoenixwire: $(systemctl is-active phoenixwire)"'
}

if [ "$mode" = rollback ]; then
  rollback
  check_site || true
  exit 0
fi

echo "--- deploying: $(git log -1 --oneline)"
if [ -n "$(git status --porcelain)" ]; then
  echo "WARNING: there are uncommitted changes in this folder; they will be part of what is deployed."
  read -r -p "Continue? [y/N] " answer
  [ "$answer" = y ] || [ "$answer" = Y ] || { echo "Cancelled."; exit 1; }
fi

if [ "$skip_tests" = 0 ]; then
  echo "--- tests"
  gofmt -l internal/cards internal/web internal/wire cmd | (! grep .) || { echo "gofmt: the files above are not formatted (run gofmt -w)"; exit 1; }
  go vet ./...
  go test ./...
  (cd engine/secureclaim && go vet ./... && go test ./...)
  if command -v node >/dev/null; then
    [ -d node_modules/jsdom ] || npm ci --ignore-scripts
    node --test --test-timeout=90000 tests/*.test.mjs engine/secureclaim/web/tests/*.cjs
  else
    echo "node is not installed: the JavaScript tests were skipped"
  fi
fi

echo "--- build"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
CGO_ENABLED=0 go build -trimpath -o "$tmp/phoenixwire-new" ./cmd/phoenixwire
sha="$(sha256sum "$tmp/phoenixwire-new" | cut -d' ' -f1)"
echo "program: $(du -h "$tmp/phoenixwire-new" | cut -f1), sha256 $sha"

if [ "$mode" = dry ]; then
  echo "Dry run: nothing was sent."
  exit 0
fi

echo "--- sending to $LAPTOP"
scp -q "$tmp/phoenixwire-new" "$LAPTOP:/tmp/phoenixwire-new"
echo "$sha" | ssh "$LAPTOP" 'cat > /tmp/wire.sha'

echo "--- installing and restarting"
ssh "$LAPTOP" 'set -e
echo "$(cat /tmp/wire.sha)  /tmp/phoenixwire-new" | sha256sum -c -
sudo cp -p /opt/phoenixwire/phoenixwire /opt/phoenixwire/phoenixwire.prev
sudo install -o root -g root -m 755 /tmp/phoenixwire-new /opt/phoenixwire/phoenixwire
rm -f /tmp/phoenixwire-new /tmp/wire.sha
sudo systemctl restart phoenixwire
sleep 7
echo "phoenixwire: $(systemctl is-active phoenixwire)"
test "$(systemctl is-active phoenixwire)" = active' || {
  echo "The service did not start: putting the previous program back."
  rollback
  exit 1
}

if check_site; then
  echo "Done: $(git log -1 --oneline) is live."
else
  echo "The site does not answer as expected. If it looks wrong: ./scripts/deploy-laptop.sh --rollback"
  exit 1
fi
