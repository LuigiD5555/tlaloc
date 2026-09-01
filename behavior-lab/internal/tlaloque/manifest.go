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
	Plan    SwarmPlan    `json:"plan,omitempty"`
}

func LoadSwarmManifest(path string) (SwarmManifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return SwarmManifest{}, err
	}
	var manifest SwarmManifest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return SwarmManifest{}, fmt.Errorf("swarm manifest: %w", err)
	}
	if manifest.Schema == "" {
		manifest.Schema = SwarmManifestSchemaR0
	}
	if manifest.Schema != SwarmManifestSchemaR0 {
		return SwarmManifest{}, fmt.Errorf("unexpected swarm manifest schema %q", manifest.Schema)
	}
	if len(manifest.Workers) == 0 {
		return SwarmManifest{}, fmt.Errorf("manifest requires workers")
	}

	for i := range manifest.Workers {
		spec := &manifest.Workers[i]
		desc, err := spec.Descriptor.Normalize()
		if err != nil {
			return SwarmManifest{}, fmt.Errorf("worker[%d]: %w", i, err)
		}
		spec.Descriptor = desc
		spec.Transport = strings.ToUpper(strings.TrimSpace(spec.Transport))
		if spec.Transport == "" {
			spec.Transport = TransportProcess
			if strings.TrimSpace(spec.Endpoint) != "" {
				spec.Transport = TransportHTTPJSON
			}
		}
		strategy, err := resolveTransportStrategy(spec.Transport)
		if err != nil {
			return SwarmManifest{}, fmt.Errorf("worker %q: %w", desc.ID, err)
		}
		if err := strategy.Validate(*spec, desc); err != nil {
			return SwarmManifest{}, err
		}
	}

	if manifest.Plan.ID != "" || len(manifest.Plan.Nodes) > 0 {
		plan, err := manifest.Plan.Normalize()
		if err != nil {
			return SwarmManifest{}, err
		}
		manifest.Plan = plan
	}
	return manifest, nil
}

func (m SwarmManifest) Registry() (*Registry, error) {
	registry := NewRegistry()
	for _, spec := range m.Workers {
		var timeout time.Duration
		if spec.TimeoutMS > 0 {
			timeout = time.Duration(spec.TimeoutMS) * time.Millisecond
		}
		strategy, err := resolveTransportStrategy(spec.Transport)
		if err != nil {
			return nil, fmt.Errorf("worker %q: %w", spec.Descriptor.ID, err)
		}
		worker := strategy.Build(spec, timeout)
		if err := registry.Register(worker); err != nil {
			return nil, err
		}
	}
	return registry, nil
}
