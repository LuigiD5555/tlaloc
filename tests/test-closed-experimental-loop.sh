#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC="$ROOT/behavior-lab/spec/CLOSED_EXPERIMENTAL_LOOP_R0.json"
DOC="$ROOT/docs/CLOSED_EXPERIMENTAL_LOOP_R0.md"
[[ -f "$SPEC" && -f "$DOC" ]]
python3 - <<'PY' "$SPEC"
import json,sys
s=json.load(open(sys.argv[1]))
assert s['schema']=='tlaloc.closed-experimental-loop.r0'
assert s['managed_cli']=='tlaloc-closed-loop'
assert s['experimental_incumbent']['canonical_authority'] is False
assert s['experimental_incumbent']['requires_no_question_score_regression'] is True
assert s['memory']['historical_observations_do_not_override_the_current_incumbent_frontier'] is True
assert s['candidate_bank']['optional_parent_specimen_id_forms_experimental_candidate_dag'] is True
assert 'INCUMBENT_NO_ACTIVE_FAILURES' in s['stop_reasons']
assert 'EXPERIMENTAL_INCUMBENT_NE_CANONICAL_ORIGAMI' in s['hard_invariants']
PY
grep -q 'Experimental incumbent' "$DOC"
grep -q 'Candidate DAG' "$DOC"
grep -q 'TLALOC_PARENT_PNG' "$DOC"
echo PASS
