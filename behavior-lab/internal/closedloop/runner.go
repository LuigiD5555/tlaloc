package closedloop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/temporalbench"
	"tlaloc.local/behaviorlab/internal/visualsearch"
)

type prepared struct {
	cfg        Config
	master     string
	conditions []string
	store      learningmemory.Store
}

type specimenRun struct {
	report SpecimenReport
	events []learningmemory.Event
	result temporalbench.Result
}

func Validate(cfg Config) error {
	_, err := prepare(cfg, false)
	return err
}

func Run(ctx context.Context, cfg Config) (Report, error) {
	p, err := prepare(cfg, true)
	if err != nil {
		return Report{}, err
	}
	if err := os.MkdirAll(p.cfg.OutputDir, 0o755); err != nil {
		return Report{}, err
	}

	incumbent := p.cfg.Baseline
	incumbentCandidateID := ""
	report := Report{
		Schema:            ReportSchema,
		RunID:             p.cfg.RunID,
		OutputDir:         p.cfg.OutputDir,
		MemoryRoot:        p.store.Root,
		InitialBaselineID: p.cfg.Baseline.ID,
		FinalIncumbentID:  p.cfg.Baseline.ID,
		Authority:         "EXPERIMENTAL_INCUMBENT_ONLY_ORIGAMI_CANONICAL_PROMOTION_REMAINS_EXTERNAL_AND_EVIDENCE_GATED",
	}
	tested := map[string]bool{}

	for gen := 1; gen <= p.cfg.MaxGenerations; gen++ {
		genDir := filepath.Join(p.cfg.OutputDir, fmt.Sprintf("generation-%03d", gen))
		if err := os.MkdirAll(genDir, 0o755); err != nil {
			return report, err
		}

		baseRun, errs, err := p.runSpecimen(ctx, gen, genDir, incumbent, incumbentCandidateID)
		report.ExecutionErrors = append(report.ExecutionErrors, errs...)
		if err != nil {
			return report, err
		}

		allEvents, err := p.store.LoadAll()
		if err != nil {
			return report, err
		}
		planBefore := adaptivesearch.BuildPlan(p.store.Root, planningEvents(allEvents, baseRun.events))
		planBeforePath := filepath.Join(genDir, "plan-before.json")
		if err := writeJSON(planBeforePath, planBefore); err != nil {
			return report, err
		}

		g := GenerationReport{
			Generation:         gen,
			PlanBeforePath:     planBeforePath,
			Baseline:           baseRun.report,
			IncumbentBeforeID:  incumbent.ID,
			IncumbentAfterID:   incumbent.ID,
			ActiveFailureCount: countActiveFailures(baseRun.result),
		}

		if baseRun.report.Scores.CleanTrials == 0 {
			g.PlanAfterPath = planBeforePath
			g.RemainingBank = len(p.availableCandidateConfigs(tested, incumbent.ID))
			report.Generations = append(report.Generations, g)
			report.FinalPlanPath = planBeforePath
			report.StopReason = "INCUMBENT_EXECUTION_UNAVAILABLE"
			break
		}

		if g.ActiveFailureCount == 0 && !p.cfg.ContinueExplorationWhenStable {
			g.PlanAfterPath = planBeforePath
			g.IncumbentReason = "current experimental incumbent has no failed benchmark questions"
			g.RemainingBank = len(p.availableCandidateConfigs(tested, incumbent.ID))
			report.Generations = append(report.Generations, g)
			report.FinalPlanPath = planBeforePath
			report.FinalIncumbentID = incumbent.ID
			report.StopReason = "INCUMBENT_NO_ACTIVE_FAILURES"
			break
		}

		visualCandidates, cfgByID := p.availableCandidates(tested, incumbent.ID)
		queue := adaptivesearch.Prioritize(planBefore, visualCandidates, p.cfg.CandidatesPerGeneration)
		queuePath := filepath.Join(genDir, "candidate-queue.json")
		if err := writeJSON(queuePath, queue); err != nil {
			return report, err
		}
		g.QueuePath = queuePath

		if len(queue.CandidateOrder) == 0 {
			g.PlanAfterPath = planBeforePath
			g.IncumbentReason = "no untested candidate is eligible for the current experimental incumbent"
			g.RemainingBank = 0
			report.Generations = append(report.Generations, g)
			report.FinalPlanPath = planBeforePath
			report.FinalIncumbentID = incumbent.ID
			report.StopReason = "NO_ELIGIBLE_CANDIDATES"
			break
		}

		changeIDs := map[string][]string{}
		changeEvents := adaptivesearch.ChangeAttemptEvents(queue, visualCandidates)
		if len(changeEvents) == 0 {
			changeEvents = p.explorationChangeEvents(queue, baseRun.events, cfgByID)
		}
		if len(changeEvents) > 0 {
			_, _, stored, putErr := p.store.PutAll(changeEvents)
			if putErr != nil {
				return report, putErr
			}
			for _, e := range stored {
				changeIDs[e.CandidateID] = append(changeIDs[e.CandidateID], e.EventID)
			}
		}

		baselineMetric := metricValue(baseRun.report.Scores, p.cfg.OutcomeMetric)
		candidateRuns := map[string]specimenRun{}
		bestID := ""
		bestAfter := -1.0

		for _, item := range queue.CandidateOrder {
			cc, ok := cfgByID[item.CandidateID]
			if !ok {
				continue
			}
			tested[cc.ID] = true
			g.SelectedIDs = append(g.SelectedIDs, cc.ID)

			if err := p.ensureCandidatePNG(ctx, cc, incumbent); err != nil {
				report.ExecutionErrors = append(report.ExecutionErrors, ExecutionError{
					Generation:  gen,
					SpecimenID:  cc.ID,
					CandidateID: cc.ID,
					Error:       "candidate build: " + err.Error(),
				})
				continue
			}

			run, runErrs, runErr := p.runSpecimen(ctx, gen, genDir, SpecimenConfig{ID: cc.ID, PNG: cc.PNG}, cc.ID)
			report.ExecutionErrors = append(report.ExecutionErrors, runErrs...)
			if runErr != nil {
				return report, runErr
			}
			candidateRuns[cc.ID] = run
			g.Candidates = append(g.Candidates, run.report)
			if run.report.Scores.CleanTrials == 0 {
				continue
			}

			after := metricValue(run.report.Scores, p.cfg.OutcomeMetric)
			delta := after - baselineMetric
			nonRegress, nonRegressReason := nonRegression(baseRun.result, run.result)
			advanceable := nonRegress && delta >= p.cfg.MinIncumbentImprovement
			reason := nonRegressReason
			if nonRegress && !advanceable {
				reason = fmt.Sprintf("improvement %.6f below minimum %.6f", delta, p.cfg.MinIncumbentImprovement)
			}
			out := CandidateOutcome{
				CandidateID: cc.ID,
				Metric:      p.cfg.OutcomeMetric,
				Before:      baselineMetric,
				After:       after,
				Delta:       delta,
				NonRegress:  nonRegress,
				Advanceable: advanceable,
				Reason:      reason,
			}

			parents := append([]string(nil), changeIDs[cc.ID]...)
			parents = append(parents, observationIDs(run.events)...)
			parents = dedupeLimit(parents, 30)
			if len(parents) >= 2 && len(changeIDs[cc.ID]) > 0 {
				beforeCopy, afterCopy, deltaCopy := baselineMetric, after, delta
				tags := []string{
					"closed-loop",
					"metric:" + p.cfg.OutcomeMetric,
					fmt.Sprintf("generation:%d", gen),
					fmt.Sprintf("non-regression:%t", nonRegress),
					fmt.Sprintf("advanceable:%t", advanceable),
				}
				ev := learningmemory.Event{
					Schema:         learningmemory.EventSchema,
					EventType:      learningmemory.EventOutcome,
					EvidenceClass:  learningmemory.EvidenceManual,
					CandidateID:    cc.ID,
					ParentEventIDs: parents,
					BeforeScore:    &beforeCopy,
					AfterScore:     &afterCopy,
					Delta:          &deltaCopy,
					Tags:           tags,
				}
				_, stored, putErr := p.store.Put(ev)
				if putErr != nil {
					return report, putErr
				}
				out.EventID = stored.EventID
			}
			g.Outcomes = append(g.Outcomes, out)

			if advanceable && (bestID == "" || after > bestAfter || (after == bestAfter && cc.ID < bestID)) {
				bestID = cc.ID
				bestAfter = after
			}
		}

		focusRun := baseRun
		if bestID != "" {
			winnerCfg := cfgByID[bestID]
			incumbent = SpecimenConfig{ID: winnerCfg.ID, PNG: winnerCfg.PNG}
			incumbentCandidateID = winnerCfg.ID
			focusRun = candidateRuns[bestID]
			g.IncumbentAdvanced = true
			g.IncumbentAfterID = winnerCfg.ID
			g.IncumbentReason = fmt.Sprintf("candidate %s became the next experimental incumbent after evidence-gated non-regression and improvement", winnerCfg.ID)
			report.FinalIncumbentID = winnerCfg.ID
		} else {
			g.IncumbentReason = "no tested candidate satisfied minimum improvement plus non-regression gates"
			report.FinalIncumbentID = incumbent.ID
		}

		allEvents, err = p.store.LoadAll()
		if err != nil {
			return report, err
		}
		planAfter := adaptivesearch.BuildPlan(p.store.Root, planningEvents(allEvents, focusRun.events))
		planAfterPath := filepath.Join(genDir, "plan-after.json")
		if err := writeJSON(planAfterPath, planAfter); err != nil {
			return report, err
		}
		g.PlanAfterPath = planAfterPath
		report.FinalPlanPath = planAfterPath
		g.RemainingBank = len(p.availableCandidateConfigs(tested, incumbent.ID))
		report.Generations = append(report.Generations, g)

		if g.IncumbentAdvanced && countActiveFailures(focusRun.result) == 0 && !p.cfg.ContinueExplorationWhenStable {
			report.StopReason = "INCUMBENT_NO_ACTIVE_FAILURES"
			break
		}
		if g.RemainingBank == 0 {
			report.StopReason = "NO_ELIGIBLE_CANDIDATES"
			break
		}
		if gen == p.cfg.MaxGenerations {
			report.StopReason = "MAX_GENERATIONS_REACHED"
		}
	}

	if report.StopReason == "" {
		report.StopReason = "COMPLETED"
	}
	if err := writeJSON(filepath.Join(p.cfg.OutputDir, "closed-loop-report.json"), report); err != nil {
		return report, err
	}
	return report, nil
}

func prepare(cfg Config, checkFiles bool) (prepared, error) {
	if cfg.Schema != "" && cfg.Schema != ConfigSchema {
		return prepared{}, fmt.Errorf("unexpected schema %q", cfg.Schema)
	}
	cfg.Schema = ConfigSchema
	if strings.TrimSpace(cfg.RunID) == "" {
		return prepared{}, fmt.Errorf("run_id is required")
	}
	if strings.TrimSpace(cfg.OutputDir) == "" {
		return prepared{}, fmt.Errorf("output_dir is required")
	}
	if cfg.BenchmarkID == "" {
		cfg.BenchmarkID = "origami-temporal-native-r0"
	}
	if cfg.TrialsPerModel <= 0 {
		cfg.TrialsPerModel = 1
	}
	if cfg.CandidatesPerGeneration <= 0 {
		cfg.CandidatesPerGeneration = 2
	}
	if cfg.MaxGenerations <= 0 {
		cfg.MaxGenerations = 1
	}
	if cfg.MinIncumbentImprovement <= 0 {
		cfg.MinIncumbentImprovement = 0.01
	}
	if cfg.OutcomeMetric == "" {
		cfg.OutcomeMetric = OutcomeNative
	}
	if cfg.OutcomeMetric != OutcomeNative && cfg.OutcomeMetric != OutcomeOverall {
		return prepared{}, fmt.Errorf("unsupported outcome_metric %q", cfg.OutcomeMetric)
	}
	if len(cfg.Conditions) == 0 {
		cfg.Conditions = []string{"NATIVE_PNG_ONLY"}
		if cfg.MasterPrompt != "" {
			cfg.Conditions = append(cfg.Conditions, "R4_ASSISTED")
		}
	}

	conditions := []string{}
	seenCondition := map[string]bool{}
	for _, c := range cfg.Conditions {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c != "NATIVE_PNG_ONLY" && c != "R4_ASSISTED" {
			return prepared{}, fmt.Errorf("unsupported condition %q", c)
		}
		if !seenCondition[c] {
			conditions = append(conditions, c)
			seenCondition[c] = true
		}
	}
	if cfg.OutcomeMetric == OutcomeNative && !seenCondition["NATIVE_PNG_ONLY"] {
		return prepared{}, fmt.Errorf("NATIVE_SCORE requires NATIVE_PNG_ONLY condition")
	}
	if strings.TrimSpace(cfg.Baseline.ID) == "" || strings.TrimSpace(cfg.Baseline.PNG) == "" {
		return prepared{}, fmt.Errorf("baseline id and png are required")
	}
	if len(cfg.Models) == 0 {
		return prepared{}, fmt.Errorf("at least one model is required")
	}

	modelNames := map[string]bool{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if m.Name == "" {
			m.Name = m.Model
		}
		if m.Provider == "" {
			m.Provider = "OPENAI_COMPAT"
		}
		if strings.ToUpper(m.Provider) != "OPENAI_COMPAT" {
			return prepared{}, fmt.Errorf("model %s provider %q unsupported in R0", m.Name, m.Provider)
		}
		if m.Model == "" {
			return prepared{}, fmt.Errorf("model %d model is required", i)
		}
		if m.Compatibility == "" {
			m.Compatibility = target.CompatibilityOpenAI
		}
		strategy, err := target.ResolveMultimodalCompatibility(m.Compatibility)
		if err != nil {
			return prepared{}, fmt.Errorf("model %s compatibility: %w", m.Name, err)
		}
		m.Compatibility = strategy.Name()
		if modelNames[m.Name] {
			return prepared{}, fmt.Errorf("duplicate model name %q", m.Name)
		}
		modelNames[m.Name] = true
		if m.TimeoutSeconds <= 0 {
			m.TimeoutSeconds = 120
		}
		if m.TransportRetries < 0 {
			m.TransportRetries = 0
		}
	}

	candidateIDs := map[string]bool{}
	for i := range cfg.Candidates {
		c := &cfg.Candidates[i]
		if c.ID == "" || c.PNG == "" {
			return prepared{}, fmt.Errorf("candidate %d requires id and png", i)
		}
		if c.ID == cfg.Baseline.ID {
			return prepared{}, fmt.Errorf("candidate %q conflicts with baseline id", c.ID)
		}
		if candidateIDs[c.ID] {
			return prepared{}, fmt.Errorf("duplicate candidate id %q", c.ID)
		}
		candidateIDs[c.ID] = true
		if c.BaseProfileID == "" {
			return prepared{}, fmt.Errorf("candidate %q base_profile_id required", c.ID)
		}
		if len(c.Mutations) == 0 {
			return prepared{}, fmt.Errorf("candidate %q requires mutations", c.ID)
		}
		for j, m := range c.Mutations {
			if !m.Experimental {
				return prepared{}, fmt.Errorf("candidate %q mutation %d must remain experimental", c.ID, j)
			}
		}
	}
	if err := validateCandidateParents(cfg.Baseline.ID, cfg.Candidates, candidateIDs); err != nil {
		return prepared{}, err
	}

	master := ""
	if seenCondition["R4_ASSISTED"] {
		if cfg.MasterPrompt == "" {
			return prepared{}, fmt.Errorf("R4_ASSISTED requires master_prompt")
		}
		if checkFiles {
			b, err := os.ReadFile(cfg.MasterPrompt)
			if err != nil {
				return prepared{}, err
			}
			master = string(b)
		}
	}
	if checkFiles {
		if _, err := readPNGMeta(cfg.Baseline.PNG); err != nil {
			return prepared{}, fmt.Errorf("baseline: %w", err)
		}
		for _, c := range cfg.Candidates {
			if _, err := os.Stat(c.PNG); err == nil {
				if _, err := readPNGMeta(c.PNG); err != nil {
					return prepared{}, fmt.Errorf("candidate %s: %w", c.ID, err)
				}
			} else if len(c.BuildCommand) == 0 {
				return prepared{}, fmt.Errorf("candidate %s png missing and no build_command: %w", c.ID, err)
			}
		}
	}
	return prepared{cfg: cfg, master: master, conditions: conditions, store: learningmemory.New(cfg.MemoryRoot)}, nil
}

func validateCandidateParents(baselineID string, candidates []CandidateConfig, ids map[string]bool) error {
	parent := map[string]string{}
	for _, c := range candidates {
		p := strings.TrimSpace(c.ParentSpecimenID)
		if p == "" {
			continue
		}
		if p == c.ID {
			return fmt.Errorf("candidate %q cannot parent itself", c.ID)
		}
		if p != baselineID && !ids[p] {
			return fmt.Errorf("candidate %q references unknown parent_specimen_id %q", c.ID, p)
		}
		parent[c.ID] = p
	}
	for id := range ids {
		seen := map[string]bool{}
		cur := id
		for {
			p := parent[cur]
			if p == "" || p == baselineID {
				break
			}
			if seen[p] {
				return fmt.Errorf("candidate parent cycle detected at %q", p)
			}
			seen[p] = true
			cur = p
		}
	}
	return nil
}

func (p prepared) runSpecimen(ctx context.Context, generation int, genDir string, s SpecimenConfig, candidateID string) (specimenRun, []ExecutionError, error) {
	meta, err := readPNGMeta(s.PNG)
	if err != nil {
		return specimenRun{}, nil, err
	}
	imageBytes := meta.bytes
	campaign := temporalbench.Campaign{Schema: temporalbench.CampaignSchema, BenchmarkID: p.cfg.BenchmarkID}
	errs := []ExecutionError{}
	questions := temporalbench.CanonicalQuestions()
	models := map[string]ModelConfig{}
	for _, m := range p.cfg.Models {
		models[m.Name] = m
	}

	for _, m := range p.cfg.Models {
		for trialN := 1; trialN <= p.cfg.TrialsPerModel; trialN++ {
			for _, condition := range p.conditions {
				trial := temporalbench.Trial{
					ID:        fmt.Sprintf("g%03d-%s-%s-%s-%02d", generation, slug(s.ID), slug(m.Name), strings.ToLower(condition), trialN),
					ModelID:   m.Name,
					Provider:  m.Provider,
					Condition: condition,
					Specimen: temporalbench.Specimen{
						ID:       s.ID,
						SHA256:   meta.sha,
						Variant:  "original",
						PNGBytes: len(imageBytes),
						Width:    meta.width,
						Height:   meta.height,
					},
				}
				system := p.systemFor(condition)
				complete := true
				for _, q := range questions {
					r, callErr := p.call(ctx, m, system, q.Text, imageBytes)
					if callErr != nil {
						errs = append(errs, ExecutionError{Generation: generation, SpecimenID: s.ID, CandidateID: candidateID, ModelID: m.Name, Condition: condition, QuestionID: q.ID, Error: callErr.Error()})
						complete = false
						break
					}
					r.QuestionID = q.ID
					trial.Responses = append(trial.Responses, r)
				}
				if complete {
					campaign.Trials = append(campaign.Trials, trial)
				}
			}
		}
	}

	clean := temporalbench.EvaluateCampaign(campaign)
	if p.cfg.DiagnosticRetries {
		byTrial := map[string]temporalbench.Trial{}
		for _, t := range campaign.Trials {
			byTrial[t.ID] = t
		}
		for _, tr := range clean.Trials {
			failed := []string{}
			for _, q := range tr.Questions {
				if !q.Pass {
					failed = append(failed, q.QuestionID)
				}
			}
			if len(failed) == 0 {
				continue
			}
			source := byTrial[tr.TrialID]
			m, ok := models[source.ModelID]
			if !ok {
				continue
			}
			diag := temporalbench.Trial{
				ID:                    source.ID + "-diag",
				ModelID:               source.ModelID,
				Provider:              source.Provider,
				Condition:             source.Condition,
				DiagnosticMode:        true,
				DiagnosticQuestionIDs: failed,
				Specimen:              source.Specimen,
			}
			system := strings.TrimSpace(p.systemFor(source.Condition) + "\n\n" + temporalbench.DiagnosticInstruction())
			diagComplete := true
			qByID := map[string]string{}
			for _, q := range questions {
				qByID[q.ID] = q.Text
			}
			for _, qid := range failed {
				r, callErr := p.call(ctx, m, system, qByID[qid], imageBytes)
				if callErr != nil {
					errs = append(errs, ExecutionError{Generation: generation, SpecimenID: s.ID, CandidateID: candidateID, ModelID: m.Name, Condition: source.Condition, QuestionID: qid, Diagnostic: true, Error: callErr.Error()})
					diagComplete = false
					break
				}
				r.QuestionID = qid
				diag.Responses = append(diag.Responses, r)
			}
			if diagComplete && len(diag.Responses) == len(failed) {
				campaign.Trials = append(campaign.Trials, diag)
			}
		}
	}

	result := temporalbench.EvaluateCampaign(campaign)
	specDir := filepath.Join(genDir, slug(s.ID))
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return specimenRun{}, errs, err
	}
	campaignPath := filepath.Join(specDir, "campaign.json")
	resultPath := filepath.Join(specDir, "result.json")
	campaignBody, _ := json.MarshalIndent(campaign, "", "  ")
	campaignBody = append(campaignBody, '\n')
	resultBody, _ := json.MarshalIndent(result, "", "  ")
	resultBody = append(resultBody, '\n')
	if err := os.WriteFile(campaignPath, campaignBody, 0o644); err != nil {
		return specimenRun{}, errs, err
	}
	if err := os.WriteFile(resultPath, resultBody, 0o644); err != nil {
		return specimenRun{}, errs, err
	}

	imported, err := learningmemory.ImportTemporalBenchmark(campaignBody, resultBody, campaign, result, learningmemory.ImportOptions{
		OrigamiVersion: p.cfg.OrigamiVersion,
		TlalocVersion:  p.cfg.TlalocVersion,
		CandidateID:    candidateID,
	})
	if err != nil {
		return specimenRun{}, errs, err
	}
	_, _, stored, err := p.store.PutAll(imported)
	if err != nil {
		return specimenRun{}, errs, err
	}
	ids := []string{}
	for _, e := range stored {
		ids = append(ids, e.EventID)
	}
	rep := SpecimenReport{
		SpecimenID:      s.ID,
		CandidateID:     candidateID,
		PNG:             s.PNG,
		SHA256:          meta.sha,
		Scores:          summarizeScores(result),
		CampaignPath:    campaignPath,
		ResultPath:      resultPath,
		MemoryEvents:    len(stored),
		MemoryEventIDs:  ids,
		ExecutionErrors: countSpecimenErrors(errs, s.ID, candidateID),
	}
	return specimenRun{report: rep, events: stored, result: result}, errs, nil
}

func (p prepared) call(parent context.Context, m ModelConfig, system, question string, image []byte) (temporalbench.Response, error) {
	compatibility, err := target.ResolveMultimodalCompatibility(m.Compatibility)
	if err != nil {
		return temporalbench.Response{}, err
	}
	client := target.OpenAICompat{BaseURL: m.BaseURL, Model: m.Model, Temperature: m.Temperature, Compatibility: compatibility}
	if m.APIKeyEnv != "" {
		client.APIKey = os.Getenv(m.APIKeyEnv)
	}
	attempts := m.TransportRetries + 1
	var last error
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(parent, time.Duration(m.TimeoutSeconds)*time.Second)
		start := time.Now()
		r, err := client.CompletePerception(ctx, target.PerceptionInput{SystemPrompt: system, Question: question, Image: image, MediaType: "image/png"})
		cancel()
		if err == nil {
			return temporalbench.Response{Text: r.Content, LatencyMS: time.Since(start).Milliseconds()}, nil
		}
		last = err
	}
	return temporalbench.Response{}, last
}

func (p prepared) systemFor(condition string) string {
	if condition == "R4_ASSISTED" {
		return p.master
	}
	return ""
}

func (p prepared) availableCandidateConfigs(tested map[string]bool, incumbentID string) []CandidateConfig {
	out := []CandidateConfig{}
	for _, c := range p.cfg.Candidates {
		if tested[c.ID] {
			continue
		}
		if c.ParentSpecimenID != "" && c.ParentSpecimenID != incumbentID {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (p prepared) availableCandidates(tested map[string]bool, incumbentID string) ([]visualsearch.Candidate, map[string]CandidateConfig) {
	out := []visualsearch.Candidate{}
	by := map[string]CandidateConfig{}
	for _, c := range p.availableCandidateConfigs(tested, incumbentID) {
		vc := c.VisualCandidate()
		out = append(out, vc)
		by[c.ID] = c
	}
	return out, by
}

func (p prepared) explorationChangeEvents(queue adaptivesearch.Queue, baseline []learningmemory.Event, by map[string]CandidateConfig) []learningmemory.Event {
	parents := observationIDs(baseline)
	parents = dedupeLimit(parents, 20)
	if len(parents) == 0 {
		return nil
	}
	out := []learningmemory.Event{}
	for _, item := range queue.CandidateOrder {
		c, ok := by[item.CandidateID]
		if !ok {
			continue
		}
		tags := []string{"closed-loop", "exploration"}
		for _, m := range c.Mutations {
			tags = append(tags, "mutation:"+string(m.Kind))
		}
		sort.Strings(tags)
		out = append(out, learningmemory.Event{
			Schema:         learningmemory.EventSchema,
			EventType:      learningmemory.EventChange,
			EvidenceClass:  learningmemory.EvidenceManual,
			CandidateID:    c.ID,
			ParentEventIDs: parents,
			ChangeSummary:  "Closed-loop exploration candidate selected before evidence scoring",
			Tags:           tags,
		})
	}
	return out
}

func (p prepared) ensureCandidatePNG(ctx context.Context, c CandidateConfig, parent SpecimenConfig) error {
	if _, err := os.Stat(c.PNG); err == nil {
		return nil
	}
	if len(c.BuildCommand) == 0 {
		return fmt.Errorf("png %s missing", c.PNG)
	}
	if err := os.MkdirAll(filepath.Dir(c.PNG), 0o755); err != nil {
		return err
	}
	mutationJSON, _ := json.Marshal(c.Mutations)
	cmd := exec.CommandContext(ctx, c.BuildCommand[0], c.BuildCommand[1:]...)
	cmd.Env = append(os.Environ(),
		"TLALOC_CANDIDATE_ID="+c.ID,
		"TLALOC_OUTPUT_PNG="+c.PNG,
		"TLALOC_MUTATIONS_JSON="+string(mutationJSON),
		"TLALOC_PARENT_SPECIMEN_ID="+parent.ID,
		"TLALOC_PARENT_PNG="+parent.PNG,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(output)))
	}
	_, err = readPNGMeta(c.PNG)
	return err
}

func planningEvents(all, current []learningmemory.Event) []learningmemory.Event {
	currentIDs := map[string]bool{}
	for _, e := range current {
		if e.EventID != "" {
			currentIDs[e.EventID] = true
		}
	}
	out := []learningmemory.Event{}
	for _, e := range all {
		if e.EventType == learningmemory.EventObservation {
			if currentIDs[e.EventID] {
				out = append(out, e)
			}
			continue
		}
		out = append(out, e)
	}
	return out
}

func countActiveFailures(r temporalbench.Result) int {
	n := 0
	for _, t := range r.Trials {
		if t.DiagnosticMode {
			continue
		}
		for _, q := range t.Questions {
			if !q.Pass {
				n++
			}
		}
	}
	return n
}

func nonRegression(base, candidate temporalbench.Result) (bool, string) {
	baseClean, candClean := cleanTrialCount(base), cleanTrialCount(candidate)
	if candClean < baseClean {
		return false, fmt.Sprintf("candidate has fewer complete clean trials (%d < %d)", candClean, baseClean)
	}
	if inventedExact(candidate) > inventedExact(base) {
		return false, "candidate increases invented exact claims"
	}
	if missingQuestions(candidate) > missingQuestions(base) {
		return false, "candidate increases missing benchmark questions"
	}
	baseQ := questionMeans(base)
	candQ := questionMeans(candidate)
	keys := make([]string, 0, len(baseQ))
	for q := range baseQ {
		keys = append(keys, q)
	}
	sort.Strings(keys)
	for _, q := range keys {
		cv, ok := candQ[q]
		if !ok {
			return false, "candidate omits question " + q
		}
		if cv+1e-12 < baseQ[q] {
			return false, fmt.Sprintf("candidate regresses question %s from %.4f to %.4f", q, baseQ[q], cv)
		}
	}
	return true, "candidate preserves every baseline question score and exactness discipline"
}

func cleanTrialCount(r temporalbench.Result) int {
	n := 0
	for _, t := range r.Trials {
		if !t.DiagnosticMode {
			n++
		}
	}
	return n
}

func inventedExact(r temporalbench.Result) int {
	n := 0
	for _, t := range r.Trials {
		if !t.DiagnosticMode {
			n += t.InventedExactClaims
		}
	}
	return n
}

func missingQuestions(r temporalbench.Result) int {
	n := 0
	for _, t := range r.Trials {
		if !t.DiagnosticMode {
			n += t.MissingQuestionCount
		}
	}
	return n
}

func questionMeans(r temporalbench.Result) map[string]float64 {
	sum := map[string]float64{}
	count := map[string]int{}
	for _, t := range r.Trials {
		if t.DiagnosticMode {
			continue
		}
		for _, q := range t.Questions {
			sum[q.QuestionID] += q.Score
			count[q.QuestionID]++
		}
	}
	out := map[string]float64{}
	for q, v := range sum {
		if count[q] > 0 {
			out[q] = v / float64(count[q])
		}
	}
	return out
}

type pngMeta struct {
	bytes         []byte
	sha           string
	width, height int
}

func readPNGMeta(path string) (pngMeta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return pngMeta{}, err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return pngMeta{}, fmt.Errorf("invalid PNG %s: %w", path, err)
	}
	sum := sha256.Sum256(b)
	return pngMeta{bytes: b, sha: hex.EncodeToString(sum[:]), width: cfg.Width, height: cfg.Height}, nil
}

func summarizeScores(r temporalbench.Result) ScoreSummary {
	out := ScoreSummary{}
	nativeN, assistN := 0, 0
	for _, t := range r.Trials {
		if t.DiagnosticMode {
			continue
		}
		out.CleanTrials++
		out.MeanOverall += t.OverallScore
		if t.Condition == "NATIVE_PNG_ONLY" {
			out.MeanNative += t.OverallScore
			nativeN++
		}
		if t.Condition == "R4_ASSISTED" {
			out.MeanAssisted += t.OverallScore
			assistN++
		}
	}
	if out.CleanTrials > 0 {
		out.MeanOverall /= float64(out.CleanTrials)
	}
	if nativeN > 0 {
		out.MeanNative /= float64(nativeN)
	}
	if assistN > 0 {
		out.MeanAssisted /= float64(assistN)
	}
	return out
}

func metricValue(s ScoreSummary, metric string) float64 {
	if metric == OutcomeOverall {
		return s.MeanOverall
	}
	return s.MeanNative
}

func observationIDs(events []learningmemory.Event) []string {
	out := []string{}
	for _, e := range events {
		if e.EventType == learningmemory.EventObservation && e.EventID != "" {
			out = append(out, e.EventID)
		}
	}
	sort.Strings(out)
	return out
}

func dedupeLimit(in []string, n int) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range in {
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
		if n > 0 && len(out) >= n {
			break
		}
	}
	return out
}

func countSpecimenErrors(in []ExecutionError, s, c string) int {
	n := 0
	for _, e := range in {
		if e.SpecimenID == s && e.CandidateID == c {
			n++
		}
	}
	return n
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
