#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY' "$ROOT/behavior-lab/spec/REAL_VLM_CAMPAIGN_R0.json"
import json,sys
s=json.load(open(sys.argv[1]))
assert s['schema']=='tlaloc.real-vlm-campaign.r0'
assert s['phases']['SMOKE']['promotion_eligible'] is False
assert s['phases']['EVIDENCE']['promotion_eligible'] is False
assert s['phases']['EVIDENCE']['minimum_trials_per_model'] >= 3
inv=set(s['invariants'])
for x in [
  'CANONICAL_SIGNAL_CHAIN_GROUND_TRUTH_REQUIRED',
  'REAL_CAMPAIGN_REJECTS_SYNTHETIC_MODEL_ID',
  'MULTIMODAL_TRANSPORT_PROBE_REQUIRED_BEFORE_RUN',
  'SMOKE_NE_PROMOTION_EVIDENCE',
  'SINGLE_MODEL_REPEATED_EVIDENCE_NE_CROSS_MODEL_EVIDENCE',
  'PROMOTION_ELIGIBLE_FALSE_IN_R0',
  'TRANSPORT_FAILURE_NE_SEMANTIC_FAILURE',
  'EXPERIMENTAL_INCUMBENT_NE_CANONICAL_ORIGAMI',
]: assert x in inv,x
assert s['origami_contract']['expected_version']=='6.0.0-alpha.15'
assert s['origami_contract']['parent_profile']=='origami.temporal-carrier.r0.profile-1'
PY
grep -q 'Real VLM Campaign R0' "$ROOT/docs/REAL_VLM_CAMPAIGN_R0.md"
grep -q 'tlaloc-real-vlm-campaign' "$ROOT/install.sh"
grep -q 'tlaloc-real-vlm-campaign' "$ROOT/uninstall.sh"
grep -q 'PhaseEvidence' "$ROOT/behavior-lab/internal/realcampaign/normalize.go"
grep -q 'PromotionEligible: false' "$ROOT/behavior-lab/internal/realcampaign/prepare.go"
grep -q 'promotion_eligible = false' "$ROOT/docs/REAL_VLM_CAMPAIGN_R0.md"
echo REAL_VLM_CAMPAIGN_R0=PASS
