#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY' "$ROOT/behavior-lab/spec/CLOSED_EXPERIMENTAL_LOOP_R0.json"
import json,sys
p=json.load(open(sys.argv[1]))
assert p['schema']=='tlaloc.closed-experimental-loop.r0'
assert p['diagnostic_policy']['only_failed_question_ids'] is True
assert p['diagnostic_policy']['excluded_from_primary_score'] is True
assert p['transport']['transport_failures_are_model_failures'] is False
assert p['memory']['change_attempts_link_to_parent_failure_evidence'] is True
assert p['memory']['outcomes_link_change_attempt_to_post_change_evidence'] is True
assert p['adaptive_search']['may_change_final_tournament_score'] is False
assert p['candidate_rendering']['tlaloc_is_pixel_authority'] is False
for inv in [
  'CLEAN_NATIVE_TRIAL_EQUALS_PNG_PLUS_QUESTION_ONLY',
  'DIAGNOSTIC_RETRY_NE_SELF_BOOTSTRAP_EVIDENCE',
  'TRANSPORT_FAILURE_NE_MODEL_SEMANTIC_FAILURE',
  'MEMORY_GUIDES_SEARCH_NE_PROMOTION',
  'FALSE_EXACT_EQUALS_0'
]: assert inv in p['hard_invariants']
PY
grep -q 'tlaloc-closed-loop' "$ROOT/behavior-lab/CLOSED_EXPERIMENTAL_LOOP_R0.md"
grep -q 'NATIVE_PNG_ONLY' "$ROOT/behavior-lab/testdata/closed-loop/LOCAL_LM_STUDIO_TEMPLATE.json"
grep -q 'DiagnosticInstruction' "$ROOT/behavior-lab/internal/closedloop/runner.go"
grep -q 'ImportTemporalBenchmark' "$ROOT/behavior-lab/internal/closedloop/runner.go"
grep -q 'EventOutcome' "$ROOT/behavior-lab/internal/closedloop/runner.go"
grep -q 'adaptivesearch.Prioritize' "$ROOT/behavior-lab/internal/closedloop/runner.go"
echo 'CLOSED_EXPERIMENTAL_LOOP_R0_OK'
