#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
python3 - <<'PY'
import json
from pathlib import Path
p = Path('behavior-lab/spec/ORIGAMI_VISUAL_EVOLUTION_R0.json')
d = json.loads(p.read_text())
assert d['contract_id'] == 'tlaloc.origami-visual-search.r0'
assert d['base_origami_profile'] == 'origami.canonical-aesthetic.r0'
assert d['tlaloc_role'].startswith('DEVELOPMENT_KIT')
assert d['tonal_role'].startswith('OPTIONAL_MULTI_TOOL')
for kind in ['PROMPT','NUMERIC_STRUCTURE','INTERFERENCE_STRUCTURE','DEPTH_STRUCTURE','TEMPORAL_STRUCTURE','EMERGENT_STRUCTURE']:
    assert kind in d['candidate_mutations']
assert 'PRIME_DERIVED_SPACING' in d['numeric_structure_examples']
assert 'MOIRE' in d['perceptual_operation_examples']['interference']
assert 'STEREO_BIND' in d['perceptual_operation_examples']['depth']
assert d['default_policy']['max_carrier_bytes'] == 512000
assert d['default_policy']['max_mean_context_tokens'] == 4000
assert d['default_policy']['min_semantic_roundtrip_rate'] == 1.0
assert d['default_policy']['min_perceptual_reveal_rate'] >= .95
assert d['default_policy']['min_real_models_for_perception'] >= 3
assert 'TLALOC_RECOMMENDATION_ONLY' in d['hard_invariants']
assert 'ORIGAMI_OWNS_PROFILE_PROMOTION' in d['hard_invariants']
assert 'TONAL_COMPOSES_DEVELOPMENT_TOOLCHAINS_NOT_ORIGAMI_SEMANTICS' in d['hard_invariants']
assert 'FALSE_EXACT=0' in d['hard_invariants']
print('visual-evolution-contract: PASS')
PY
