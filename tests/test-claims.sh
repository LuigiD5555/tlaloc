#!/usr/bin/env bash
set -euo pipefail

REPOSITORY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLAIMS_TOOL="$REPOSITORY_ROOT/tools/claims.py"
CAPABILITY_DOCUMENT="$REPOSITORY_ROOT/docs/CAPABILITY_STATUS.md"

python3 "$CLAIMS_TOOL" validate
python3 "$CLAIMS_TOOL" check

python3 - "$REPOSITORY_ROOT" <<'PY'
import copy
import sys
from pathlib import Path

repository_root = Path(sys.argv[1])
sys.path.insert(0, str(repository_root))
from tools import claims as claims_module

baseline_claims = claims_module.load_claims(repository_root / "state" / "CLAIMS.json")

duplicate_claims = copy.deepcopy(baseline_claims)
duplicate_claims.append(copy.deepcopy(duplicate_claims[0]))
assert any("duplicate claim id" in error for error in claims_module.validate_claims(duplicate_claims))

invalid_status_claims = copy.deepcopy(baseline_claims)
invalid_status_claims[0]["status"] = "claimed"
assert any("status is not allowed" in error for error in claims_module.validate_claims(invalid_status_claims))

missing_evidence_claims = copy.deepcopy(baseline_claims)
implemented_claim = next(
    claim for claim in missing_evidence_claims if claim["status"] == "implemented"
)
implemented_claim["evidence"] = []
assert any("requires evidence" in error for error in claims_module.validate_claims(missing_evidence_claims))

missing_test_claims = copy.deepcopy(baseline_claims)
implemented_claim = next(
    claim for claim in missing_test_claims if claim["status"] == "implemented"
)
implemented_claim["evidence"] = [
    "test:behavior-lab/internal/compiler:TestThatDoesNotExist"
]
assert any("function does not exist" in error for error in claims_module.validate_claims(missing_test_claims))
PY

test_directory="$(mktemp -d)"
drifted_document="$test_directory/CAPABILITY_STATUS.md"
cp "$CAPABILITY_DOCUMENT" "$drifted_document"
sed -i '0,/`implemented`/s//`designed`/' "$drifted_document"

if python3 "$CLAIMS_TOOL" check --document "$drifted_document" >/dev/null 2>&1; then
  echo "claims check accepted a manually edited generated table" >&2
  exit 1
fi

python3 "$CLAIMS_TOOL" generate --document "$drifted_document" >/dev/null
python3 "$CLAIMS_TOOL" check --document "$drifted_document" >/dev/null
echo PASS
