#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TLALOC_VERSION="$(tr -d '\r\n' < "$HERE/VERSION")"
[[ -n "$TLALOC_VERSION" ]] || { echo "VERSION is empty" >&2; exit 1; }
CLEAN_LEGACY=0
YES=0
PRUNE_OLD=0

HOME_DIR="${HOME:?HOME is required}"
DATA_HOME="${XDG_DATA_HOME:-$HOME_DIR/.local/share}"
STATE_HOME="${XDG_STATE_HOME:-$HOME_DIR/.local/state}"
BIN_HOME="${TLALOC_BIN_HOME:-$HOME_DIR/.local/bin}"
TLALOC_ROOT="$DATA_HOME/tlaloc"
TLALOC_DST="$TLALOC_ROOT/versions/$TLALOC_VERSION"

usage() {
  cat <<'USAGE'
Usage: ./install.sh [options]

  --clean-legacy    After successful install, remove high-confidence legacy Origami/OHF residue
  --prune-old       Remove older managed Tlaloc versions after successful install
  --yes             Required for --clean-legacy; suppress confirmation
  -h, --help        Show help

Installs Tlaloc only, entirely under XDG user directories; sudo is not required.
Origami is independent and optional. BPFW/PipeCraft are external and never modified.
Learning memory and blackboard runtime evidence are under XDG_STATE_HOME and are preserved across version upgrades.
USAGE
}
while (($#)); do
  case "$1" in
    --clean-legacy) CLEAN_LEGACY=1 ;;
    --prune-old) PRUNE_OLD=1 ;;
    --yes) YES=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done
if [[ "$CLEAN_LEGACY" -eq 1 && "$YES" -ne 1 ]]; then
  echo "--clean-legacy requires --yes. Run ./tools/legacy-cleanup.sh --scan first." >&2
  exit 2
fi

for req in cp mkdir ln sha256sum readlink go tar; do command -v "$req" >/dev/null 2>&1 || { echo "Missing required command: $req" >&2; exit 1; }; done
[[ -d "$HERE/behavior-lab" ]] || { echo "Missing behavior-lab source" >&2; exit 1; }
[[ -x "$HERE/tools/tlaloc" && -x "$HERE/tools/doctor.sh" && -x "$HERE/tools/legacy-cleanup.sh" ]] || { echo "Missing Tlaloc tools" >&2; exit 1; }

mkdir -p "$BIN_HOME" "$TLALOC_ROOT/versions"
printf 'managed-by=tlaloc-installer-v1\n' > "$TLALOC_ROOT/.tlaloc-managed-v1"
rm -rf -- "$TLALOC_DST.tmp"
mkdir -p "$TLALOC_DST.tmp"

(
  cd "$HERE"
  tar --exclude='./.git' --exclude='./.github' -cf - .
) | (cd "$TLALOC_DST.tmp" && tar -xf -)
mkdir -p "$TLALOC_DST.tmp/bin" "$TLALOC_DST.tmp/tools"

(
  cd "$TLALOC_DST.tmp/behavior-lab"
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-behavior-lab" ./cmd/behaviorlab
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-origami" ./cmd/tlaloc-origami
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-perception-campaign" ./cmd/tlaloc-perception-campaign
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-visual-search" ./cmd/tlaloc-visual-search
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-native-eval" ./cmd/tlaloc-native-eval
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-protocol-eval" ./cmd/tlaloc-protocol-eval
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-automaton-distill" ./cmd/tlaloc-automaton-distill
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-temporal-bench" ./cmd/tlaloc-temporal-bench
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-learning-memory" ./cmd/tlaloc-learning-memory
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-adaptive-search" ./cmd/tlaloc-adaptive-search
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-closed-loop" ./cmd/tlaloc-closed-loop
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-real-vlm-campaign" ./cmd/tlaloc-real-vlm-campaign
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-tlaloque-swarm" ./cmd/tlaloc-tlaloque-swarm
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-learn" ./cmd/tlaloc-learn
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-prompt" ./cmd/tlaloc-prompt
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-lfm2-worker" ./cmd/tlaloc-lfm2-worker
  CGO_ENABLED=0 go build -trimpath -o "$TLALOC_DST.tmp/bin/tlaloc-lfm2-boundary" ./cmd/tlaloc-lfm2-boundary
)
cp -a "$HERE/tools/tlaloc" "$TLALOC_DST.tmp/bin/tlaloc"
cp -a "$HERE/tools/doctor.sh" "$TLALOC_DST.tmp/tools/doctor.sh"
cp -a "$HERE/tools/legacy-cleanup.sh" "$TLALOC_DST.tmp/tools/legacy-cleanup.sh"
cp -a "$HERE/uninstall.sh" "$TLALOC_DST.tmp/tools/uninstall.sh"
chmod +x "$TLALOC_DST.tmp/bin/"* "$TLALOC_DST.tmp/tools/"*
printf 'Tlaloc\t%s\n' "$TLALOC_VERSION" > "$TLALOC_DST.tmp/.tlaloc-managed-version"
(
  cd "$TLALOC_DST.tmp"
  find . -type f ! -name INSTALL_MANIFEST.sha256 -print0 | sort -z | xargs -0 sha256sum > INSTALL_MANIFEST.sha256
)
rm -rf -- "$TLALOC_DST"
mv "$TLALOC_DST.tmp" "$TLALOC_DST"
ln -sfn "$TLALOC_DST" "$TLALOC_ROOT/current"
for b in tlaloc tlaloc-behavior-lab tlaloc-origami tlaloc-perception-campaign tlaloc-visual-search tlaloc-native-eval tlaloc-protocol-eval tlaloc-automaton-distill tlaloc-temporal-bench tlaloc-learning-memory tlaloc-adaptive-search tlaloc-closed-loop tlaloc-real-vlm-campaign tlaloc-learn tlaloc-prompt tlaloc-lfm2-worker tlaloc-lfm2-boundary; do
  ln -sfn "$TLALOC_DST/bin/$b" "$BIN_HOME/$b"
done
ln -sfn "$TLALOC_DST/tools/uninstall.sh" "$BIN_HOME/tlaloc-uninstall"

rm -f -- "$STATE_HOME/tlaloc/install-manifest-v1.tsv"
rmdir "$STATE_HOME/tlaloc" 2>/dev/null || true

if [[ "$PRUNE_OLD" -eq 1 ]]; then
  find "$TLALOC_ROOT/versions" -mindepth 1 -maxdepth 1 -type d ! -path "$TLALOC_DST" -exec rm -rf -- {} +
fi

"$TLALOC_DST/tools/doctor.sh"
echo
echo "Installed Tlaloc $TLALOC_VERSION -> $TLALOC_DST"
echo "CLI dir: $BIN_HOME"
case ":$PATH:" in *":$BIN_HOME:"*) ;; *) echo "NOTE: add $BIN_HOME to PATH" ;; esac

"$TLALOC_DST/tools/legacy-cleanup.sh" --scan || true
if [[ "$CLEAN_LEGACY" -eq 1 ]]; then
  "$TLALOC_DST/tools/legacy-cleanup.sh" --remove --yes
fi
