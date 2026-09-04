// Command tlaloc-tonalt1-freshcorpus runs the TONAL T1 fresh-corpus gate:
// build a canonical store for a fresh born-digital PDF, deterministically
// scan it (D3 v2 selector), freeze a page-level bridge/primary partition
// BEFORE any model call, run the isolated document-specific perceptual
// bridge (isolated EXTRACT_NUMBER, NOT T1 workflows), and freeze the fresh
// primary held-out universe with a constructive allocation-feasibility
// proof.
//
//	tlaloc-tonalt1-freshcorpus \
//	  -root . -pdf "/path/to/fresh.pdf" -rank 1 -proxy 812 \
//	  -out experiments/tonal-t1/fresh-corpus \
//	  -endpoint http://127.0.0.1:1234 -model lfm2-vl-1.6b \
//	  -bridge-n 60
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
	"tlaloc.local/behaviorlab/internal/perceptenvelope"
	"tlaloc.local/behaviorlab/internal/tonalt1"
)

func main() {
	root := flag.String("root", ".", "behavior-lab repo root")
	pdf := flag.String("pdf", "", "fresh born-digital source PDF (absolute path)")
	rank := flag.Int("rank", 1, "deterministic selection rank of this document")
	proxy := flag.Int("proxy", 0, "prefilter eligible_operand_proxy for this document")
	out := flag.String("out", "experiments/tonal-t1/fresh-corpus", "output dir (relative to root)")
	endpoint := flag.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := flag.String("model", "lfm2-vl-1.6b", "model id")
	bridgeN := flag.Int("bridge-n", 60, "target bridge instances per morphology")
	flag.Parse()

	if *pdf == "" {
		fail(fmt.Errorf("-pdf is required"))
	}
	outDir := filepath.Join(*root, *out)
	corpusID := tonalt1Slug(filepath.Base(*pdf))
	storeDir := tonalt1.FreshStoreDir(*root, corpusID)
	if err := os.MkdirAll(filepath.Join(outDir, "run"), 0o755); err != nil {
		fail(err)
	}

	sourceSHA := fileSHA(*pdf)
	if sourceSHA == "7143a3d446074a33c7b3e945de0881ad759265536a55fec7025cd20ed6c0a8e9" {
		fail(fmt.Errorf("this is the D2L source; a FRESH document is required"))
	}

	// 1. Build the canonical store (deterministic content hashing).
	_, err := tonalt1.EnsureFreshStore(*pdf, storeDir, corpusID, func(pdfPath, dir, carrier string) error {
		res, err := pdfmemory.BuildPDF(pdfPath, dir, carrier)
		if err != nil {
			return err
		}
		_ = res
		return nil
	})
	if err != nil {
		fail(fmt.Errorf("store build: %w", err))
	}
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		fail(fmt.Errorf("load store: %w", err))
	}
	store := tonalt1.StoreIdentity{
		StoreDir: storeDir, CarrierID: manifest.CarrierID,
		SourcePDFSHA256: sourceSHA, StoreRootSHA256: manifest.StoreRootSHA256,
		PageCount: manifest.PageCount, RegionCount: manifest.RegionCount,
	}

	// 2. Deterministic full scan (no prior-use index — fresh document,
	//    disjoint source sha).
	scan, err := tonalt1.Scan(storeDir, nil)
	if err != nil {
		fail(fmt.Errorf("scan: %w", err))
	}
	writeJSON(filepath.Join(outDir, "FRESH_CANDIDATES_ALL.json"), scan.Candidates)

	pages := tonalt1.EligiblePages(scan)
	eligibleTotal := 0
	for _, c := range scan.Candidates {
		if c.Eligibility.Eligible {
			eligibleTotal++
		}
	}
	fmt.Fprintf(os.Stderr, "scan: total=%d eligible=%d eligible_pages=%d\n", len(scan.Candidates), eligibleTotal, len(pages))

	// 3. FREEZE the bridge/primary page partition BEFORE inference.
	partition := tonalt1.PartitionPages(sourceSHA, pages, tonalt1.FreshBridgeFraction)
	writeJSON(filepath.Join(outDir, "FRESH_PAGE_PARTITION.json"), partition)

	identity := loadWeightsSHA(*root)

	// 4. FREEZE the bridge dataset BEFORE inference.
	bridge := tonalt1.BuildBridgeDataset(scan, partition, *bridgeN, identity)
	writeJSON(filepath.Join(outDir, "FRESH_BRIDGE_DATASET.json"), bridge)
	writeJSON(filepath.Join(outDir, "FRESH_BRIDGE_CRITERIA.json"), map[string]any{
		"schema":            "tonal.t1.fresh-corpus.bridge-criteria.r1",
		"frozen_before":     "the first bridge model call",
		"question":          "does the earned Parrot EXTRACT_NUMBER capability remain valid under this document's typography when rendered with the frozen presentation core?",
		"not_asked":         "whether T1 workflows work; no arithmetic / Blackboard / Arm A/B/C / DAG",
		"per_morphology":    bridge.PerMorphologyN,
		"promotion_rule":    bridge.PromotionRule,
		"renderer":          bridge.Renderer,
		"prompt":            bridge.Prompt,
		"model":             bridge.Model,
		"model_weights_sha": identity,
		"temperature":       0.0,
		"max_output_tokens": 32,
		"call_budget":       bridge.CallBudget,
		"dataset_hash":      bridge.DatasetHash,
		"partition_hash":    partition.PartitionHash,
	})

	if bridge.CallBudget == 0 {
		freezeAndReport(outDir, tonalt1.FreshCorpusFreeze(
			sourceDoc(*pdf, sourceSHA, manifest.PageCount, *rank, *proxy, scan),
			store, scan, partition, bridge, nil, tonalt1.PrimaryUniverse{}, tonalt1.CapacityCheck{NRequired: 144}))
		fail(fmt.Errorf("bridge dataset empty: no eligible MULTI_DIGIT_INTEGER/DECIMAL candidates on bridge pages — document unsuitable"))
	}

	// 5. Endpoint guard + bridge inference (isolated capability, NOT T1).
	if err := guardEndpoint(*endpoint, *model); err != nil {
		fail(fmt.Errorf("endpoint guard: %w", err))
	}
	alloc := perceptenvelope.R1CAllocation{
		Schema: "tlaloc.parrot-perceptual-envelope-r1.r1c-allocation.r1", ExperimentID: "tonal-t1-fresh-corpus-bridge",
		Seed: tonalt1.Seed, RankRule: "sha256(seed || candidate_id) ascending",
		LineHeightPx: perceptenvelope.R1CLineHeightPx, ContextLevel: perceptenvelope.R1CContextLevel, CanvasPx: 512,
		Families: bridgeFamilies(bridge),
	}
	writeJSON(filepath.Join(outDir, "FRESH_BRIDGE_ALLOCATION.json"), alloc)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Minute)
	defer cancel()
	cfg := perceptenvelope.RunConfig{
		StoreDir: storeDir, Endpoint: *endpoint, Model: *model,
		Temperature: 0.0, MaxTokens: 32, RunDir: filepath.Join(outDir, "run"),
	}
	records, err := perceptenvelope.RunR1C(ctx, cfg, alloc, nil)
	if err != nil {
		fail(fmt.Errorf("bridge run: %w", err))
	}
	writeJSON(filepath.Join(outDir, "FRESH_BRIDGE_RECORDS.json"), records)

	// 6. Aggregate per morphology with the frozen R1-C verdict logic.
	table := perceptenvelope.AggregateR1C(records)
	writeJSON(filepath.Join(outDir, "FRESH_BRIDGE_MORPHOLOGY_TABLE.json"), table)

	var results []tonalt1.BridgeMorphologyResult
	var qualified []tonalt1.MorphologyFamily
	for _, morph := range []tonalt1.MorphologyFamily{tonalt1.MorphMultiDigitInteger, tonalt1.MorphDecimal} {
		famKey := string(morph)
		var row perceptenvelope.R1CRow
		found := false
		for _, r := range table.Rows {
			if r.Family == famKey && r.Provenance == "REAL_DOCUMENT" {
				row, found = r, true
			}
		}
		if !found {
			continue
		}
		verdict := ""
		for _, v := range table.Verdicts {
			if v.Family == famKey {
				verdict = v.Verdict
			}
		}
		q := verdict == "USABLE_WITH_CONSTRAINTS" || verdict == "RELIABLE"
		results = append(results, tonalt1.BridgeMorphologyResult{
			Morphology: morph, N: row.Value.N, Correct: row.Value.Count,
			Accuracy: row.Value.Accuracy, CI95Low: row.Value.CI95Low, CI95High: row.Value.CI95High,
			ContractSuccess: row.ContractSuccess, Verdict: verdict, Qualified: q,
		})
		if q {
			qualified = append(qualified, morph)
		}
	}
	writeJSON(filepath.Join(outDir, "FRESH_BRIDGE_RESULTS.json"), results)

	// 7. Primary held-out universe (bridge-qualified morphologies only).
	universe := tonalt1.BuildPrimaryUniverse(scan, partition, bridge, qualified)
	writeJSON(filepath.Join(outDir, "FRESH_PRIMARY_UNIVERSE.json"), universe)

	// 8. Constructive allocation feasibility.
	capacity := tonalt1.CheckAllocationFeasible(universe)
	capacity.NAvailable = universe.N

	// 9. Freeze.
	man := tonalt1.FreshCorpusFreeze(
		sourceDoc(*pdf, sourceSHA, manifest.PageCount, *rank, *proxy, scan),
		store, scan, partition, bridge, results, universe, capacity)
	freezeAndReport(outDir, man)
}

func freezeAndReport(outDir string, man tonalt1.FreshCorpusManifest) {
	writeJSON(filepath.Join(outDir, "TONAL_T1_FRESH_CORPUS_FREEZE.json"), man)
	report := map[string]any{
		"TONAL_T1_FRESH_CORPUS_FROZEN": man.TONALT1FreshCorpusFrozen,
		"T1_D4_CAN_PROCEED":            man.T1D4CanProceed,
		"source":                       man.Source.Path,
		"source_sha256":                man.Source.SourceSHA256,
		"pages":                        man.Source.PageCount,
		"scan_total":                   man.ScanTotal,
		"eligible_total":               man.EligibleTotal,
		"eligible_pages":               man.EligiblePages,
		"bridge_pages":                 len(man.Partition.BridgePages),
		"primary_pages":                len(man.Partition.PrimaryPages),
		"bridge_per_morphology":        man.Bridge.PerMorphologyN,
		"bridge_call_budget":           man.Bridge.CallBudget,
		"n_primary_available":          man.Primary.N,
		"primary_by_morphology":        man.Primary.ByMorphology,
		"distinct_primary_pages":       man.Primary.DistinctPages,
		"bridge_leakage":               man.Primary.BridgeLeakage,
		"bridge_page_leakage":          man.Primary.BridgePageLeakage,
		"headroom_ratio":               man.Capacity.HeadroomRatio,
		"allocation_feasible":          man.Capacity.AllocationFeasible,
		"hard_invariants":              man.HardInvariants,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
}

func bridgeFamilies(bridge tonalt1.BridgeSpec) []perceptenvelope.R1CFamilyAllocation {
	byMorph := map[tonalt1.MorphologyFamily][]perceptenvelope.R1CBase{}
	for _, base := range bridge.Bases {
		cc := base.Candidate
		start := 0
		if idx := strings.Index(cc.LineText, cc.Token); idx >= 0 {
			start = len([]rune(cc.LineText[:idx]))
		}
		byMorph[base.Morphology] = append(byMorph[base.Morphology], perceptenvelope.R1CBase{
			BaseID: base.BaseID, Family: string(base.Morphology),
			Provenance: perceptenvelope.ProvReal, Stratum: perceptenvelope.StratumLexical,
			GoldSurface: base.Gold, RankKey: base.RankKey, Candidate: &cc,
			CharOffsetStart: start, CharOffsetEnd: start + len([]rune(cc.Token)),
		})
	}
	var out []perceptenvelope.R1CFamilyAllocation
	for _, morph := range []tonalt1.MorphologyFamily{tonalt1.MorphMultiDigitInteger, tonalt1.MorphDecimal} {
		if len(byMorph[morph]) == 0 {
			continue
		}
		out = append(out, perceptenvelope.R1CFamilyAllocation{
			Family: string(morph), Stratum: perceptenvelope.StratumLexical,
			RealAvailable: len(byMorph[morph]), Band: "FRESH_CORPUS_BRIDGE", RealBases: byMorph[morph],
		})
	}
	return out
}

func sourceDoc(path, sha string, pages, rank, proxy int, scan tonalt1.ScanResult) tonalt1.SourceDoc {
	return tonalt1.SourceDoc{Path: path, SourceSHA256: sha, PageCount: pages, SelectionRank: rank, ProxyEligible: proxy}
}

func loadWeightsSHA(root string) string {
	body, err := os.ReadFile(filepath.Join(root, "experiments/parrot-perceptual-envelope-r1/MODEL_IDENTITY.json"))
	if err != nil {
		return ""
	}
	var doc struct {
		Model struct {
			WeightsGGUF struct {
				SHA256 string `json:"sha256"`
			} `json:"weights_gguf"`
		} `json:"model"`
	}
	_ = json.Unmarshal(body, &doc)
	return doc.Model.WeightsGGUF.SHA256
}

func guardEndpoint(endpoint, model string) error {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(strings.TrimRight(endpoint, "/") + "/v1/models")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return err
	}
	for _, m := range list.Data {
		if m.ID == model {
			return nil
		}
	}
	return fmt.Errorf("model %q not served by %s", model, endpoint)
}

func fileSHA(path string) string {
	f, err := os.Open(path)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		fail(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func tonalt1Slug(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '.':
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if s == "" {
		s = "fresh-doc"
	}
	return s
}

func writeJSON(path string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "tlaloc-tonalt1-freshcorpus:", err)
	os.Exit(1)
}
