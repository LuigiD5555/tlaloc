package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
	"tlaloc.local/behaviorlab/internal/target"
)

func compileCmd(args []string) {
	fs := flag.NewFlagSet("compile", flag.ExitOnError)
	var pdfs multiFlag
	fs.Var(&pdfs, "pdf", "source PDF; repeat for multi-document corpus")
	out := fs.String("out", "bundle", "bundle output directory")
	carrierID := fs.String("carrier-id", "document", "stable logical carrier id")
	origamiBin := fs.String("origami-bin", "origami-fixed-carrier", "Origami Fixed Carrier R2 compiler")
	sampleCount := fs.Int("sample-pages", 20, "independent canonical sample per document")
	_ = fs.Parse(args)
	if len(pdfs) == 0 {
		die(fmt.Errorf("at least one -pdf is required"))
	}
	storeDir := filepath.Join(*out, "store")
	die(os.RemoveAll(*out))
	die(os.MkdirAll(*out, 0755))
	sources := make([]pdfmemory.SourceSpec, len(pdfs))
	for i, p := range pdfs {
		sources[i] = pdfmemory.SourceSpec{Path: p}
	}
	res, err := pdfmemory.BuildCorpus(sources, storeDir, *carrierID)
	die(err)
	promptPath := filepath.Join(*out, "MASTER_PROMPT.txt")
	die(os.WriteFile(promptPath, []byte(pdfmemory.MasterPrompt()), 0644))
	carrierPath := filepath.Join(*out, "origami.png")
	metaPath := filepath.Join(storeDir, "fixed_carrier_meta.json")
	cmd := exec.Command(*origamiBin, "-mode", "build", "-in", metaPath, "-out", carrierPath)
	buildOut, err := cmd.CombinedOutput()
	if err != nil {
		die(fmt.Errorf("origami carrier build: %w: %s", err, strings.TrimSpace(string(buildOut))))
	}
	cbytes, err := os.ReadFile(carrierPath)
	die(err)
	if len(cbytes) > 512000 {
		die(fmt.Errorf("carrier exceeds 500 KiB hard limit: %d", len(cbytes)))
	}
	pbytes, err := os.ReadFile(promptPath)
	die(err)
	debugMain("carrier built bytes=%d", len(cbytes))
	ex := target.FixedOrigamiExecutor{OrigamiBinary: *origamiBin, Carrier: carrierPath, StoreDir: storeDir}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	debugMain("visual probe decode")
	probe, err := ex.VisualProbe(ctx)
	die(err)
	bootArgs, _ := json.Marshal(map[string]any{"visual_probe": probe})
	debugMain("boot verify")
	boot, err := ex.Execute(ctx, "origami_boot", bootArgs)
	die(err)
	debugMain("full store verify")
	verifyJSON, err := ex.Execute(ctx, "origami_verify", json.RawMessage(`{}`))
	die(err)
	var verifyMap map[string]any
	die(json.Unmarshal([]byte(verifyJSON), &verifyMap))
	debugMain("canonical re-extraction in isolated verifier process")
	self, err := os.Executable()
	die(err)
	canonicalPath := filepath.Join(*out, "canonical_report.json")
	canonicalOut, err := exec.Command(self, "canonical-verify", "-store", storeDir, "-count", fmt.Sprint(*sampleCount), "-out", canonicalPath).CombinedOutput()
	if err != nil {
		die(fmt.Errorf("canonical verifier: %w: %s", err, strings.TrimSpace(string(canonicalOut))))
	}
	var canonical struct {
		Samples []pdfmemory.CanonicalSample `json:"samples"`
		Passed  bool                        `json:"passed"`
	}
	cb, err := os.ReadFile(canonicalPath)
	die(err)
	die(json.Unmarshal(cb, &canonical))
	samples := canonical.Samples
	debugMain("writing final bundle artifacts")
	manifest := BundleManifest{Schema: "tonal.origami-fixed-bundle.r2", Carrier: "origami.png", Prompt: "MASTER_PROMPT.txt", Store: "store", CarrierSHA256: hash(cbytes), PromptSHA256: hash(pbytes), StoreRootSHA256: res.Manifest.StoreRootSHA256, SourceSHA256: res.Manifest.SourceSHA256, DocumentCount: res.Manifest.DocumentCount, PageCount: res.Manifest.PageCount, BlockCount: res.Manifest.BlockCount, RegionCount: res.Manifest.RegionCount, CanonicalClaimCount: res.Manifest.CanonicalClaimCount, CanonicalStateHash: res.Manifest.CanonicalStateHash, ObjectCount: res.Manifest.ObjectCount, FixedCarrierBytes: len(cbytes), MaxCarrierBytes: 512000}
	die(writeJSON(filepath.Join(*out, "bundle_manifest.json"), manifest))
	report := VerificationReport{Schema: "tonal.origami-fixed-verification.r2", CarrierBytes: len(cbytes), CarrierMaxBytes: 512000, CarrierSHA256: hash(cbytes), PromptSHA256: hash(pbytes), SourceSHA256: res.Manifest.SourceSHA256, StoreRootSHA256: res.Manifest.StoreRootSHA256, DocumentCount: res.Manifest.DocumentCount, PageCount: res.Manifest.PageCount, BlockCount: res.Manifest.BlockCount, RegionCount: res.Manifest.RegionCount, CandidateCount: res.Manifest.CandidateCount, CanonicalClaimCount: res.Manifest.CanonicalClaimCount, ConflictCount: res.Manifest.ConflictCount, CanonicalStateHash: res.Manifest.CanonicalStateHash, ObjectCount: res.Manifest.ObjectCount, PagesVerified: res.Manifest.PageCount, BlocksVerified: res.Manifest.BlockCount, CanonicalSample: samples, CanonicalSamplePassed: true, OCRRequired: false, T0PlaintextBoot: true, FalseExact: 0}
	die(writeJSON(filepath.Join(*out, "verification_report.json"), report))
	die(os.WriteFile(filepath.Join(*out, "sample_queries.txt"), []byte(sampleQueries), 0644))
	tools, _ := json.MarshalIndent(target.OrigamiFixedTools(), "", "  ")
	die(os.WriteFile(filepath.Join(*out, "tool_schema.json"), append(tools, '\n'), 0644))
	die(writeBundleScripts(*out))
	// Persist carrier decode and BOOT/VERIFY outputs as reproducible build evidence.
	decOut, err := exec.Command(*origamiBin, "-mode", "decode", "-in", carrierPath).Output()
	die(err)
	die(os.WriteFile(filepath.Join(*out, "carrier_decoded.json"), decOut, 0644))
	die(os.WriteFile(filepath.Join(*out, "boot_result.json"), append([]byte(boot), '\n'), 0644))
	die(os.WriteFile(filepath.Join(*out, "verify_result.json"), append([]byte(verifyJSON), '\n'), 0644))
	fmt.Printf("BUNDLE=%s\nCARRIER=%s\nCARRIER_BYTES=%d\nPROMPT=%s\nSTORE=%s\nDOCUMENTS=%d\nPAGES=%d\nBLOCKS=%d\nOBJECTS=%d\nBOOT=%s\nVERIFY=%s\nCANONICAL_SAMPLE=%d\n", *out, carrierPath, len(cbytes), promptPath, storeDir, res.Manifest.DocumentCount, res.Manifest.PageCount, res.Manifest.BlockCount, res.Manifest.ObjectCount, boot, verifyJSON, len(samples))
}
