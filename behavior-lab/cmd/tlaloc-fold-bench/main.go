package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/foldtest"
	"tlaloc.local/behaviorlab/internal/foldtest/swarmask"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "build":
		cmdBuild(os.Args[2:])
	case "cover":
		cmdCover(os.Args[2:])
	case "ask":
		cmdAsk(os.Args[2:])
	case "swarm-ask":
		cmdSwarmAsk(os.Args[2:])
	case "validate":
		cmdValidate(os.Args[2:])
	case "dump-pages":
		cmdDumpPages(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  tlaloc-fold-bench build -pdf <file> -out <store>
  tlaloc-fold-bench cover -store <store>
  tlaloc-fold-bench ask -store <store> -model <model> -question <q> [-turns N] [-url URL]
  tlaloc-fold-bench swarm-ask -store <store> -model <model> -question <q> [-turns N] [-url URL]
  tlaloc-fold-bench validate -store <store> -model <model> [-pages N] [-seed SEED] [-flexibility F] [-url URL]
  tlaloc-fold-bench dump-pages -store <store> [-limit N] [-stride S]
`)
}

// cmdDumpPages writes one JSON object per line ({page, address, content}) for
// pages in the store, resolving each page's real content via the same
// foldtest.ExtractPageContent path the swarm consolidator uses. It exists to
// feed offline tooling (e.g. tools/grounding_dataset.py) real page text
// without duplicating the content-addressed store reader in Python.
func cmdDumpPages(args []string) {
	fs := flag.NewFlagSet("dump-pages", flag.ExitOnError)
	storeDir := fs.String("store", "", "Store directory")
	limit := fs.Int("limit", 0, "Maximum pages to emit (0 = all)")
	stride := fs.Int("stride", 1, "Emit every Nth page (spacing across the book)")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *storeDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: tlaloc-fold-bench dump-pages -store <store> [-limit N] [-stride S]\n")
		os.Exit(1)
	}
	if *stride < 1 {
		*stride = 1
	}

	manifest, _, err := pdfmemory.Load(*storeDir)
	if err != nil {
		log.Fatalf("Failed to load store: %v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	emitted := 0
	for index, page := range manifest.Pages {
		if index%*stride != 0 {
			continue
		}
		if *limit > 0 && emitted >= *limit {
			break
		}
		content, err := foldtest.ExtractPageContent(*storeDir, manifest, page.Number)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip page %d: %v\n", page.Number, err)
			continue
		}
		if err := encoder.Encode(map[string]any{
			"page":    page.Number,
			"address": page.Address,
			"content": content,
		}); err != nil {
			log.Fatalf("encode page %d: %v", page.Number, err)
		}
		emitted++
	}
	fmt.Fprintf(os.Stderr, "emitted %d pages\n", emitted)
}

func cmdBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	pdfPath := fs.String("pdf", "", "PDF file path")
	outDir := fs.String("out", "", "Output store directory")

	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *pdfPath == "" || *outDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: tlaloc-fold-bench build -pdf <file> -out <store>\n")
		os.Exit(1)
	}

	// Build the PDF memory store
	result, err := pdfmemory.BuildCorpus([]pdfmemory.SourceSpec{
		{Path: *pdfPath, ID: "doc1"},
	}, *outDir, "fold-bench")
	if err != nil {
		log.Fatalf("Failed to build corpus: %v", err)
	}

	// Output result
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("%s\n", out)
}

func cmdCover(args []string) {
	fs := flag.NewFlagSet("cover", flag.ExitOnError)
	storeDir := fs.String("store", "", "Store directory")

	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *storeDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: tlaloc-fold-bench cover -store <store>\n")
		os.Exit(1)
	}

	// Load manifest
	manifest, _, err := pdfmemory.Load(*storeDir)
	if err != nil {
		log.Fatalf("Failed to load store: %v", err)
	}

	// Generate cover
	cover, err := foldtest.BuildCoverText(*storeDir, manifest, 800)
	if err != nil {
		log.Fatalf("Failed to build cover: %v", err)
	}

	fmt.Println(cover)
}

func cmdAsk(args []string) {
	fs := flag.NewFlagSet("ask", flag.ExitOnError)
	storeDir := fs.String("store", "", "Store directory")
	model := fs.String("model", "lfm2-vl-1.6b", "Model name")
	question := fs.String("question", "", "Question to ask")
	baseURL := fs.String("url", "http://127.0.0.1:1234/v1", "LM Studio API URL")
	maxTurns := fs.Int("turns", 6, "Maximum turns")

	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *storeDir == "" || *question == "" {
		fmt.Fprintf(os.Stderr, "Usage: tlaloc-fold-bench ask -store <store> -model <model> -question <q> [-turns N] [-url URL]\n")
		os.Exit(1)
	}

	// Load manifest and cover
	manifest, _, err := pdfmemory.Load(*storeDir)
	if err != nil {
		log.Fatalf("Failed to load store: %v", err)
	}

	cover, err := foldtest.BuildCoverText(*storeDir, manifest, 800)
	if err != nil {
		log.Fatalf("Failed to build cover: %v", err)
	}

	// Create work directory
	workDir := filepath.Join(*storeDir, ".foldtest-work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("Failed to create work dir: %v", err)
	}

	// Configure session
	config := foldtest.SessionConfig{
		WorkDir:  workDir,
		StoreDir: *storeDir,
		Manifest: manifest,
		Cover:    cover,
		Model:    *model,
		BaseURL:  *baseURL,
		MaxTurns: *maxTurns,
		Budget:   4000,
	}

	// Run session
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	result, err := foldtest.RunSession(ctx, config, *question)
	elapsed := time.Since(start)

	if err != nil {
		log.Fatalf("Session failed: %v", err)
	}

	result.TotalLatencyMs = elapsed.Milliseconds()

	// Output result as JSON
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("%s\n", out)
}

func cmdSwarmAsk(args []string) {
	fs := flag.NewFlagSet("swarm-ask", flag.ExitOnError)
	storeDir := fs.String("store", "", "Store directory")
	model := fs.String("model", "lfm2-vl-1.6b", "Model name")
	question := fs.String("question", "", "Question to ask")
	baseURL := fs.String("url", "http://127.0.0.1:1234/v1", "LM Studio API URL")
	maxTurns := fs.Int("turns", 6, "Maximum turns")
	classifierService := fs.String("classifier-service", "", "questionclass-charcnn-r0 /execute URL (empty = rule-based question classifier)")
	classifierCalibration := fs.String("classifier-calibration", "", "CalibrationProfile JSON the classifier consults before trusting the model")
	groundingService := fs.String("grounding-service", "", "groundingscore-distilled-r0 /execute URL (empty = answerscore grounding judge)")
	groundingCalibration := fs.String("grounding-calibration", "", "CalibrationProfile JSON the consolidator consults before trusting the distilled score")

	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *storeDir == "" || *question == "" {
		fmt.Fprintf(os.Stderr, "Usage: tlaloc-fold-bench swarm-ask -store <store> -model <model> -question <q> [-turns N] [-url URL] [-classifier-service URL]\n")
		os.Exit(1)
	}

	manifest, _, err := pdfmemory.Load(*storeDir)
	if err != nil {
		log.Fatalf("Failed to load store: %v", err)
	}

	cover, err := foldtest.BuildCoverText(*storeDir, manifest, 800)
	if err != nil {
		log.Fatalf("Failed to build cover: %v", err)
	}

	workDir := filepath.Join(*storeDir, ".foldtest-work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("Failed to create work dir: %v", err)
	}

	store := blackboard.New(filepath.Join(*storeDir, ".foldtest-blackboard"))
	runID := fmt.Sprintf("swarm-ask-%d", time.Now().UnixNano())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	answer, report, err := swarmask.Ask(ctx, store, runID, swarmask.AskInput{
		Question:                  *question,
		Cover:                     cover,
		WorkDir:                   workDir,
		StoreDir:                  *storeDir,
		Manifest:                  manifest,
		Model:                     *model,
		BaseURL:                   *baseURL,
		MaxTurns:                  *maxTurns,
		Budget:                    4000,
		ClassifierEndpoint:        *classifierService,
		ClassifierCalibrationPath: *classifierCalibration,
		GroundingEndpoint:         *groundingService,
		GroundingCalibrationPath:  *groundingCalibration,
	})
	if err != nil {
		log.Fatalf("swarm-ask failed: %v", err)
	}

	out, _ := json.MarshalIndent(map[string]any{"answer": answer, "swarm_report": report}, "", "  ")
	fmt.Printf("%s\n", out)
}

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	storeDir := fs.String("store", "", "Store directory")
	model := fs.String("model", "lfm2-vl-1.6b", "Model name")
	baseURL := fs.String("url", "http://127.0.0.1:1234/v1", "LM Studio API URL")
	pages := fs.Int("pages", 5, "Number of pages to sample")
	seed := fs.Int64("seed", 0, "Random seed for reproducibility (0 = current time)")
	flexibility := fs.Float64("flexibility", 0.8, "Answer validation flexibility (0.0-1.0)")
	maxTurns := fs.Int("turns", 6, "Maximum turns per question")
	outFile := fs.String("out", "", "Output JSON file (default: stdout)")
	embeddingService := fs.String("embedding-service", "", "Optional URL of a tlaloc-embedding-scorer /execute endpoint")

	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *storeDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: tlaloc-fold-bench validate -store <store> -model <model> [-pages N] [-seed SEED] [-flexibility F] [-url URL]\n")
		os.Exit(1)
	}

	// Load manifest and cover
	manifest, _, err := pdfmemory.Load(*storeDir)
	if err != nil {
		log.Fatalf("Failed to load store: %v", err)
	}

	cover, err := foldtest.BuildCoverText(*storeDir, manifest, 800)
	if err != nil {
		log.Fatalf("Failed to build cover: %v", err)
	}

	// Create work directory
	workDir := filepath.Join(*storeDir, ".foldtest-validation")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("Failed to create work dir: %v", err)
	}

	// Configure validation
	config := foldtest.ValidationConfig{
		WorkDir:             workDir,
		StoreDir:            *storeDir,
		Manifest:            manifest,
		Cover:               cover,
		Model:               *model,
		BaseURL:             *baseURL,
		MaxTurns:            *maxTurns,
		Budget:              4000,
		PageCount:           manifest.PageCount,
		SampleSize:          *pages,
		RandomSeed:          *seed,
		FlexibilityScore:    *flexibility,
		TimeoutSecs:         30,
		EmbeddingServiceURL: *embeddingService,
	}

	// Run validation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Running validation test with %d random spaced pages...\n", *pages)
	start := time.Now()
	result, err := foldtest.RunValidationTest(ctx, config)
	elapsed := time.Since(start)

	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Validation completed in %v\n", elapsed)
	fmt.Fprintf(os.Stderr, "Average score: %.2f (pages selected: %v)\n",
		result.AggregateMetrics.AverageScore, result.SelectedPages)

	// Output result as JSON
	out, _ := json.MarshalIndent(result, "", "  ")
	if *outFile != "" {
		if err := os.WriteFile(*outFile, out, 0644); err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Results saved to: %s\n", *outFile)
	} else {
		fmt.Printf("%s\n", out)
	}
}
