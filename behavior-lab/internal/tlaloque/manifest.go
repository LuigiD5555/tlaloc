package tlaloque

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	SwarmManifestSchemaR0 = "tlaloc.tlaloque-swarm-manifest.r0"
	TransportProcess      = "PROCESS"
)

type WorkerSpec struct {
	Descriptor CapabilityDescriptor `json:"descriptor"`
	Transport  string               `json:"transport,omitempty"`
	Command    []string             `json:"command,omitempty"`
	Endpoint   string               `json:"endpoint,omitempty"`
	TimeoutMS  int                  `json:"timeout_ms,omitempty"`
}

// Backward-compatible source alias for early R0 callers.
type ProcessWorkerSpec = WorkerSpec

type SwarmManifest struct {
	Schema  string       `json:"schema"`
	Workers []WorkerSpec `json:"workers"`
	Plan    SwarmPlan    `json:"plan"`
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
		spec := &manifest.Workers[i]
		d, err := spec.Descriptor.Normalize()
		if err != nil { return SwarmManifest{}, fmt.Errorf("worker[%d]: %w", i, err) }
		spec.Descriptor = d
		spec.Transport = strings.ToUpper(strings.TrimSpace(spec.Transport))
		if spec.Transport == "" {
			if strings.TrimSpace(spec.Endpoint) != "" { spec.Transport = TransportHTTPJSON } else { spec.Transport = TransportProcess }
		}
		switch spec.Transport {
		case TransportProcess:
			if len(spec.Command) == 0 { return SwarmManifest{}, fmt.Errorf("worker %q command is required for PROCESS transport", d.ID) }
		case TransportHTTPJSON:
			if strings.TrimSpace(spec.Endpoint) == "" { return SwarmManifest{}, fmt.Errorf("worker %q endpoint is required for HTTP_JSON transport", d.ID) }
		default:
			return SwarmManifest{}, fmt.Errorf("worker %q unsupported transport %q", d.ID, spec.Transport)
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
		var worker CapabilityWorker
		switch spec.Transport {
		case TransportHTTPJSON:
			worker = HTTPWorker{Desc: spec.Descriptor, Endpoint: spec.Endpoint, Timeout: timeout}
		default:
			worker = ProcessWorker{Desc: spec.Descriptor, Command: spec.Command, Timeout: timeout}
		}
		if err := registry.Register(worker); err != nil { return nil, err }
	}
	return registry, nil
}
