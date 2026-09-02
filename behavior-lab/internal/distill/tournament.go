package distill

import "fmt"

type Fitness struct {
	BootstrapSuccess  float64 `json:"bootstrap_success"`
	RosettaSuccess    float64 `json:"rosetta_success"`
	Navigation        float64 `json:"navigation"`
	Correctness       float64 `json:"correctness"`
	Evidence          float64 `json:"evidence"`
	UnknownAccuracy   float64 `json:"unknown_accuracy"`
	FalseExact        int     `json:"false_exact"`
	PeakActiveTokenEq int     `json:"peak_active_token_eq"`
	ToolCost          float64 `json:"tool_cost"`
	Contaminated      bool    `json:"contaminated"`
}

type ScoredCandidate struct {
	Candidate Candidate `json:"candidate"`
	Fitness   Fitness   `json:"fitness"`
	Score     float64   `json:"score"`
	Eligible  bool      `json:"eligible"`
	Reason    string    `json:"reason,omitempty"`
}

func score(f Fitness, workingWindow int) (float64, bool, string) {
	if f.Contaminated {
		return 0, false, "contaminated trial"
	}
	if f.FalseExact != 0 {
		return 0, false, "FALSE_EXACT must equal zero"
	}
	if workingWindow <= 0 {
		return 0, false, "working window must be positive"
	}
	if f.PeakActiveTokenEq > workingWindow {
		return 0, false, "active interface budget exceeded"
	}
	metrics := []float64{f.BootstrapSuccess, f.RosettaSuccess, f.Navigation, f.Correctness, f.Evidence, f.UnknownAccuracy}
	for _, v := range metrics {
		if v < 0 || v > 1 {
			return 0, false, "fitness rates must be in [0,1]"
		}
	}
	// Safety and semantic quality dominate. Cost is a small tie-breaker rather
	// than a reason to prefer an incorrect or unverifiable candidate.
	s := 0.15*f.BootstrapSuccess +
		0.15*f.RosettaSuccess +
		0.15*f.Navigation +
		0.25*f.Correctness +
		0.15*f.Evidence +
		0.15*f.UnknownAccuracy
	if f.ToolCost > 0 {
		s -= 0.001 * f.ToolCost
	}
	return s, true, ""
}

func Rank(candidates []ScoredCandidate, workingWindow int) ([]ScoredCandidate, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("at least one receiver candidate is required")
	}
	out := append([]ScoredCandidate(nil), candidates...)
	for i := range out {
		s, ok, reason := score(out[i].Fitness, workingWindow)
		out[i].Score = s
		out[i].Eligible = ok
		out[i].Reason = reason
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if better(out[j], out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func better(a, b ScoredCandidate) bool {
	if a.Eligible != b.Eligible {
		return a.Eligible
	}
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return a.Candidate.ID < b.Candidate.ID
}

func Winner(candidates []ScoredCandidate, workingWindow int) (ScoredCandidate, error) {
	ranked, err := Rank(candidates, workingWindow)
	if err != nil {
		return ScoredCandidate{}, err
	}
	if !ranked[0].Eligible {
		return ScoredCandidate{}, fmt.Errorf("no eligible receiver candidate")
	}
	return ranked[0], nil
}
