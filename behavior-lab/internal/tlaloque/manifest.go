package tlaloque

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const SwarmManifestSchemaR0 = "tlaloc.tlaloque-swarm-manifest.r0"

type ProcessWorkerSpec struct {
	Descriptor CapabilityDescriptor `json:"descriptor"`
	Command    []string             `json:"command"`
	TimeoutMS  int                  `json:"timeout_ms,omitempty"`
}

type SwarmManifest struct {
	Schema  string              `json:"schema"`
	Workers []ProcessWorkerSpec `json:"workers"`
	Plan    SwarmPlan           `json:"plan"`
}

func LoadSwarmManifest(path string) (SwarmManifest, error) {
	body, err := os.ReadFile(path)
	if err != nil { return SwarmManifest{}, err }
	var manifest SwarmManifest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return SwarmManifest{}, fmt.Errorf("swarm manifest: %w", err)
	}
	if manifest.Schema == "" { manifest.Schema = SwarmManifestSchemaR0 }
	if manifest.Schema != SwarmManifestSchemaR0 {
		return SwarmManifest{}, fmt.Errorf("unexpected swarm manifest schema %q", manifest.Schema)
	}
	if len(manifest.Workers) == 0 { return SwarmManifest{}, fmt.Errorf("manifest requires workers") }
	for i := range manifest.Workers {
		d, err := manifest.Workers[i].Descriptor.Normalize()
		if err != nil { return SwarmManifest{}, fmt.Errorf("worker[%d]: %w", i, err) }
		manifest.Workers[i].Descriptor = d
		if len(manifest.Workers[i].Command) == 0 {
			return SwarmManifest{}, fmt.Errorf("worker %q command is required", d.ID)
		}
	}
	plan, err := manifest.Plan.Normalize()
	if err != nil { return SwarmManifest{}, err }
	manifest.Plan = plan
	return manifest, nil
}

func (m SwarmManifest) Registry() (*Registry, error) {
	registry := NewRegistry()
	for _, spec := range m.Workers {
		var timeout time.Duration
		if spec.TimeoutMS > 0 { timeout = time.Duration(spec.TimeoutMS) * time.Millisecond }
		if err := registry.Register(ProcessWorker{Desc: spec.Descriptor, Command: spec.Command, Timeout: timeout}); err != nil {
			return nil, err
		}
	}
	return registry, nil
}
