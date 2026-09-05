#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
[[ -d "$ROOT/behavior-lab/internal/tlaloque" ]]
[[ -d "$ROOT/behavior-lab/internal/reference" ]]
[[ ! -e "$ROOT/behavior-lab/internal/swarm" ]]
[[ ! -e "$ROOT/behavior-lab/internal/oracle" ]]
# Match the retired package names exactly. Names such as "swarmask" are
# legitimate current packages and must not be rejected merely for sharing a
# prefix with the retired "swarm" namespace.
if grep -RInE 'internal/(swarm|oracle)(/|$)|package (swarm|oracle)([[:space:]]|$)' "$ROOT/behavior-lab" --include='*.go'; then
  echo "retired implementation namespace found" >&2
  exit 1
fi
for f in "$ROOT/docs/ARCHITECTURE.md" "$ROOT/docs/BEHAVIOR_COMPILATION.md" "$ROOT/docs/CAPABILITY_STATUS.md" "$ROOT/docs/ORIGAMI_INTEGRATION_CONTRACT.md" "$ROOT/README.md"; do
  if grep -inE '\b(oracle|adivino|báculo|baculo)\b' "$f"; then
    echo "retired current-architecture term found in $f" >&2
    exit 1
  fi
done
grep -q 'Tlaloque' "$ROOT/docs/NOMENCLATURE.md"
grep -q 'reference semantics' "$ROOT/docs/NOMENCLATURE.md"
echo PASS
