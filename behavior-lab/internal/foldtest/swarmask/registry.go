package swarmask

import (
	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/groundingscore"
	"tlaloc.local/behaviorlab/internal/tlaloque/questionclass"
)

// RegistryConfig points the two model-backed nodes (question classifier,
// consolidator grounding judge) at their optional resident HTTP services
// and calibration profiles. Every field empty = the fully in-process,
// rule-and-answerscore behavior.
type RegistryConfig struct {
	ClassifierEndpoint        string
	ClassifierCalibrationPath string
	GroundingEndpoint         string
	GroundingCalibrationPath  string
	// DisableGroundingAutomaton turns off the deterministic
	// grounding-automaton-r0 first pass in the consolidator (default: on).
	DisableGroundingAutomaton bool
}

// NewRegistry builds a tlaloque.Registry with every swarmask worker
// registered in-process — same convention as answerscore/questiongen and
// internal/lfm2boundary/campaign.go: constructed directly in Go, no
// manifest JSON.
//
//   - ClassifierEndpoint backs the question classifier node with the trained
//     questionclass-charcnn-r0 HTTP service (rule-based classifyQuestion stays
//     as its honest fallback); ClassifierCalibrationPath is the profile it
//     consults before trusting a model prediction.
//   - GroundingEndpoint backs the consolidator with the distilled
//     groundingscore-distilled-r0 judge (independent of the parrot);
//     GroundingCalibrationPath is the profile it consults before trusting the
//     distilled score instead of falling back to answerscore.
func NewRegistry(config RegistryConfig) (*tlaloque.Registry, error) {
	registry := tlaloque.NewRegistry()
	_ = registry.Register(PageScoutWorker{})
	_ = registry.Register(EntityScoutWorker{})

	classifier := QuestionClassifierWorker{}
	if config.ClassifierEndpoint != "" {
		classifier.ModelRegistry = questionclass.NewRegistry(config.ClassifierEndpoint)
	}
	if config.ClassifierCalibrationPath != "" {
		profile, err := questionclass.LoadProfile(config.ClassifierCalibrationPath)
		if err != nil {
			return nil, err
		}
		classifier.Profile = &profile
	}
	_ = registry.Register(classifier)

	consolidator := ConsolidatorWorker{DisableAutomaton: config.DisableGroundingAutomaton}
	if config.GroundingEndpoint != "" {
		consolidator.GroundingRegistry = groundingscore.NewRegistry(config.GroundingEndpoint)
	}
	if config.GroundingCalibrationPath != "" {
		profile, err := groundingscore.LoadProfile(config.GroundingCalibrationPath)
		if err != nil {
			return nil, err
		}
		consolidator.GroundingProfile = &profile
	}

	_ = registry.Register(ParrotAnswerWorker{})
	_ = registry.Register(consolidator)
	return registry, nil
}
