#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
python3 - <<'PY'
import json
from pathlib import Path
p = Path('behavior-lab/spec/PROMPT_FIRST_DISTILLATION_R0.json')
d = json.loads(p.read_text())
assert d['contract_id'] == 'tlaloc.prompt-first-distillation.r0'
assert d['portable_default'] == 'L0_PROMPT_ONLY'
assert d['swarm_role'] == 'REFERENCE_LABORATORY_NOT_DEFAULT_PRODUCTION_RUNTIME'
assert d['distillation_target'] == 'BEHAVIOR_NOT_TRACE_TEXT'
levels = [x['level'] for x in d['deployment_ladder']]
assert levels == ['L0','L1','L2','L3','L4']
assert d['deployment_ladder'][0]['artifact'] == 'PROMPT_ONLY'
assert d['deployment_ladder'][0]['allowed_dependencies'] == ['LLM_TEXT_INTERFACE']
assert 'ORIGAMI_IS_A_TARGET_NOT_TLALOC_IDENTITY' in d['hard_invariants']
assert 'L0_REQUIRES_NO_SANDBOX' in d['hard_invariants']
assert 'L0_REQUIRES_NO_TOOLS' in d['hard_invariants']
assert 'L0_REQUIRES_NO_TLALOC_RUNTIME' in d['hard_invariants']
print('prompt-first-contract: PASS')
PY
