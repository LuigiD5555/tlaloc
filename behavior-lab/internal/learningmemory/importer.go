package learningmemory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"tlaloc.local/behaviorlab/internal/temporalbench"
)

type ImportOptions struct {
	OrigamiVersion   string
	TlalocVersion    string
	CandidateID      string
	RecordedAt       string
	IncludeSynthetic bool
}

func ImportTemporalBenchmark(campaignBody, resultBody []byte, c temporalbench.Campaign, r temporalbench.Result, opt ImportOptions) ([]Event, error) {
	campaignSHA := shaHex(campaignBody)
	resultSHA := shaHex(resultBody)
	trials := map[string]temporalbench.Trial{}
	for _, t := range c.Trials {
		trials[t.ID] = t
	}
	out := []Event{}
	for _, tr := range r.Trials {
		source, ok := trials[tr.TrialID]
		if !ok {
			return nil, fmt.Errorf("result trial %q not found in campaign", tr.TrialID)
		}
		class, placeholder := classifyModel(source.ModelID)
		if placeholder {
			continue
		}
		if class == EvidenceSynthetic && !opt.IncludeSynthetic {
			continue
		}
		debugByQ := map[string]temporalbench.DebugResult{}
		for _, d := range tr.DebugReports {
			debugByQ[d.QuestionID] = d
		}
		for _, q := range tr.Questions {
			pass := q.Pass
			overall := tr.OverallScore
			e := Event{
				Schema: EventSchema, EventType: EventObservation, EvidenceClass: class, RecordedAt: opt.RecordedAt,
				SourceCampaignSHA: campaignSHA, SourceResultSHA: resultSHA, BenchmarkID: r.BenchmarkID,
				TrialID: tr.TrialID, ModelID: source.ModelID, Provider: source.Provider, Condition: source.Condition,
				SpecimenID: source.Specimen.ID, SpecimenSHA256: source.Specimen.SHA256, QuestionID: q.QuestionID,
				ScoreLayer: q.Layer, Pass: &pass, OverallScore: &overall, OrigamiVersion: opt.OrigamiVersion,
				TlalocVersion: opt.TlalocVersion, CandidateID: opt.CandidateID,
			}
			if d, ok := debugByQ[q.QuestionID]; ok {
				e.Status = d.Status
				e.LastCompletedStage = d.LastCompletedStage
				e.FailureCode = d.FailureCode
				e.SelectedCodec = d.SelectedCodec
			}
			if !q.Pass && e.FailureCode == "" {
				e.FailureCode = "BENCHMARK_ASSERTION_FAILED"
			}
			out = append(out, e)
		}
	}
	return out, nil
}

func classifyModel(id string) (class string, placeholder bool) {
	u := strings.ToUpper(strings.TrimSpace(id))
	if u == "" || strings.Contains(u, "REPLACE_WITH") || strings.Contains(u, "PLACEHOLDER") {
		return "", true
	}
	if strings.HasPrefix(u, "SYNTHETIC") {
		return EvidenceSynthetic, false
	}
	return EvidenceRealModel, false
}

func shaHex(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }
