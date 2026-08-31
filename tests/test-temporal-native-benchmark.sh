#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY' "$ROOT/behavior-lab/spec/TEMPORAL_NATIVE_BENCHMARK_R0.json"
import json,sys
p=json.load(open(sys.argv[1]))
assert p['contract_id']=='tlaloc.temporal-native-benchmark.r0'
assert p['conditions'][0]=='NATIVE_PNG_ONLY'
assert 'R4_ASSISTED' in p['conditions']
assert set(p['score_layers'])=={'P_PERCEPTION','R_PROTOCOL','S_SEMANTIC','T_TEMPORAL','X_EXACTNESS'}
assert p['questions'][-1]['id']=='Q8'
assert p['diagnostic_mode']['orthogonal_to_condition'] is True
assert p['diagnostic_mode']['included_in_primary_condition_comparison'] is False
assert p['diagnostic_mode']['reports_private_reasoning'] is False
assert p['diagnostic_mode']['reports_chain_of_thought'] is False
assert 'ROSETTA' in p['diagnostic_mode']['stages']
assert 'T2_NOT_FOUND' in p['diagnostic_mode']['failure_codes']
assert 'DEBUG_TRACE_MUST_NOT_REQUEST_PRIVATE_REASONING_OR_CHAIN_OF_THOUGHT' in p['hard_invariants']
assert 'NO_LLM_JUDGE' in p['hard_invariants']
assert 'FALSE_EXACT=0' in p['hard_invariants']
print('temporal-native-benchmark-contract: PASS')
PY
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
(
  cd "$ROOT/behavior-lab"
  go run ./cmd/tlaloc-temporal-bench \
    -in testdata/temporal-benchmark/perfect-campaign.json \
    -out "$TMP/result.json"
  go run ./cmd/tlaloc-temporal-bench \
    -in testdata/temporal-benchmark/diagnostic-failure-campaign.json \
    -out "$TMP/diagnostic.json"
  go run ./cmd/tlaloc-temporal-bench -print-debug-instruction > "$TMP/debug-instruction.txt"
  go run ./cmd/tlaloc-temporal-bench -print-debug-example > "$TMP/debug-example.txt"
)
python3 - <<'PY' "$TMP/result.json" "$TMP/diagnostic.json" "$TMP/debug-instruction.txt" "$TMP/debug-example.txt"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.temporal-native-benchmark.r0.result'
assert r['real_evidence'] is False
assert len(r['trials'])==2
for t in r['trials']:
    assert t['overall_score']==1
    assert t['self_bootstrap_score']==1
    assert t['temporal_reasoning_score']==1
    assert t['exact_honesty_score']==1
    assert t['invented_exact_claims']==0
assert len(r['comparisons'])==1
assert r['comparisons'][0]['assistance_gain']==0

d=json.load(open(sys.argv[2]))
assert d['real_evidence'] is False
assert len(d['trials'])==1
trial=d['trials'][0]
assert trial['diagnostic_mode'] is True
assert trial['debug_summary']['trace_coverage']==1
assert trial['debug_summary']['trace_consistency_score']==1
assert trial['debug_summary']['dominant_failure_frontier']=='ROSETTA'
assert trial['debug_summary']['earliest_failure_frontier']=='ROSETTA'
assert trial['debug_summary']['furthest_completed_stage']=='EXACT_BOUNDARY'
assert trial['debug_summary']['most_common_failure_code']=='T2_NOT_FOUND'
assert trial['debug_summary']['missing_trace_count']==0
assert trial['debug_summary']['invalid_trace_count']==0
assert trial['debug_summary']['answer_trace_mismatch_count']==0
assert len(trial['debug_reports'])==9
# Diagnostic repetitions must not enter Native-vs-R4 primary comparison.
assert d.get('comparisons',[])==[]

instruction=open(sys.argv[3]).read()
assert 'ORIGAMI_DEBUG_R0=' in instruction
assert 'not private reasoning' in instruction.lower()
example=open(sys.argv[4]).read().strip()
assert example.startswith('ORIGAMI_DEBUG_R0=')
assert 'T2_NOT_FOUND' in example
print('temporal-native-benchmark-reference: PASS')
PY
