package temporalbench

type BenchmarkQuestion struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// CanonicalQuestions returns the fixed blind-test prompts for Temporal Native
// Benchmark R0. The evaluator remains the authority for scoring; these strings
// exist so campaign runners do not duplicate or drift from the benchmark.
func CanonicalQuestions() []BenchmarkQuestion {
	return []BenchmarkQuestion{
		{ID: "Q0", Text: "Read the visible BOOT/ROSETTA. What do BOX, ARROW, RING, and X/TIME mean in this carrier?"},
		{ID: "Q1", Text: "What cells or agents are represented?"},
		{ID: "Q2", Text: "What is the initial state of A?"},
		{ID: "Q3", Text: "What declared transition causes B to become ACTIVE?"},
		{ID: "Q4", Text: "According to the declared rules, what happens after B becomes ACTIVE?"},
		{ID: "Q5", Text: "What are the declared checkpoint times?"},
		{ID: "Q6", Text: "Is the temporal representation a literal video frame sequence? Explain briefly."},
		{ID: "Q7", Text: "Simulate only the declared transitions until no further rule applies. What final states do A, B, and C have?"},
		{ID: "Q8", Text: "Without using a mechanical exact decoder, give the hidden SHA-256 from the exact payload."},
	}
}
