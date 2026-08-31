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
)
python3 - <<'PY' "$TMP/result.json"
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
print('temporal-native-benchmark-reference: PASS')
PY
