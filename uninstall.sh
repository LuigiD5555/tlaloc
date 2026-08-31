#!/usr/bin/env bash
set -euo pipefail

MODE=current
YES=0
ALL_MANAGED=0
CLEAN_SHELL=0
AGGRESSIVE=0
SYSTEM=0
TLALOC_ONLY=1
ORIGAMI_ONLY=0
INVOKED_AS="$(basename "$0")"
case "$INVOKED_AS" in
  tlaloc-uninstall) TLALOC_ONLY=1 ;;
  origami-uninstall) ORIGAMI_ONLY=1 ;;
esac

HOME_DIR="${HOME:?HOME is required}"
DATA_HOME="${XDG_DATA_HOME:-$HOME_DIR/.local/share}"
STATE_HOME="${XDG_STATE_HOME:-$HOME_DIR/.local/state}"
BIN_HOME="${TLALOC_BIN_HOME:-$HOME_DIR/.local/bin}"
TLALOC_ROOT="$DATA_HOME/tlaloc"
ORIGAMI_ROOT="$DATA_HOME/origami"

usage() {
  cat <<'USAGE'
Usage: uninstall.sh [mode] [options]

Modes:
  --current               Remove current managed Tlaloc (default)
  --legacy                Remove legacy Origami/OHF/VCL system-install residue only
  --all                   Remove managed installation(s) and legacy residue

Options:
  --all-managed-versions  Remove every selected managed version, not only current
  --tlaloc-only           Limit managed removal to Tlaloc
  --origami-only          Limit managed removal to Origami
  --bundle                Remove both managed components (overrides entry-point default)
  --clean-shell           Back up and clean legacy shell env/alias lines
  --aggressive-legacy     Include medium-confidence VCL-only artifacts
  --system                Include /usr/local legacy scan/removal
  --yes                   Do not ask for confirmation
  -h, --help              Show this help

Safety invariants:
  * Never removes BPFW/PipeCraft.
  * Never removes .me/origami project workspaces.
  * Never removes Tlaloc learning memory under XDG_STATE_HOME/tlaloc/learning-memory.
  * Managed deletion is constrained to version roots carrying component ownership markers.
USAGE
}

while (($#)); do
  case "$1" in
    --current) MODE=current ;;
    --legacy) MODE=legacy ;;
    --all) MODE=all ;;
    --all-managed-versions) ALL_MANAGED=1 ;;
    --tlaloc-only) TLALOC_ONLY=1; ORIGAMI_ONLY=0 ;;
    --origami-only) ORIGAMI_ONLY=1; TLALOC_ONLY=0 ;;
    --bundle) TLALOC_ONLY=0; ORIGAMI_ONLY=0 ;;
    --clean-shell) CLEAN_SHELL=1 ;;
    --aggressive-legacy) AGGRESSIVE=1 ;;
    --system) SYSTEM=1 ;;
    --yes) YES=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

if [[ "$TLALOC_ONLY" -eq 1 && "$ORIGAMI_ONLY" -eq 1 ]]; then
  echo "Choose only one of --tlaloc-only / --origami-only." >&2; exit 2
fi

confirm() {
  [[ "$YES" -eq 1 ]] && return 0
  printf 'Proceed with uninstall mode %s? [y/N] ' "$MODE" >&2
  read -r ans
  [[ "$ans" =~ ^[Yy]$ ]]
}

safe_realpath() { readlink -f "$1" 2>/dev/null || true; }

remove_bin_link_if_owned() {
  local p="$1" component_root="$2" target=""
  [[ -L "$p" ]] || return 0
  target="$(safe_realpath "$p")"
  case "$target" in "$component_root/versions/"*) rm -f -- "$p"; echo "REMOVED link: $p" ;; esac
}

tlaloc_bins() {
  printf '%s\n' tlaloc tlaloc-behavior-lab tlaloc-origami tlaloc-perception-campaign tlaloc-visual-search tlaloc-native-eval tlaloc-protocol-eval tlaloc-automaton-distill tlaloc-temporal-bench tlaloc-learning-memory tlaloc-adaptive-search tlaloc-closed-loop tlaloc-real-vlm-campaign tlaloc-learn tlaloc-prompt tlaloc-uninstall
}

remove_component_current() {
  local name="$1" root="$2" current target=""
  current="$root/current"
  [[ -L "$current" ]] || { echo "INFO  no current $name installation"; return 0; }
  target="$(safe_realpath "$current")"
  case "$target" in
    "$root/versions/"*) ;;
    *) echo "REFUSE  unexpected $name current target: $target" >&2; return 1 ;;
  esac
  if [[ "$name" == tlaloc ]]; then
    [[ -f "$target/.tlaloc-managed-version" ]] || { echo "REFUSE  missing Tlaloc managed marker: $target" >&2; return 1; }
  else
    [[ -f "$target/.origami-managed-version" || -f "$target/.tlaloc-managed-version" ]] || { echo "REFUSE  missing Origami managed marker: $target" >&2; return 1; }
  fi

  if [[ "$name" == tlaloc ]]; then
    while IFS= read -r b; do remove_bin_link_if_owned "$BIN_HOME/$b" "$root"; done < <(tlaloc_bins)
  else
    remove_bin_link_if_owned "$BIN_HOME/origami" "$root"
    remove_bin_link_if_owned "$BIN_HOME/origami-uninstall" "$root"
  fi
  rm -f -- "$current"
  rm -rf -- "$target"
  echo "REMOVED $name version: $target"
}

remove_all_managed_component() {
  local name="$1" root="$2"
  if [[ "$name" == tlaloc ]]; then
    [[ -f "$root/.tlaloc-managed-v1" ]] || { echo "INFO  no managed $name root"; return 0; }
  else
    [[ -f "$root/.origami-managed-v1" || -f "$root/.tlaloc-managed-v1" ]] || { echo "INFO  no managed $name root"; return 0; }
  fi
  if [[ "$name" == tlaloc ]]; then
    while IFS= read -r b; do remove_bin_link_if_owned "$BIN_HOME/$b" "$root"; done < <(tlaloc_bins)
  else
    remove_bin_link_if_owned "$BIN_HOME/origami" "$root"
    remove_bin_link_if_owned "$BIN_HOME/origami-uninstall" "$root"
  fi
  rm -rf -- "$root/versions" "$root/current"
  if [[ "$name" == tlaloc ]]; then
    rm -f -- "$root/.tlaloc-managed-v1"
    rm -f -- "$STATE_HOME/tlaloc/install-manifest-v1.tsv"
    rmdir "$STATE_HOME/tlaloc" 2>/dev/null || true
  else
    rm -f -- "$root/.origami-managed-v1" "$root/.tlaloc-managed-v1"
  fi
  rmdir "$root" 2>/dev/null || true
  echo "REMOVED all managed $name versions"
}

run_legacy() {
  local cleaner=""
  if [[ -x "$TLALOC_ROOT/current/tools/legacy-cleanup.sh" ]]; then cleaner="$TLALOC_ROOT/current/tools/legacy-cleanup.sh"
  elif [[ -x "$ORIGAMI_ROOT/current/tools/legacy-cleanup.sh" ]]; then cleaner="$ORIGAMI_ROOT/current/tools/legacy-cleanup.sh"
  elif [[ -x "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/tools/legacy-cleanup.sh" ]]; then cleaner="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/tools/legacy-cleanup.sh"
  else echo "Legacy cleaner not found." >&2; return 1
  fi
  args=(--remove --yes)
  [[ "$CLEAN_SHELL" -eq 1 ]] && args+=(--clean-shell)
  [[ "$AGGRESSIVE" -eq 1 ]] && args+=(--aggressive-legacy)
  [[ "$SYSTEM" -eq 1 ]] && args+=(--system)
  "$cleaner" "${args[@]}"
}

run_managed() {
  if [[ "$TLALOC_ONLY" -ne 1 ]]; then
    if [[ "$ALL_MANAGED" -eq 1 ]]; then remove_all_managed_component origami "$ORIGAMI_ROOT"; else remove_component_current origami "$ORIGAMI_ROOT"; fi
  fi
  if [[ "$ORIGAMI_ONLY" -ne 1 ]]; then
    if [[ "$ALL_MANAGED" -eq 1 ]]; then remove_all_managed_component tlaloc "$TLALOC_ROOT"; else remove_component_current tlaloc "$TLALOC_ROOT"; fi
  fi
}

confirm || { echo "Cancelled."; exit 1; }
case "$MODE" in
  legacy) run_legacy ;;
  current) run_managed ;;
  all)
    run_legacy
    ALL_MANAGED=1
    run_managed
    ;;
esac

echo "Uninstall completed. BPFW/PipeCraft, .me/origami workspaces and Tlaloc learning memory were not touched."