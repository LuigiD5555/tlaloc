// Package exocortex implements the Tlaloc Exocortex R0 vertical slice
// (E0-E6, docs/EXOCORTEX_R0.md): a runtime CapabilityProfile contract, a
// Micro-ISA opcode vocabulary, a ModelAdapter, a WorkingSetBuilder, a
// CapabilityProfile-aware routing strategy, and the five R0 Tlaloques. It
// deliberately reuses internal/tlaloque (Registry, ResolveGoal, SwarmRunner,
// SwarmPlan, CapabilityWorker) and internal/blackboard rather than
// duplicating them (E0.15).
package exocortex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	// ProfileSchemaR0 identifies a runtime CapabilityProfile document.
	ProfileSchemaR0 = "tlaloc.exocortex-capability-profile.r0"

	// MicroISAArtifactSchemaR0 identifies the frozen P2-A evidence artifact
	// this package compiles CapabilityProfiles from. P2-A itself (results,
	// experiment definition, hashes) is owned by the Behavior Lab campaign
	// that produced it and is never modified here.
	MicroISAArtifactSchemaR0 = "tlaloc.parrot-microisa.r0"
)

// Deployment recommendations a CapabilityRouter can act on.
const (
	DeploymentDeploy            = "DEPLOY"
	DeploymentDeployConstrained = "DEPLOY_WITH_CONSTRAINTS"
	DeploymentExternalize       = "EXTERNALIZE"
	DeploymentDoNotDeploy       = "DO_NOT_DEPLOY"
)

// Intrinsic/transfer verdict vocabulary, preserved verbatim from P2-A.
const (
	VerdictStrong   = "STRONG"
	VerdictUsable   = "USABLE"
	VerdictFragile  = "FRAGILE"
	VerdictUnusable = "UNUSABLE"

	TransferPartial         = "PARTIAL"
	TransferDoesNotTransfer = "DOES_NOT_TRANSFER"
	TransferNotEvaluated    = "NOT_EVALUATED"
)

// MicroISAArtifact is the read-only schema for the frozen P2-A evidence file
// (e.g. results/PARROT_MICRO_ISA_R0.json from experiment
// parrot-microisa-r0.1). It is an import DTO only: this package never writes
// to it and never regenerates, rescales, or "improves" it.
type MicroISAArtifact struct {
	Schema             string                           `json:"schema"`
	ExperimentID       string                           `json:"experiment_id"`
	Records            int                              `json:"records"`
	ExecutionErrors    int                              `json:"execution_errors"`
	Frozen             bool                             `json:"frozen"`
	MaxSafeOpsSemantic int                              `json:"max_safe_ops_semantic"`
	MaxSafeOpsContract int                              `json:"max_safe_ops_contract"`
	Opcodes            map[string]MicroISAOpcodeFinding `json:"opcodes"`
}

// MicroISAOpcodeFinding is one opcode's empirical capability finding from
// P2-A. Fields are pointers/omitempty because not every opcode reports
// every axis (e.g. SELECT_ONE has a choice-width envelope; READ_SHORT_TEXT
// has a character-length curve).
type MicroISAOpcodeFinding struct {
	IntrinsicVerdict          string             `json:"intrinsic_verdict"`
	SyntheticAccuracy         *float64           `json:"synthetic_accuracy,omitempty"`
	PDFTransferVerdict        string             `json:"pdf_transfer_verdict,omitempty"`
	TightCropAccuracy         *float64           `json:"tight_crop_accuracy,omitempty"`
	FullPageAccuracy          *float64           `json:"full_page_accuracy,omitempty"`
	FormalMaxSafeChoiceWidth  *int               `json:"formal_max_safe_choice_width,omitempty"`
	ObservedTestedChoiceWidth *int               `json:"observed_tested_choice_width,omitempty"`
	MaxUsefulChars            *int               `json:"max_useful_chars,omitempty"`
	CharAccuracyCurve         map[string]float64 `json:"char_accuracy_curve,omitempty"`
	ResponseCollapse          bool               `json:"response_collapse,omitempty"`
	ExternalizeCandidate      bool               `json:"externalize_candidate,omitempty"`
	Notes                     string             `json:"notes,omitempty"`
}

// CapabilityProfile is the runtime-readable contract compiled from an
// executor's empirical evidence artifact (E1). It can represent any
// executor, not only Parrot.
type CapabilityProfile struct {
	Schema             string            `json:"schema"`
	ProfileID          string            `json:"profile_id"`
	ProfileVersion     string            `json:"profile_version"`
	ExecutorID         string            `json:"executor_id"`
	ExecutorKind       string            `json:"executor_kind"`
	ModelID            string            `json:"model_id,omitempty"`
	SourceExperiment   string            `json:"source_experiment"`
	SourceArtifactPath string            `json:"source_artifact_path"`
	SourceArtifactHash string            `json:"source_artifact_hash_sha256"`
	MaxSafeOps         int               `json:"max_safe_ops"`
	Capabilities       []CapabilityEntry `json:"capabilities"`
}

// CapabilityEntry describes one opcode's deployment contract for one
// executor.
type CapabilityEntry struct {
	Opcode                   string             `json:"opcode"`
	IntrinsicVerdict         string             `json:"intrinsic_verdict"`
	PDFTransferVerdict       string             `json:"pdf_transfer_verdict,omitempty"`
	DeploymentRecommendation string             `json:"deployment_recommendation"`
	Constraints              Constraints        `json:"constraints"`
	PreferredInput           PreferredInput     `json:"preferred_input"`
	OutputContract           OutputContract     `json:"output_contract"`
	ResponseCollapse         bool               `json:"response_collapse"`
	Confidence               ConfidenceMetadata `json:"confidence,omitempty"`
	FallbackSuggestions      []string           `json:"fallback_suggestions,omitempty"`
}

// Constraints bounds the operand an executor may legally receive for an
// opcode. FormalMaxChoiceWidth is the preregistered conservative rung a
// CapabilityRouter must enforce; ObservedChoiceWidthEnvelope is evidence
// only and must never be substituted for it (P2-A: SELECT_ONE_OF_N showed
// no degradation through width 8, but the formal rung stays 2 under the
// preregistered conservative Wilson rule).
type Constraints struct {
	MaxChars                    int      `json:"max_chars,omitempty"`
	FormalMaxChoiceWidth        int      `json:"formal_max_choice_width,omitempty"`
	ObservedChoiceWidthEnvelope int      `json:"observed_choice_width_envelope,omitempty"`
	AllowedVisualField          []string `json:"allowed_visual_field,omitempty"`
	PreferredVisualField        string   `json:"preferred_visual_field,omitempty"`
	MaxRegionCount              int      `json:"max_region_count,omitempty"`
	RequiredInputType           string   `json:"required_input_type,omitempty"`
	ForbiddenInputModes         []string `json:"forbidden_input_modes,omitempty"`
}

type PreferredInput struct {
	Type      string `json:"type,omitempty"`
	FieldSize string `json:"field_size,omitempty"`
}

type OutputContract struct {
	Type string `json:"type,omitempty"`
}

// ConfidenceMetadata carries the statistical evidence behind a deployment
// recommendation. It is descriptive only; it never widens Constraints.
type ConfidenceMetadata struct {
	SyntheticAccuracy float64 `json:"synthetic_accuracy,omitempty"`
	TightCropAccuracy float64 `json:"tight_crop_accuracy,omitempty"`
	FullPageAccuracy  float64 `json:"full_page_accuracy,omitempty"`
	SampleNote        string  `json:"sample_note,omitempty"`
}

const (
	VisualFieldTightCrop = "TIGHT_CROP"
	VisualFieldFullPage  = "FULL_PAGE"
)

// LoadMicroISAArtifact reads and hash-verifies a frozen P2-A evidence file.
// It rejects unknown fields and an unfrozen or error-containing artifact:
// R0 must not compile a runtime profile from evidence that is still moving
// or that contains execution errors.
func LoadMicroISAArtifact(path string) (MicroISAArtifact, string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return MicroISAArtifact{}, "", fmt.Errorf("read micro-ISA artifact %s: %w", path, err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])

	var artifact MicroISAArtifact
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&artifact); err != nil {
		return MicroISAArtifact{}, "", fmt.Errorf("decode micro-ISA artifact %s: %w", path, err)
	}
	if artifact.Schema != MicroISAArtifactSchemaR0 {
		return MicroISAArtifact{}, "", fmt.Errorf("micro-ISA artifact %s: unexpected schema %q, want %q", path, artifact.Schema, MicroISAArtifactSchemaR0)
	}
	if strings.TrimSpace(artifact.ExperimentID) == "" {
		return MicroISAArtifact{}, "", fmt.Errorf("micro-ISA artifact %s: experiment_id is required", path)
	}
	if !artifact.Frozen {
		return MicroISAArtifact{}, "", fmt.Errorf("micro-ISA artifact %s: experiment %s is not frozen; R0 cannot compile a profile from a moving artifact", path, artifact.ExperimentID)
	}
	if artifact.Records <= 0 {
		return MicroISAArtifact{}, "", fmt.Errorf("micro-ISA artifact %s: records must be positive", path)
	}
	if artifact.ExecutionErrors != 0 {
		return MicroISAArtifact{}, "", fmt.Errorf("micro-ISA artifact %s: %d execution errors recorded; P2-A is expected to freeze at zero", path, artifact.ExecutionErrors)
	}
	if artifact.MaxSafeOpsContract <= 0 || artifact.MaxSafeOpsSemantic <= 0 {
		return MicroISAArtifact{}, "", fmt.Errorf("micro-ISA artifact %s: max_safe_ops_semantic and max_safe_ops_contract must be positive", path)
	}
	if len(artifact.Opcodes) == 0 {
		return MicroISAArtifact{}, "", fmt.Errorf("micro-ISA artifact %s: no opcode findings", path)
	}
	return artifact, hash, nil
}

// CompileParrotProfile compiles the frozen P2-A artifact at path into a
// runtime CapabilityProfile for one Parrot deployment (an LFM2-VL-class
// visual specialist reached at a fixed model id). It is a pure,
// evidence-preserving transform (E1): it never invents a number that is
// not present in the artifact, and it never collapses the formal/observed
// choice-width distinction or a confirmed response collapse into an
// ordinary accuracy figure.
func CompileParrotProfile(artifactPath, executorID, modelID, profileVersion string) (CapabilityProfile, error) {
	artifact, hash, err := LoadMicroISAArtifact(artifactPath)
	if err != nil {
		return CapabilityProfile{}, err
	}
	executorID = strings.TrimSpace(executorID)
	modelID = strings.TrimSpace(modelID)
	profileVersion = strings.TrimSpace(profileVersion)
	if executorID == "" || modelID == "" || profileVersion == "" {
		return CapabilityProfile{}, fmt.Errorf("executor_id, model_id and profile_version are required")
	}

	opcodes := make([]string, 0, len(artifact.Opcodes))
	for op := range artifact.Opcodes {
		opcodes = append(opcodes, op)
	}
	sort.Strings(opcodes)

	entries := make([]CapabilityEntry, 0, len(opcodes))
	for _, op := range opcodes {
		finding := artifact.Opcodes[op]
		entry, err := compileEntry(op, finding)
		if err != nil {
			return CapabilityProfile{}, fmt.Errorf("opcode %s: %w", op, err)
		}
		entries = append(entries, entry)
	}

	maxSafeOps := artifact.MaxSafeOpsContract
	if artifact.MaxSafeOpsSemantic < maxSafeOps {
		maxSafeOps = artifact.MaxSafeOpsSemantic
	}

	profile := CapabilityProfile{
		Schema:             ProfileSchemaR0,
		ProfileID:          executorID + "@" + profileVersion,
		ProfileVersion:     profileVersion,
		ExecutorID:         executorID,
		ExecutorKind:       "MODEL",
		ModelID:            modelID,
		SourceExperiment:   artifact.ExperimentID,
		SourceArtifactPath: artifactPath,
		SourceArtifactHash: hash,
		MaxSafeOps:         maxSafeOps,
		Capabilities:       entries,
	}
	if err := ValidateProfile(profile); err != nil {
		return CapabilityProfile{}, err
	}
	return profile, nil
}

func compileEntry(opcode string, f MicroISAOpcodeFinding) (CapabilityEntry, error) {
	opcode = strings.ToUpper(strings.TrimSpace(opcode))
	if opcode == "" {
		return CapabilityEntry{}, fmt.Errorf("opcode is required")
	}
	verdict := strings.ToUpper(strings.TrimSpace(f.IntrinsicVerdict))
	switch verdict {
	case VerdictStrong, VerdictUsable, VerdictFragile, VerdictUnusable:
	default:
		return CapabilityEntry{}, fmt.Errorf("unknown intrinsic_verdict %q", f.IntrinsicVerdict)
	}
	transfer := strings.ToUpper(strings.TrimSpace(f.PDFTransferVerdict))

	if f.FormalMaxSafeChoiceWidth != nil && f.ObservedTestedChoiceWidth != nil {
		if *f.ObservedTestedChoiceWidth < *f.FormalMaxSafeChoiceWidth {
			return CapabilityEntry{}, fmt.Errorf("observed_tested_choice_width (%d) is narrower than formal_max_safe_choice_width (%d); the formal rung must be the more conservative of the two", *f.ObservedTestedChoiceWidth, *f.FormalMaxSafeChoiceWidth)
		}
	}

	deployment := DeploymentDeploy
	switch {
	case f.ResponseCollapse || f.ExternalizeCandidate:
		deployment = DeploymentExternalize
	case verdict == VerdictUnusable:
		deployment = DeploymentDoNotDeploy
	case transfer == TransferDoesNotTransfer:
		deployment = DeploymentExternalize
	case transfer == TransferPartial || verdict == VerdictFragile:
		deployment = DeploymentDeployConstrained
	}

	constraints := Constraints{}
	if f.MaxUsefulChars != nil {
		constraints.MaxChars = *f.MaxUsefulChars
	}
	if f.FormalMaxSafeChoiceWidth != nil {
		constraints.FormalMaxChoiceWidth = *f.FormalMaxSafeChoiceWidth
	}
	if f.ObservedTestedChoiceWidth != nil {
		constraints.ObservedChoiceWidthEnvelope = *f.ObservedTestedChoiceWidth
	}
	if f.TightCropAccuracy != nil || f.FullPageAccuracy != nil {
		if f.TightCropAccuracy != nil && f.FullPageAccuracy != nil && *f.TightCropAccuracy > *f.FullPageAccuracy {
			constraints.AllowedVisualField = []string{VisualFieldTightCrop}
			constraints.PreferredVisualField = VisualFieldTightCrop
		} else {
			constraints.AllowedVisualField = []string{VisualFieldTightCrop, VisualFieldFullPage}
			constraints.PreferredVisualField = VisualFieldTightCrop
		}
	}

	confidence := ConfidenceMetadata{}
	if f.SyntheticAccuracy != nil {
		confidence.SyntheticAccuracy = *f.SyntheticAccuracy
	}
	if f.TightCropAccuracy != nil {
		confidence.TightCropAccuracy = *f.TightCropAccuracy
	}
	if f.FullPageAccuracy != nil {
		confidence.FullPageAccuracy = *f.FullPageAccuracy
	}
	confidence.SampleNote = f.Notes

	var fallbacks []string
	if f.ResponseCollapse {
		fallbacks = append(fallbacks, "RESPONSE_COLLAPSE_CONFIRMED: route to deterministic or externalized alternative")
	}
	if f.ExternalizeCandidate {
		fallbacks = append(fallbacks, "EXTERNALIZE_CANDIDATE: prefer deterministic Tlaloque over Parrot")
	}

	outputType := "plain_text"
	if constraints.FormalMaxChoiceWidth > 0 {
		outputType = "single_token_choice"
	}

	return CapabilityEntry{
		Opcode:                   opcode,
		IntrinsicVerdict:         verdict,
		PDFTransferVerdict:       transfer,
		DeploymentRecommendation: deployment,
		Constraints:              constraints,
		PreferredInput:           PreferredInput{Type: "image_crop", FieldSize: constraints.PreferredVisualField},
		OutputContract:           OutputContract{Type: outputType},
		ResponseCollapse:         f.ResponseCollapse,
		Confidence:               confidence,
		FallbackSuggestions:      fallbacks,
	}, nil
}

// ValidateProfile checks structural and semantic invariants a runtime
// CapabilityProfile must satisfy before a CapabilityRouter may use it.
func ValidateProfile(p CapabilityProfile) error {
	if p.Schema != ProfileSchemaR0 {
		return fmt.Errorf("capability profile: unexpected schema %q, want %q", p.Schema, ProfileSchemaR0)
	}
	if strings.TrimSpace(p.ProfileID) == "" || strings.TrimSpace(p.ExecutorID) == "" || strings.TrimSpace(p.ExecutorKind) == "" {
		return fmt.Errorf("capability profile: profile_id, executor_id and executor_kind are required")
	}
	if strings.TrimSpace(p.SourceExperiment) == "" || strings.TrimSpace(p.SourceArtifactHash) == "" {
		return fmt.Errorf("capability profile: source_experiment and source_artifact_hash_sha256 are required")
	}
	if p.MaxSafeOps <= 0 {
		return fmt.Errorf("capability profile: max_safe_ops must be positive")
	}
	if len(p.Capabilities) == 0 {
		return fmt.Errorf("capability profile: at least one capability entry is required")
	}
	seen := map[string]bool{}
	for _, c := range p.Capabilities {
		if c.Opcode == "" {
			return fmt.Errorf("capability profile: entry with empty opcode")
		}
		if seen[c.Opcode] {
			return fmt.Errorf("capability profile: duplicate opcode %q", c.Opcode)
		}
		seen[c.Opcode] = true
		switch c.DeploymentRecommendation {
		case DeploymentDeploy, DeploymentDeployConstrained, DeploymentExternalize, DeploymentDoNotDeploy:
		default:
			return fmt.Errorf("capability profile: opcode %s has unknown deployment_recommendation %q", c.Opcode, c.DeploymentRecommendation)
		}
		if c.Constraints.FormalMaxChoiceWidth > 0 && c.Constraints.ObservedChoiceWidthEnvelope > 0 &&
			c.Constraints.ObservedChoiceWidthEnvelope < c.Constraints.FormalMaxChoiceWidth {
			return fmt.Errorf("capability profile: opcode %s observed_choice_width_envelope (%d) narrower than formal_max_choice_width (%d)", c.Opcode, c.Constraints.ObservedChoiceWidthEnvelope, c.Constraints.FormalMaxChoiceWidth)
		}
	}
	return nil
}

// LoadProfile reads and validates a runtime CapabilityProfile JSON document
// from disk. Unlike LoadMicroISAArtifact it disallows unknown fields on the
// already-compiled runtime shape, not on the raw P2-A evidence shape.
func LoadProfile(path string) (CapabilityProfile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return CapabilityProfile{}, fmt.Errorf("read capability profile %s: %w", path, err)
	}
	var profile CapabilityProfile
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&profile); err != nil {
		return CapabilityProfile{}, fmt.Errorf("decode capability profile %s: %w", path, err)
	}
	if err := ValidateProfile(profile); err != nil {
		return CapabilityProfile{}, fmt.Errorf("%s: %w", path, err)
	}
	return profile, nil
}

// WriteProfile writes a validated CapabilityProfile as canonical indented
// JSON. Callers are expected to validate first; WriteProfile validates
// again defensively so a caller cannot persist a broken contract.
func WriteProfile(path string, profile CapabilityProfile) error {
	if err := ValidateProfile(profile); err != nil {
		return err
	}
	body, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

// VerifySourceArtifact recomputes the SHA-256 of the artifact a profile was
// compiled from and confirms it still matches SourceArtifactHash. This is
// the hash-validation step the T0 doctor command runs before treating a
// profile as trustworthy evidence.
func VerifySourceArtifact(profile CapabilityProfile) error {
	if strings.TrimSpace(profile.SourceArtifactPath) == "" {
		return fmt.Errorf("capability profile %s has no source_artifact_path to verify", profile.ProfileID)
	}
	body, err := os.ReadFile(profile.SourceArtifactPath)
	if err != nil {
		return fmt.Errorf("verify source artifact for %s: %w", profile.ProfileID, err)
	}
	sum := sha256.Sum256(body)
	actual := hex.EncodeToString(sum[:])
	if actual != profile.SourceArtifactHash {
		return fmt.Errorf("capability profile %s: source artifact hash mismatch: got %s, want %s", profile.ProfileID, actual, profile.SourceArtifactHash)
	}
	return nil
}

// Entry looks up one opcode's CapabilityEntry.
func (p CapabilityProfile) Entry(opcode string) (CapabilityEntry, bool) {
	opcode = strings.ToUpper(strings.TrimSpace(opcode))
	for _, c := range p.Capabilities {
		if c.Opcode == opcode {
			return c, true
		}
	}
	return CapabilityEntry{}, false
}
