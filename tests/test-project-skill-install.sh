#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

export HOME="$TMP/home"
export XDG_DATA_HOME="$HOME/.local/share"
mkdir -p "$XDG_DATA_HOME/tlaloc/versions/test/.claude/skills" "$TMP/project"
cp -a "$ROOT/.claude/skills/." "$XDG_DATA_HOME/tlaloc/versions/test/.claude/skills/"
ln -s "$XDG_DATA_HOME/tlaloc/versions/test" "$XDG_DATA_HOME/tlaloc/current"

git -C "$TMP/project" init -q

"$ROOT/tools/tlaloc" skills list | grep -qx 'tlaloc-project'
if "$ROOT/tools/tlaloc" skills list | grep -qx 'repo-flow'; then
  echo "repo-flow must not be listed by Tlaloc" >&2
  exit 1
fi
(
  cd "$TMP/project"
  "$ROOT/tools/tlaloc" skills install tlaloc-project
)
TARGET="$TMP/project/.claude/skills/tlaloc-project/SKILL.md"
[[ -f "$TARGET" ]]
cmp "$ROOT/.claude/skills/tlaloc-project/SKILL.md" "$TARGET"

# Idempotent when unchanged.
(
  cd "$TMP/project"
  "$ROOT/tools/tlaloc" skills install tlaloc-project >/dev/null
)

# Protect local edits unless force is explicit.
printf '\nlocal edit\n' >> "$TARGET"
if (cd "$TMP/project" && "$ROOT/tools/tlaloc" skills install tlaloc-project >/dev/null 2>&1); then
  echo "project skill install overwrote protection gate" >&2
  exit 1
fi
(
  cd "$TMP/project"
  "$ROOT/tools/tlaloc" skills install tlaloc-project --force >/dev/null
)
cmp "$ROOT/.claude/skills/tlaloc-project/SKILL.md" "$TARGET"

# repo-flow has a deliberate ownership migration error.
if (cd "$TMP/project" && "$ROOT/tools/tlaloc" skills install repo-flow >"$TMP/repo-flow.out" 2>&1); then
  echo "Tlaloc unexpectedly installed Tonal-owned repo-flow" >&2
  exit 1
fi
grep -q 'repo-flow moved to Tonal' "$TMP/repo-flow.out"

echo PASS
