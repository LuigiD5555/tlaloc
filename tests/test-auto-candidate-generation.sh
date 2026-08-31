#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY' "$ROOT/behavior-lab/spec/AUTO_CANDIDATE_GENERATION_R0.json"
import json,sys
s=json.load(open(sys.argv[1]))
assert s['schema']=='tlaloc.auto-candidate-generation.r0'
inv=set(s['invariants'])
for x in [
  'AUTO_CANDIDATE_MODE_IS_OPT_IN',
  'TLALOC_GENERATES_MUTATION_INTENT_NOT_CANONICAL_PIXELS',
  'UNSUPPORTED_MUTATION_KIND_IS_FILTERED_BEFORE_INFERENCE',
  'AUTO_CANDIDATE_HAS_EXACTLY_ONE_MUTATION',
  'BUILDER_MUST_DECLARE_EXACT_PLANE_MUTATION_FALSE',
  'CANDIDATE_BUILD_SUCCESS_NE_MODEL_IMPROVEMENT',
  'EXPERIMENTAL_INCUMBENT_NE_CANONICAL_ORIGAMI',
]: assert x in inv, x
assert s['builder_protocol']['required_capability_schema']=='origami.experimental-candidate.r0.capabilities'
assert s['default_parent_profile']=='origami.temporal-carrier.r0.profile-1'
PY
grep -q 'Auto Candidate Generation R0' "$ROOT/docs/AUTO_CANDIDATE_GENERATION_R0.md"
grep -q 'AutoCandidates' "$ROOT/behavior-lab/internal/closedloop/model.go"
grep -q 'RunAuto' "$ROOT/behavior-lab/internal/closedloop/auto_runner.go"
grep -q 'candidate builder capabilities' "$ROOT/behavior-lab/internal/closedloop/autocandidates.go"
echo AUTO_CANDIDATE_GENERATION_R0=PASS
