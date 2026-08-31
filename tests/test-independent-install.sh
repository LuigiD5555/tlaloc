#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
export HOME="$TMP/home"
export XDG_DATA_HOME="$HOME/.local/share"
export XDG_STATE_HOME="$HOME/.local/state"
export XDG_CONFIG_HOME="$HOME/.config"
export XDG_CACHE_HOME="$HOME/.cache"
mkdir -p "$HOME"
EXPECTED_VERSION="$(tr -d '\r\n' < "$ROOT/VERSION")"
PATH="$HOME/.local/bin:$PATH" "$ROOT/install.sh"
[[ -L "$HOME/.local/share/tlaloc/current" ]]
CURRENT_TARGET="$(readlink -f "$HOME/.local/share/tlaloc/current")"
[[ "$CURRENT_TARGET" == "$HOME/.local/share/tlaloc/versions/$EXPECTED_VERSION" ]]
grep -qx $'Tlaloc\t'"$EXPECTED_VERSION" "$CURRENT_TARGET/.tlaloc-managed-version"
PATH="$HOME/.local/bin:$PATH" tlaloc version | grep -qx "Tlaloc $EXPECTED_VERSION"
for cli in tlaloc tlaloc-behavior-lab tlaloc-origami tlaloc-perception-campaign tlaloc-visual-search tlaloc-native-eval tlaloc-protocol-eval tlaloc-automaton-distill tlaloc-temporal-bench tlaloc-learning-memory tlaloc-adaptive-search tlaloc-closed-loop tlaloc-real-vlm-campaign tlaloc-learn tlaloc-prompt tlaloc-uninstall; do
  [[ -L "$HOME/.local/bin/$cli" ]] || { echo "missing managed CLI: $cli" >&2; exit 1; }
  [[ -x "$(readlink -f "$HOME/.local/bin/$cli")" ]] || { echo "managed CLI is not executable: $cli" >&2; exit 1; }
done
PATH="$HOME/.local/bin:$PATH" tlaloc-learning-memory summary > "$TMP/memory-summary.json"
python3 - <<'PY' "$TMP/memory-summary.json"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.learning-memory.r0.summary'
assert r['total_events']==0
PY
PATH="$HOME/.local/bin:$PATH" tlaloc-adaptive-search plan > "$TMP/adaptive-plan.json"
python3 - <<'PY' "$TMP/adaptive-plan.json"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.adaptive-search.r0.plan'
assert r['adaptive'] is False
assert len(r['mutation_priorities']) > 0
assert abs(sum(x['weight'] for x in r['mutation_priorities'])-1.0) < 1e-9
PY
PATH="$HOME/.local/bin:$PATH" tlaloc-learn status -genome "$CURRENT_TARGET/behavior-lab/profiles/prompt-genome-r1.json" > "$TMP/learn-status.json"
python3 - <<'PY' "$TMP/learn-status.json"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.learning-cycle.r1.status'
assert r['policy']['schema']=='tlaloc.learning-policy.r1'
assert any(x['target']=='SEMANTIC_PARITY_GATE' for x in r['policy']['rules'] if x['kind']=='REQUIRE')
PY
PATH="$HOME/.local/bin:$PATH" tlaloc-prompt compile -genome "$CURRENT_TARGET/behavior-lab/profiles/prompt-genome-r1.json" -relevant TEMPORAL_GRAMMAR,EXECUTION_POLICY > "$TMP/prompt.json"
python3 - <<'PY' "$TMP/prompt.json"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.compiled-master-prompt.r1'
assert 'TEMPORAL_GRAMMAR' in [x['id'] for x in r['modules']]
assert 'EXECUTION_POLICY' in [x['id'] for x in r['modules']]
PY
PATH="$HOME/.local/bin:$PATH" tlaloc-closed-loop example > "$TMP/closed-loop-example.json"
python3 - <<'PY' "$TMP/closed-loop-example.json"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.closed-experimental-loop.r0.config'
assert r['models'][0]['provider']=='OPENAI_COMPAT'
assert r['diagnostic_retries'] is True
assert 'NATIVE_PNG_ONLY' in r['conditions']
PY
PATH="$HOME/.local/bin:$PATH" tlaloc-real-vlm-campaign example > "$TMP/real-vlm-example.json"
python3 - <<'PY' "$TMP/real-vlm-example.json"
import json,sys
r=json.load(open(sys.argv[1]))
assert r['schema']=='tlaloc.real-vlm-campaign.r0.spec'
assert r['phase']=='SMOKE'
assert r['endpoint']=='http://127.0.0.1:1234/v1'
assert r['temporal_carrier']=='origami-temporal-carrier'
assert r['candidate_builder']=='origami-candidate-build'
PY
PATH="$HOME/.local/bin:$PATH" tlaloc skills list | grep -qx 'tlaloc-project'
if PATH="$HOME/.local/bin:$PATH" tlaloc skills list | grep -qx 'repo-flow'; then
  echo "repo-flow must not be distributed by Tlaloc" >&2
  exit 1
fi
[[ ! -e "$CURRENT_TARGET/.claude/skills/repo-flow" ]]
mkdir -p "$TMP/project"
git -C "$TMP/project" init -q
(
  cd "$TMP/project"
  PATH="$HOME/.local/bin:$PATH" tlaloc skills install tlaloc-project >/dev/null
)
cmp "$CURRENT_TARGET/.claude/skills/tlaloc-project/SKILL.md" "$TMP/project/.claude/skills/tlaloc-project/SKILL.md"
if PATH="$HOME/.local/bin:$PATH" tlaloc skills install repo-flow --project "$TMP/project" >"$TMP/repo-flow.out" 2>&1; then
  echo "Tlaloc unexpectedly installed Tonal-owned repo-flow" >&2
  exit 1
fi
grep -q 'repo-flow moved to Tonal' "$TMP/repo-flow.out"
[[ ! -e "$HOME/.local/share/origami/current" ]]
PATH="$HOME/.local/bin:$PATH" tlaloc doctor

# Simulate the current standalone Origami installer contract. Tlaloc must
# recognize and protect it even though there is no origami/versions/current.
ORIGAMI_PROJECT="$TMP/origami-project"
ORIGAMI_STATE="$XDG_DATA_HOME/origami/install-state-v1/manifest.tsv"
mkdir -p "$ORIGAMI_PROJECT" "$(dirname "$ORIGAMI_STATE")" "$HOME/.local/bin"
printf '6.0.0-alpha.15\n' > "$ORIGAMI_PROJECT/VERSION"
for name in origami-fixed-carrier origami-temporal-carrier origami-candidate-build ohf-lab; do
  printf '#!/usr/bin/env sh\nexit 0\n' > "$HOME/.local/bin/$name"
  chmod +x "$HOME/.local/bin/$name"
done
{
  printf 'META\tformat\t1\t-\t-\t-\n'
  printf 'META\tproject\t%s\t-\t-\t-\n' "$ORIGAMI_PROJECT"
  for name in origami-fixed-carrier origami-temporal-carrier origami-candidate-build ohf-lab; do
    printf 'BIN\t%s\t%s\tdeadbeef\t0\t-\n' "$name" "$HOME/.local/bin/$name"
  done
} > "$ORIGAMI_STATE"
PATH="$HOME/.local/bin:$PATH" tlaloc doctor > "$TMP/doctor-with-origami.out"
grep -q 'PASS  Standalone Origami installation detected: 6.0.0-alpha.15' "$TMP/doctor-with-origami.out"
PATH="$HOME/.local/bin:$PATH" tlaloc legacy-scan > "$TMP/legacy-scan.out"
grep -q 'standalone Origami install: PROTECTED' "$TMP/legacy-scan.out"
if grep -Fq "$HOME/.local/bin/origami-fixed-carrier" "$TMP/legacy-scan.out"; then
  echo "standalone Origami binary was incorrectly classified as legacy" >&2
  exit 1
fi
if grep -Fq "$HOME/.local/bin/ohf-lab" "$TMP/legacy-scan.out"; then
  echo "standalone OHF binary tracked by Origami was incorrectly classified as legacy" >&2
  exit 1
fi

mkdir -p "$XDG_STATE_HOME/tlaloc/learning-memory"
printf 'preserve-me\n' > "$XDG_STATE_HOME/tlaloc/learning-memory/uninstall-probe"
PATH="$HOME/.local/bin:$PATH" tlaloc-uninstall --yes
[[ ! -e "$HOME/.local/share/tlaloc/current" ]]
for cli in tlaloc tlaloc-behavior-lab tlaloc-origami tlaloc-perception-campaign tlaloc-visual-search tlaloc-native-eval tlaloc-protocol-eval tlaloc-automaton-distill tlaloc-temporal-bench tlaloc-learning-memory tlaloc-adaptive-search tlaloc-closed-loop tlaloc-real-vlm-campaign tlaloc-learn tlaloc-prompt tlaloc-uninstall; do
  [[ ! -e "$HOME/.local/bin/$cli" ]] || { echo "uninstall left managed CLI: $cli" >&2; exit 1; }
done
[[ -x "$HOME/.local/bin/origami-fixed-carrier" ]]
[[ -x "$HOME/.local/bin/ohf-lab" ]]
grep -qx 'preserve-me' "$XDG_STATE_HOME/tlaloc/learning-memory/uninstall-probe"