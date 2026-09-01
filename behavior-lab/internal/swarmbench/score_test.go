package swarmbench

import "testing"

func mustItem(t *testing.T) Item {
	t.Helper()
	dataset, err := Generate("score", 11, 1)
	if err != nil {
		t.Fatal(err)
	}
	return dataset.Items[0]
}

func TestScoreItemPerfectRecoveryIsExactMatch(t *testing.T) {
	item := mustItem(t)
	scored := ScoreItem(item, item.Expected)
	if !scored.ExactMatch {
		t.Fatalf("expected an exact match: %+v", scored)
	}
	if !scored.RouteCorrect {
		t.Fatal("route should be correct")
	}
	for _, field := range scored.Fields {
		if !field.Correct {
			t.Fatalf("field %s reported incorrect on perfect recovery", field.Field)
		}
	}
}

// A missing/errored field must score as incorrect, not be silently absent —
// this is what keeps the accuracy metric honest under partial swarm failure.
func TestScoreItemMissingFieldScoresIncorrect(t *testing.T) {
	item := mustItem(t)
	scored := ScoreItem(item, Fields{})
	if scored.ExactMatch {
		t.Fatal("an empty recovery must not be an exact match")
	}
	for _, field := range scored.Fields {
		if field.Correct {
			t.Fatalf("field %s reported correct on empty recovery", field.Field)
		}
	}
}

// One wrong upstream field (intent) must still let a correctly recovered
// amount score independently correct: field accuracy is per-field, not
// all-or-nothing.
func TestScoreItemIsolatesFieldErrors(t *testing.T) {
	item := mustItem(t)
	got := item.Expected
	got.Intent = "WRONG_INTENT"
	scored := ScoreItem(item, got)
	if scored.ExactMatch {
		t.Fatal("a wrong field must break exact match")
	}
	for _, field := range scored.Fields {
		if field.Field == "amount" && !field.Correct {
			t.Fatal("amount field must still score correct despite an unrelated intent error")
		}
		if field.Field == "intent" && field.Correct {
			t.Fatal("intent field must score incorrect")
		}
	}
}

func TestScoreDatasetAggregatesAcrossItems(t *testing.T) {
	dataset, err := Generate("score-agg", 21, 50)
	if err != nil {
		t.Fatal(err)
	}
	score := ScoreDataset(dataset, func(item Item) Fields { return item.Expected })
	if score.ItemCount != 50 {
		t.Fatalf("item_count=%d", score.ItemCount)
	}
	if score.ExactMatchRate != 1.0 {
		t.Fatalf("exact_match_rate=%v, want 1.0 for perfect recovery", score.ExactMatchRate)
	}
	if score.RouteAccuracy != 1.0 {
		t.Fatalf("route_accuracy=%v, want 1.0", score.RouteAccuracy)
	}
	for _, field := range score.FieldAccuracies {
		if field.Accuracy != 1.0 {
			t.Fatalf("field %s accuracy=%v, want 1.0", field.Field, field.Accuracy)
		}
	}
}

// A recoverer that always returns nothing must drive every metric to zero,
// never leave it at some default that could be mistaken for partial success.
func TestScoreDatasetWithTotalFailureIsZero(t *testing.T) {
	dataset, err := Generate("score-fail", 33, 30)
	if err != nil {
		t.Fatal(err)
	}
	score := ScoreDataset(dataset, func(Item) Fields { return Fields{} })
	if score.ExactMatchRate != 0 {
		t.Fatalf("exact_match_rate=%v, want 0", score.ExactMatchRate)
	}
	if score.ExactMatchCount != 0 {
		t.Fatalf("exact_match_count=%d, want 0", score.ExactMatchCount)
	}
}

// This is the exponential-decay claim from the depth-vs-width discussion made
// concrete: on a critical path, two independently-imperfect stages compound
// multiplicatively (P(both correct) = p1*p2), rather than averaging. It
// documents the mechanism the topology metrics (E, D, W) are meant to predict:
// accuracy decays with chain depth, not with node count. Intent and date are
// scored independently and left decoupled from route derivation here, so the
// measured compounding reflects only chain depth, not ResolveRoute's business
// rules (route depends on date only for the PAGAR intent).
func TestChainedFieldErrorsCompoundMultiplicatively(t *testing.T) {
	dataset, err := Generate("chain", 55, 2000)
	if err != nil {
		t.Fatal(err)
	}
	const perStageAccuracy = 0.9
	stageFailed := func(item Item, stage int) bool {
		var seed uint64
		for _, character := range item.ID {
			seed = seed*131 + uint64(character)
		}
		sequence := deterministicSequence{state: seed + uint64(stage)*0x1000}
		return sequence.pick(100) < int((1-perStageAccuracy)*100)
	}
	score := ScoreDataset(dataset, func(item Item) Fields {
		got := item.Expected // organization, amount and route recovered perfectly
		if stageFailed(item, 0) {
			got.Intent = "CORRUPTED"
		}
		if stageFailed(item, 1) {
			got.DateISO = "1900-01-01"
		}
		return got
	})
	expected := perStageAccuracy * perStageAccuracy
	if diff := score.ExactMatchRate - expected; diff > 0.05 || diff < -0.05 {
		t.Fatalf("exact_match_rate=%v, want close to the compounded rate %v", score.ExactMatchRate, expected)
	}
	// The compounded rate must be visibly worse than either single stage in
	// isolation: this is the depth penalty, not a rounding artefact.
	if score.ExactMatchRate >= perStageAccuracy {
		t.Fatalf("exact_match_rate=%v did not degrade below the single-stage rate %v", score.ExactMatchRate, perStageAccuracy)
	}
}
