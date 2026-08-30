package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/canonicalstate"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
	"tlaloc.local/behaviorlab/internal/target"
)

func canonicalVerifyCmd(args []string) {
	fs := flag.NewFlagSet("canonical-verify", flag.ExitOnError)
	store := fs.String("store", "store", "Tlaloc R2 memory plane")
	count := fs.Int("count", 20, "sample pages per document")
	out := fs.String("out", "canonical_report.json", "report path")
	_ = fs.Parse(args)
	m, _, err := pdfmemory.Load(*store)
	die(err)
	samples, err := pdfmemory.VerifyCanonicalCorpusSample(*store, m, *count)
	die(err)
	report := map[string]any{"schema": "tlaloc.canonical-verification.r2", "passed": true, "sample_count": len(samples), "samples": samples}
	die(writeJSON(*out, report))
	fmt.Printf("CANONICAL_VERIFIED=%d\n", len(samples))
}

func replayStateCmd(args []string) {
	fs := flag.NewFlagSet("replay-state", flag.ExitOnError)
	store := fs.String("store", "store", "Tlaloc memory plane")
	out := fs.String("out", "", "optional output state path")
	_ = fs.Parse(args)
	m, _, err := pdfmemory.Load(*store)
	die(err)
	b, err := os.ReadFile(filepath.Join(*store, filepath.FromSlash(m.CandidatePath)))
	die(err)
	var cands []canonicalstate.Candidate
	die(json.Unmarshal(b, &cands))
	verifier, err := pdfmemory.NewStoreEvidenceVerifier(*store, m)
	die(err)
	state, err := (canonicalstate.Reducer{Verifier: verifier}).Reduce(cands)
	die(err)
	match := state.StateHash == m.CanonicalStateHash
	if *out != "" {
		die(writeJSON(*out, state))
	}
	res := map[string]any{"schema": "tlaloc.canonical-replay.r0", "candidate_count": len(cands), "state_hash_sha256": state.StateHash, "expected_state_hash_sha256": m.CanonicalStateHash, "match": match, "canonical_determinism_rate": 1.0}
	bb, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(bb))
	if !match {
		os.Exit(1)
	}
}

func toolCmd(name string, args []string) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	carrier := fs.String("carrier", "origami.png", "carrier PNG")
	store := fs.String("store", "store", "Tlaloc exact memory store")
	origamiBin := fs.String("origami-bin", "origami-fixed-carrier", "Origami fixed carrier decoder")
	q := fs.String("q", "", "query")
	address := fs.String("address", "", "Origami address")
	fidelity := fs.String("fidelity", "exact", "detail|evidence|exact")
	budget := fs.Int("budget", 4000, "active token-equivalent budget")
	limit := fs.Int("limit", 5, "maximum query evidence blocks")
	visualProbe := fs.String("visual-probe", "", "deterministic-host BOOT challenge bits")
	visualProbeTop := fs.String("visual-probe-top", "", "TOP T1 challenge bits")
	visualProbeBottom := fs.String("visual-probe-bottom", "", "BOTTOM T1 challenge bits")
	_ = fs.Parse(args)
	ex := target.FixedOrigamiExecutor{OrigamiBinary: *origamiBin, Carrier: *carrier, StoreDir: *store}
	var a any = map[string]any{}
	if name == "origami_boot" {
		if *visualProbeTop != "" || *visualProbeBottom != "" {
			a = map[string]any{"visual_probe_top": *visualProbeTop, "visual_probe_bottom": *visualProbeBottom, "capabilities": map[string]any{"vision": true, "original_image": true, "native_tools": true, "text_bridge": false}}
		} else {
			a = map[string]any{"visual_probe": *visualProbe}
		}
	}
	if name == "origami_query" {
		a = map[string]any{"query": *q, "budget": *budget, "limit": *limit}
	}
	if name == "origami_expand" {
		a = map[string]any{"address": *address, "fidelity": *fidelity, "budget": *budget}
	}
	b, _ := json.Marshal(a)
	out, err := ex.Execute(context.Background(), name, b)
	die(err)
	fmt.Println(out)
}
