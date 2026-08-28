#!/usr/bin/env bash
set -euo pipefail
DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
BIN_HOME="${TLALOC_BIN_HOME:-$HOME/.local/bin}"
fail=0
check() { if "$@" >/dev/null 2>&1; then printf 'PASS  %s\n' "$*"; else printf 'FAIL  %s\n' "$*"; fail=1; fi; }
[[ -L "$DATA_HOME/tlaloc/current" ]] && echo "PASS  Tlaloc current link" || { echo "FAIL  Tlaloc current link"; fail=1; }
check "$BIN_HOME/tlaloc" version
[[ -f "$DATA_HOME/tlaloc/current/.claude/skills/tlaloc-project/SKILL.md" ]] && echo "PASS  Tlaloc project skills" || { echo "FAIL  Tlaloc project skills"; fail=1; }
check "$BIN_HOME/tlaloc-behavior-lab" compile -spec "$DATA_HOME/tlaloc/current/behavior-lab/profiles/origami/quantum-inspired-r0.json" -out "${TMPDIR:-/tmp}/tlaloc-doctor-prompt.md" -target doctor
if [[ -L "$DATA_HOME/origami/current" ]]; then
  echo "INFO  Optional Origami installation detected: $(cat "$DATA_HOME/origami/current/VERSION" 2>/dev/null || echo unknown)"
else
  echo "INFO  Origami not installed (optional; Tlaloc remains valid)"
fi
if [[ -e "$DATA_HOME/bpfw" || -e "$BIN_HOME/bpfw" ]]; then echo "PASS  BPFW present and external/protected"; else echo "INFO  BPFW not detected (not a Tlaloc dependency)"; fi
exit "$fail"
