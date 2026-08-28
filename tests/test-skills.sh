#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skills=(tlaloc-project tlaloc-behavior tlaloc-tlaloque origami-semantics tlaloc-release)
[[ -f "$ROOT/CLAUDE.md" ]]
for name in "${skills[@]}"; do
  f="$ROOT/.claude/skills/$name/SKILL.md"
  [[ -f "$f" ]]
  grep -q '^---$' "$f"
  grep -q "^name: $name$" "$f"
  grep -q '^description: .\+' "$f"
  grep -q '^version: [0-9]' "$f"
done
grep -q 'Project-local Claude Code skills.*R0 implemented' "$ROOT/docs/CAPABILITY_STATUS.md"
grep -q 'SkillIR / generated Claude Skills.*not implemented' "$ROOT/docs/CAPABILITY_STATUS.md"
echo PASS
