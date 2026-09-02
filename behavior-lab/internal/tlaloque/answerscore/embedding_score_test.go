package answerscore

import "testing"

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float64
		want float64
	}{
		{name: "identical vectors", a: []float64{1, 2, 3}, b: []float64{1, 2, 3}, want: 1.0},
		{name: "orthogonal vectors", a: []float64{1, 0}, b: []float64{0, 1}, want: 0.0},
		{name: "opposite vectors", a: []float64{1, 0}, b: []float64{-1, 0}, want: -1.0},
		{name: "mismatched length", a: []float64{1, 2}, b: []float64{1}, want: 0.0},
		{name: "zero vector", a: []float64{0, 0}, b: []float64{1, 1}, want: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if diff := got - tt.want; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("cosineSimilarity(%v, %v) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestScoreByBestPassageSimilarity(t *testing.T) {
	answerVec := []float64{1, 2, 3}
	passages := [][]float64{
		{-1, -2, -3}, // unrelated passage
		{1, 2, 3},    // the matching passage
		{0, 1, 0},    // another unrelated passage
	}

	out := ScoreByBestPassageSimilarity(answerVec, passages)
	if out.Score != 1.0 {
		t.Errorf("expected the best-matching passage to win with score 1.0, got %f", out.Score)
	}

	// Opposite vectors would yield a negative cosine similarity; the score
	// must be clamped to the [0,1] range the rest of the harness expects.
	clamped := ScoreByBestPassageSimilarity([]float64{1, 0}, [][]float64{{-1, 0}})
	if clamped.Score != 0.0 {
		t.Errorf("expected score clamped to 0.0 for opposite vectors, got %f", clamped.Score)
	}

	// No passages at all must not panic and must report a floor score.
	empty := ScoreByBestPassageSimilarity(answerVec, nil)
	if empty.Score != 0.0 {
		t.Errorf("expected score 0.0 when there are no passages to compare against, got %f", empty.Score)
	}
}

func TestSplitIntoPassages(t *testing.T) {
	got := SplitIntoPassages("Un swarm coordina agentes distribuidos. Ok. Las reglas locales generan comportamiento emergente.")
	if len(got) != 2 {
		t.Fatalf("expected 2 passages (the short fragment 'Ok' filtered out), got %d: %v", len(got), got)
	}
}
