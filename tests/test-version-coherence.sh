#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXPECTED="$(tr -d '\r\n' < "$ROOT/VERSION")"
[[ -n "$EXPECTED" ]]
grep -q 'TLALOC_VERSION="$(tr -d' "$ROOT/install.sh"
if grep -Eq 'TLALOC_VERSION="[0-9]+\.[0-9]+\.[0-9]+' "$ROOT/install.sh"; then
  echo "installer duplicates a hard-coded release version" >&2
  exit 1
fi
grep -q "# Tlaloc $EXPECTED" "$ROOT/README.md"
grep -q "Capability status — Tlaloc $EXPECTED" "$ROOT/docs/CAPABILITY_STATUS.md"
echo PASS
