#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skills=(tlaloc-project tlaloc-behavior tlaloc-tlaloque origami-semantics tlaloc-release)
[[ -f "$ROOT/CLAUDE.md" ]]
[[ ! -e "$ROOT/.claude/skills/repo-flow" ]]
for name in "${skills[@]}"; do
  f="$ROOT/.claude/skills/$name/SKILL.md"
  [[ -f "$f" ]]
  grep -q '^---$' "$f"
  grep -q "^name: $name$" "$f"
  grep -q '^description: .\+' "$f"
  grep -q '^version: [0-9]' "$f"
done
python3 - "$ROOT/state/CLAIMS.json" <<'PY'
import json, sys
claims = {claim['id']: claim for claim in json.load(open(sys.argv[1]))}
skill_ir = claims['TLALOC.IR.SKILL']
assert skill_ir['status'] == 'designed'
assert not skill_ir.get('evidence')
PY
grep -q 'TLALOC.IR.SKILL' "$ROOT/docs/CAPABILITY_STATUS.md"
echo PASS
