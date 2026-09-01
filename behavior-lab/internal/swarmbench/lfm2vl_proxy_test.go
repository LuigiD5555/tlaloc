package swarmbench

import (
	"context"
	"testing"
)

// The proxy is a faithful stand-in for the real probe only if it reproduces
// what the probe log actually recorded — not a plausible-looking invention.
// The probe (runs/lm-studio/lfm2-vl-1.6b-probe-001.json) measured intent
// accuracy 10/40=0.25 by collapsing to CONSULTAR; over a large balanced
// dataset the constant-CONSULTAR proxy should land at the same rate.
func TestIntentProxyMatchesObservedCollapseRate(t *testing.T) {
	dataset, err := Generate("proxy-calibration", 909, 4000)
	if err != nil {
		t.Fatal(err)
	}
	correct := 0
	for _, item := range dataset.Items {
		got, _ := IntentWorkerLogicLFM2VLProxy(item.Text)
		if got != IntentQuery {
			t.Fatalf("proxy returned %q, want the constant collapse label %q", got, IntentQuery)
		}
		if got == item.Expected.Intent {
			correct++
		}
	}
	rate := float64(correct) / float64(len(dataset.Items))
	if diff := rate - 0.25; diff > 0.03 || diff < -0.03 {
		t.Fatalf("proxy intent accuracy=%v, want close to the observed 0.25 (dataset is uniform across 4 intents)", rate)
	}
}

// The real model hijacked the literal substring "CFDI" for the organization
// on the CANCELAR template that contains it — this must reproduce exactly,
// since it is a deterministic, generalizable failure, not a probabilistic one.
func TestEntityProxyHijacksCFDILiteral(t *testing.T) {
	text := "Solicito cancelar el CFDI de ACME Servicios por $500.00 emitido hoy"
	got, confidence := EntityWorkerLogicLFM2VLProxy(text)
	if got != "CFDI" {
		t.Fatalf("got %q, want the observed CFDI hijack", got)
	}
	if confidence <= 0 {
		t.Fatal("hijacked answer must still carry a confidence, matching a real wrong-but-confident model response")
	}
}

// Outside the CFDI-bearing template, the proxy must fall back to a real
// gazetteer read most of the time — the failure was specific to that phrase,
// not a general inability to read organization names.
func TestEntityProxyFallsBackToGazetteerOutsideCFDIContext(t *testing.T) {
	text := "Necesito pagar la factura de ACME Servicios por $500.00 hoy"
	got, _ := EntityWorkerLogicLFM2VLProxy(text)
	if got != "ACME Servicios" && got != "Servicios" {
		t.Fatalf("got %q, want the real organization or its calibrated truncation", got)
	}
}

func TestEntityProxyReturnsEmptyWithoutGazetteerHit(t *testing.T) {
	got, confidence := EntityWorkerLogicLFM2VLProxy("Documento sin ninguna organizacion reconocible")
	if got != "" || confidence != 0 {
		t.Fatalf("got=%q confidence=%v, want empty/zero", got, confidence)
	}
}

// This is the actual question the experiment exists to answer: does
// composing router + verifier on top of a genuinely broken intent stage
// improve anything? Route depends on intent; a verifier that only re-derives
// route from the fields it is handed cannot recover an intent the upstream
// classifier never produced. This test measures — not assumes — how much
// composition can and cannot rescue.
func TestSwarmCompositionCannotRescueACollapsedIntentStage(t *testing.T) {
	registry, err := BuildInProcessRegistryWithLogic(1_600_000_000, 18_000_000, IntentWorkerLogicLFM2VLProxy, nil)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("composition-rescue", 4242, 2000)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("composition-rescue", 4)

	run, err := Execute(context.Background(), registry, plan, dataset, FanInTerminalNode)
	if err != nil {
		t.Fatal(err)
	}

	// With intent collapsed to CONSULTAR, route always resolves to ARCHIVE
	// (CONSULTAR and BUSCAR both map there) — route accuracy is bounded by
	// the fraction of the dataset whose true route is ARCHIVE, not by
	// anything the router or verifier contribute.
	archiveShare := 0.0
	for _, item := range dataset.Items {
		if item.Expected.Route == RouteArchive {
			archiveShare++
		}
	}
	archiveShare /= float64(len(dataset.Items))

	if diff := run.Score.RouteAccuracy - archiveShare; diff > 0.02 || diff < -0.02 {
		t.Fatalf("route_accuracy=%v, want it capped at the ARCHIVE base rate %v — composition should not be able to exceed what a collapsed intent stage allows", run.Score.RouteAccuracy, archiveShare)
	}
	// The verifier must not silently mask this: it only catches ROUTE
	// arithmetic mistakes given the (wrong) intent, it has no way to
	// re-derive the true intent, so it cannot lift the ceiling above.
	if run.Score.RouteAccuracy >= 0.9 {
		t.Fatalf("route_accuracy=%v is implausibly high given a collapsed intent stage; the verifier should not be able to fully mask an upstream classifier failure", run.Score.RouteAccuracy)
	}
}

// The same swarm topology, same everything, but with the exhaustive lexicon
// instead of the calibrated real-model proxy — this is the control that
// makes the previous test's finding attributable to the intent stage
// specifically, not to some other change.
func TestSwarmCompositionRecoversFullyWithTheExhaustiveLexicon(t *testing.T) {
	registry, err := BuildInProcessRegistry(12_000_000, 18_000_000)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := Generate("composition-control", 4242, 2000)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildFanInPlan("composition-control", 4)
	run, err := Execute(context.Background(), registry, plan, dataset, FanInTerminalNode)
	if err != nil {
		t.Fatal(err)
	}
	if run.Score.RouteAccuracy != 1.0 {
		t.Fatalf("route_accuracy=%v, want 1.0 with a working intent stage — isolates the collapse test's finding to the intent classifier", run.Score.RouteAccuracy)
	}
}
