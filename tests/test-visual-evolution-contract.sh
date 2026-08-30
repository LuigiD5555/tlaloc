#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY'
import json
from pathlib import Path
p = Path('behavior-lab/spec/ORIGAMI_VISUAL_EVOLUTION_R0.json')
d = json.loads(p.read_text())
assert d['contract_id'] == 'tlaloc.origami-visual-search.r0'
assert d['base_origami_profile'] == 'origami.canonical-aesthetic.r0'
assert 'PROMPT' in d['candidate_mutations']
assert 'NUMERIC_STRUCTURE' in d['candidate_mutations']
assert 'PRIME_DERIVED_SPACING' in d['numeric_structure_examples']
assert d['default_policy']['max_carrier_bytes'] == 512000
assert d['default_policy']['max_mean_context_tokens'] == 4000
assert d['default_policy']['min_semantic_roundtrip_rate'] == 1.0
assert d['default_policy']['min_real_models_for_perception'] >= 3
assert 'TLALOC_RECOMMENDATION_ONLY' in d['hard_invariants']
assert 'ORIGAMI_VALIDATES' in d['hard_invariants']
assert 'TONAL_PROMOTES' in d['hard_invariants']
assert 'FALSE_EXACT=0' in d['hard_invariants']
print('visual-evolution-contract: PASS')
PY
