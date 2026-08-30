#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY' "$ROOT/behavior-lab/spec/AUTOMATON_DISTILLATION_R0.json"
import json,sys
p=json.load(open(sys.argv[1]))
assert p['contract_id']=='tlaloc.automaton-distillation.r0'
assert p['output']=='origami.automaton.r0-compatible IR'
assert p['boundary']['tlaloque_runtime_required_after_distillation'] is False
assert p['boundary']['origami_runtime_required_by_tlaloc'] is False
assert p['boundary']['target_owns_final_semantics'] is True
assert 'UNDECLARED_DEPENDENCY_IS_NOT_INFERRED' in p['hard_invariants']
assert 'DISTILLED_AUTOMATON_NE_ORIGINAL_SWARM_TRACE' in p['hard_invariants']
print('automaton-distillation-contract: PASS')
PY
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
(
  cd "$ROOT/behavior-lab"
  go run ./cmd/tlaloc-automaton-distill \
    -in testdata/automata/signal-chain-trace.json \
    -out "$TMP/result.json"
)
python3 - <<'PY' "$TMP/result.json"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.automaton-distillation-result.r0'
assert r['automaton']['schema']=='origami.automaton.r0'
assert r['temporal_program']['schema']=='origami.temporal-program.r0'
assert r['temporal_program']['automaton']['id']==r['automaton']['id']
assert r['temporal_program']['max_steps']==4
assert r['metrics']['trace_steps']==4
assert r['metrics']['trace_max_step']==3
assert r['metrics']['unique_cells']==3
assert r['metrics']['unique_rules']==3
assert r['metrics']['repeated_transitions_removed']==1
assert len(r['automaton']['source_trace_sha256'])==64
print('automaton-distillation-reference: PASS')
PY
