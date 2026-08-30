package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func writeBundleScripts(out string) error {
	verify := `#!/usr/bin/env bash\nset -euo pipefail\nHERE="$(cd "$(dirname "$0")" && pwd)"\n"$HERE/bin/tlaloc-origami" verify -carrier "$HERE/origami.png" -store "$HERE/store" -origami-bin "$HERE/bin/origami-fixed-carrier"\n`
	ask := `#!/usr/bin/env bash\nset -euo pipefail\nHERE="$(cd "$(dirname "$0")" && pwd)"\nMODEL="${1:?model name required}"; shift\nQUESTION="${*:?question required}"\nMODE="${TOOL_MODE:-functions}"\n"$HERE/bin/tlaloc-origami" chat -carrier "$HERE/origami.png" -store "$HERE/store" -prompt "$HERE/MASTER_PROMPT.txt" -origami-bin "$HERE/bin/origami-fixed-carrier" -model "$MODEL" -question "$QUESTION" -tool-mode "$MODE"\n`
	if err := os.WriteFile(filepath.Join(out, "verify.sh"), []byte(strings.ReplaceAll(verify, "\\n", "\n")), 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "ask.sh"), []byte(strings.ReplaceAll(ask, "\\n", "\n")), 0755)
}
func debugMain(format string, args ...any) {
	if os.Getenv("TLALOC_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[compile %s] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
	}
}

func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

const sampleQueries = `R2 test sequence:
1. BOOT from the actual image: read T0, then both T1 challenge rows.
2. What does PDF page 235 discuss? Use exact address expansion.
3. What does PDF page 637 discuss? Cite the exact Origami address.
4. Where does the corpus discuss dynamic programming and approximate string matching?
5. Find evidence about minimum spanning trees.
6. Compare shortest-path material with dynamic-programming material using only selected blocks/pages.
7. Verify the carrier/store Merkle binding and report FALSE_EXACT.
8. Ask for ZXQJ-NEVER-EXISTS-918273645 and confirm UNKNOWN.
`
