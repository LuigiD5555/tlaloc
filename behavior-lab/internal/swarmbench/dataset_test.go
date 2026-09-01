package swarmbench

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The measurement instrument's own source of truth must rebuild
// byte-identically, or the scaling sweep is not reproducible.
func TestGenerateIsDeterministic(t *testing.T) {
	first, err := Generate("bench", 42, 200)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate("bench", 42, 200)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstJSON, secondJSON) {
		t.Fatal("Generate is not deterministic across identical seed and count")
	}
}

func TestGenerateVariesWithSeed(t *testing.T) {
	first, err := Generate("bench", 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate("bench", 2, 100)
	if err != nil {
		t.Fatal(err)
	}
	if first.Items[0].Text == second.Items[0].Text && first.Items[0].Expected == second.Items[0].Expected {
		t.Fatal("two different seeds produced an identical first item")
	}
}

func TestGeneratedDatasetIsInternallyConsistent(t *testing.T) {
	dataset, err := Generate("bench", 7, 500)
	if err != nil {
		t.Fatal(err)
	}
	if err := dataset.Validate(); err != nil {
		t.Fatal(err)
	}
}

// Every intent and every side of the payment threshold must actually appear,
// or the dataset cannot exercise the routing logic it claims to test.
func TestGeneratedDatasetCoversEveryIntentAndThreshold(t *testing.T) {
	dataset, err := Generate("bench", 99, 500)
	if err != nil {
		t.Fatal(err)
	}
	seenIntents := map[string]bool{}
	seenRoutes := map[string]bool{}
	aboveThreshold, belowThreshold := false, false
	for _, item := range dataset.Items {
		seenIntents[item.Expected.Intent] = true
		seenRoutes[item.Expected.Route] = true
		if item.Expected.AmountCents >= TreasuryThresholdCents {
			aboveThreshold = true
		} else {
			belowThreshold = true
		}
	}
	for _, intent := range []string{IntentPay, IntentCancel, IntentQuery, IntentSearch} {
		if !seenIntents[intent] {
			t.Fatalf("dataset never generated intent %s", intent)
		}
	}
	for _, route := range []string{RouteTreasury, RouteCompliance, RouteArchive, RouteSupport} {
		if !seenRoutes[route] {
			t.Fatalf("dataset never generated route %s", route)
		}
	}
	if !aboveThreshold || !belowThreshold {
		t.Fatal("dataset does not straddle the treasury threshold")
	}
}

func TestGenerateRejectsInvalidInput(t *testing.T) {
	if _, err := Generate("", 1, 10); err == nil {
		t.Fatal("expected an empty dataset id to be rejected")
	}
	if _, err := Generate("bench", 1, 0); err == nil {
		t.Fatal("expected a zero item count to be rejected")
	}
}

func TestValidateRejectsTamperedDataset(t *testing.T) {
	dataset, err := Generate("bench", 3, 20)
	if err != nil {
		t.Fatal(err)
	}
	tampered := dataset
	tampered.Items = append([]Item(nil), dataset.Items...)
	tampered.Items[0].Expected.Route = "SOMEWHERE_ELSE"
	if err := tampered.Validate(); err == nil {
		t.Fatal("expected a tampered ground truth route to be rejected")
	}
}

func TestValidateRejectsDuplicateItemIDs(t *testing.T) {
	dataset, err := Generate("bench", 5, 10)
	if err != nil {
		t.Fatal(err)
	}
	dataset.Items = append(dataset.Items, dataset.Items[0])
	if err := dataset.Validate(); err == nil {
		t.Fatal("expected a duplicate item id to be rejected")
	}
}

func TestValidateRejectsWrongSchemaAndEmptyDataset(t *testing.T) {
	if err := (Dataset{Schema: "something.else", Items: []Item{{}}}).Validate(); err == nil {
		t.Fatal("expected an unexpected schema to be rejected")
	}
	if err := (Dataset{Schema: DatasetSchemaR0}).Validate(); err == nil {
		t.Fatal("expected an empty dataset to be rejected")
	}
}
