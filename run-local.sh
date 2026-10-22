#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
wire_env_file="${PHOENIXWIRE_ENV_FILE:-${XDG_CONFIG_HOME:-$HOME/.config}/phoenixwire/local.env}"
if [[ -e "$wire_env_file" ]]; then
  if [[ ! -f "$wire_env_file" || -L "$wire_env_file" || "$(stat -c '%a' -- "$wire_env_file")" != 600 || "$(stat -c '%u' -- "$wire_env_file")" != "$(id -u)" ]]; then
    printf '%s\n' 'Phoenix Wire requires an owner-only environment file (chmod 600).' >&2
    exit 1
  fi
  set -a
  source "$wire_env_file"
  set +a
fi
exec go run ./cmd/phoenixwire
