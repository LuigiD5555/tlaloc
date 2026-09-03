package decompositionlab

import "testing"

func TestAggregateCondition_ComputesCountsAndAccuracy(t *testing.T) {
	outcomes := []RecordOutcome{
		{BaseID: "a", Category: CategoryNumeric, Attempted: true, ContractSuccess: true, SemanticCorrect: true, LatencyMS: 100, VisualExposureRatio: 0.1},
		{BaseID: "b", Category: CategoryNumeric, Attempted: true, ContractSuccess: true, SemanticCorrect: false, LatencyMS: 200, VisualExposureRatio: 0.2},
		{BaseID: "c", Category: CategoryEntity, Attempted: true, ContractSuccess: false, UnsupportedAssertion: true, Abstained: true, LatencyMS: 300, VisualExposureRatio: 1.0},
	}
	agg := AggregateCondition(ConditionC1OracleCrop, outcomes)
	if agg.N != 3 || agg.Attempted != 3 {
		t.Fatalf("unexpected n/attempted: %+v", agg)
	}
	if agg.ContractSuccess != 2 || agg.SemanticCorrect != 1 {
		t.Fatalf("unexpected success counts: %+v", agg)
	}
	if agg.ContractAccuracy != 2.0/3.0 {
		t.Fatalf("contract_accuracy = %v, want 2/3", agg.ContractAccuracy)
	}
	if agg.SemanticAccuracy != 1.0/3.0 {
		t.Fatalf("semantic_accuracy = %v, want 1/3", agg.SemanticAccuracy)
	}
	if agg.Abstained != 1 || agg.UnsupportedAssertion != 1 {
		t.Fatalf("unexpected abstain/unsupported counts: %+v", agg)
	}
	if agg.MeanLatencyMS != 200 {
		t.Fatalf("mean_latency_ms = %v, want 200", agg.MeanLatencyMS)
	}
	if agg.ByCategory[CategoryNumeric].N != 2 || agg.ByCategory[CategoryEntity].N != 1 {
		t.Fatalf("unexpected category breakdown: %+v", agg.ByCategory)
	}
	if agg.ExternalizedCapabilities != 1 {
		t.Fatalf("externalized_capabilities = %d, want 1 for C1", agg.ExternalizedCapabilities)
	}
}

func TestAggregateCondition_EmptyIsHonestlyZero(t *testing.T) {
	agg := AggregateCondition(ConditionC0ParrotDirect, nil)
	if agg.N != 0 || agg.ContractAccuracy != 0 || agg.SemanticAccuracy != 0 {
		t.Fatalf("expected an all-zero aggregate for no records, got %+v", agg)
	}
}

func TestPairTransition_MatchesByBaseIDAndComputesMcNemar(t *testing.T) {
	c0 := []RecordOutcome{
		{BaseID: "a", ContractSuccess: false, SemanticCorrect: false},
		{BaseID: "b", ContractSuccess: true, SemanticCorrect: true},
		{BaseID: "c", ContractSuccess: false, SemanticCorrect: false},
	}
	c1 := []RecordOutcome{
		{BaseID: "a", ContractSuccess: true, SemanticCorrect: true},
		{BaseID: "b", ContractSuccess: true, SemanticCorrect: true},
		{BaseID: "c", ContractSuccess: false, SemanticCorrect: false},
	}
	transition := PairTransition(ConditionC0ParrotDirect, ConditionC1OracleCrop, IndexByBaseID(c0), IndexByBaseID(c1))
	if transition.SemanticMcNemar.WrongToCorrect != 1 || transition.SemanticMcNemar.CorrectToWrong != 0 {
		t.Fatalf("unexpected mcnemar cells: %+v", transition.SemanticMcNemar)
	}
	want := 1.0 / 3.0
	if transition.SemanticDelta < want-1e-9 || transition.SemanticDelta > want+1e-9 {
		t.Fatalf("semantic_delta = %v, want %v", transition.SemanticDelta, want)
	}
}

func TestPairTransition_ExcludesUnmatchedBaseIDs(t *testing.T) {
	c0 := []RecordOutcome{{BaseID: "a", SemanticCorrect: true}, {BaseID: "only-in-c0", SemanticCorrect: true}}
	c1 := []RecordOutcome{{BaseID: "a", SemanticCorrect: true}, {BaseID: "only-in-c1", SemanticCorrect: false}}
	transition := PairTransition(ConditionC0ParrotDirect, ConditionC1OracleCrop, IndexByBaseID(c0), IndexByBaseID(c1))
	total := transition.SemanticMcNemar.CorrectToCorrect + transition.SemanticMcNemar.CorrectToWrong +
		transition.SemanticMcNemar.WrongToCorrect + transition.SemanticMcNemar.WrongToWrong
	if total != 1 {
		t.Fatalf("expected exactly one matched pair, got %d", total)
	}
}
