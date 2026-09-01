package swarmbench

import "testing"

func TestIntentWorkerLogicRecognizesEveryDatasetIntent(t *testing.T) {
	cases := map[string]string{
		"Necesito pagar la factura de ACME por $1,200.00 hoy":                IntentPay,
		"Favor de programar el pago a ACME de $500.00 para manana":           IntentPay,
		"Solicito cancelar el CFDI de ACME por $500.00 emitido hoy":          IntentCancel,
		"Hay que cancelar el comprobante de ACME de $500.00 con fecha hoy":   IntentCancel,
		"Pueden consultar el saldo de ACME, son $500.00 con vencimiento hoy": IntentQuery,
		"Consulta el estado de cuenta de ACME por $500.00 al corte hoy":      IntentQuery,
		"Busca los comprobantes de ACME por $500.00 de hoy":                  IntentSearch,
		"Localiza la documentacion de ACME de $500.00 emitida hoy":           IntentSearch,
	}
	for text, want := range cases {
		got, confidence := IntentWorkerLogic(text)
		if got != want {
			t.Errorf("IntentWorkerLogic(%q)=%s, want %s", text, got, want)
		}
		if confidence <= 0 {
			t.Errorf("IntentWorkerLogic(%q) returned zero confidence for a matched cue", text)
		}
	}
}

func TestIntentWorkerLogicReturnsZeroConfidenceOnNoMatch(t *testing.T) {
	intent, confidence := IntentWorkerLogic("Documento sin ningun verbo reconocible")
	if intent != "" || confidence != 0 {
		t.Fatalf("intent=%q confidence=%v, want empty/zero on no match", intent, confidence)
	}
}

func TestEntityWorkerLogicMatchesGazetteer(t *testing.T) {
	for _, organization := range Organizations {
		text := "Necesito pagar la factura de " + organization + " por $500.00 hoy"
		got, confidence := EntityWorkerLogic(text)
		if got != organization {
			t.Fatalf("EntityWorkerLogic did not recover %q from %q, got %q", organization, text, got)
		}
		if confidence <= 0 {
			t.Fatalf("EntityWorkerLogic(%q) returned zero confidence on a real match", text)
		}
	}
}

// A substring gazetteer risks matching the wrong candidate when one
// organization name is a prefix of another; EntityWorkerLogic must prefer
// the longest match actually present in the text.
func TestEntityWorkerLogicPrefersLongestMatch(t *testing.T) {
	got, _ := EntityWorkerLogic("Pago a Consultoria Zapopan por servicios")
	if got != "Consultoria Zapopan" {
		t.Fatalf("got %q", got)
	}
}

func TestEntityWorkerLogicReturnsZeroConfidenceWithoutGazetteerHit(t *testing.T) {
	organization, confidence := EntityWorkerLogic("Un documento sin ninguna organizacion conocida")
	if organization != "" || confidence != 0 {
		t.Fatalf("organization=%q confidence=%v, want empty/zero", organization, confidence)
	}
}

func TestDateNumberWorkerLogicRecoversBothFields(t *testing.T) {
	dateISO, amountCents, err := DateNumberWorkerLogic("Pagar a ACME $12,450.75 en 30 dias", "2026-03-10")
	if err != nil {
		t.Fatal(err)
	}
	if dateISO != "2026-04-09" {
		t.Fatalf("date=%s", dateISO)
	}
	if amountCents != 1_245_075 {
		t.Fatalf("amount=%d", amountCents)
	}
}

func TestDateNumberWorkerLogicFailsWithoutEitherField(t *testing.T) {
	if _, _, err := DateNumberWorkerLogic("Documento sin fecha ni importe reconocible", "2026-03-10"); err == nil {
		t.Fatal("expected an error when neither field is recoverable")
	}
}

func TestRouteWorkerLogicMatchesReferenceRoute(t *testing.T) {
	dataset, err := Generate("route-worker", 61, 300)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range dataset.Items {
		fields, err := RouteWorkerLogic(item.Expected.Intent, item.Expected.Organization, item.Expected.AmountCents, item.Expected.DateISO, item.ReferenceDate)
		if err != nil {
			t.Fatalf("item %s: %v", item.ID, err)
		}
		if fields != item.Expected {
			t.Fatalf("item %s: got %+v, want %+v", item.ID, fields, item.Expected)
		}
	}
}

// The verifier's entire purpose: catch a router-stage error and correct it
// without re-running intent or entity extraction.
func TestVerifyWorkerLogicCorrectsAWrongRoute(t *testing.T) {
	fields := Fields{Intent: IntentPay, Organization: "ACME", AmountCents: 1_000_000, DateISO: "2026-04-01", Route: RouteArchive}
	corrected, changed, err := VerifyWorkerLogic(fields, "2026-03-01")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected the verifier to flag a correction")
	}
	if corrected.Route != RouteTreasury {
		t.Fatalf("route=%s, want %s", corrected.Route, RouteTreasury)
	}
	// Every other field must pass through untouched.
	corrected.Route = fields.Route
	if corrected != fields {
		t.Fatalf("verifier altered a field other than route: %+v vs %+v", corrected, fields)
	}
}

func TestVerifyWorkerLogicLeavesACorrectRouteUnchanged(t *testing.T) {
	fields := Fields{Intent: IntentCancel, Organization: "ACME", AmountCents: 500_00, DateISO: "2026-03-01", Route: RouteCompliance}
	corrected, changed, err := VerifyWorkerLogic(fields, "2026-03-01")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("verifier flagged a change on an already-correct route")
	}
	if corrected != fields {
		t.Fatalf("got %+v, want unchanged %+v", corrected, fields)
	}
}

func TestVerifyWorkerLogicRejectsUnroutableFields(t *testing.T) {
	fields := Fields{Intent: "NOT_A_REAL_INTENT", Route: RouteArchive}
	if _, _, err := VerifyWorkerLogic(fields, "2026-03-01"); err == nil {
		t.Fatal("expected an error for fields the router logic cannot resolve")
	}
}
