#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TLALOC_ROOT="$(cd "$HERE/.." && pwd)"
CAMPAIGN="$TLALOC_ROOT/bin/tlaloc-real-vlm-campaign"

ENDPOINT="${TLALOC_ENDPOINT:-http://127.0.0.1:1234/v1}"
MODEL="${TLALOC_MODEL:-}"
PROGRAM="${ORIGAMI_TEMPORAL_PROGRAM:-}"
ORIGAMI_ROOT="${ORIGAMI_ROOT:-}"
OUT="${TLALOC_SMOKE_OUT:-$PWD/runs/real-vlm/local-smoke}"
RUN_RECORD_ROOT="${TLALOC_RUN_RECORD_ROOT:-$PWD/runs}"

usage() {
  cat <<'USAGE'
Usage: tlaloc real-vlm smoke [options]

Runs the existing Real VLM Campaign locally as an evidence smoke:
  doctor -> SMOKE -> Run Record replay verification

Defaults target LM Studio/OpenAI-compatible HTTP at http://127.0.0.1:1234/v1.
The canonical Origami temporal program is discovered from the tracked Origami
installer state, ORIGAMI_ROOT, or a nearby source checkout.

Options:
  --model ID              Exact loaded model id. Required only if endpoint exposes >1 model.
  --endpoint URL          OpenAI-compatible base URL.
  --program PATH          Canonical origami.temporal-program.r0 JSON.
  --origami-root PATH     Origami source checkout containing experiments/.
  --out PATH              Campaign output directory.
  --run-record-root PATH  Immutable Run Record root.
  -h, --help              Show this help.

Environment equivalents:
  TLALOC_MODEL, TLALOC_ENDPOINT, ORIGAMI_TEMPORAL_PROGRAM, ORIGAMI_ROOT,
  TLALOC_SMOKE_OUT, TLALOC_RUN_RECORD_ROOT.
USAGE
}

while (($#)); do
  case "$1" in
    --model) shift; (($#)) || { echo '--model requires a value' >&2; exit 2; }; MODEL="$1" ;;
    --endpoint) shift; (($#)) || { echo '--endpoint requires a value' >&2; exit 2; }; ENDPOINT="$1" ;;
    --program) shift; (($#)) || { echo '--program requires a value' >&2; exit 2; }; PROGRAM="$1" ;;
    --origami-root) shift; (($#)) || { echo '--origami-root requires a value' >&2; exit 2; }; ORIGAMI_ROOT="$1" ;;
    --out) shift; (($#)) || { echo '--out requires a value' >&2; exit 2; }; OUT="$1" ;;
    --run-record-root) shift; (($#)) || { echo '--run-record-root requires a value' >&2; exit 2; }; RUN_RECORD_ROOT="$1" ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

[[ -x "$CAMPAIGN" ]] || {
  echo "Tlaloc campaign binary is missing: $CAMPAIGN" >&2
  echo "Update the Tlaloc checkout and run ./install.sh first." >&2
  exit 1
}

ORIGAMI_STATE="${ORIGAMI_INSTALL_STATE:-$HOME/.local/share/origami/install-state-v1/manifest.tsv}"
ORIGAMI_PREFIX=""
if [[ -f "$ORIGAMI_STATE" ]]; then
  if [[ -z "$ORIGAMI_ROOT" ]]; then
    ORIGAMI_ROOT="$(awk -F '\t' '$1=="META" && $2=="project" {print $3; exit}' "$ORIGAMI_STATE")"
  fi
  ORIGAMI_PREFIX="$(awk -F '\t' '$1=="META" && $2=="prefix" {print $3; exit}' "$ORIGAMI_STATE")"
fi

if [[ -z "$PROGRAM" && -n "$ORIGAMI_ROOT" ]]; then
  candidate="$ORIGAMI_ROOT/experiments/temporal-automaton-r0/signal-chain.json"
  [[ -f "$candidate" ]] && PROGRAM="$candidate"
fi
if [[ -z "$PROGRAM" ]]; then
  for candidate in \
    "$PWD/origami/experiments/temporal-automaton-r0/signal-chain.json" \
    "$PWD/../origami/experiments/temporal-automaton-r0/signal-chain.json" \
    "$PWD/experiments/temporal-automaton-r0/signal-chain.json"; do
    if [[ -f "$candidate" ]]; then PROGRAM="$candidate"; break; fi
  done
fi
[[ -f "$PROGRAM" ]] || {
  echo "Could not find Origami's canonical temporal program." >&2
  echo "Install Origami from its current checkout, or pass --origami-root / --program." >&2
  exit 1
}

find_origami_binary() {
  local name="$1" resolved=""
  resolved="$(command -v "$name" 2>/dev/null || true)"
  if [[ -n "$resolved" ]]; then printf '%s\n' "$resolved"; return 0; fi
  if [[ -n "$ORIGAMI_PREFIX" && -x "$ORIGAMI_PREFIX/bin/$name" ]]; then
    printf '%s\n' "$ORIGAMI_PREFIX/bin/$name"
    return 0
  fi
  return 1
}

CARRIER="$(find_origami_binary origami-temporal-carrier || true)"
BUILDER="$(find_origami_binary origami-candidate-build || true)"
[[ -x "$CARRIER" && -x "$BUILDER" ]] || {
  echo "Origami temporal binaries are not installed/current." >&2
  echo "From the Origami checkout run: ./install.sh" >&2
  exit 1
}

mkdir -p "$OUT" "$RUN_RECORD_ROOT"
COMMON=(
  --endpoint "$ENDPOINT"
  --program "$PROGRAM"
  --carrier "$CARRIER"
  --builder "$BUILDER"
  --out "$OUT"
  --run-record-root "$RUN_RECORD_ROOT"
)
if [[ -n "$MODEL" ]]; then COMMON+=(--model "$MODEL"); fi

printf '=== TLALOC LOCAL REAL-VLM DOCTOR ===\n'
"$CAMPAIGN" doctor "${COMMON[@]}"

printf '\n=== TLALOC LOCAL REAL-VLM SMOKE ===\n'
export TLALOC_RUN_RECORD_REPLAY_EXECUTABLE="$CAMPAIGN"
"$CAMPAIGN" run --phase SMOKE "${COMMON[@]}"

INDEX="$RUN_RECORD_ROOT/index.jsonl"
[[ -s "$INDEX" ]] || { echo "Run Record index was not created: $INDEX" >&2; exit 1; }
RECORD_REL="$(tail -n 1 "$INDEX" | sed -n 's/.*"path":"\([^"]*\)".*/\1/p')"
[[ -n "$RECORD_REL" ]] || { echo "Could not resolve latest Run Record from $INDEX" >&2; exit 1; }
RECORD="$RUN_RECORD_ROOT/$RECORD_REL"
[[ -f "$RECORD" ]] || { echo "Run Record is missing: $RECORD" >&2; exit 1; }

printf '\n=== RUN RECORD REPLAY (C3) ===\n'
"$CAMPAIGN" replay-record --record "$RECORD"

printf '\nLOCAL_REAL_VLM_SMOKE=PASS\n'
printf 'ENDPOINT=%s\n' "$ENDPOINT"
printf 'MODEL=%s\n' "${MODEL:-auto}"
printf 'PROGRAM=%s\n' "$PROGRAM"
printf 'RUN_RECORD=%s\n' "$RECORD"
printf 'CAMPAIGN_OUT=%s\n' "$OUT"
