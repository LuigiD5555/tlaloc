package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// buildOnce compiles the CLI once per test binary invocation and returns the
// path to the executable, following the same self-build pattern the
// tlaloque package's process worker tests use to exercise a real binary.
func buildCLI(t *testing.T) string {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "tlaloc-swarm-bench")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, stderr.String())
	}
	return binaryPath
}

func runCLI(t *testing.T, binary string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func TestCLIDatasetRunSweepManifestEndToEnd(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	datasetPath := filepath.Join(dir, "dataset.json")

	if _, stderr, err := runCLI(t, binary, "dataset", "--seed", "2026", "--count", "60", "--out", datasetPath); err != nil {
		t.Fatalf("dataset: %v\n%s", err, stderr)
	}
	if _, err := os.Stat(datasetPath); err != nil {
		t.Fatal(err)
	}

	baselinePath := filepath.Join(dir, "run-baseline.json")
	if _, stderr, err := runCLI(t, binary, "run", "--dataset", datasetPath, "--out", baselinePath); err != nil {
		t.Fatalf("run: %v\n%s", err, stderr)
	}
	var baseline struct {
		Score struct {
			ExactMatchRate float64 `json:"exact_match_rate"`
		} `json:"score"`
		Topology struct {
			Nodes int `json:"nodes"`
		} `json:"topology"`
	}
	readJSON(t, baselinePath, &baseline)
	if baseline.Score.ExactMatchRate != 1.0 {
		t.Fatalf("baseline exact_match_rate=%v", baseline.Score.ExactMatchRate)
	}
	if baseline.Topology.Nodes != 5 {
		t.Fatalf("baseline nodes=%d, want 5", baseline.Topology.Nodes)
	}

	decomposedPath := filepath.Join(dir, "run-decomposed.json")
	if _, stderr, err := runCLI(t, binary, "run", "--dataset", datasetPath, "--decomposed", "--out", decomposedPath); err != nil {
		t.Fatalf("run --decomposed: %v\n%s", err, stderr)
	}
	var decomposed struct {
		Topology struct {
			Nodes int `json:"nodes"`
			Depth int `json:"depth"`
		} `json:"topology"`
	}
	readJSON(t, decomposedPath, &decomposed)
	if decomposed.Topology.Nodes != 8 || decomposed.Topology.Depth != 4 {
		t.Fatalf("decomposed topology=%+v, want nodes=8 depth=4", decomposed.Topology)
	}

	// The calibrated real-model profile must reproduce a visibly degraded,
	// non-1.0 accuracy — proof the CLI actually wires the proxy, not a
	// silent fallback to the exhaustive lexicon.
	proxyPath := filepath.Join(dir, "run-proxy.json")
	if _, stderr, err := runCLI(t, binary, "run", "--dataset", datasetPath, "--profile", "lfm2vl-proxy", "--out", proxyPath); err != nil {
		t.Fatalf("run --profile lfm2vl-proxy: %v\n%s", err, stderr)
	}
	var proxy struct {
		Score struct {
			ExactMatchRate float64 `json:"exact_match_rate"`
		} `json:"score"`
	}
	readJSON(t, proxyPath, &proxy)
	if proxy.Score.ExactMatchRate >= 1.0 {
		t.Fatalf("lfm2vl-proxy exact_match_rate=%v, want a visibly degraded score", proxy.Score.ExactMatchRate)
	}

	sweepDir := filepath.Join(dir, "sweep")
	if _, stderr, err := runCLI(t, binary, "sweep", "--dataset", datasetPath, "--widths", "1,4,16", "--out-dir", sweepDir); err != nil {
		t.Fatalf("sweep: %v\n%s", err, stderr)
	}
	var summary struct {
		Rows []struct {
			Replicas       int     `json:"replicas"`
			ExactMatchRate float64 `json:"exact_match_rate"`
		} `json:"rows"`
	}
	readJSON(t, filepath.Join(sweepDir, "summary.json"), &summary)
	if len(summary.Rows) != 3 {
		t.Fatalf("sweep rows=%d, want 3", len(summary.Rows))
	}
	for _, row := range summary.Rows {
		if row.ExactMatchRate != 1.0 {
			t.Fatalf("sweep replicas=%d exact_match_rate=%v, want 1.0 (replication preserves accuracy)", row.Replicas, row.ExactMatchRate)
		}
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	if _, stderr, err := runCLI(t, binary, "manifest", "--worker-path", "/opt/worker", "--out", manifestPath); err != nil {
		t.Fatalf("manifest: %v\n%s", err, stderr)
	}
	var manifest struct {
		Schema  string `json:"schema"`
		Workers []struct {
			Descriptor struct {
				ID string `json:"id"`
			} `json:"descriptor"`
			Transport string   `json:"transport"`
			Command   []string `json:"command,omitempty"`
			Endpoint  string   `json:"endpoint,omitempty"`
		} `json:"workers"`
		Plan struct {
			Nodes []struct {
				ID string `json:"id"`
			} `json:"nodes"`
		} `json:"plan"`
	}
	readJSON(t, manifestPath, &manifest)
	if len(manifest.Workers) != 5 || len(manifest.Plan.Nodes) != 5 {
		t.Fatalf("manifest workers=%d plan.nodes=%d, want 5/5", len(manifest.Workers), len(manifest.Plan.Nodes))
	}
	for _, worker := range manifest.Workers {
		if worker.Transport == "PROCESS" && (len(worker.Command) == 0 || worker.Command[0] != "/opt/worker") {
			t.Fatalf("worker %s command=%v, want the --worker-path override", worker.Descriptor.ID, worker.Command)
		}
	}
}

func TestCLIRejectsMissingRequiredFlags(t *testing.T) {
	binary := buildCLI(t)
	if _, _, err := runCLI(t, binary, "dataset"); err == nil {
		t.Fatal("expected dataset without --out to fail")
	}
	if _, _, err := runCLI(t, binary, "run", "--out", "x.json"); err == nil {
		t.Fatal("expected run without --dataset to fail")
	}
	if _, _, err := runCLI(t, binary); err == nil {
		t.Fatal("expected no subcommand to fail")
	}
}

func TestCLIRejectsUnknownProfile(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	datasetPath := filepath.Join(dir, "dataset.json")
	if _, stderr, err := runCLI(t, binary, "dataset", "--seed", "1", "--count", "10", "--out", datasetPath); err != nil {
		t.Fatalf("dataset: %v\n%s", err, stderr)
	}
	if _, _, err := runCLI(t, binary, "run", "--dataset", datasetPath, "--profile", "not-a-real-profile", "--out", filepath.Join(dir, "out.json")); err == nil {
		t.Fatal("expected an unknown profile to be rejected")
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
}
