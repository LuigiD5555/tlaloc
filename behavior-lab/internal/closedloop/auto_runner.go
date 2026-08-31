package closedloop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/learningmemory"
)

// ValidateAutoReady validates the normal closed-loop inputs plus the explicit
// Origami-owned candidate builder capability contract. It performs no model
// inference and does not build candidates.
func ValidateAutoReady(ctx context.Context, cfg Config) error {
	p, err := prepare(cfg, true)
	if err != nil { return err }
	normalizeAutoConfig(&p.cfg)
	if !p.cfg.AutoCandidates { return nil }
	_, err = validateAutoConfig(ctx, p)
	return err
}

// RunAuto preserves the alpha.19 evidence, diagnostics, memory and incumbent
// gates while sourcing additional one-mutation candidate configs from the
// current adaptive plan. Tlaloc still does not render Origami pixels: it calls
// the explicitly configured Origami-owned builder through ensureCandidatePNG.
func RunAuto(ctx context.Context, cfg Config) (Report, error) {
	p, err := prepare(cfg, true)
	if err != nil { return Report{}, err }
	normalizeAutoConfig(&p.cfg)
	caps, err := validateAutoConfig(ctx, p)
	if err != nil { return Report{}, err }
	if !p.cfg.AutoCandidates {
		return Run(ctx, cfg)
	}
	if err := os.MkdirAll(p.cfg.OutputDir, 0o755); err != nil { return Report{}, err }

	incumbent := p.cfg.Baseline
	incumbentCandidateID := ""
	report := Report{
		Schema: ReportSchema, RunID: p.cfg.RunID, OutputDir: p.cfg.OutputDir,
		MemoryRoot: p.store.Root, InitialBaselineID: p.cfg.Baseline.ID,
		FinalIncumbentID: p.cfg.Baseline.ID,
		Authority: "EXPERIMENTAL_INCUMBENT_ONLY_ORIGAMI_CANONICAL_PROMOTION_REMAINS_EXTERNAL_AND_EVIDENCE_GATED",
	}
	tested := map[string]bool{}

	for gen := 1; gen <= p.cfg.MaxGenerations; gen++ {
		genDir := filepath.Join(p.cfg.OutputDir, fmt.Sprintf("generation-%03d", gen))
		if err := os.MkdirAll(genDir, 0o755); err != nil { return report, err }

		baseRun, errs, err := p.runSpecimen(ctx, gen, genDir, incumbent, incumbentCandidateID)
		report.ExecutionErrors = append(report.ExecutionErrors, errs...)
		if err != nil { return report, err }

		allEvents, err := p.store.LoadAll()
		if err != nil { return report, err }
		planBefore := adaptivesearch.BuildPlan(p.store.Root, planningEvents(allEvents, baseRun.events))
		planBeforePath := filepath.Join(genDir, "plan-before.json")
		if err := writeJSON(planBeforePath, planBefore); err != nil { return report, err }

		manual := p.availableCandidateConfigs(tested, incumbent.ID)
		automatic, err := p.autoCandidateConfigs(planBefore, caps, incumbent, genDir)
		if err != nil { return report, err }
		available, err := mergeCandidateConfigs(manual, automatic, tested, incumbent.ID)
		if err != nil { return report, err }
		visualCandidates, cfgByID := visualCandidatesFromConfigs(available)

		g := GenerationReport{
			Generation: gen, PlanBeforePath: planBeforePath, Baseline: baseRun.report,
			IncumbentBeforeID: incumbent.ID, IncumbentAfterID: incumbent.ID,
			ActiveFailureCount: countActiveFailures(baseRun.result), RemainingBank: len(available),
		}

		if baseRun.report.Scores.CleanTrials == 0 {
			g.PlanAfterPath = planBeforePath
			report.Generations = append(report.Generations, g)
			report.FinalPlanPath = planBeforePath
			report.StopReason = "INCUMBENT_EXECUTION_UNAVAILABLE"
			break
		}
		if g.ActiveFailureCount == 0 && !p.cfg.ContinueExplorationWhenStable {
			g.PlanAfterPath = planBeforePath
			g.IncumbentReason = "current experimental incumbent has no failed benchmark questions"
			report.Generations = append(report.Generations, g)
			report.FinalPlanPath = planBeforePath
			report.FinalIncumbentID = incumbent.ID
			report.StopReason = "INCUMBENT_NO_ACTIVE_FAILURES"
			break
		}

		queue := adaptivesearch.Prioritize(planBefore, visualCandidates, p.cfg.CandidatesPerGeneration)
		queuePath := filepath.Join(genDir, "candidate-queue.json")
		if err := writeJSON(queuePath, queue); err != nil { return report, err }
		g.QueuePath = queuePath
		if len(queue.CandidateOrder) == 0 {
			g.PlanAfterPath = planBeforePath
			g.IncumbentReason = "no supported untested candidate is eligible for the current experimental incumbent"
			g.RemainingBank = 0
			report.Generations = append(report.Generations, g)
			report.FinalPlanPath = planBeforePath
			report.FinalIncumbentID = incumbent.ID
			report.StopReason = "NO_ELIGIBLE_CANDIDATES"
			break
		}

		changeIDs := map[string][]string{}
		changeEvents := adaptivesearch.ChangeAttemptEvents(queue, visualCandidates)
		if len(changeEvents) == 0 { changeEvents = p.explorationChangeEvents(queue, baseRun.events, cfgByID) }
		if len(changeEvents) > 0 {
			_, _, stored, putErr := p.store.PutAll(changeEvents)
			if putErr != nil { return report, putErr }
			for _, e := range stored { changeIDs[e.CandidateID] = append(changeIDs[e.CandidateID], e.EventID) }
		}

		baselineMetric := metricValue(baseRun.report.Scores, p.cfg.OutcomeMetric)
		candidateRuns := map[string]specimenRun{}
		bestID := ""
		bestAfter := -1.0

		for _, item := range queue.CandidateOrder {
			cc, ok := cfgByID[item.CandidateID]
			if !ok { continue }
			tested[cc.ID] = true
			g.SelectedIDs = append(g.SelectedIDs, cc.ID)
			if err := p.ensureCandidatePNG(ctx, cc, incumbent); err != nil {
				report.ExecutionErrors = append(report.ExecutionErrors, ExecutionError{Generation:gen,SpecimenID:cc.ID,CandidateID:cc.ID,Error:"candidate build: "+err.Error()})
				continue
			}

			run, runErrs, runErr := p.runSpecimen(ctx, gen, genDir, SpecimenConfig{ID:cc.ID,PNG:cc.PNG}, cc.ID)
			report.ExecutionErrors = append(report.ExecutionErrors, runErrs...)
			if runErr != nil { return report, runErr }
			candidateRuns[cc.ID] = run
			g.Candidates = append(g.Candidates, run.report)
			if run.report.Scores.CleanTrials == 0 { continue }

			after := metricValue(run.report.Scores, p.cfg.OutcomeMetric)
			delta := after - baselineMetric
			nonRegress, nonRegressReason := nonRegression(baseRun.result, run.result)
			advanceable := nonRegress && delta >= p.cfg.MinIncumbentImprovement
			reason := nonRegressReason
			if nonRegress && !advanceable { reason = fmt.Sprintf("improvement %.6f below minimum %.6f",delta,p.cfg.MinIncumbentImprovement) }
			out := CandidateOutcome{CandidateID:cc.ID,Metric:p.cfg.OutcomeMetric,Before:baselineMetric,After:after,Delta:delta,NonRegress:nonRegress,Advanceable:advanceable,Reason:reason}

			parents := append([]string(nil), changeIDs[cc.ID]...)
			parents = append(parents, observationIDs(run.events)...)
			parents = dedupeLimit(parents, 30)
			if len(parents) >= 2 && len(changeIDs[cc.ID]) > 0 {
				beforeCopy, afterCopy, deltaCopy := baselineMetric, after, delta
				ev := learningmemory.Event{
					Schema:learningmemory.EventSchema, EventType:learningmemory.EventOutcome,
					EvidenceClass:learningmemory.EvidenceManual, CandidateID:cc.ID,
					ParentEventIDs:parents, BeforeScore:&beforeCopy, AfterScore:&afterCopy, Delta:&deltaCopy,
					Tags:[]string{"closed-loop","auto-candidate","metric:"+p.cfg.OutcomeMetric,fmt.Sprintf("generation:%d",gen),fmt.Sprintf("non-regression:%t",nonRegress),fmt.Sprintf("advanceable:%t",advanceable)},
				}
				_, stored, putErr := p.store.Put(ev)
				if putErr != nil { return report, putErr }
				out.EventID = stored.EventID
			}
			g.Outcomes = append(g.Outcomes, out)
			if advanceable && (bestID == "" || after > bestAfter || (after == bestAfter && cc.ID < bestID)) {
				bestID, bestAfter = cc.ID, after
			}
		}

		focusRun := baseRun
		if bestID != "" {
			winnerCfg := cfgByID[bestID]
			incumbent = SpecimenConfig{ID:winnerCfg.ID,PNG:winnerCfg.PNG}
			incumbentCandidateID = winnerCfg.ID
			focusRun = candidateRuns[bestID]
			g.IncumbentAdvanced = true
			g.IncumbentAfterID = winnerCfg.ID
			g.IncumbentReason = fmt.Sprintf("candidate %s became the next experimental incumbent after evidence-gated non-regression and improvement",winnerCfg.ID)
			report.FinalIncumbentID = winnerCfg.ID
		} else {
			g.IncumbentReason = "no tested candidate satisfied minimum improvement plus non-regression gates"
			report.FinalIncumbentID = incumbent.ID
		}

		allEvents, err = p.store.LoadAll()
		if err != nil { return report, err }
		planAfter := adaptivesearch.BuildPlan(p.store.Root, planningEvents(allEvents, focusRun.events))
		planAfterPath := filepath.Join(genDir, "plan-after.json")
		if err := writeJSON(planAfterPath, planAfter); err != nil { return report, err }
		g.PlanAfterPath = planAfterPath
		report.FinalPlanPath = planAfterPath

		nextDir := filepath.Join(p.cfg.OutputDir, fmt.Sprintf("generation-%03d", gen+1))
		nextManual := p.availableCandidateConfigs(tested, incumbent.ID)
		nextAuto, err := p.autoCandidateConfigs(planAfter, caps, incumbent, nextDir)
		if err != nil { return report, err }
		nextAvailable, err := mergeCandidateConfigs(nextManual, nextAuto, tested, incumbent.ID)
		if err != nil { return report, err }
		g.RemainingBank = len(nextAvailable)
		report.Generations = append(report.Generations, g)

		if g.IncumbentAdvanced && countActiveFailures(focusRun.result) == 0 && !p.cfg.ContinueExplorationWhenStable {
			report.StopReason = "INCUMBENT_NO_ACTIVE_FAILURES"
			break
		}
		if g.RemainingBank == 0 {
			report.StopReason = "NO_ELIGIBLE_CANDIDATES"
			break
		}
		if gen == p.cfg.MaxGenerations { report.StopReason = "MAX_GENERATIONS_REACHED" }
	}

	if report.StopReason == "" { report.StopReason = "COMPLETED" }
	if err := writeJSON(filepath.Join(p.cfg.OutputDir,"closed-loop-report.json"),report); err != nil { return report, err }
	return report, nil
}
