package exocortex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// This file is the deterministic import adapter for the REAL frozen P2-A
// artifact (`experiments/parrot-microisa-r0.1/results/PARROT_MICRO_ISA_R0.json`).
// Its on-disk schema differs from the assumed `MicroISAArtifact` shape in
// capability_profile.go (that one was written before the real artifact was
// available). Per the T0 protocol section 27 ("if schemas differ, implement
// explicit deterministic adapters"), this file adapts the real shape into
// the same runtime `CapabilityProfile` the router/adapter already consume.
// It never edits, regenerates, rescales or "improves" the frozen artifact.

// RealMicroISAArtifact is the read-only DTO for the real P2-A file. Only the
// fields T0 needs are modelled; the rest of the artifact is ignored.
type RealMicroISAArtifact struct {
	ExperimentID string `json:"experiment_id"`
	Model        string `json:"model"`
	MaxSafeOps   int    `json:"max_safe_ops"`

	DeploymentRecommendation map[string]struct {
		Recommendation string `json:"recommendation"`
	} `json:"deployment_recommendation"`

	IntrinsicVerdict map[string]struct {
		Accuracy         float64 `json:"accuracy"`
		SemanticAccuracy float64 `json:"semantic_accuracy"`
		Class            string  `json:"class"`
		NeedsMoreN       bool    `json:"needs_more_n"`
	} `json:"intrinsic_verdict"`

	PDFTransferVerdict map[string]struct {
		Verdict  string `json:"verdict"`
		ByExtent map[string]struct {
			Accuracy         float64 `json:"accuracy"`
			SemanticAccuracy float64 `json:"semantic_accuracy"`
		} `json:"by_extent"`
	} `json:"pdf_transfer_verdict"`

	VisualField map[string]struct {
		MaxUsefulField string `json:"max_useful_field"`
		ByField        map[string]struct {
			Accuracy float64 `json:"accuracy"`
		} `json:"by_field"`
	} `json:"visual_field"`

	Limits struct {
		VisualTextChars *int `json:"visual_text_chars"`
		ChoiceWidth     *int `json:"choice_width"`
		RegionCount     *int `json:"region_count"`
	} `json:"limits"`

	Ladders map[string]struct {
		Dimension   string         `json:"dimension"`
		Capability  string         `json:"capability"`
		ByRung      map[string]any `json:"by_rung"`
		MaxSafeRung *int           `json:"max_safe_rung"`
	} `json:"ladders"`
}

// realOpcodeToR0 maps the real P2-A opcode vocabulary onto the R0 Micro-ISA
// vocabulary in opcode.go. It is a fixed, pre-registered table.
var realOpcodeToR0 = map[string]string{
	"EXTRACT_NUMBER":       OpExtractNumber,
	"EXTRACT_ENTITY":       OpExtractEntity,
	"SELECT_ONE_OF_N":      OpSelectOne,
	"COMPARE_TWO_VALUES":   OpCompareNumbers,
	"READ_SHORT_TEXT":      OpReadShortText,
	"READ_SHORT_LABEL":     OpReadShortLabel,
	"SAME_DIFFERENT":       OpSameDifferent,
	"FOLLOW_ONE_REFERENCE": OpFollowReference,
	"VISUAL_IDENTIFY":      OpVisualIdentify,
	"VISUAL_LOCATE":        OpVisualLocate,
}

// realRecommendationToDeployment maps the real P2-A deployment vocabulary
// onto the router's DeploymentRecommendation vocabulary. INSUFFICIENT_EVIDENCE
// maps conservatively to EXTERNALIZE: there is no measured basis to deploy.
var realRecommendationToDeployment = map[string]string{
	"KEEP":                  DeploymentDeploy,
	"FRAGILE":               DeploymentDeployConstrained,
	"EXTERNALIZE_CANDIDATE": DeploymentExternalize,
	"INSUFFICIENT_EVIDENCE": DeploymentExternalize,
	"DO_NOT_DEPLOY":         DeploymentDoNotDeploy,
}

// realClassToVerdict maps the real intrinsic class vocabulary onto the
// exocortex IntrinsicVerdict vocabulary. WEAK collapses onto FRAGILE.
var realClassToVerdict = map[string]string{
	"STRONG":   VerdictStrong,
	"USABLE":   VerdictUsable,
	"FRAGILE":  VerdictFragile,
	"WEAK":     VerdictFragile,
	"UNUSABLE": VerdictUnusable,
}

var realTransferToVerdict = map[string]string{
	"DOES_NOT_TRANSFER": TransferDoesNotTransfer,
	"PARTIAL_TRANSFER":  TransferPartial,
	"EXPLORATORY":       TransferNotEvaluated,
	"":                  "",
}

// LoadMicroISAArtifactReal reads and hash-verifies the real frozen P2-A
// artifact. It requires the co-located FREEZE.json (a sibling of results/)
// to confirm the experiment is frozen — R0 must not compile a runtime
// profile from a moving artifact.
func LoadMicroISAArtifactReal(path string) (RealMicroISAArtifact, string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return RealMicroISAArtifact{}, "", fmt.Errorf("read real micro-ISA artifact %s: %w", path, err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])

	var artifact RealMicroISAArtifact
	if err := json.Unmarshal(body, &artifact); err != nil {
		return RealMicroISAArtifact{}, "", fmt.Errorf("decode real micro-ISA artifact %s: %w", path, err)
	}
	if strings.TrimSpace(artifact.ExperimentID) == "" {
		return RealMicroISAArtifact{}, "", fmt.Errorf("real micro-ISA artifact %s: experiment_id is required", path)
	}
	if artifact.MaxSafeOps <= 0 {
		return RealMicroISAArtifact{}, "", fmt.Errorf("real micro-ISA artifact %s: max_safe_ops must be positive", path)
	}
	if len(artifact.DeploymentRecommendation) == 0 {
		return RealMicroISAArtifact{}, "", fmt.Errorf("real micro-ISA artifact %s: no deployment_recommendation findings", path)
	}
	if err := verifyRealArtifactFrozen(path); err != nil {
		return RealMicroISAArtifact{}, "", err
	}
	return artifact, hash, nil
}

// verifyRealArtifactFrozen confirms the FREEZE.json ledger sibling of the
// results/ directory marks the experiment frozen.
func verifyRealArtifactFrozen(artifactPath string) error {
	freezePath := filepath.Join(filepath.Dir(filepath.Dir(artifactPath)), "FREEZE.json")
	body, err := os.ReadFile(freezePath)
	if err != nil {
		return fmt.Errorf("real micro-ISA artifact: cannot read freeze ledger %s: %w", freezePath, err)
	}
	var ledger struct {
		Global struct {
			Frozen bool `json:"frozen"`
		} `json:"global"`
	}
	if err := json.Unmarshal(body, &ledger); err != nil {
		return fmt.Errorf("real micro-ISA artifact: decode freeze ledger %s: %w", freezePath, err)
	}
	if !ledger.Global.Frozen {
		return fmt.Errorf("real micro-ISA artifact: %s reports global.frozen=false; R0 cannot compile a profile from a moving artifact", freezePath)
	}
	return nil
}

// CompileParrotProfileReal compiles the real frozen P2-A artifact into a
// runtime CapabilityProfile. It is a pure, evidence-preserving transform: it
// invents no number that is not in the artifact and it preserves the
// formal/observed choice-width distinction and confirmed capability collapse.
func CompileParrotProfileReal(artifactPath, executorID, modelID, profileVersion string) (CapabilityProfile, error) {
	artifact, hash, err := LoadMicroISAArtifactReal(artifactPath)
	if err != nil {
		return CapabilityProfile{}, err
	}
	executorID = strings.TrimSpace(executorID)
	modelID = strings.TrimSpace(modelID)
	profileVersion = strings.TrimSpace(profileVersion)
	if executorID == "" || modelID == "" || profileVersion == "" {
		return CapabilityProfile{}, fmt.Errorf("executor_id, model_id and profile_version are required")
	}

	realOpcodes := make([]string, 0, len(artifact.DeploymentRecommendation))
	for op := range artifact.DeploymentRecommendation {
		realOpcodes = append(realOpcodes, op)
	}
	sort.Strings(realOpcodes)

	entries := make([]CapabilityEntry, 0, len(realOpcodes))
	for _, realOp := range realOpcodes {
		entry, err := compileRealEntry(artifact, realOp)
		if err != nil {
			return CapabilityProfile{}, fmt.Errorf("opcode %s: %w", realOp, err)
		}
		entries = append(entries, entry)
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
		MaxSafeOps:         artifact.MaxSafeOps,
		Capabilities:       entries,
	}
	if err := ValidateProfile(profile); err != nil {
		return CapabilityProfile{}, err
	}
	return profile, nil
}

func compileRealEntry(artifact RealMicroISAArtifact, realOp string) (CapabilityEntry, error) {
	r0op, ok := realOpcodeToR0[realOp]
	if !ok {
		return CapabilityEntry{}, fmt.Errorf("real opcode %q has no R0 vocabulary mapping", realOp)
	}

	rec := strings.ToUpper(strings.TrimSpace(artifact.DeploymentRecommendation[realOp].Recommendation))
	deployment, ok := realRecommendationToDeployment[rec]
	if !ok {
		return CapabilityEntry{}, fmt.Errorf("unknown deployment recommendation %q", rec)
	}

	verdict := VerdictFragile
	var syntheticAccuracy float64
	if iv, ok := artifact.IntrinsicVerdict[realOp]; ok {
		syntheticAccuracy = iv.Accuracy
		if mapped, ok := realClassToVerdict[strings.ToUpper(strings.TrimSpace(iv.Class))]; ok {
			verdict = mapped
		}
		if verdict == VerdictUnusable {
			deployment = DeploymentDoNotDeploy
		}
	}

	transfer := ""
	if pt, ok := artifact.PDFTransferVerdict[realOp]; ok {
		transfer = realTransferToVerdict[strings.ToUpper(strings.TrimSpace(pt.Verdict))]
		if transfer == TransferDoesNotTransfer && deployment == DeploymentDeployConstrained {
			deployment = DeploymentExternalize
		}
	}

	constraints := Constraints{
		AllowedVisualField:   []string{VisualFieldTightCrop, VisualFieldFullPage},
		PreferredVisualField: VisualFieldTightCrop,
	}
	// Choice-width: the formal preregistered rung is limits.choice_width;
	// the observed tested envelope is the widest rung the ladder walked.
	if r0op == OpSelectOne {
		if artifact.Limits.ChoiceWidth != nil {
			constraints.FormalMaxChoiceWidth = *artifact.Limits.ChoiceWidth
		}
		if ladder, ok := artifact.Ladders["choice_width"]; ok {
			widest := 0
			for rung := range ladder.ByRung {
				if v := atoiSafe(rung); v > widest {
					widest = v
				}
			}
			if widest > constraints.FormalMaxChoiceWidth {
				constraints.ObservedChoiceWidthEnvelope = widest
			}
		}
	}

	confidence := ConfidenceMetadata{SyntheticAccuracy: syntheticAccuracy}
	if vf, ok := artifact.VisualField[realOp]; ok {
		if tight, ok := vf.ByField["tight"]; ok {
			confidence.TightCropAccuracy = tight.Accuracy
		}
		if page, ok := vf.ByField["page"]; ok {
			confidence.FullPageAccuracy = page.Accuracy
		}
	}
	if pt, ok := artifact.PDFTransferVerdict[realOp]; ok {
		if rt, ok := pt.ByExtent["real_tight"]; ok && confidence.TightCropAccuracy == 0 {
			confidence.TightCropAccuracy = rt.Accuracy
		}
		if fp, ok := pt.ByExtent["full_page"]; ok && confidence.FullPageAccuracy == 0 {
			confidence.FullPageAccuracy = fp.Accuracy
		}
	}

	var fallbacks []string
	switch deployment {
	case DeploymentExternalize:
		fallbacks = append(fallbacks, "P2-A "+rec+": route to a deterministic or externalized alternative, not Parrot")
	case DeploymentDoNotDeploy:
		fallbacks = append(fallbacks, "P2-A intrinsic class UNUSABLE: do not deploy on Parrot")
	}

	outputType := "plain_text"
	if constraints.FormalMaxChoiceWidth > 0 {
		outputType = "single_token_choice"
	}

	return CapabilityEntry{
		Opcode:                   r0op,
		IntrinsicVerdict:         verdict,
		PDFTransferVerdict:       transfer,
		DeploymentRecommendation: deployment,
		Constraints:              constraints,
		PreferredInput:           PreferredInput{Type: "image_crop", FieldSize: constraints.PreferredVisualField},
		OutputContract:           OutputContract{Type: outputType},
		ResponseCollapse:         verdict == VerdictUnusable,
		Confidence:               confidence,
		FallbackSuggestions:      fallbacks,
	}, nil
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
