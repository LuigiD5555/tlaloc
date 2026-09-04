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

// CapabilityProfile R1 — a machine-readable runtime evidence contract
// compiled from the full LFM2-VL 1.6B characterisation campaign (P1, P2,
// T0-A, T0-B, R1-A0…R1-G). It is a DISTINCT document from the R0
// CapabilityProfile (ProfileSchemaR0), which is preserved and never
// overwritten. R1 carries scale/context/repeatability envelopes and the
// R1-G recovery/prevention rules, each with full per-rule provenance.

// CapabilityProfileR1Schema identifies an R1 runtime evidence contract.
const CapabilityProfileR1Schema = "tlaloc.capability-profile.r1"

// Evidence classes (protocol §1). An adapter rule may only be given formal
// runtime authority when its evidence class is EARNED or a REJECT/DO_NOT_USE
// safety rule; every other class is advisory.
const (
	EvidenceEarned                  = "EARNED"
	EvidencePromising               = "PROMISING"
	EvidenceObservedExploratory     = "OBSERVED_EXPLORATORY"
	EvidencePreventivePractice      = "PREVENTIVE_PRACTICE"
	EvidenceSafeEngineeringDefault  = "SAFE_ENGINEERING_DEFAULT"
	EvidenceNoMeasuredBenefit       = "NO_MEASURED_BENEFIT"
	EvidenceSyntheticProxyLimit     = "SYNTHETIC_PROXY_LIMITATION"
	EvidenceRejectBeforeCall        = "REJECT_BEFORE_CALL"
	EvidenceDoNotUse                = "DO_NOT_USE"
	EvidenceUnknown                 = "UNKNOWN"
	EvidenceUnproven                = "UNPROVEN"
)

var validEvidenceClasses = map[string]bool{
	EvidenceEarned: true, EvidencePromising: true, EvidenceObservedExploratory: true,
	EvidencePreventivePractice: true, EvidenceSafeEngineeringDefault: true,
	EvidenceNoMeasuredBenefit: true, EvidenceSyntheticProxyLimit: true,
	EvidenceRejectBeforeCall: true, EvidenceDoNotUse: true, EvidenceUnknown: true, EvidenceUnproven: true,
}

// EvidenceRef is the provenance every profile rule must carry (protocol §3).
type EvidenceRef struct {
	SourceExperiment string `json:"source_experiment"`
	ArtifactPath     string `json:"artifact_path"`
	ArtifactSHA256   string `json:"artifact_sha256"`
	SampleSize       int    `json:"sample_size"`
	EvidenceClass    string `json:"evidence_class"`
	MeasuredMetric   string `json:"measured_metric"`
	Limitations      string `json:"limitations"`
}

// ExecutorIdentityR1 pins the exact executor the profile describes.
type ExecutorIdentityR1 struct {
	ModelID          string `json:"model_id"`
	Family           string `json:"family"`
	Publisher        string `json:"publisher"`
	Quantization     string `json:"quantization"`
	Architecture     string `json:"architecture"`
	WeightsGGUFSHA   string `json:"weights_gguf_sha256"`
	MMProjGGUFSHA    string `json:"mmproj_gguf_sha256"`
	LMStudioVersion  string `json:"lm_studio_version"`
	BackendVersion   string `json:"inference_backend_version"`
	RuntimeLibHashes map[string]string `json:"runtime_binary_hashes,omitempty"`
	ContextLength    int    `json:"context_length_loaded"`
	Endpoint         string `json:"endpoint"`
	Temperature      float64 `json:"temperature"`
	MaxOutputTokens  int    `json:"max_output_tokens"`
	IdentitySHA256   string `json:"model_identity_artifact_sha256"`
}

// SourceExperimentRef records one campaign artifact the profile compiled from.
type SourceExperimentRef struct {
	ID             string `json:"id"`
	ArtifactPath   string `json:"artifact_path"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	Records        int    `json:"records,omitempty"`
	Frozen         bool   `json:"frozen"`
	Status         string `json:"status,omitempty"`
}

// GlobalExecutorRules (protocol §4).
type GlobalExecutorRules struct {
	MaxCognitiveTransformationsPerCall int         `json:"max_cognitive_transformations_per_call"`
	Rule                               string      `json:"rule"`
	FormattingIsExternalDeterministic  bool        `json:"formatting_normalization_serialization_is_external_deterministic"`
	SequenceWorkingMemoryIsTlaloc      bool        `json:"sequence_and_working_memory_is_tlaloc_responsibility"`
	ModelResultIsObservationNotFact    bool        `json:"model_result_is_observation_never_authoritative_fact"`
	Evidence                           EvidenceRef `json:"evidence"`
}

// ScaleRung is one measured EXTRACT_NUMBER scale point.
type ScaleRung struct {
	LineHeightPx int     `json:"line_height_px"`
	Accuracy     float64 `json:"accuracy"`
	Verdict      string  `json:"verdict"`
}

// ExtractNumberProfile (protocol §5).
type ExtractNumberProfile struct {
	ScaleRungs                []ScaleRung `json:"scale_rungs"`
	FormalSafeScalePx         int         `json:"formal_safe_scale_px"`
	PreferredScalePx          int         `json:"preferred_scale_px"`
	ObservedOperatingRegionPx [2]int      `json:"observed_operating_region_px"`
	Scope                     string      `json:"scope"`
	Evidence                  EvidenceRef `json:"evidence"`
}

// ContextEnvelope is one context track's measured curve.
type ContextEnvelope struct {
	Name     string             `json:"name"`
	Points   map[string]float64 `json:"points"`
	Pipeline string             `json:"pipeline"`
}

// ContextProfile (protocol §6) — two envelopes, never causally decomposed.
type ContextProfile struct {
	NaturalVisualField        ContextEnvelope `json:"natural_visual_field"`
	FixedScaleLocalContext    ContextEnvelope `json:"fixed_scale_local_context"`
	AllowedConclusion         string          `json:"allowed_conclusion"`
	ForbiddenDecomposition    string          `json:"forbidden_causal_decomposition"`
	RuntimePreference         string          `json:"runtime_context_preference"`
	AggressiveReductionClass  string          `json:"aggressive_context_reduction_class"`
	Evidence                  []EvidenceRef   `json:"evidence"`
}

// ReadAssociatedNumberProfile (protocol §7).
type ReadAssociatedNumberProfile struct {
	Opcode                      string            `json:"opcode"`
	MicroISAPromoted            bool              `json:"micro_isa_promoted"`
	VisualDependence            bool              `json:"visual_dependence"`
	Evidence                    map[string]string `json:"visual_dependence_evidence"`
	Verdict                     string            `json:"verdict"`
	TestedEnvelope              []string          `json:"tested_envelope"`
	ObservedExploratoryExitK    int               `json:"observed_exploratory_exit_k"`
	FormalMaxDistractors        string            `json:"formal_max_distractors"`
	CompetitorRemovalProvenance string            `json:"competitor_removal_provenance"`
	EvidenceRefs                []EvidenceRef     `json:"evidence_refs"`
}

// MorphologyFamilyProfile (protocol §8) — real and synthetic never pooled.
type MorphologyFamilyProfile struct {
	Family             string     `json:"family"`
	RealN              int        `json:"real_document_n"`
	SyntheticN         int        `json:"synthetic_realistic_n"`
	RealValueAccuracy  *float64   `json:"real_value_accuracy,omitempty"`
	RealSurfaceAccuracy *float64  `json:"real_surface_accuracy,omitempty"`
	RealValueCI95      *[2]float64 `json:"real_value_ci95,omitempty"`
	SynValueAccuracy   *float64   `json:"synthetic_value_accuracy,omitempty"`
	SynSurfaceAccuracy *float64   `json:"synthetic_surface_accuracy,omitempty"`
	ProvisionalVerdict string     `json:"provisional_verdict"`
	FailureModes       []string   `json:"failure_modes,omitempty"`
	NeverPooled        bool       `json:"real_and_synthetic_never_pooled"`
}

// RepeatabilityProfile (protocol §9).
type RepeatabilityProfile struct {
	Sentinels       int         `json:"sentinels"`
	Repeats         int         `json:"repeats_per_sentinel"`
	ByteStable      string      `json:"byte_stable"`
	SemanticStable  string      `json:"semantic_stable"`
	FailedSentinels int         `json:"failed_sentinels"`
	Recoveries      int         `json:"exact_retry_recoveries"`
	RuntimeRule     string      `json:"runtime_rule"`
	EvidenceScope   string      `json:"evidence_scope"`
	Evidence        EvidenceRef `json:"evidence"`
}

// RecoveryRule is one R1-G recovery/prevention policy (protocol §10).
type RecoveryRule struct {
	ID              string      `json:"id"`
	DetectIf        string      `json:"detect_if"`
	PreferredAction string      `json:"preferred_action"`
	Classification  string      `json:"classification"`
	ModelCallsAfter int         `json:"model_calls_after_action"` // -1 = one call as usual, 0 = zero calls (reject)
	Evidence        EvidenceRef `json:"evidence"`
}

// CapabilityProfileR1 is the full frozen R1 runtime evidence contract.
type CapabilityProfileR1 struct {
	Schema            string                      `json:"schema"`
	ProfileID         string                      `json:"profile_id"`
	ProfileVersion    string                      `json:"profile_version"`
	CompiledAt        string                      `json:"compiled_at"`
	PreservesR0       string                      `json:"preserves_r0"`
	Executor          ExecutorIdentityR1          `json:"executor"`
	SourceExperiments []SourceExperimentRef       `json:"source_experiments"`
	GlobalRules       GlobalExecutorRules         `json:"global_executor_rules"`
	ExtractNumber     ExtractNumberProfile        `json:"extract_number"`
	Context           ContextProfile              `json:"context"`
	ReadAssociated    ReadAssociatedNumberProfile `json:"read_associated_number"`
	Morphology        []MorphologyFamilyProfile   `json:"morphology"`
	MorphologyEvidence EvidenceRef                `json:"morphology_evidence"`
	Repeatability     RepeatabilityProfile        `json:"repeatability"`
	RecoveryRules     []RecoveryRule              `json:"recovery_rules"`
	ProfileHash       string                      `json:"profile_hash_sha256"`
}

// ComputeProfileR1Hash returns the canonical sha256 of the profile with the
// ProfileHash field cleared (so the hash is stable and self-consistent).
func ComputeProfileR1Hash(p CapabilityProfileR1) (string, error) {
	p.ProfileHash = ""
	body, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

// r1RecoveryRuleIDs is the frozen set of recovery-rule IDs the profile must
// carry.
var r1RequiredRecoveryRules = []string{
	"LOW_SCALE", "NUMERIC_COMPETITORS", "HIGH_CONTEXT", "VALUE_CUE",
	"MISSING_VISUAL_OPERAND", "EXACT_RETRY", "CURRENT_TESSERACT_OCR",
}

func detectIfMentionsGroundTruth(s string) bool {
	low := strings.ToLower(s)
	for _, bad := range []string{"base_id", "expected", "gold", "answer", "scorer", "dataset_family", "benchmark"} {
		if strings.Contains(low, bad) {
			return true
		}
	}
	return false
}

// ValidateCapabilityProfileR1 enforces every structural + semantic invariant
// a CapabilityRouter must be able to rely on (protocol §11). It rejects
// malformed evidence links, unknown opcodes, impossible ranges, missing
// source hashes, conflicting formal rules, and exploratory evidence used as
// a formal constraint.
func ValidateCapabilityProfileR1(p CapabilityProfileR1) error {
	if p.Schema != CapabilityProfileR1Schema {
		return fmt.Errorf("profile r1: schema %q, want %q", p.Schema, CapabilityProfileR1Schema)
	}
	if strings.TrimSpace(p.ProfileID) == "" || strings.TrimSpace(p.ProfileVersion) == "" {
		return fmt.Errorf("profile r1: profile_id and profile_version required")
	}
	if !strings.Contains(strings.ToLower(p.PreservesR0), "not overwrit") && !strings.Contains(strings.ToLower(p.PreservesR0), "preserved") {
		return fmt.Errorf("profile r1: preserves_r0 must state R0 is preserved / not overwritten")
	}

	// executor identity
	e := p.Executor
	if e.ModelID == "" || e.WeightsGGUFSHA == "" || e.MMProjGGUFSHA == "" || e.Architecture == "" || e.IdentitySHA256 == "" {
		return fmt.Errorf("profile r1: executor identity incomplete (model_id/weights_sha/mmproj_sha/architecture/identity_sha all required)")
	}
	if len(e.WeightsGGUFSHA) != 64 || len(e.MMProjGGUFSHA) != 64 {
		return fmt.Errorf("profile r1: executor gguf hashes must be 64 hex chars")
	}

	// source experiments
	if len(p.SourceExperiments) < 8 {
		return fmt.Errorf("profile r1: expected the full campaign in source_experiments, got %d", len(p.SourceExperiments))
	}
	for _, s := range p.SourceExperiments {
		if s.ID == "" || len(s.ArtifactSHA256) != 64 {
			return fmt.Errorf("profile r1: source experiment %q missing id / 64-hex artifact sha", s.ID)
		}
		if !s.Frozen {
			return fmt.Errorf("profile r1: source experiment %q is not frozen", s.ID)
		}
	}

	checkEvidence := func(where string, ev EvidenceRef, allowZeroSample bool) error {
		if ev.SourceExperiment == "" || len(ev.ArtifactSHA256) != 64 || ev.MeasuredMetric == "" {
			return fmt.Errorf("profile r1: %s evidence link malformed (source/artifact_sha/metric required)", where)
		}
		if !validEvidenceClasses[ev.EvidenceClass] {
			return fmt.Errorf("profile r1: %s evidence_class %q not recognised", where, ev.EvidenceClass)
		}
		if ev.SampleSize <= 0 && !allowZeroSample {
			return fmt.Errorf("profile r1: %s evidence sample_size must be positive", where)
		}
		return nil
	}

	// global rules
	if p.GlobalRules.MaxCognitiveTransformationsPerCall != 1 {
		return fmt.Errorf("profile r1: global max_cognitive_transformations_per_call must be 1 (P1 formal_safe)")
	}
	if !p.GlobalRules.FormattingIsExternalDeterministic || !p.GlobalRules.SequenceWorkingMemoryIsTlaloc || !p.GlobalRules.ModelResultIsObservationNotFact {
		return fmt.Errorf("profile r1: global executor rules must all hold (formatting external, sequence=tlaloc, result=observation)")
	}
	if err := checkEvidence("global_rules", p.GlobalRules.Evidence, false); err != nil {
		return err
	}

	// extract number
	en := p.ExtractNumber
	if len(en.ScaleRungs) < 5 {
		return fmt.Errorf("profile r1: extract_number needs the measured scale ladder")
	}
	for i := 1; i < len(en.ScaleRungs); i++ {
		if en.ScaleRungs[i].LineHeightPx <= en.ScaleRungs[i-1].LineHeightPx {
			return fmt.Errorf("profile r1: extract_number scale rungs must be strictly ascending")
		}
	}
	for _, r := range en.ScaleRungs {
		if r.Accuracy < 0 || r.Accuracy > 1 {
			return fmt.Errorf("profile r1: extract_number rung %dpx accuracy %.3f out of [0,1]", r.LineHeightPx, r.Accuracy)
		}
	}
	if en.FormalSafeScalePx <= 0 || en.PreferredScalePx < en.FormalSafeScalePx {
		return fmt.Errorf("profile r1: extract_number formal_safe_scale_px (%d) / preferred_scale_px (%d) invalid", en.FormalSafeScalePx, en.PreferredScalePx)
	}
	if en.ObservedOperatingRegionPx[0] > en.FormalSafeScalePx || en.ObservedOperatingRegionPx[1] < en.PreferredScalePx {
		return fmt.Errorf("profile r1: extract_number observed operating region %v does not contain [safe %d, preferred %d]", en.ObservedOperatingRegionPx, en.FormalSafeScalePx, en.PreferredScalePx)
	}
	if !strings.Contains(strings.ToLower(en.Scope), "lfm2") {
		return fmt.Errorf("profile r1: extract_number scope must name the tested LFM2-VL runtime / presentation family / renderer")
	}
	if err := checkEvidence("extract_number", en.Evidence, false); err != nil {
		return err
	}

	// context — no causal decomposition, aggressive reduction is a practice
	if strings.TrimSpace(p.Context.ForbiddenDecomposition) == "" || strings.TrimSpace(p.Context.AllowedConclusion) == "" {
		return fmt.Errorf("profile r1: context must record the allowed conclusion and the forbidden causal decomposition")
	}
	if p.Context.AggressiveReductionClass != EvidencePreventivePractice {
		return fmt.Errorf("profile r1: aggressive context reduction must be classed PREVENTIVE_PRACTICE, not %q", p.Context.AggressiveReductionClass)
	}
	if len(p.Context.Evidence) < 2 {
		return fmt.Errorf("profile r1: context needs both envelope evidence links")
	}
	for i, ev := range p.Context.Evidence {
		if err := checkEvidence(fmt.Sprintf("context[%d]", i), ev, false); err != nil {
			return err
		}
	}

	// read associated number — exploratory not promoted to a formal max
	ra := p.ReadAssociated
	if ra.Opcode != "READ_ASSOCIATED_NUMBER" {
		return fmt.Errorf("profile r1: read_associated_number.opcode must be READ_ASSOCIATED_NUMBER")
	}
	if !ra.VisualDependence {
		return fmt.Errorf("profile r1: read_associated_number.visual_dependence must be true (R1-E)")
	}
	if strings.ToUpper(strings.TrimSpace(ra.FormalMaxDistractors)) != "UNKNOWN" {
		return fmt.Errorf("profile r1: read_associated_number.formal_max_distractors must be UNKNOWN (R1-D ladder was exploratory)")
	}
	if ra.ObservedExploratoryExitK != 1 {
		return fmt.Errorf("profile r1: read_associated_number.observed_exploratory_exit_k must record the observed K=1")
	}
	if !strings.Contains(strings.ToLower(ra.CompetitorRemovalProvenance), "intervention reuse") {
		return fmt.Errorf("profile r1: read_associated_number competitor-removal provenance must state it is R1-D intervention reuse, not a fresh independent estimate")
	}
	for i, ev := range ra.EvidenceRefs {
		if err := checkEvidence(fmt.Sprintf("read_associated_number[%d]", i), ev, false); err != nil {
			return err
		}
	}

	// morphology — never pooled
	if len(p.Morphology) == 0 {
		return fmt.Errorf("profile r1: morphology profile is empty")
	}
	for _, m := range p.Morphology {
		if !m.NeverPooled {
			return fmt.Errorf("profile r1: morphology family %q must set real_and_synthetic_never_pooled", m.Family)
		}
		if m.ProvisionalVerdict == "" {
			return fmt.Errorf("profile r1: morphology family %q missing provisional_verdict", m.Family)
		}
	}
	if err := checkEvidence("morphology", p.MorphologyEvidence, false); err != nil {
		return err
	}

	// repeatability
	rp := p.Repeatability
	if rp.Sentinels != 20 || rp.Repeats != 5 {
		return fmt.Errorf("profile r1: repeatability must record 20 sentinels x 5 repeats (R1-F)")
	}
	if rp.RuntimeRule != "DO_NOT_RETRY_IDENTICAL_INPUT" {
		return fmt.Errorf("profile r1: repeatability runtime_rule must be DO_NOT_RETRY_IDENTICAL_INPUT")
	}
	if !strings.Contains(strings.ToLower(rp.EvidenceScope), "temperature 0") {
		return fmt.Errorf("profile r1: repeatability evidence_scope must be bounded to temperature 0 / byte-identical input")
	}
	if err := checkEvidence("repeatability", rp.Evidence, false); err != nil {
		return err
	}

	// recovery rules
	byID := map[string]RecoveryRule{}
	for _, r := range p.RecoveryRules {
		if _, dup := byID[r.ID]; dup {
			return fmt.Errorf("profile r1: duplicate recovery rule %q", r.ID)
		}
		byID[r.ID] = r
		if detectIfMentionsGroundTruth(r.DetectIf) {
			return fmt.Errorf("profile r1: recovery rule %q detect_if references ground truth / base id / scorer", r.ID)
		}
		allowZero := r.ID == "EXACT_RETRY" || r.ID == "MISSING_VISUAL_OPERAND" || r.ID == "CURRENT_TESSERACT_OCR"
		if err := checkEvidence("recovery_rule "+r.ID, r.Evidence, allowZero); err != nil {
			return err
		}
	}
	for _, id := range r1RequiredRecoveryRules {
		if _, ok := byID[id]; !ok {
			return fmt.Errorf("profile r1: missing required recovery rule %q", id)
		}
	}
	// conflicting formal rules / exploratory used as formal
	if byID["LOW_SCALE"].Classification != "PREVENTIVE+EARNED" {
		return fmt.Errorf("profile r1: LOW_SCALE must be classified PREVENTIVE+EARNED")
	}
	if !strings.Contains(byID["LOW_SCALE"].DetectIf, fmt.Sprintf("< %d", en.FormalSafeScalePx)) {
		return fmt.Errorf("profile r1: LOW_SCALE detect_if threshold must match extract_number.formal_safe_scale_px (%d)", en.FormalSafeScalePx)
	}
	if byID["LOW_SCALE"].Evidence.EvidenceClass != EvidenceEarned {
		return fmt.Errorf("profile r1: LOW_SCALE evidence_class must be EARNED")
	}
	if byID["HIGH_CONTEXT"].Classification == "PREVENTIVE+EARNED" || byID["HIGH_CONTEXT"].Evidence.EvidenceClass == EvidenceEarned {
		return fmt.Errorf("profile r1: HIGH_CONTEXT must NOT be EARNED (R1-G showed NO_MEASURED_BENEFIT on the fresh sample)")
	}
	if byID["VALUE_CUE"].Evidence.EvidenceClass != EvidenceSafeEngineeringDefault {
		return fmt.Errorf("profile r1: VALUE_CUE must be SAFE_ENGINEERING_DEFAULT, not an earned recovery")
	}
	if byID["EXACT_RETRY"].Classification != "REJECT" || byID["EXACT_RETRY"].ModelCallsAfter != 0 {
		return fmt.Errorf("profile r1: EXACT_RETRY must be REJECT with 0 model calls")
	}
	if byID["MISSING_VISUAL_OPERAND"].ModelCallsAfter != 0 {
		return fmt.Errorf("profile r1: MISSING_VISUAL_OPERAND must produce 0 model calls")
	}
	if byID["CURRENT_TESSERACT_OCR"].Evidence.EvidenceClass != EvidenceDoNotUse {
		return fmt.Errorf("profile r1: CURRENT_TESSERACT_OCR must be DO_NOT_USE")
	}
	if byID["NUMERIC_COMPETITORS"].Evidence.EvidenceClass != EvidenceEarned {
		return fmt.Errorf("profile r1: NUMERIC_COMPETITORS must be EARNED (R1-G real intervention track)")
	}

	// self-consistent hash
	if p.ProfileHash != "" {
		want, err := ComputeProfileR1Hash(p)
		if err != nil {
			return err
		}
		if want != p.ProfileHash {
			return fmt.Errorf("profile r1: profile_hash_sha256 mismatch (stored %s, computed %s)", p.ProfileHash, want)
		}
	}
	return nil
}

// LoadCapabilityProfileR1 reads and validates an R1 profile document.
func LoadCapabilityProfileR1(path string) (CapabilityProfileR1, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return CapabilityProfileR1{}, err
	}
	var p CapabilityProfileR1
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return CapabilityProfileR1{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := ValidateCapabilityProfileR1(p); err != nil {
		return CapabilityProfileR1{}, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

// WriteCapabilityProfileR1 validates, stamps the deterministic hash, and
// writes the profile as canonical indented JSON. Returns the profile hash.
func WriteCapabilityProfileR1(path string, p CapabilityProfileR1) (string, error) {
	p.ProfileHash = ""
	if err := ValidateCapabilityProfileR1(p); err != nil {
		return "", err
	}
	hash, err := ComputeProfileR1Hash(p)
	if err != nil {
		return "", err
	}
	p.ProfileHash = hash
	if err := ValidateCapabilityProfileR1(p); err != nil {
		return "", err
	}
	body, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return hash, nil
}

// RecoveryRule looks up one rule by id.
func (p CapabilityProfileR1) RecoveryRule(id string) (RecoveryRule, bool) {
	for _, r := range p.RecoveryRules {
		if r.ID == id {
			return r, true
		}
	}
	return RecoveryRule{}, false
}

// KnownOpcodes returns the opcode set the R1 profile has evidence for.
func (p CapabilityProfileR1) KnownOpcodes() []string {
	ops := []string{OpExtractNumber, p.ReadAssociated.Opcode}
	sort.Strings(ops)
	return ops
}
