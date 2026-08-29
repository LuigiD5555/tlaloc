#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - <<'PY' "$ROOT/gatekeeper.json"
import json,sys
p=json.load(open(sys.argv[1]))
assert p['schema']=='tonal.gatekeeper.r0'
assert p['owner']=='LuigiD5555'
assert p['authority']=='LuigiD5555/tonal'
assert p['policy']['OWNER']['may_explicitly_override_promotion_gate'] is True
assert p['policy']['EXTERNAL']['may_explicitly_override_promotion_gate'] is False
assert 'owner_approval' in p['policy']['EXTERNAL']['requirements']
PY
grep -q 'pull_request_review:' "$ROOT/.github/workflows/gatekeeper.yml"
echo 'PASS Tlaloc gatekeeper mirror'
