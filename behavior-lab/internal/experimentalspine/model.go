// Package experimentalspine provides the small common output contract used
// by prototype experiments. It does not replace experiment-specific raw
// records; it projects them into a reusable manifest + episodes + summary
// bundle for the next iteration.
package experimentalspine

import (
	"errors"
	"fmt"
)

const (
	ManifestSchema = "tlaloc.experimental-spine.r0.manifest"
	SummarySchema  = "tlaloc.experimental-spine.r0.summary"
)

// RunManifest records why a run exists and exactly which prototype/config
// produced its Episodes. Fields that are genuinely unknown may remain empty;
// the spine never invents provenance.
type RunManifest struct {
	Schema           string    `json:"schema"`
	RunID            string    `json:"run_id"`
	SourceExperiment string    `json:"source_experiment"`
	Prototype        Prototype `json:"prototype"`
	Repos            Repos     `json:"repos"`
	Model            Model     `json:"model"`
	ConfigHash       string    `json:"config_hash,omitempty"`
	RunClass         string    `json:"run_class,omitempty"`
	EvidenceClass    string    `json:"evidence_class,omitempty"`
	StartedAt        string    `json:"started_at,omitempty"`
	FinishedAt       string    `json:"finished_at,omitempty"`
}

type Prototype struct {
	ID            string `json:"id"`
	Version       string `json:"version,omitempty"`
	ParentVersion string `json:"parent_version,omitempty"`
	Hypothesis    string `json:"hypothesis,omitempty"`
}

type Repos struct {
	Tonal   string `json:"tonal,omitempty"`
	Tlaloc  string `json:"tlaloc,omitempty"`
	Origami string `json:"origami,omitempty"`
}

type Model struct {
	Requested string `json:"requested,omitempty"`
	Reported  string `json:"reported,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
}

func (m RunManifest) Validate() error {
	if m.Schema != ManifestSchema {
		return fmt.Errorf("experimentalspine: manifest schema = %q, want %q", m.Schema, ManifestSchema)
	}
	if m.RunID == "" {
		return errors.New("experimentalspine: manifest run_id is empty")
	}
	if m.SourceExperiment == "" {
		return errors.New("experimentalspine: manifest source_experiment is empty")
	}
	if m.Prototype.ID == "" {
		return errors.New("experimentalspine: manifest prototype.id is empty")
	}
	return nil
}
