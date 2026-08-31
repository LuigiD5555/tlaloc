#!/usr/bin/env bash
set -euo pipefail

ACTION=scan
YES=0
AGGRESSIVE=0
CLEAN_SHELL=0
SYSTEM=0
HOME_DIR="${HOME:?HOME is required}"
DATA_HOME="${XDG_DATA_HOME:-$HOME_DIR/.local/share}"
CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME_DIR/.config}"
CACHE_HOME="${XDG_CACHE_HOME:-$HOME_DIR/.cache}"
STATE_HOME="${XDG_STATE_HOME:-$HOME_DIR/.local/state}"
BIN_HOME="${TLALOC_BIN_HOME:-$HOME_DIR/.local/bin}"
GO_BIN="${GOBIN:-${GOPATH:-$HOME_DIR/go}/bin}"
CARGO_BIN="$HOME_DIR/.cargo/bin"
USER_BIN="$HOME_DIR/bin"
LOCAL_LIB="$HOME_DIR/.local/lib"
ORIGAMI_INSTALL_STATE="$DATA_HOME/origami/install-state-v1/manifest.tsv"

usage() {
  cat <<'USAGE'
Usage: legacy-cleanup.sh [options]

  --scan                 Inventory only (default; makes no changes)
  --remove               Remove detected legacy artifacts
  --yes                  Required with --remove
  --aggressive-legacy    Also remove medium-confidence VCL-only artifacts
  --clean-shell          Back up and remove legacy Origami/OHF/VCL env/alias lines
  --system               Also scan /usr/local; removal requires appropriate permissions
  -h, --help             Show this help

Never removes BPFW/PipeCraft. Never removes project .me/origami workspaces.
Current managed Tlaloc/Origami installations are protected.
USAGE
}

while (($#)); do
  case "$1" in
    --scan) ACTION=scan ;;
    --remove) ACTION=remove ;;
    --yes) YES=1 ;;
    --aggressive-legacy) AGGRESSIVE=1 ;;
    --clean-shell) CLEAN_SHELL=1 ;;
    --system) SYSTEM=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

if [[ "$ACTION" == remove && "$YES" -ne 1 ]]; then
  echo "Refusing destructive cleanup without --yes." >&2
  exit 2
fi

origami_manifest_tracks() {
  local p="$1"
  [[ -f "$ORIGAMI_INSTALL_STATE" ]] || return 1
  awk -F '\t' -v want="$p" '$1=="BIN" && $3==want {found=1} END{exit found?0:1}' "$ORIGAMI_INSTALL_STATE"
}

is_protected() {
  local p="$1" r=""
  case "$p" in
    "$DATA_HOME/bpfw"|"$DATA_HOME/bpfw/"*|"$BIN_HOME/bpfw"|"$BIN_HOME/bpfw-uninstall"|*"/pipecraft"|*"/pipecraft/"*) return 0 ;;
  esac
  [[ "$p" == *"/.me/origami" || "$p" == *"/.me/origami/"* ]] && return 0
  case "$p" in
    "$DATA_HOME/tlaloc"|"$DATA_HOME/tlaloc/"*) return 0 ;;
  esac
  if [[ -f "$ORIGAMI_INSTALL_STATE" ]]; then
    case "$p" in
      "$DATA_HOME/origami"|"$DATA_HOME/origami/"*) return 0 ;;
    esac
    origami_manifest_tracks "$p" && return 0
  fi
  if [[ -e "$p" || -L "$p" ]]; then
    r="$(readlink -f "$p" 2>/dev/null || true)"
    case "$r" in
      "$DATA_HOME/tlaloc/"*|"$DATA_HOME/origami/versions/"*) return 0 ;;
    esac
  fi
  if [[ "$p" == "$DATA_HOME/origami" && ( -f "$DATA_HOME/origami/.origami-managed-v1" || -f "$DATA_HOME/origami/.tlaloc-managed-v1" ) ]]; then
    return 0
  fi
  return 1
}

CANDIDATES=()
CONFIDENCE=()
REASONS=()
add_candidate() {
  local p="$1" c="$2" reason="$3"
  [[ -e "$p" || -L "$p" ]] || return 0
  is_protected "$p" && return 0
  local i
  for i in "${!CANDIDATES[@]}"; do [[ "${CANDIDATES[$i]}" == "$p" ]] && return 0; done
  CANDIDATES+=("$p"); CONFIDENCE+=("$c"); REASONS+=("$reason")
}

for base in "$DATA_HOME" "$CONFIG_HOME" "$CACHE_HOME" "$STATE_HOME" "$LOCAL_LIB"; do
  add_candidate "$base/ohf" HIGH "legacy OHF root"
  add_candidate "$base/origami-hyperfold" HIGH "legacy Origami HyperFold root"
  add_candidate "$base/origami_hyperfold" HIGH "legacy Origami HyperFold root"
  add_candidate "$base/vcl" MEDIUM "legacy VCL root; generic name requires aggressive mode"
  if [[ "$base" != "$DATA_HOME" || ! ( -f "$DATA_HOME/origami/.origami-managed-v1" || -f "$DATA_HOME/origami/.tlaloc-managed-v1" || -f "$ORIGAMI_INSTALL_STATE" ) ]]; then
    add_candidate "$base/origami" HIGH "legacy Origami root"
  fi
done

if [[ -f "$DATA_HOME/origami/.origami-managed-v1" || -f "$DATA_HOME/origami/.tlaloc-managed-v1" ]]; then
  shopt -s nullglob dotglob
  for child in "$DATA_HOME/origami"/*; do
    case "$(basename "$child")" in versions|current|.origami-managed-v1|.tlaloc-managed-v1) continue ;; esac
    add_candidate "$child" HIGH "loose pre-manifest content inside managed Origami root"
  done
  shopt -u nullglob dotglob
fi

add_candidate "$HOME_DIR/.origami" HIGH "legacy hidden Origami root"
add_candidate "$HOME_DIR/.ohf" HIGH "legacy hidden OHF root"
add_candidate "$HOME_DIR/.origami-hyperfold" HIGH "legacy hidden Origami HyperFold root"
add_candidate "$HOME_DIR/.vcl" MEDIUM "legacy hidden VCL root; generic name requires aggressive mode"
add_candidate "$STATE_HOME/tlaloc/install-manifest-v1.tsv" HIGH "obsolete Tlaloc alpha.2 installer state manifest"

for d in "$BIN_HOME" "$GO_BIN" "$CARGO_BIN" "$USER_BIN"; do
  [[ -d "$d" ]] || continue
  for name in origami origami-cli origami-lab origami-uninstall ohf ohf-lab ohf-cli ohf-uninstall origami-ohf origami-hyperfold perception-lab origami-perception-lab; do
    add_candidate "$d/$name" HIGH "legacy Origami/OHF executable"
  done
  for name in vcl vcl-cli vcl-go; do
    add_candidate "$d/$name" MEDIUM "legacy VCL executable; generic name requires aggressive mode"
  done
  shopt -s nullglob
  for p in "$d"/origami-* "$d"/ohf-*; do add_candidate "$p" HIGH "legacy prefixed executable"; done
  shopt -u nullglob
done

shopt -s nullglob
for p in \
  "$CONFIG_HOME/systemd/user"/origami*.service "$CONFIG_HOME/systemd/user"/ohf*.service \
  "$CONFIG_HOME/systemd/user"/origami*.timer "$CONFIG_HOME/systemd/user"/ohf*.timer \
  "$CONFIG_HOME/systemd/user"/origami*.socket "$CONFIG_HOME/systemd/user"/ohf*.socket \
  "$CONFIG_HOME/systemd/user"/origami*.path "$CONFIG_HOME/systemd/user"/ohf*.path \
  "$DATA_HOME/applications"/origami*.desktop "$DATA_HOME/applications"/ohf*.desktop \
  "$CONFIG_HOME/autostart"/origami*.desktop "$CONFIG_HOME/autostart"/ohf*.desktop \
  "$DATA_HOME/bash-completion/completions"/origami* "$DATA_HOME/bash-completion/completions"/ohf* \
  "$CONFIG_HOME/fish/completions"/origami*.fish "$CONFIG_HOME/fish/completions"/ohf*.fish; do
  add_candidate "$p" HIGH "legacy user integration/completion file"
done
shopt -u nullglob

if [[ "$SYSTEM" -eq 1 ]]; then
  for p in /usr/local/bin/origami /usr/local/bin/ohf /usr/local/bin/ohf-lab /usr/local/share/origami /usr/local/share/ohf /usr/local/share/origami-hyperfold; do
    add_candidate "$p" HIGH "legacy system-wide artifact"
  done
fi

printf 'Legacy Origami/OHF/VCL inventory\n'
printf '  action: %s\n' "$ACTION"
printf '  BPFW/PipeCraft: PROTECTED\n'
printf '  .me/origami workspaces: PROTECTED\n'
if [[ -f "$ORIGAMI_INSTALL_STATE" ]]; then printf '  standalone Origami install: PROTECTED\n'; fi
printf '\n'

if ((${#CANDIDATES[@]} == 0)); then
  echo "No legacy system-install artifacts detected in known locations."
else
  for i in "${!CANDIDATES[@]}"; do
    printf '[%s] %s\n      %s\n' "${CONFIDENCE[$i]}" "${CANDIDATES[$i]}" "${REASONS[$i]}"
  done
fi

SHELL_MATCHES=()
for rc in "$HOME_DIR/.bashrc" "$HOME_DIR/.bash_profile" "$HOME_DIR/.profile" "$HOME_DIR/.zshrc" "$HOME_DIR/.zprofile"; do
  [[ -f "$rc" ]] || continue
  if grep -Eqi 'ORIGAMI_ROOT|OHF_ROOT|VCL_ROOT|\.local/(share|bin)/(origami|ohf)|origami-hyperfold|ohf-lab' "$rc"; then
    SHELL_MATCHES+=("$rc")
  fi
done
if ((${#SHELL_MATCHES[@]})); then
  echo
  echo "Shell startup files with legacy-looking references:"
  printf '  %s\n' "${SHELL_MATCHES[@]}"
fi

if [[ "$ACTION" == remove ]]; then
  echo
  echo "Removing eligible legacy artifacts..."
  for i in "${!CANDIDATES[@]}"; do
    p="${CANDIDATES[$i]}"; c="${CONFIDENCE[$i]}"
    if [[ "$c" == MEDIUM && "$AGGRESSIVE" -ne 1 ]]; then
      echo "SKIP (medium confidence): $p"
      continue
    fi
    is_protected "$p" && { echo "PROTECTED: $p"; continue; }
    rm -rf -- "$p"
    echo "REMOVED: $p"
  done

  if [[ "$CLEAN_SHELL" -eq 1 ]]; then
    stamp="$(date +%Y%m%d%H%M%S)"
    for rc in "${SHELL_MATCHES[@]}"; do
      cp -a -- "$rc" "$rc.tlaloc-backup-$stamp"
      awk '
        BEGIN { IGNORECASE=1 }
        /^[[:space:]]*export[[:space:]]+(ORIGAMI_ROOT|OHF_ROOT|VCL_ROOT)=/ { next }
        /^[[:space:]]*alias[[:space:]]+(origami|ohf|ohf-lab|vcl)=/ { next }
        /^[[:space:]]*export[[:space:]]+PATH=.*(ORIGAMI_ROOT|OHF_ROOT|VCL_ROOT|\.local\/share\/(origami|ohf)|origami-hyperfold)/ { next }
        { print }
      ' "$rc" > "$rc.tmp.tlaloc"
      mv -- "$rc.tmp.tlaloc" "$rc"
      echo "CLEANED shell file (backup kept): $rc"
    done
  fi
fi
