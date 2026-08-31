#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC="$ROOT/behavior-lab/spec/ADAPTIVE_SEARCH_R0.json"
DOC="$ROOT/behavior-lab/ADAPTIVE_SEARCH_R0.md"
CLI="$ROOT/behavior-lab/cmd/tlaloc-adaptive-search/main.go"
[[ -f "$SPEC" && -f "$DOC" && -f "$CLI" ]]
grep -q '"contract_id": "tlaloc.adaptive-search.r0"' "$SPEC"
grep -q 'MEMORY_GUIDES_EXPERIMENT_BUDGET_NOT_PROMOTION_SCORE' "$SPEC"
grep -q 'FINAL_TOURNAMENT_REMAINS_EVIDENCE_GATED' "$SPEC"
grep -q 'EXPLORATION_FLOOR_GT_0' "$SPEC"
grep -q 'SYNTHETIC_EVIDENCE_NE_EMPIRICAL_SEARCH_TARGET' "$SPEC"
grep -q 'memory priority != promotion score' "$DOC"
(
  cd "$ROOT/behavior-lab"
  go test ./internal/adaptivesearch
)
echo PASS
