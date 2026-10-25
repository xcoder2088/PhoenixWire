#!/usr/bin/env bash
# Makes an encrypted backup of everything Phoenix Wire and PhoenixMail keep on the laptop (programs, keys, data, services, web and tunnel
# configuration). Not the operating system. Run it from your computer, with the backup disk plugged in:
#
#   ./scripts/backup-laptop.sh                  backup to the default folder, then read it back to check it
#   ./scripts/backup-laptop.sh --with-ai        also include the (large) PhoenixAI data
#   ./scripts/backup-laptop.sh --check FILE     only check that a backup can be read, and lists its key files
#
# Where to: the folder of the backup disk is private to whoever runs this. Put it once in ~/.config/phoenixwire/deploy.env (the same file the
# deploy script reads; WIRE_DEPLOY_ENV points at another one):
#   WIRE_LAPTOP=user@host                 the laptop (ssh, with sudo for tar)
#   WIRE_BACKUP_DIR="/path/of/the/disk"   where the backup is written (quote it if it has a space)
# The backup holds the master key and the relay keys: it is encrypted with a passphrase you type (never printed, never put on a command line).
set -euo pipefail

deploy_env="${WIRE_DEPLOY_ENV:-${XDG_CONFIG_HOME:-$HOME/.config}/phoenixwire/deploy.env}"
if [ -f "$deploy_env" ]; then
  # shellcheck disable=SC1090
  . "$deploy_env"
fi
LAPTOP="${WIRE_LAPTOP:-}"
DEST="${WIRE_BACKUP_DIR:-}"

with_ai=0
check_only=""
while [ $# -gt 0 ]; do
  case "$1" in
    --with-ai) with_ai=1 ;;
    --check) shift; check_only="${1:-}"; [ -n "$check_only" ] || { echo "--check needs a file" >&2; exit 2; } ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done

ask_passphrase() { # $1 = prompt; reads without echo, into $pass
  printf '%s' "$1" >&2
  IFS= read -rs pass
  printf '\n' >&2
  [ -n "$pass" ] || { echo "An empty passphrase is refused." >&2; exit 1; }
}

# Reads a backup back with the passphrase in $pass and says whether the master key and the relay keys are in it.
read_back() { # $1 = file
  local listing
  listing="$(openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 -pass fd:3 -in "$1" 3<<<"$pass" 2>/dev/null | tar tz 2>/dev/null)" || { echo "The backup cannot be read: wrong passphrase, or the file is damaged." >&2; return 1; }
  echo "$listing" | grep -E 'master\.key|relayer-[a-z]+\.key' | head -8 || true
  if ! echo "$listing" | grep -q 'master\.key' || ! echo "$listing" | grep -q 'relayer-'; then
    echo "WARNING: the master key or the relay keys are not in this backup." >&2
    return 1
  fi
  echo "OK: $(echo "$listing" | wc -l) entries, the master key and the relay keys are there."
}

if [ -n "$check_only" ]; then
  ask_passphrase "Passphrase to read the backup: "
  read_back "$check_only"
  exit $?
fi

if [ -z "$LAPTOP" ] || [ -z "$DEST" ]; then
  echo "Set WIRE_LAPTOP (user@host) and WIRE_BACKUP_DIR (the backup disk's folder) in $deploy_env, or in the environment." >&2
  exit 2
fi
if [ ! -d "$DEST" ] || [ ! -w "$DEST" ]; then
  echo "The backup folder is not there (or not writable): $DEST" >&2
  echo "Plug in the disk, open it once in the file manager, then try again." >&2
  exit 2
fi

excludes="--exclude=/var/lib/phoenixmail-ai"
[ "$with_ai" = 1 ] && excludes=""

paths="/opt/phoenixwire /opt/phoenixmail* /opt/nimiqsecureclaim /etc/phoenixwire /etc/phoenixmail* /etc/nimiqsecureclaim /var/lib/phoenixwire /var/lib/phoenixmail* /var/lib/nimiqsecureclaim /etc/systemd/system/phoenixwire.service /etc/systemd/system/phoenixmail* /etc/systemd/system/nimiq* /etc/systemd/system/cloudflared* /etc/caddy /etc/cloudflared"
out="$DEST/phoenix-backup-$(date +%Y%m%d-%H%M).tgz.enc"

ask_passphrase "Choose the passphrase (keep it in your password manager): "
first="$pass"
ask_passphrase "Type it again: "
[ "$pass" = "$first" ] || { echo "The two passphrases differ. Nothing was written." >&2; exit 1; }
unset first

echo "Copying from $LAPTOP ..." >&2
# The globs are expanded on the laptop, as root. A path that does not exist there is only a warning.
ssh "$LAPTOP" "sudo sh -c 'tar czf - --ignore-failed-read $excludes $paths'" \
  | openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt -pass fd:3 -out "$out" 3<<<"$pass"
chmod 600 "$out"
echo "Written: $out ($(du -h "$out" | cut -f1))" >&2

echo "Reading it back ..." >&2
read_back "$out" || { echo "Do not trust this backup: run the script again." >&2; exit 1; }
echo "Keep the passphrase, and copy this file to a second place away from the laptop."
