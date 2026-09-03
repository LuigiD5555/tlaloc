package parrotlab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

// microisagg turns microisa_visual run records into the Parrot Micro-ISA.
// Three verdicts are kept strictly separate and their accuracies are never
// pooled:
//
//   INTRINSIC_VERDICT       — from the A1 canonical synthetic baseline only.
//   PDF_TRANSFER_VERDICT    — from the A4 real-PDF transfer only.
//   DEPLOYMENT_RECOMMENDATION — derived from A1 class + A2 operating limits +
//                              A3 field behaviour + A4 transfer, as a
//                              decision over those axes, not an average.

// MicroISAResult is StageResult.MicroISA for the microisa_visual stage.
type MicroISAResult struct {
	IntrinsicVerdict map[string]CapIntrinsic `json:"intrinsic_verdict"`
	TransferVerdict  map[string]CapTransfer  `json:"pdf_transfer_verdict"`
	Deployment       map[string]CapDeploy    `json:"deployment_recommendation"`
	Ladders          map[string]LadderResult `json:"ladders"`
	ReferenceTypes   *ReferenceResult        `json:"reference_types,omitempty"`
	VisualField      map[string]FieldResult  `json:"visual_field"`
	Limits           MicroISALimits          `json:"limits"`
	Counts           MicroISACounts          `json:"counts"`
}

type MicroISACounts struct {
	BySubExperiment map[string]SubCount `json:"by_sub_experiment"`
}

type SubCount struct {
	IndependentBaseStimuli int `json:"independent_base_stimuli"`
	TotalObservations      int `json:"total_observations"`
}

type CapIntrinsic struct {
	BaseStimuli              int        `json:"independent_base_stimuli"`
	Observations             int        `json:"total_observations"`
	Accuracy                 float64    `json:"accuracy"`
	SemanticAccuracy         float64    `json:"semantic_accuracy"`
	WilsonCI                 [2]float64 `json:"wilson_ci95"`
	FormatFailureRate        float64    `json:"format_failure_rate"`
	UnsupportedAssertionRate float64    `json:"unsupported_assertion_rate"`
	MeanLatencyMS            int64      `json:"mean_latency_ms"`
	Class                    string     `json:"class"`
	NeedsMoreN               bool       `json:"needs_more_n"`
}

type CapTransfer struct {
	BaseStimuli    int                  `json:"independent_base_stimuli"`
	Exploratory    bool                 `json:"exploratory"`
	SyntheticA1    float64              `json:"synthetic_a1_accuracy"`
	ByExtent       map[string]GroupStat `json:"by_extent"`
	Transitions    []PairedTransition   `json:"paired_transitions_from_real_tight"`
	FullPageWilson [2]float64           `json:"full_page_wilson_ci95"`
	Verdict        string               `json:"verdict"`
}

type CapDeploy struct {
	Recommendation  string            `json:"recommendation"`
	Rationale       []string          `json:"rationale"`
	OperatingLimits map[string]string `json:"operating_limits,omitempty"`
}

type LadderResult struct {
	Dimension   string              `json:"dimension"`
	Capability  string              `json:"capability"`
	ByRung      map[string]RungStat `json:"by_rung"`
	Transitions []PairedTransition  `json:"paired_transitions"`
	MaxSafeRung *int                `json:"max_safe_rung"`
}

type RungStat struct {
	Rung         int        `json:"rung"`
	BaseStimuli  int        `json:"independent_base_stimuli"`
	Observations int        `json:"total_observations"`
	Accuracy     float64    `json:"accuracy"`
	WilsonCI     [2]float64 `json:"wilson_ci95"`
}

type ReferenceResult struct {
	BaseStimuli int                  `json:"independent_base_stimuli"`
	ByType      map[string]GroupStat `json:"by_type"`
	VsArrow     map[string]float64   `json:"delta_vs_arrow"`
}

type FieldResult struct {
	Capability     string               `json:"capability"`
	ByField        map[string]GroupStat `json:"by_field"`
	Transitions    []PairedTransition   `json:"paired_transitions_from_tight"`
	MaxUsefulField string               `json:"max_useful_field"`
}

type MicroISALimits struct {
	VisualTextChars *int `json:"visual_text_chars"`
	ChoiceWidth     *int `json:"choice_width"`
	RegionCount     *int `json:"region_count"`
}

var fieldOrder = map[string]int{"tight": 0, "medium": 1, "block": 2, "page": 3}

func buildMicroISA(records []RunRecord, thresholds Thresholds) *MicroISAResult {
	bySub := map[string][]RunRecord{}
	for _, record := range records {
		bySub[record.SubExperiment] = append(bySub[record.SubExperiment], record)
	}

	result := &MicroISAResult{
		IntrinsicVerdict: map[string]CapIntrinsic{},
		TransferVerdict:  map[string]CapTransfer{},
		Deployment:       map[string]CapDeploy{},
		Ladders:          map[string]LadderResult{},
		VisualField:      map[string]FieldResult{},
		Counts:           MicroISACounts{BySubExperiment: map[string]SubCount{}},
	}
	for sub, bucket := range bySub {
		result.Counts.BySubExperiment[sub] = SubCount{
			IndependentBaseStimuli: countBaseStimuli(bucket),
			TotalObservations:      len(bucket),
		}
	}

	cuts := thresholds.CapabilityClass
	result.buildIntrinsic(bySub["A1"], cuts)
	result.buildLadders(bySub["A2"])
	result.buildReferenceTypes(bySub["A2"])
	result.buildVisualField(bySub["A3"])
	result.buildTransfer(bySub["A4"], bySub["A1"], cuts)
	result.buildDeployment(cuts)
	return result
}

func countBaseStimuli(records []RunRecord) int {
	seen := map[string]bool{}
	for _, record := range records {
		key := record.BaseID
		if key == "" {
			key = record.CaseID
		}
		seen[key] = true
	}
	return len(seen)
}

func (result *MicroISAResult) buildIntrinsic(records []RunRecord, cuts CapabilityClassCuts) {
	byCap := map[string][]RunRecord{}
	for _, record := range records {
		if len(record.Capabilities) == 1 {
			byCap[record.Capabilities[0]] = append(byCap[record.Capabilities[0]], record)
		}
	}
	for capability, bucket := range byCap {
		stat := computeGroup(bucket)
		interval := calibration.WilsonInterval(stat.Correct, stat.Attempted)
		class, needsMore := classifyInterval(interval, cuts)
		result.IntrinsicVerdict[capability] = CapIntrinsic{
			BaseStimuli:              countBaseStimuli(bucket),
			Observations:             len(bucket),
			Accuracy:                 stat.Accuracy,
			SemanticAccuracy:         stat.SemanticAccuracy,
			WilsonCI:                 [2]float64{interval.Low, interval.High},
			FormatFailureRate:        stat.FormatFailureRate,
			UnsupportedAssertionRate: stat.UnsupportedAssertionRate,
			MeanLatencyMS:            stat.MeanLatencyMS,
			Class:                    class,
			NeedsMoreN:               needsMore,
		}
	}
}

// classifyInterval mirrors classify() but also flags the "CI straddles the
// usable/weak boundary" case as NEEDS_MORE_N (spec item 4).
func classifyInterval(interval calibration.ProportionInterval, cuts CapabilityClassCuts) (string, bool) {
	switch {
	case interval.Low >= cuts.StrongLowerCI:
		return "STRONG", false
	case interval.Low >= cuts.UsableLowerCI:
		return "USABLE", false
	case interval.High < cuts.UnusableUpperCI:
		return "UNUSABLE", false
	case interval.High < cuts.WeakUpperCI:
		return "WEAK", false
	default:
		// FRAGILE: straddles a decision boundary → more data would resolve it.
		return "FRAGILE", true
	}
}

func (result *MicroISAResult) buildLadders(records []RunRecord) {
	byDim := map[string][]RunRecord{}
	for _, record := range records {
		if record.VariedDim == "" || record.VariedDim == "reference_type" {
			continue
		}
		byDim[record.VariedDim] = append(byDim[record.VariedDim], record)
	}
	for dim, bucket := range byDim {
		ladder := LadderResult{Dimension: dim, ByRung: map[string]RungStat{}}
		if len(bucket) > 0 {
			ladder.Capability = bucket[0].Capabilities[0]
		}
		byRung := map[int][]RunRecord{}
		correctByBase := map[string]map[int]bool{}
		for _, record := range bucket {
			rung := conditionInt(record.Condition)
			byRung[rung] = append(byRung[rung], record)
			if correctByBase[record.BaseID] == nil {
				correctByBase[record.BaseID] = map[int]bool{}
			}
			prior, seen := correctByBase[record.BaseID][rung]
			value := record.Score.ContractSuccess
			if seen {
				value = prior && value
			}
			correctByBase[record.BaseID][rung] = value
		}
		rungs := sortedKeys(byRung)
		for _, rung := range rungs {
			stat := computeGroup(byRung[rung])
			interval := calibration.WilsonInterval(stat.Correct, stat.Attempted)
			ladder.ByRung[strconv.Itoa(rung)] = RungStat{
				Rung: rung, BaseStimuli: countBaseStimuli(byRung[rung]), Observations: len(byRung[rung]),
				Accuracy: stat.Accuracy, WilsonCI: [2]float64{interval.Low, interval.High},
			}
		}
		var maxSafe *int
		for index := 1; index < len(rungs); index++ {
			transition := pairedTransition(rungs[index-1], rungs[index], correctByBase)
			ladder.Transitions = append(ladder.Transitions, transition)
		}
		// max_safe_rung: highest rung whose Wilson-low >= usable cut and with
		// no significant paired drop from the rung below.
		for index, rung := range rungs {
			stat := ladder.ByRung[strconv.Itoa(rung)]
			if stat.WilsonCI[0] < 0.70 {
				break
			}
			if index > 0 && ladder.Transitions[index-1].Significant {
				break
			}
			value := rung
			maxSafe = &value
		}
		ladder.MaxSafeRung = maxSafe
		result.Ladders[dim] = ladder
	}
	result.Limits = MicroISALimits{
		VisualTextChars: ladderLimit(result.Ladders["visual_text_chars"]),
		ChoiceWidth:     ladderLimit(result.Ladders["choice_width"]),
		RegionCount:     ladderLimit(result.Ladders["region_count"]),
	}
}

func ladderLimit(ladder LadderResult) *int { return ladder.MaxSafeRung }

func (result *MicroISAResult) buildReferenceTypes(records []RunRecord) {
	var bucket []RunRecord
	for _, record := range records {
		if record.VariedDim == "reference_type" {
			bucket = append(bucket, record)
		}
	}
	if len(bucket) == 0 {
		return
	}
	byType := map[string][]RunRecord{}
	for _, record := range bucket {
		byType[strings.TrimPrefix(record.Condition, "reftype=")] = append(byType[strings.TrimPrefix(record.Condition, "reftype=")], record)
	}
	reference := &ReferenceResult{
		BaseStimuli: countBaseStimuli(bucket),
		ByType:      map[string]GroupStat{},
		VsArrow:     map[string]float64{},
	}
	for name, group := range byType {
		reference.ByType[name] = computeGroup(group)
	}
	arrow := reference.ByType["arrow"].Accuracy
	for name, stat := range reference.ByType {
		reference.VsArrow[name] = stat.Accuracy - arrow
	}
	result.ReferenceTypes = reference
}

func (result *MicroISAResult) buildVisualField(records []RunRecord) {
	byCap := map[string][]RunRecord{}
	for _, record := range records {
		if len(record.Capabilities) == 1 {
			byCap[record.Capabilities[0]] = append(byCap[record.Capabilities[0]], record)
		}
	}
	for capability, bucket := range byCap {
		field := FieldResult{Capability: capability, ByField: map[string]GroupStat{}}
		byField := map[string][]RunRecord{}
		correctByBase := map[string]map[int]bool{}
		for _, record := range bucket {
			name := strings.TrimPrefix(record.Condition, "field=")
			byField[name] = append(byField[name], record)
			index := fieldOrder[name]
			if correctByBase[record.BaseID] == nil {
				correctByBase[record.BaseID] = map[int]bool{}
			}
			correctByBase[record.BaseID][index] = record.Score.ContractSuccess
		}
		for name, group := range byField {
			field.ByField[name] = computeGroup(group)
		}
		field.MaxUsefulField = "tight"
		for _, name := range []string{"medium", "block", "page"} {
			transition := pairedTransition(fieldOrder["tight"], fieldOrder[name], correctByBase)
			field.Transitions = append(field.Transitions, transition)
			if !transition.Significant {
				field.MaxUsefulField = name
			}
		}
		result.VisualField[capability] = field
	}
}

func (result *MicroISAResult) buildTransfer(a4, a1 []RunRecord, cuts CapabilityClassCuts) {
	a1Acc := map[string]float64{}
	a1ByCap := map[string][]RunRecord{}
	for _, record := range a1 {
		if len(record.Capabilities) == 1 {
			a1ByCap[record.Capabilities[0]] = append(a1ByCap[record.Capabilities[0]], record)
		}
	}
	for capability, bucket := range a1ByCap {
		a1Acc[capability] = computeGroup(bucket).Accuracy
	}

	byCap := map[string][]RunRecord{}
	for _, record := range a4 {
		if len(record.Capabilities) == 1 {
			byCap[record.Capabilities[0]] = append(byCap[record.Capabilities[0]], record)
		}
	}
	for capability, bucket := range byCap {
		transfer := CapTransfer{
			BaseStimuli: countBaseStimuli(bucket),
			SyntheticA1: a1Acc[capability],
			ByExtent:    map[string]GroupStat{},
		}
		transfer.Exploratory = transfer.BaseStimuli < 8
		byExtent := map[string][]RunRecord{}
		correctByBase := map[string]map[int]bool{}
		extentIndex := map[string]int{"real_tight": 0, "real_block": 1, "full_page": 2}
		for _, record := range bucket {
			name := strings.TrimPrefix(record.Condition, "crop=")
			byExtent[name] = append(byExtent[name], record)
			if correctByBase[record.BaseID] == nil {
				correctByBase[record.BaseID] = map[int]bool{}
			}
			correctByBase[record.BaseID][extentIndex[name]] = record.Score.ContractSuccess
		}
		for name, group := range byExtent {
			transfer.ByExtent[name] = computeGroup(group)
		}
		for _, name := range []string{"real_block", "full_page"} {
			transfer.Transitions = append(transfer.Transitions,
				pairedTransition(extentIndex["real_tight"], extentIndex[name], correctByBase))
		}
		fullPage := computeGroup(byExtent["full_page"])
		interval := calibration.WilsonInterval(fullPage.Correct, fullPage.Attempted)
		transfer.FullPageWilson = [2]float64{interval.Low, interval.High}
		switch {
		case transfer.Exploratory:
			transfer.Verdict = "EXPLORATORY"
		case interval.Low >= cuts.UsableLowerCI:
			transfer.Verdict = "TRANSFERS"
		case interval.High < cuts.WeakUpperCI:
			transfer.Verdict = "DOES_NOT_TRANSFER"
		default:
			transfer.Verdict = "PARTIAL_TRANSFER"
		}
		result.TransferVerdict[capability] = transfer
	}
}

func (result *MicroISAResult) buildDeployment(cuts CapabilityClassCuts) {
	for _, capability := range MicroISACapabilities {
		intrinsic, hasIntrinsic := result.IntrinsicVerdict[capability]
		deploy := CapDeploy{OperatingLimits: map[string]string{}}
		if !hasIntrinsic {
			deploy.Recommendation = "INSUFFICIENT_EVIDENCE"
			deploy.Rationale = append(deploy.Rationale, "no A1 canonical baseline for this capability")
			result.Deployment[capability] = deploy
			continue
		}

		for dim, ladder := range result.Ladders {
			if ladder.Capability != capability {
				continue
			}
			if ladder.MaxSafeRung != nil {
				deploy.OperatingLimits[dim] = fmt.Sprintf("<= %d", *ladder.MaxSafeRung)
			} else {
				deploy.OperatingLimits[dim] = "no safe rung"
			}
		}
		if field, ok := result.VisualField[capability]; ok {
			deploy.OperatingLimits["max_useful_field"] = field.MaxUsefulField
		}
		transfer, hasTransfer := result.TransferVerdict[capability]

		switch intrinsic.Class {
		case "WEAK", "UNUSABLE":
			deploy.Recommendation = "EXTERNALIZE_CANDIDATE"
			deploy.Rationale = append(deploy.Rationale, "A1 intrinsic class "+intrinsic.Class)
		case "FRAGILE":
			deploy.Recommendation = "FRAGILE"
			deploy.Rationale = append(deploy.Rationale, "A1 intrinsic CI straddles the usable/weak boundary (NEEDS_MORE_N)")
		default: // USABLE / STRONG
			corroborated := false
			if hasTransfer {
				switch transfer.Verdict {
				case "TRANSFERS":
					deploy.Recommendation = "KEEP_IN_PARROT"
					deploy.Rationale = append(deploy.Rationale, "A1 "+intrinsic.Class+" and A4 full-page transfer holds (Wilson-low >= usable cut)")
					corroborated = true
				case "PARTIAL_TRANSFER":
					deploy.Recommendation = "FRAGILE"
					deploy.Rationale = append(deploy.Rationale, "A1 "+intrinsic.Class+" but A4 transfer only partial on real pages")
					corroborated = true
				case "DOES_NOT_TRANSFER":
					deploy.Recommendation = "EXTERNALIZE_CANDIDATE"
					deploy.Rationale = append(deploy.Rationale, "A1 "+intrinsic.Class+" but fails on real PDF pages (A4)")
					corroborated = true
				case "EXPLORATORY":
					deploy.Recommendation = "FRAGILE"
					deploy.Rationale = append(deploy.Rationale, "A1 "+intrinsic.Class+" but A4 transfer evidence is exploratory (<8 real base stimuli)")
					corroborated = true
				}
			}
			if !corroborated {
				// No transfer axis. Fall back to ladder/field corroboration.
				hasOperatingEvidence := false
				for _, value := range deploy.OperatingLimits {
					if value != "" {
						hasOperatingEvidence = true
					}
				}
				if hasOperatingEvidence {
					deploy.Recommendation = "KEEP_IN_PARROT_WITHIN_LIMITS"
					deploy.Rationale = append(deploy.Rationale, "A1 "+intrinsic.Class+" with A2/A3 operating limits characterised; PDF transfer untested for this synthetic-only capability")
				} else {
					deploy.Recommendation = "INSUFFICIENT_EVIDENCE"
					deploy.Rationale = append(deploy.Rationale, "A1 "+intrinsic.Class+" only; no A2/A3/A4 corroboration — not KEEP on A1 alone")
				}
			}
		}
		if intrinsic.FormatFailureRate >= 0.15 {
			deploy.Rationale = append(deploy.Rationale, fmt.Sprintf("A1 format-failure rate %.2f", intrinsic.FormatFailureRate))
		}
		result.Deployment[capability] = deploy
	}
}

// WriteMicroISAArtifacts writes results/PARROT_MICRO_ISA_R0.json and
// results/MICROISA_TABLE.md from an aggregated microisa_visual StageResult.
func WriteMicroISAArtifacts(exp *Experiment, result StageResult) ([]string, error) {
	if result.MicroISA == nil {
		return nil, fmt.Errorf("stage result has no microisa block")
	}
	dir := filepath.Join(exp.Root, "results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	micro := result.MicroISA

	supported := map[string]any{}
	fragile := map[string]any{}
	unsupported := map[string]any{}
	needsMoreN := []string{}
	for _, capability := range MicroISACapabilities {
		verdict, ok := micro.IntrinsicVerdict[capability]
		if !ok {
			continue
		}
		entry := map[string]any{
			"accuracy": verdict.Accuracy, "semantic_accuracy": verdict.SemanticAccuracy,
			"wilson_ci95": verdict.WilsonCI, "independent_base_stimuli": verdict.BaseStimuli,
		}
		switch verdict.Class {
		case "STRONG", "USABLE":
			supported[capability] = entry
		case "WEAK", "UNUSABLE":
			unsupported[capability] = entry
		default:
			fragile[capability] = entry
		}
		if verdict.NeedsMoreN {
			needsMoreN = append(needsMoreN, capability)
		}
	}

	document := map[string]any{
		"model":                     exp.Model.ID,
		"experiment_id":             exp.Manifest.ExperimentID,
		"max_safe_ops":              1,
		"classification_basis":      "INTRINSIC_VERDICT from A1 canonical synthetic baseline only; DEPLOYMENT_RECOMMENDATION never KEEP on A1 alone",
		"intrinsic_verdict":         micro.IntrinsicVerdict,
		"pdf_transfer_verdict":      micro.TransferVerdict,
		"deployment_recommendation": micro.Deployment,
		"supported":                 supported,
		"fragile":                   fragile,
		"unsupported":               unsupported,
		"needs_more_n":              needsMoreN,
		"limits":                    micro.Limits,
		"ladders":                   micro.Ladders,
		"reference_types":           micro.ReferenceTypes,
		"visual_field":              micro.VisualField,
		"counts":                    micro.Counts,
		"image_instruction":         map[string]any{"supported": nil},
	}
	jsonPath := filepath.Join(dir, "PARROT_MICRO_ISA_R0.json")
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(jsonPath, append(raw, '\n'), 0o644); err != nil {
		return nil, err
	}

	tablePath := filepath.Join(dir, "MICROISA_TABLE.md")
	if err := os.WriteFile(tablePath, []byte(renderMicroISATable(exp, micro)), 0o644); err != nil {
		return nil, err
	}
	return []string{jsonPath, tablePath}, nil
}

func renderMicroISATable(exp *Experiment, micro *MicroISAResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Parrot Micro-ISA R0 — %s\n\n", exp.Manifest.ExperimentID)
	fmt.Fprintf(&builder, "Model: `%s`  ·  max_safe_ops = 1\n\n", exp.Model.ID)
	builder.WriteString("Three verdicts are separate: INTRINSIC (A1 only) · PDF_TRANSFER (A4 only) · ")
	builder.WriteString("DEPLOYMENT (decision over A1 class + A2 limits + A3 field + A4 transfer, not a pooled accuracy).\n\n")

	builder.WriteString("| capability | base stimuli (A1) | obs (A1) | A1 acc | A1 semantic | A1 Wilson CI95 | A4 real_tight | A4 full_page | latency ms | operating limits | intrinsic | deployment |\n")
	builder.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, capability := range MicroISACapabilities {
		intrinsic := micro.IntrinsicVerdict[capability]
		realTight, fullPage := "—", "—"
		if transfer, ok := micro.TransferVerdict[capability]; ok {
			if stat, has := transfer.ByExtent["real_tight"]; has {
				realTight = fmt.Sprintf("%.2f", stat.Accuracy)
			}
			if stat, has := transfer.ByExtent["full_page"]; has {
				fullPage = fmt.Sprintf("%.2f", stat.Accuracy)
			}
			if transfer.Exploratory {
				fullPage += " (exploratory)"
			}
		}
		limits := []string{}
		if deploy, ok := micro.Deployment[capability]; ok {
			names := make([]string, 0, len(deploy.OperatingLimits))
			for name := range deploy.OperatingLimits {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				limits = append(limits, name+" "+deploy.OperatingLimits[name])
			}
		}
		deployRec := "—"
		if deploy, ok := micro.Deployment[capability]; ok {
			deployRec = deploy.Recommendation
		}
		fmt.Fprintf(&builder, "| %s | %d | %d | %.2f | %.2f | [%.2f, %.2f] | %s | %s | %d | %s | %s | %s |\n",
			capability, intrinsic.BaseStimuli, intrinsic.Observations, intrinsic.Accuracy, intrinsic.SemanticAccuracy,
			intrinsic.WilsonCI[0], intrinsic.WilsonCI[1], realTight, fullPage, intrinsic.MeanLatencyMS,
			strings.Join(limits, "; "), intrinsic.Class, deployRec)
	}

	builder.WriteString("\n## Operating limits (A2 ladders)\n\n")
	fmt.Fprintf(&builder, "- visual_text_chars: %s\n- choice_width: %s\n- region_count: %s\n\n",
		intPtrString(micro.Limits.VisualTextChars), intPtrString(micro.Limits.ChoiceWidth), intPtrString(micro.Limits.RegionCount))

	builder.WriteString("## Useful visual field (A3)\n\n")
	for _, capability := range MicroISACapabilities {
		if field, ok := micro.VisualField[capability]; ok {
			fmt.Fprintf(&builder, "- %s: max useful field = **%s**\n", capability, field.MaxUsefulField)
		}
	}
	builder.WriteString("\n## Counts\n\n")
	subs := make([]string, 0, len(micro.Counts.BySubExperiment))
	for name := range micro.Counts.BySubExperiment {
		subs = append(subs, name)
	}
	sort.Strings(subs)
	for _, name := range subs {
		count := micro.Counts.BySubExperiment[name]
		fmt.Fprintf(&builder, "- %s: %d independent base stimuli, %d observations\n",
			name, count.IndependentBaseStimuli, count.TotalObservations)
	}
	return builder.String()
}

func intPtrString(value *int) string {
	if value == nil {
		return "null (no safe rung)"
	}
	return strconv.Itoa(*value)
}

func conditionInt(condition string) int {
	parts := strings.SplitN(condition, "=", 2)
	if len(parts) != 2 {
		return 0
	}
	value, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return value
}

func sortedKeys(byRung map[int][]RunRecord) []int {
	keys := make([]int, 0, len(byRung))
	for key := range byRung {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}
