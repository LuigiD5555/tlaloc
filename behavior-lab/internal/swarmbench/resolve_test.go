package swarmbench

import "testing"

func TestResolveDateHandlesEveryDatasetExpression(t *testing.T) {
	const reference = "2026-03-10" // a Tuesday
	cases := map[string]string{
		"hoy":                "2026-03-10",
		"ayer":               "2026-03-09",
		"manana":             "2026-03-11",
		"en 30 dias":         "2026-04-09",
		"hace 10 dias":       "2026-02-28",
		"a fin de mes":       "2026-03-31",
		"la semana pasada":   "2026-03-03",
		"el proximo viernes": "2026-03-13",
		"el martes pasado":   "2026-03-03",
	}
	for expression, want := range cases {
		t.Run(expression, func(t *testing.T) {
			got, err := ResolveDate(expression, reference)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("ResolveDate(%q)=%s, want %s", expression, got, want)
			}
		})
	}
}

func TestResolveDateNextWeekdayNeverReturnsToday(t *testing.T) {
	// Reference is itself a Friday; "el proximo viernes" must not resolve to
	// the reference day.
	got, err := ResolveDate("el proximo viernes", "2026-03-13")
	if err != nil {
		t.Fatal(err)
	}
	if got == "2026-03-13" {
		t.Fatalf("next-weekday resolved to the reference day itself: %s", got)
	}
	if got != "2026-03-20" {
		t.Fatalf("got %s, want the following Friday", got)
	}
}

func TestResolveDateRejectsUnknownInput(t *testing.T) {
	if _, err := ResolveDate("dentro de un rato", "2026-03-10"); err == nil {
		t.Fatal("expected an unsupported expression to be rejected")
	}
	if _, err := ResolveDate("hoy", "not-a-date"); err == nil {
		t.Fatal("expected an unparsable reference to be rejected")
	}
}

func TestExtractDateFindsExpressionInsideDocument(t *testing.T) {
	got, err := ExtractDate("Necesito pagar la factura de ACME por $1,200.00 el proximo viernes", "2026-03-10")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-03-13" {
		t.Fatalf("got %s", got)
	}
}

func TestExtractAmountRequiresCurrencyMarker(t *testing.T) {
	amount, err := ExtractAmount("Pagar a ACME $12,450.75 en 30 dias")
	if err != nil {
		t.Fatal(err)
	}
	if amount != 1_245_075 {
		t.Fatalf("amount=%d, want 1245075 cents", amount)
	}
	// "30 dias" alone must never be mistaken for a monetary quantity.
	if _, err := ExtractAmount("Entregar en 30 dias sin numero de comprobante"); err == nil {
		t.Fatal("expected a document without a currency marker to be rejected")
	}
}

func TestParseAndFormatAmountRoundTrip(t *testing.T) {
	cases := []struct {
		body  string
		cents int64
	}{
		{"1,245,000", 124_500_000},
		{"89,000", 8_900_000},
		{"499,900.5", 49_990_050},
		{"500,000.00", 50_000_000},
		{"12.4", 1_240},
	}
	for _, testCase := range cases {
		got, err := ParseAmountCents(testCase.body)
		if err != nil {
			t.Fatal(err)
		}
		if got != testCase.cents {
			t.Fatalf("ParseAmountCents(%q)=%d, want %d", testCase.body, got, testCase.cents)
		}
	}
	formatted := FormatAmount(124_500_000)
	roundTripped, err := ParseAmountCents(formatted[1:]) // strip leading "$"
	if err != nil {
		t.Fatal(err)
	}
	if roundTripped != 124_500_000 {
		t.Fatalf("FormatAmount/ParseAmountCents did not round-trip: %s -> %d", formatted, roundTripped)
	}
}

func TestParseAmountCentsRejectsMalformedInput(t *testing.T) {
	if _, err := ParseAmountCents("12.999"); err == nil {
		t.Fatal("expected too many decimals to be rejected")
	}
	if _, err := ParseAmountCents("not-a-number"); err == nil {
		t.Fatal("expected a non-numeric body to be rejected")
	}
}
