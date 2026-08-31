#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY' "$ROOT/behavior-lab/spec/LEARNING_MEMORY_R0.json"
import json,sys
p=json.load(open(sys.argv[1]))
assert p['contract_id']=='tlaloc.learning-memory.r0'
assert p['storage']['event_files_immutable'] is True
assert p['storage']['destructive_forgetting'] is False
assert 'EVIDENCE_LEDGER' in p['memory_layers']
assert 'PATTERN_INDEX' in p['memory_layers']
assert 'EXPERIMENT_HISTORY' in p['memory_layers']
for inv in ['EVENT_ID_IS_CONTENT_ADDRESSED','REINGEST_SAME_EVIDENCE_IS_IDEMPOTENT','SYNTHETIC_EVIDENCE_NE_REAL_MODEL_EVIDENCE','MEMORY_NE_AUTOMATIC_PROMOTION','OLD_FAILURES_ARE_NOT_DELETED_WHEN_FIXED']:
    assert inv in p['hard_invariants']
print('learning-memory-contract: PASS')
PY
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
(
  cd "$ROOT/behavior-lab"
  go run ./cmd/tlaloc-temporal-bench -no-memory -in testdata/temporal-benchmark/perfect-campaign.json -out "$TMP/result.json"
  go run ./cmd/tlaloc-learning-memory ingest-benchmark -campaign testdata/temporal-benchmark/perfect-campaign.json -result "$TMP/result.json" -include-synthetic -store "$TMP/memory" > "$TMP/ingest1.json"
  go run ./cmd/tlaloc-learning-memory ingest-benchmark -campaign testdata/temporal-benchmark/perfect-campaign.json -result "$TMP/result.json" -include-synthetic -store "$TMP/memory" > "$TMP/ingest2.json"
  go run ./cmd/tlaloc-learning-memory summary -store "$TMP/memory" > "$TMP/summary1.json"
  go run ./cmd/tlaloc-learning-memory events -store "$TMP/memory" > "$TMP/events.json"
)
python3 - <<'PY' "$TMP/ingest1.json" "$TMP/ingest2.json" "$TMP/summary1.json" "$TMP/events.json" "$TMP/parent.txt"
import json,sys
first=json.load(open(sys.argv[1])); second=json.load(open(sys.argv[2])); summary=json.load(open(sys.argv[3])); events=json.load(open(sys.argv[4]))
assert first['events_considered']==18 and first['added']==18 and first['already_present']==0
assert second['added']==0 and second['already_present']==18
assert summary['total_events']==18
assert summary['synthetic_observations']==18
assert summary['real_model_observations']==0
assert summary.get('top_real_failure_patterns',[])==[]
assert len(events)==18
open(sys.argv[5],'w').write(events[0]['event_id'])
print('learning-memory-idempotent-ingest: PASS')
PY
PARENT="$(cat "$TMP/parent.txt")"
(
  cd "$ROOT/behavior-lab"
  go run ./cmd/tlaloc-learning-memory record-change -store "$TMP/memory" -candidate-id candidate-t2-route -summary 'make T2 route more visible' -parents "$PARENT" > "$TMP/change.json"
)
CHANGE_ID="$(python3 - <<'PY' "$TMP/change.json"
import json,sys
print(json.load(open(sys.argv[1]))['event']['event_id'])
PY
)"
(
  cd "$ROOT/behavior-lab"
  go run ./cmd/tlaloc-learning-memory record-outcome -store "$TMP/memory" -candidate-id candidate-t2-route -parents "$CHANGE_ID,$PARENT" -before 0.4 -after 0.7 > "$TMP/outcome.json"
  go run ./cmd/tlaloc-learning-memory summary -store "$TMP/memory" > "$TMP/summary2.json"
)
python3 - <<'PY' "$TMP/summary2.json" "$TMP/memory"
import json,sys,pathlib
s=json.load(open(sys.argv[1]))
assert s['total_events']==20
assert s['change_attempts']==1 and s['outcome_links']==1
assert len(s['candidate_outcomes'])==1
c=s['candidate_outcomes'][0]
assert c['candidate_id']=='candidate-t2-route' and c['outcomes']==1
assert abs(c['mean_delta']-0.3)<1e-9
files=list((pathlib.Path(sys.argv[2])/'events').glob('*.json'))
assert len(files)==20
print('learning-memory-experiment-history: PASS')
PY
