package swarmbench

// FieldScore is one field's outcome for one item. Field-level accuracy is
// what lets an error be attributed to the specific individual that produced
// it, instead of only to the final route.
type FieldScore struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Got      string `json:"got"`
	Correct  bool   `json:"correct"`
}

// ItemScore is one document's outcome across every recovered field.
type ItemScore struct {
	ItemID       string       `json:"item_id"`
	Fields       []FieldScore `json:"fields"`
	ExactMatch   bool         `json:"exact_match"`
	RouteCorrect bool         `json:"route_correct"`
}

// FieldAccuracy is one field's accuracy across the whole dataset.
type FieldAccuracy struct {
	Field    string  `json:"field"`
	Correct  int     `json:"correct"`
	Total    int     `json:"total"`
	Accuracy float64 `json:"accuracy"`
}

// Score is the aggregate outcome of one swarm topology against one dataset.
type Score struct {
	Schema          string          `json:"schema"`
	ItemCount       int             `json:"item_count"`
	ExactMatchCount int             `json:"exact_match_count"`
	ExactMatchRate  float64         `json:"exact_match_rate"`
	RouteAccuracy   float64         `json:"route_accuracy"`
	FieldAccuracies []FieldAccuracy `json:"field_accuracies"`
	Items           []ItemScore     `json:"items"`
}

// ScoreItem compares one recovered Fields against ground truth. A field left
// empty by the caller (because its producing node failed or was never
// scheduled) scores as incorrect rather than being silently skipped, so a
// partial swarm failure is never invisible to the accuracy metric.
func ScoreItem(item Item, got Fields) ItemScore {
	fields := []FieldScore{
		{Field: "intent", Expected: item.Expected.Intent, Got: got.Intent, Correct: got.Intent == item.Expected.Intent},
		{Field: "organization", Expected: item.Expected.Organization, Got: got.Organization, Correct: got.Organization == item.Expected.Organization},
		scoreAmount(item.Expected.AmountCents, got.AmountCents),
		{Field: "date", Expected: item.Expected.DateISO, Got: got.DateISO, Correct: got.DateISO == item.Expected.DateISO},
		{Field: "route", Expected: item.Expected.Route, Got: got.Route, Correct: got.Route == item.Expected.Route},
	}
	exact := true
	routeCorrect := false
	for _, field := range fields {
		if !field.Correct {
			exact = false
		}
		if field.Field == "route" {
			routeCorrect = field.Correct
		}
	}
	return ItemScore{ItemID: item.ID, Fields: fields, ExactMatch: exact, RouteCorrect: routeCorrect}
}

func scoreAmount(expected, got int64) FieldScore {
	return FieldScore{
		Field:    "amount",
		Expected: FormatAmount(expected),
		Got:      FormatAmount(got),
		Correct:  expected == got,
	}
}

// ScoreDataset scores every item and aggregates per-field and route accuracy.
// recover is called once per item and must return the fields the swarm under
// test recovered for it; a missing or errored field should come back as its
// zero value so it scores as incorrect rather than as absent.
func ScoreDataset(dataset Dataset, recover func(Item) Fields) Score {
	score := Score{Schema: ScoreSchemaR0, ItemCount: len(dataset.Items)}
	fieldTotals := map[string]*FieldAccuracy{}
	fieldOrder := []string{}

	for _, item := range dataset.Items {
		got := recover(item)
		itemScore := ScoreItem(item, got)
		score.Items = append(score.Items, itemScore)
		if itemScore.ExactMatch {
			score.ExactMatchCount++
		}
		for _, field := range itemScore.Fields {
			accumulator, exists := fieldTotals[field.Field]
			if !exists {
				accumulator = &FieldAccuracy{Field: field.Field}
				fieldTotals[field.Field] = accumulator
				fieldOrder = append(fieldOrder, field.Field)
			}
			accumulator.Total++
			if field.Correct {
				accumulator.Correct++
			}
		}
	}

	if score.ItemCount > 0 {
		score.ExactMatchRate = float64(score.ExactMatchCount) / float64(score.ItemCount)
	}
	if accumulator, ok := fieldTotals["route"]; ok && accumulator.Total > 0 {
		score.RouteAccuracy = float64(accumulator.Correct) / float64(accumulator.Total)
	}
	for _, field := range fieldOrder {
		accumulator := fieldTotals[field]
		if accumulator.Total > 0 {
			accumulator.Accuracy = float64(accumulator.Correct) / float64(accumulator.Total)
		}
		score.FieldAccuracies = append(score.FieldAccuracies, *accumulator)
	}
	return score
}
