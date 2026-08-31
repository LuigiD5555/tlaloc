package realcampaign

import "strings"

const ModelInteropSchema = "tlaloc.model-interop-profile.r0"

const (
	ModelFamilyLiquidAI = "LIQUIDAI"
	ModelFamilyDeepSeek = "DEEPSEEK"
	ModelFamilyQwen     = "QWEN"
	ModelFamilyUnknown  = "UNKNOWN"

	TransportDirectImageAPI = "DIRECT_IMAGE_API"
	TransportPlatformMediated = "PLATFORM_MEDIATED"
	TransportUnknown = "UNKNOWN"
)

// ModelInteropProfile keeps model identity separate from transport identity.
// Tlaloc may infer a broad family from the advertised model id, but behavioral
// traits remain observations produced by experiments rather than assumptions
// encoded here.
type ModelInteropProfile struct {
	Schema                string `json:"schema"`
	ModelID               string `json:"model_id"`
	Family                string `json:"family"`
	TransportCondition    string `json:"transport_condition"`
	CompatibilityStrategy string `json:"compatibility_strategy"`
	SpecimenKey           string `json:"specimen_key"`
	EvidenceRule          string `json:"evidence_rule"`
	ComparisonRule        string `json:"comparison_rule"`
}

func DetectModelFamily(modelID string) string {
	id := strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case strings.Contains(id, "deepseek"):
		return ModelFamilyDeepSeek
	case strings.Contains(id, "qwen"):
		return ModelFamilyQwen
	case strings.Contains(id, "liquid") || strings.Contains(id, "lfm"):
		return ModelFamilyLiquidAI
	default:
		return ModelFamilyUnknown
	}
}

func NormalizeTransportCondition(raw, compatibility string) string {
	v := strings.ToUpper(strings.TrimSpace(raw))
	switch v {
	case "", "AUTO":
		if strings.TrimSpace(compatibility) != "" {
			return TransportDirectImageAPI
		}
		return TransportUnknown
	case "DIRECT", "DIRECT_IMAGE", "DIRECT_IMAGE_API", "NATIVE_API":
		return TransportDirectImageAPI
	case "PLATFORM", "PLATFORM_MEDIATED", "WEB", "APP":
		return TransportPlatformMediated
	default:
		return v
	}
}

func BuildModelInteropProfile(modelID, compatibility, transport string) ModelInteropProfile {
	family := DetectModelFamily(modelID)
	condition := NormalizeTransportCondition(transport, compatibility)
	compat := strings.ToLower(strings.TrimSpace(compatibility))
	model := strings.TrimSpace(modelID)
	key := strings.Join([]string{family, model, condition, compat}, "::")
	return ModelInteropProfile{
		Schema: ModelInteropSchema,
		ModelID: model,
		Family: family,
		TransportCondition: condition,
		CompatibilityStrategy: compat,
		SpecimenKey: key,
		EvidenceRule: "OBSERVE_BEHAVIOR_DO_NOT_ASSUME_INTERNALS",
		ComparisonRule: "COMPARE_EXACT_MODEL_AND_TRANSPORT_FIRST",
	}
}
