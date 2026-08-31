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
for cli in tlaloc tlaloc-behavior-lab tlaloc-origami tlaloc-perception-campaign tlaloc-visual-search tlaloc-native-eval tlaloc-protocol-eval tlaloc-automaton-distill tlaloc-temporal-bench tlaloc-uninstall; do
  [[ -L "$HOME/.local/bin/$cli" ]] || { echo "missing managed CLI: $cli" >&2; exit 1; }
  [[ -x "$(readlink -f "$HOME/.local/bin/$cli")" ]] || { echo "managed CLI is not executable: $cli" >&2; exit 1; }
done
PATH="$HOME/.local/bin:$PATH" tlaloc skills list | grep -qx 'tlaloc-project'
if PATH="$HOME/.local/bin:$PATH" tlaloc skills list | grep -qx 'repo-flow'; then
  echo "repo-flow must not be distributed by Tlaloc" >&2
  exit 1
fi
[[ ! -e "$CURRENT_TARGET/.claude/skills/repo-flow" ]]
mkdir -p "$TMP/project"
git -C "$TMP/project" init -q
(
  cd "$TMP/project"
  PATH="$HOME/.local/bin:$PATH" tlaloc skills install tlaloc-project >/dev/null
)
cmp "$CURRENT_TARGET/.claude/skills/tlaloc-project/SKILL.md" "$TMP/project/.claude/skills/tlaloc-project/SKILL.md"
if PATH="$HOME/.local/bin:$PATH" tlaloc skills install repo-flow --project "$TMP/project" >"$TMP/repo-flow.out" 2>&1; then
  echo "Tlaloc unexpectedly installed Tonal-owned repo-flow" >&2
  exit 1
fi
grep -q 'repo-flow moved to Tonal' "$TMP/repo-flow.out"
[[ ! -e "$HOME/.local/share/origami/current" ]]
PATH="$HOME/.local/bin:$PATH" tlaloc doctor
PATH="$HOME/.local/bin:$PATH" tlaloc-uninstall --yes
[[ ! -e "$HOME/.local/share/tlaloc/current" ]]
for cli in tlaloc tlaloc-behavior-lab tlaloc-origami tlaloc-perception-campaign tlaloc-visual-search tlaloc-native-eval tlaloc-protocol-eval tlaloc-automaton-distill tlaloc-temporal-bench tlaloc-uninstall; do
  [[ ! -e "$HOME/.local/bin/$cli" ]] || { echo "uninstall left managed CLI: $cli" >&2; exit 1; }
done
