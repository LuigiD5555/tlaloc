#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORIGAMI_ROOT="${ORIGAMI_ROOT:?ORIGAMI_ROOT must point to a pinned Origami checkout}"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/tlaloc-origami-crossrepo.XXXXXX")"
SERVER_PID=""
cleanup(){
  if [[ -n "$SERVER_PID" ]]; then kill "$SERVER_PID" >/dev/null 2>&1 || true; fi
  rm -rf "$TMP"
}
trap cleanup EXIT

[[ "$(tr -d '\r\n' < "$ROOT/VERSION")" == "6.0.0-alpha.20" ]]
[[ "$(tr -d '\r\n' < "$ORIGAMI_ROOT/VERSION")" == "6.0.0-alpha.15" ]]

mkdir -p "$TMP/bin" "$TMP/run" "$TMP/memory"
(
  cd "$ORIGAMI_ROOT"
  go build -trimpath -o "$TMP/bin/origami-temporal-carrier" ./cmd/origami-temporal-carrier
  go build -trimpath -o "$TMP/bin/origami-candidate-build" ./cmd/origami-candidate-build
)
(
  cd "$ROOT/behavior-lab"
  go build -trimpath -o "$TMP/bin/tlaloc-closed-loop" ./cmd/tlaloc-closed-loop
)

"$TMP/bin/origami-temporal-carrier" \
  -mode build \
  -in "$ORIGAMI_ROOT/experiments/temporal-automaton-r0/signal-chain.json" \
  -out "$TMP/baseline.png" > "$TMP/baseline-build.json"
[[ "$(wc -c < "$TMP/baseline.png" | tr -d ' ')" == "8192" ]]
BASELINE_SHA256="$(sha256sum "$TMP/baseline.png" | awk '{print $1}')"
export BASELINE_SHA256

python3 "$ROOT/tests/fixtures/fake_openai_vlm.py" --port 18765 >"$TMP/fake-vlm.log" 2>&1 &
SERVER_PID=$!
for _ in $(seq 1 50); do
  if python3 - <<'PY' >/dev/null 2>&1
import urllib.request
urllib.request.urlopen('http://127.0.0.1:18765/health', timeout=.25).read()
PY
  then break; fi
  sleep .1
done
python3 - <<'PY'
import urllib.request
assert b'"ok"' in urllib.request.urlopen('http://127.0.0.1:18765/health', timeout=1).read()
PY

cat > "$TMP/config.json" <<JSON
{
  "schema": "tlaloc.closed-experimental-loop.r0.config",
  "run_id": "origami-alpha15-crossrepo-r0",
  "benchmark_id": "origami-temporal-native-r0",
  "output_dir": "$TMP/run",
  "memory_root": "$TMP/memory",
  "origami_version": "6.0.0-alpha.15",
  "tlaloc_version": "6.0.0-alpha.20",
  "trials_per_model": 1,
  "candidates_per_generation": 1,
  "max_generations": 1,
  "min_incumbent_improvement": 0.01,
  "continue_exploration_when_stable": false,
  "diagnostic_retries": true,
  "conditions": ["NATIVE_PNG_ONLY"],
  "outcome_metric": "NATIVE_SCORE",
  "models": [{
    "name": "SYNTHETIC_CROSS_REPO_VLM",
    "provider": "OPENAI_COMPAT",
    "base_url": "http://127.0.0.1:18765/v1",
    "model": "synthetic-cross-repo-vlm",
    "temperature": 0,
    "timeout_seconds": 10,
    "transport_retries": 0
  }],
  "baseline": {"id": "signal-chain-r0", "png": "$TMP/baseline.png"},
  "auto_candidates": true,
  "candidate_builder": ["$TMP/bin/origami-candidate-build"],
  "auto_candidate_base_profile_id": "origami.temporal-carrier.r0.profile-1",
  "auto_candidates_per_generation": 4
}
JSON

"$TMP/bin/tlaloc-closed-loop" validate -config "$TMP/config.json" > "$TMP/validate.txt"
grep -q '^CLOSED_LOOP_READY$' "$TMP/validate.txt"
"$TMP/bin/tlaloc-closed-loop" run -config "$TMP/config.json" > "$TMP/run-stdout.json"

REPORT="$TMP/run/closed-loop-report.json"
[[ -f "$REPORT" ]]
CANDIDATE_PNG="$(python3 - <<'PY' "$REPORT"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.closed-experimental-loop.r0.report'
assert not r.get('execution_errors'), r.get('execution_errors')
assert len(r['generations'])==1
_g=r['generations'][0]
assert _g['active_failure_count']>0
assert len(_g['selected_candidate_ids'])==1
cid=_g['selected_candidate_ids'][0]
assert cid.startswith('auto-'), cid
assert len(_g.get('candidates',[]))==1
assert len(_g.get('outcomes',[]))==1
out=_g['outcomes'][0]
assert out['candidate_id']==cid
assert out['delta']>0, out
assert out['non_regression'] is True, out
assert out['advanceable'] is True, out
assert _g['incumbent_advanced'] is True
assert r['final_incumbent_id']==cid
print(_g['candidates'][0]['png'])
PY
)"
[[ -f "$CANDIDATE_PNG" ]]
[[ "$(wc -c < "$CANDIDATE_PNG" | tr -d ' ')" == "8192" ]]
[[ "$(sha256sum "$CANDIDATE_PNG" | awk '{print $1}')" != "$BASELINE_SHA256" ]]

BASELINE_RESULT="$(python3 - <<'PY' "$REPORT"
import json,sys
print(json.load(open(sys.argv[1]))['generations'][0]['baseline']['result_path'])
PY
)"
CANDIDATE_RESULT="$(python3 - <<'PY' "$REPORT"
import json,sys
print(json.load(open(sys.argv[1]))['generations'][0]['candidates'][0]['result_path'])
PY
)"

"$TMP/bin/origami-temporal-carrier" -mode decode -in "$TMP/baseline.png" -out "$TMP/baseline-program.json" > "$TMP/baseline-decode.json"
"$TMP/bin/origami-temporal-carrier" -mode decode -in "$CANDIDATE_PNG" -out "$TMP/candidate-program.json" > "$TMP/candidate-decode.json"
python3 - <<'PY' "$TMP/baseline-program.json" "$TMP/candidate-program.json" "$BASELINE_RESULT" "$CANDIDATE_RESULT" "$TMP/memory"
import json, pathlib, sys
base=json.load(open(sys.argv[1])); cand=json.load(open(sys.argv[2]))
assert base==cand, 'TemporalProgram changed across candidate build'
base_result=json.load(open(sys.argv[3])); cand_result=json.load(open(sys.argv[4]))
assert base_result['real_evidence'] is False, base_result['real_evidence']
assert cand_result['real_evidence'] is False, cand_result['real_evidence']
root=pathlib.Path(sys.argv[5])
raw='\n'.join(p.read_text(errors='ignore') for p in root.rglob('*') if p.is_file())
assert '"evidence_class":"REAL_MODEL"' not in raw.replace(' ',''), 'synthetic fixture contaminated persistent memory as REAL_MODEL'
PY

printf 'ORIGAMI_TLALOC_CROSS_REPO_R0=PASS\n'
printf 'ORIGAMI_VERSION=6.0.0-alpha.15\n'
printf 'TLALOC_VERSION=6.0.0-alpha.20\n'
printf 'PARENT_BYTES=8192\n'
printf 'CANDIDATE_BYTES=8192\n'
printf 'EXACT_TEMPORAL_PROGRAM_PRESERVED=YES\n'
printf 'MODEL_EVIDENCE=SYNTHETIC_ONLY_NOT_PERSISTED_AS_REAL_MODEL\n'
