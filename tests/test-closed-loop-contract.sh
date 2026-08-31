#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY' "$ROOT/behavior-lab/spec/CLOSED_EXPERIMENTAL_LOOP_R0.json"
import json,sys
p=json.load(open(sys.argv[1]))
assert p['schema']=='tlaloc.closed-experimental-loop.r0'
assert p['managed_cli']=='tlaloc-closed-loop'
assert p['diagnostic_retry']['question_scope']=='FAILED_QUESTIONS_ONLY'
assert p['diagnostic_retry']['separate_from_primary_score'] is True
assert p['model_transport']['transport_failures_are_model_failures'] is False
assert p['memory']['historical_observations_do_not_override_the_current_incumbent_frontier'] is True
assert p['adaptive_search']['may_change_final_promotion_score'] is False
assert p['candidate_bank']['tlaloc_is_pixel_authority'] is False
assert p['candidate_bank']['optional_parent_specimen_id_forms_experimental_candidate_dag'] is True
assert 'TLALOC_PARENT_PNG' in p['candidate_bank']['build_environment']
assert p['experimental_incumbent']['canonical_authority'] is False
assert p['experimental_incumbent']['requires_no_question_score_regression'] is True
assert p['experimental_incumbent']['next_generation_baseline']=='BEST_ADVANCEABLE_CANDIDATE'
for inv in [
  'EXPERIMENTAL_INCUMBENT_NE_CANONICAL_ORIGAMI',
  'CLEAN_NATIVE_TRIAL_EQUALS_PNG_PLUS_QUESTION_ONLY',
  'DIAGNOSTIC_RETRY_NE_SELF_BOOTSTRAP_EVIDENCE',
  'TRANSPORT_FAILURE_NE_MODEL_SEMANTIC_FAILURE',
  'MEMORY_GUIDES_SEARCH_NE_PROMOTION',
  'FALSE_EXACT_EQUALS_0'
]: assert inv in p['hard_invariants']
PY
grep -q 'tlaloc-closed-loop' "$ROOT/behavior-lab/CLOSED_EXPERIMENTAL_LOOP_R0.md"
grep -q 'Experimental incumbent' "$ROOT/behavior-lab/CLOSED_EXPERIMENTAL_LOOP_R0.md"
grep -q 'Candidate DAG' "$ROOT/behavior-lab/CLOSED_EXPERIMENTAL_LOOP_R0.md"
grep -q 'TLALOC_PARENT_PNG' "$ROOT/behavior-lab/CLOSED_EXPERIMENTAL_LOOP_R0.md"
grep -q 'NATIVE_PNG_ONLY' "$ROOT/behavior-lab/testdata/closed-loop/LOCAL_LM_STUDIO_TEMPLATE.json"
grep -q 'DiagnosticInstruction' "$ROOT/behavior-lab/internal/closedloop/runner.go"
grep -q 'ImportTemporalBenchmark' "$ROOT/behavior-lab/internal/closedloop/runner.go"
grep -q 'IncumbentAdvanced' "$ROOT/behavior-lab/internal/closedloop/runner.go"
grep -q 'parent_specimen_id' "$ROOT/behavior-lab/internal/closedloop/model.go"
grep -q 'adaptivesearch.Prioritize' "$ROOT/behavior-lab/internal/closedloop/runner.go"
echo 'CLOSED_EXPERIMENTAL_LOOP_R0_OK'
