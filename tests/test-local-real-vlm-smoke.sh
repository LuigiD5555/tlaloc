#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

FAKE_ROOT="$TMP/tlaloc"
FAKE_HOME="$TMP/home"
ORIGAMI_ROOT="$TMP/origami"
PREFIX="$FAKE_HOME/.local"
RUNS="$TMP/runs"
OUT="$TMP/out"
mkdir -p "$FAKE_ROOT/tools" "$FAKE_ROOT/bin" "$PREFIX/bin" \
  "$PREFIX/share/origami/install-state-v1" \
  "$ORIGAMI_ROOT/experiments/temporal-automaton-r0"
cp "$ROOT/tools/local-real-vlm-smoke.sh" "$FAKE_ROOT/tools/"
chmod +x "$FAKE_ROOT/tools/local-real-vlm-smoke.sh"

cat > "$ORIGAMI_ROOT/experiments/temporal-automaton-r0/signal-chain.json" <<'JSON'
{"schema":"origami.temporal-program.r0","id":"fixture"}
JSON
cat > "$PREFIX/share/origami/install-state-v1/manifest.tsv" <<EOF
META	format	1	-	-	-
META	prefix	$PREFIX	-	-	-
META	project	$ORIGAMI_ROOT	-	-	-
EOF
for name in origami-temporal-carrier origami-candidate-build; do
  cat > "$PREFIX/bin/$name" <<'SH'
#!/usr/bin/env sh
exit 0
SH
  chmod +x "$PREFIX/bin/$name"
done

cat > "$FAKE_ROOT/bin/tlaloc-real-vlm-campaign" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
action="${1:-}"; shift || true
printf '%s\n' "$action" >> "${FAKE_CAMPAIGN_LOG:?}"
case "$action" in
  doctor)
    printf '{"ready":true,"selected_model":"fixture-model"}\n'
    ;;
  run)
    root=""
    while (($#)); do
      case "$1" in
        --run-record-root) shift; root="$1" ;;
      esac
      shift || true
    done
    [[ -n "$root" ]]
    mkdir -p "$root/2026-09"
    printf '{"run_id":"fixture","env_hash":"sha256:fixture"}\n' > "$root/2026-09/fixture.json"
    printf '{"run_id":"fixture","env_hash":"sha256:fixture","path":"2026-09/fixture.json","verdict":"verify_pass"}\n' >> "$root/index.jsonl"
    printf '{"report":"ok"}\n'
    ;;
  replay-record)
    [[ "${1:-}" == "--record" && -f "${2:-}" ]]
    printf '{"verified":true}\n'
    ;;
  *) exit 2 ;;
esac
SH
chmod +x "$FAKE_ROOT/bin/tlaloc-real-vlm-campaign"

export HOME="$FAKE_HOME"
export FAKE_CAMPAIGN_LOG="$TMP/campaign.log"
export PATH="$PREFIX/bin:/usr/bin:/bin"
"$FAKE_ROOT/tools/local-real-vlm-smoke.sh" \
  --out "$OUT" \
  --run-record-root "$RUNS" > "$TMP/output.txt"

grep -q '^LOCAL_REAL_VLM_SMOKE=PASS$' "$TMP/output.txt"
grep -Fq "PROGRAM=$ORIGAMI_ROOT/experiments/temporal-automaton-r0/signal-chain.json" "$TMP/output.txt"
grep -Fq "RUN_RECORD=$RUNS/2026-09/fixture.json" "$TMP/output.txt"
[[ "$(tr '\n' ' ' < "$FAKE_CAMPAIGN_LOG")" == "doctor run replay-record " ]]

echo PASS
