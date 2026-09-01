package runrecord

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var allowedVerdicts = map[string]bool{
	"verify_pass": true,
	"verify_fail": true,
	"parse_fail":  true,
	"unknown":     true,
}

func Finalize(record Record, observedAt time.Time) (Record, error) {
	if record.Schema == "" {
		record.Schema = Schema
	}
	if record.Trace == nil {
		record.Trace = []TransitionEvent{}
	}
	if record.Sampling.Stop == nil {
		record.Sampling.Stop = []string{}
	}
	if record.Repetitions.VerdictDistribution == nil {
		record.Repetitions.VerdictDistribution = map[string]int{}
	}
	environmentHash, err := ComputeEnvHash(record)
	if err != nil {
		return Record{}, err
	}
	record.EnvHash = environmentHash
	if record.RunID == "" {
		record.RunID = NewRunID(observedAt, environmentHash)
	}
	if err := Validate(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func Validate(record Record) error {
	if record.Schema != Schema {
		return fmt.Errorf("run record schema: got %q, want %q", record.Schema, Schema)
	}
	if record.RunID == "" || record.EnvHash == "" || record.VariableAxis == "" {
		return errors.New("run record identity fields are required")
	}
	if record.Component.Tlaloc == "" || record.Component.Origami == "" || record.Component.TonalLock == "" {
		return errors.New("run record component versions are required")
	}
	if record.Model.Provider == "" || record.Model.IDRequested == "" || record.Model.IDReported == "" || record.Model.Endpoint == "" || record.Model.ObservedAt == "" {
		return errors.New("run record model identity is incomplete")
	}
	if record.Prompt.BehaviorSpecID == "" || record.Prompt.PromptIRHash == "" || record.Prompt.CompiledPromptHash == "" {
		return errors.New("run record prompt identity is incomplete")
	}
	if record.Fixture.ID == "" || record.Fixture.SHA256 == "" {
		return errors.New("run record fixture identity is incomplete")
	}
	if !allowedVerdicts[record.Outcome.Verdict] {
		return fmt.Errorf("run record verdict is invalid: %q", record.Outcome.Verdict)
	}
	if record.Outcome.OutputHash == "" || record.Repetitions.N < 1 || record.Replay == "" {
		return errors.New("run record outcome, repetitions, and replay are required")
	}
	expectedHash, err := ComputeEnvHash(record)
	if err != nil {
		return err
	}
	if record.EnvHash != expectedHash {
		return fmt.Errorf("env_hash mismatch: got %s, want %s", record.EnvHash, expectedHash)
	}
	return nil
}

// ComputeEnvHash hashes the complete comparable environment projection. The
// record identity, outcome, and declared variable axis are excluded to avoid
// circular identity and to make exactly one experimental variable comparable.
func ComputeEnvHash(record Record) (string, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	var projection map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&projection); err != nil {
		return "", err
	}
	delete(projection, "run_id")
	delete(projection, "env_hash")
	delete(projection, "outcome")
	if err := deletePath(projection, record.VariableAxis); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func deletePath(root map[string]any, dottedPath string) error {
	parts := strings.Split(dottedPath, ".")
	if len(parts) < 2 {
		return fmt.Errorf("variable_axis must be a dotted field path: %q", dottedPath)
	}
	current := root
	for _, part := range parts[:len(parts)-1] {
		nested, ok := current[part].(map[string]any)
		if !ok {
			return fmt.Errorf("variable_axis path does not exist: %q", dottedPath)
		}
		current = nested
	}
	leaf := parts[len(parts)-1]
	if _, ok := current[leaf]; !ok {
		return fmt.Errorf("variable_axis path does not exist: %q", dottedPath)
	}
	delete(current, leaf)
	return nil
}

func NewRunID(observedAt time.Time, environmentHash string) string {
	shortHash := strings.TrimPrefix(environmentHash, "sha256:")
	if len(shortHash) > 6 {
		shortHash = shortHash[:6]
	}
	return observedAt.UTC().Format(time.RFC3339Nano) + ":" + shortHash
}

func HashOutput(value any) (string, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	encoded = append(encoded, '\n')
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func CaptureHost() Host {
	pythonVersion := commandVersion("python3", "--version")
	return Host{
		OS: runtime.GOOS, CPU: runtime.GOARCH, GPU: "unknown", RAMGB: 0,
		Go: runtime.Version(), Python: pythonVersion,
	}
}

func EncodeReplay(arguments []string) (string, error) {
	if len(arguments) == 0 || arguments[0] == "" {
		return "", errors.New("replay command requires an executable")
	}
	encoded, err := json.Marshal(arguments)
	return string(encoded), err
}

func ReplayAndVerify(ctx context.Context, recordPath string) error {
	record, err := Load(recordPath)
	if err != nil {
		return err
	}
	var arguments []string
	if err := json.Unmarshal([]byte(record.Replay), &arguments); err != nil {
		return fmt.Errorf("decode replay command: %w", err)
	}
	if len(arguments) == 0 {
		return errors.New("decode replay command: command is empty")
	}
	command := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("execute replay: %w", err)
	}
	sum := sha256.Sum256(output)
	actualOutputHash := "sha256:" + hex.EncodeToString(sum[:])
	if actualOutputHash != record.Outcome.OutputHash {
		return fmt.Errorf("replay output_hash mismatch: got %s, want %s", actualOutputHash, record.Outcome.OutputHash)
	}
	return Validate(record)
}

func commandVersion(name string, arguments ...string) string {
	output, err := exec.Command(name, arguments...).CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func HashBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}
