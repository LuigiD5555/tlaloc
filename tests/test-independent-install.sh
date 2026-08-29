#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
export HOME="$TMP/home"
export XDG_DATA_HOME="$HOME/.local/share"
export XDG_STATE_HOME="$HOME/.local/state"
export XDG_CONFIG_HOME="$HOME/.config"
export XDG_CACHE_HOME="$HOME/.cache"
mkdir -p "$HOME"
EXPECTED_VERSION="$(tr -d '\r\n' < "$ROOT/VERSION")"
PATH="$HOME/.local/bin:$PATH" "$ROOT/install.sh"
[[ -L "$HOME/.local/share/tlaloc/current" ]]
CURRENT_TARGET="$(readlink -f "$HOME/.local/share/tlaloc/current")"
[[ "$CURRENT_TARGET" == "$HOME/.local/share/tlaloc/versions/$EXPECTED_VERSION" ]]
grep -qx $'Tlaloc\t'"$EXPECTED_VERSION" "$CURRENT_TARGET/.tlaloc-managed-version"
PATH="$HOME/.local/bin:$PATH" tlaloc version | grep -qx "Tlaloc $EXPECTED_VERSION"
[[ ! -e "$HOME/.local/share/origami/current" ]]
PATH="$HOME/.local/bin:$PATH" tlaloc doctor
PATH="$HOME/.local/bin:$PATH" tlaloc-uninstall --yes
[[ ! -e "$HOME/.local/share/tlaloc/current" ]]
