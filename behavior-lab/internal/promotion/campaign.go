package promotion

import (
	"fmt"
	"sort"
)

const SchemaR1 = "tlaloc.perception-campaign.r1"

type EvaluatorReport struct {
	Schema                   string `json:"schema"`
	Model                    string `json:"model"`
	Trial                    int    `json:"trial"`
	Transport                string `json:"transport"`
	EvidenceKind             string `json:"evidence_kind"`
	HybridMechanicalPass     bool   `json:"hybrid_mechanical_pass"`
	NativeT3MechanicalPass   bool   `json:"native_t3_mechanical_pass"`
	HybridTrialPromotionOK   bool   `json:"hybrid_trial_promotion_eligible"`
	NativeT3TrialPromotionOK bool   `json:"native_t3_trial_promotion_eligible"`
}

type TrialRecord struct {
	Model          string          `json:"model"`
	Trial          int             `json:"trial"`
	Transport      string          `json:"transport"`
	EvidenceKind   string          `json:"evidence_kind"`
	Evaluation     EvaluatorReport `json:"evaluation"`
	ToolLoopPassed bool            `json:"tool_loop_passed"`
	ToolCalls      int             `json:"tool_calls,omitempty"`
	AnswerPresent  bool            `json:"answer_present,omitempty"`
}

type RoutingEvidence struct {
	Documents            int     `json:"documents"`
	Queries              int     `json:"queries"`
	PrimaryDocHitRate    float64 `json:"primary_doc_hit_rate"`
	VerifiedEvidenceRate float64 `json:"verified_evidence_rate"`
	BudgetViolations     int     `json:"budget_violations"`
	FalseExact           int     `json:"false_exact"`
}

type Policy struct {
	MinModels               int      `json:"min_models"`
	TrialsPerModel          int      `json:"trials_per_model"`
	RequiredTransports      []string `json:"required_transports"`
	MinRoutingDocuments     int      `json:"min_routing_documents"`
	MinRoutingQueries       int      `json:"min_routing_queries"`
	MinPrimaryDocHitRate    float64  `json:"min_primary_doc_hit_rate"`
	MinVerifiedEvidenceRate float64  `json:"min_verified_evidence_rate"`
}

func DefaultPolicy() Policy {
	return Policy{
		MinModels: 3, TrialsPerModel: 3,
		RequiredTransports:  []string{"original", "resize-75", "resize-50", "jpeg-preview"},
		MinRoutingDocuments: 5, MinRoutingQueries: 6,
		MinPrimaryDocHitRate: .95, MinVerifiedEvidenceRate: .95,
	}
}

type Gate struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"`
}

type ModelSummary struct {
	Model                      string `json:"model"`
	RealOriginalHybridTrials   int    `json:"real_original_hybrid_trials"`
	RealOriginalNativeT3Trials int    `json:"real_original_native_t3_trials"`
	RealToolLoopPasses         int    `json:"real_tool_loop_passes"`
}

type CampaignReport struct {
	Schema                         string          `json:"schema"`
	Policy                         Policy          `json:"policy"`
	Models                         []ModelSummary  `json:"models"`
	Transports                     map[string]int  `json:"eligible_transport_trials"`
	Routing                        RoutingEvidence `json:"routing"`
	Gates                          []Gate          `json:"gates"`
	HybridSupportedCandidate       bool            `json:"hybrid_supported_candidate"`
	NativeVisualSupportedCandidate bool            `json:"native_visual_supported_candidate"`
	PromotionAuthority             string          `json:"promotion_authority"`
}

func Evaluate(records []TrialRecord, routing RoutingEvidence, policy Policy) (CampaignReport, error) {
	policy = normalize(policy)
	byModel := map[string]*ModelSummary{}
	transports := map[string]int{}
	for _, record := range records {
		if record.Model == "" || record.Trial <= 0 {
			return CampaignReport{}, fmt.Errorf("invalid trial identity")
		}
		if record.Evaluation.Model != "" && record.Evaluation.Model != record.Model {
			return CampaignReport{}, fmt.Errorf("evaluator/model mismatch for %s trial %d", record.Model, record.Trial)
		}
		s := byModel[record.Model]
		if s == nil {
			s = &ModelSummary{Model: record.Model}
			byModel[record.Model] = s
		}
		real := record.EvidenceKind == "REAL_MODEL" && record.Evaluation.EvidenceKind == "REAL_MODEL"
		if real && record.Evaluation.HybridTrialPromotionOK {
			transports[record.Transport]++
			if record.Transport == "original" {
				s.RealOriginalHybridTrials++
			}
		}
		if real && record.Transport == "original" && record.Evaluation.NativeT3TrialPromotionOK {
			s.RealOriginalNativeT3Trials++
		}
		if real && record.ToolLoopPassed && record.ToolCalls > 0 && record.AnswerPresent {
			s.RealToolLoopPasses++
		}
	}
	models := make([]ModelSummary, 0, len(byModel))
	for _, s := range byModel {
		models = append(models, *s)
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Model < models[j].Model })
	qualifiedHybrid := 0
	qualifiedNative := 0
	toolModels := 0
	for _, s := range models {
		if s.RealOriginalHybridTrials >= policy.TrialsPerModel {
			qualifiedHybrid++
		}
		if s.RealOriginalNativeT3Trials >= policy.TrialsPerModel {
			qualifiedNative++
		}
		if s.RealToolLoopPasses > 0 {
			toolModels++
		}
	}
	crossHybrid := qualifiedHybrid >= policy.MinModels
	transportPass := true
	for _, name := range policy.RequiredTransports {
		if transports[name] < 1 {
			transportPass = false
		}
	}
	toolPass := toolModels >= policy.MinModels
	routingPass := routing.Documents >= policy.MinRoutingDocuments && routing.Queries >= policy.MinRoutingQueries && routing.PrimaryDocHitRate >= policy.MinPrimaryDocHitRate && routing.VerifiedEvidenceRate >= policy.MinVerifiedEvidenceRate && routing.BudgetViolations == 0 && routing.FalseExact == 0
	nativeCross := qualifiedNative >= policy.MinModels
	report := CampaignReport{Schema: SchemaR1 + ".report", Policy: policy, Models: models, Transports: transports, Routing: routing, PromotionAuthority: "TONAL_FINAL_STACK_PROMOTION"}
	report.Gates = []Gate{
		{Name: "CROSS_MODEL_HYBRID_3x3", Pass: crossHybrid, Reason: gateReason(crossHybrid, "need at least 3 real models with 3 original Hybrid-eligible trials each")},
		{Name: "TRANSPORT_ROBUSTNESS", Pass: transportPass, Reason: gateReason(transportPass, "each required transport needs at least one real Hybrid-eligible trial")},
		{Name: "REAL_TOOL_LOOP", Pass: toolPass, Reason: gateReason(toolPass, "need a real successful tool loop for each of at least 3 models")},
		{Name: "HELD_OUT_ROUTING", Pass: routingPass, Reason: gateReason(routingPass, "held-out multi-document routing/evidence/budget/FALSE_EXACT thresholds not met")},
		{Name: "CROSS_MODEL_NATIVE_T3_3x3", Pass: nativeCross, Reason: gateReason(nativeCross, "need at least 3 real models with 3 original Native-T3-eligible trials each")},
	}
	report.HybridSupportedCandidate = crossHybrid && transportPass && toolPass && routingPass
	report.NativeVisualSupportedCandidate = report.HybridSupportedCandidate && nativeCross
	return report, nil
}

func normalize(p Policy) Policy {
	d := DefaultPolicy()
	if p.MinModels <= 0 {
		p.MinModels = d.MinModels
	}
	if p.TrialsPerModel <= 0 {
		p.TrialsPerModel = d.TrialsPerModel
	}
	if len(p.RequiredTransports) == 0 {
		p.RequiredTransports = d.RequiredTransports
	}
	if p.MinRoutingDocuments <= 0 {
		p.MinRoutingDocuments = d.MinRoutingDocuments
	}
	if p.MinRoutingQueries <= 0 {
		p.MinRoutingQueries = d.MinRoutingQueries
	}
	if p.MinPrimaryDocHitRate <= 0 {
		p.MinPrimaryDocHitRate = d.MinPrimaryDocHitRate
	}
	if p.MinVerifiedEvidenceRate <= 0 {
		p.MinVerifiedEvidenceRate = d.MinVerifiedEvidenceRate
	}
	return p
}
func gateReason(pass bool, reason string) string {
	if pass {
		return ""
	}
	return reason
}
