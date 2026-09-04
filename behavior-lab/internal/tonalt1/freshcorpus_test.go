package tonalt1

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
)

func mkOperand(id string, page int, region, value string, morph MorphologyFamily) Candidate {
	cand := Candidate{
		CandidateID:  id,
		Corpus:       CandidateCorpus{Page: page, PageWidth: 756, PageHeight: 1080},
		Identity:     PhysicalIdentity{Page: page, RegionID: region, NormalizedSpanHash: id + "-hash"},
		Source:       CandidateSource{NumericNormalized: value, NumericRaw: value, ContainingLineText: "the value is " + value + " here in prose"},
		Geometry:     CandidateGeometry{ContainingLineBBox: canonicaldoc.BBox{X1: 100, Y1: float64(page), X2: 400, Y2: float64(page) + 16}},
		Presentation: CandidatePresentation{MorphologyFamily: morph, R1EnvelopeSupported: true},
		Eligibility:  CandidateEligibility{Eligible: true},
	}
	return cand
}

// Deterministic page partition: same inputs -> identical, zero overlap,
// ~20% bridge.
func TestFresh_PartitionPagesDeterministic(t *testing.T) {
	pages := []int{}
	for p := 1; p <= 100; p++ {
		pages = append(pages, p)
	}
	a := PartitionPages("abc123", pages, 0.20)
	b := PartitionPages("abc123", pages, 0.20)
	if a.PartitionHash != b.PartitionHash {
		t.Fatal("partition not deterministic")
	}
	if len(a.BridgePages) != 20 {
		t.Errorf("bridge pages = %d, want 20", len(a.BridgePages))
	}
	if len(a.BridgePages)+len(a.PrimaryPages) != 100 {
		t.Error("partition does not cover all pages")
	}
	if !a.ZeroPageOverlap {
		t.Error("partition reports page overlap")
	}
	seen := map[int]bool{}
	for _, p := range append(append([]int{}, a.BridgePages...), a.PrimaryPages...) {
		if seen[p] {
			t.Fatalf("page %d in both sets", p)
		}
		seen[p] = true
	}
	// different corpus sha -> different split
	c := PartitionPages("xyz789", pages, 0.20)
	if a.PartitionHash == c.PartitionHash {
		t.Error("different corpus sha produced identical partition")
	}
}

// Primary universe excludes bridge pages and bridge physical instances.
func TestFresh_PrimaryUniverseNoBridgeLeakage(t *testing.T) {
	var cands []Candidate
	for p := 1; p <= 40; p++ {
		cands = append(cands, mkOperand("c"+itoa(p), p, "text-1", "512", MorphMultiDigitInteger))
	}
	scan := ScanResult{Candidates: cands}
	partition := PagePartition{BridgePages: []int{1, 2, 3, 4, 5}, PrimaryPages: nil}
	for p := 6; p <= 40; p++ {
		partition.PrimaryPages = append(partition.PrimaryPages, p)
	}
	bridge := BridgeSpec{Bases: []BridgeBase{{CandidateID: "c6", Page: 6}}} // simulate a stray bridge instance on a primary page
	universe := BuildPrimaryUniverse(scan, partition, bridge, []MorphologyFamily{MorphMultiDigitInteger})

	for _, cand := range universe.Operands {
		for _, bp := range partition.BridgePages {
			if cand.Corpus.Page == bp {
				t.Fatalf("bridge page %d leaked into primary", bp)
			}
		}
		if cand.CandidateID == "c6" {
			t.Fatal("bridge physical instance c6 leaked into primary")
		}
	}
	if universe.BridgeLeakage != 1 {
		t.Errorf("bridge leakage count = %d, want 1", universe.BridgeLeakage)
	}
	if universe.N != 34 { // 40 - 5 bridge pages - 1 leaked instance
		t.Errorf("primary N = %d, want 34", universe.N)
	}
}

// Allocation feasibility: a witness exists at healthy headroom, and the
// check fails when N < 144.
func TestFresh_AllocationFeasibility(t *testing.T) {
	// 300 operands across 150 pages (2 per page) -> feasible.
	var ops []Candidate
	for p := 1; p <= 150; p++ {
		ops = append(ops, mkOperand("a"+itoa(p), p, "text-1", "512", MorphMultiDigitInteger))
		ops = append(ops, mkOperand("b"+itoa(p), p, "text-2", "256", MorphMultiDigitInteger))
	}
	feasible := CheckAllocationFeasible(PrimaryUniverse{Operands: ops, N: len(ops), QualifiedMorphologies: []MorphologyFamily{MorphMultiDigitInteger}})
	if !feasible.AllocationFeasible {
		t.Fatalf("300 operands / 150 pages should be allocation-feasible: %+v", feasible.WitnessSummary)
	}
	if feasible.HeadroomRatio < 2.0 {
		t.Errorf("headroom = %.2f, want >= 2", feasible.HeadroomRatio)
	}

	// 100 operands -> infeasible (< 144).
	short := CheckAllocationFeasible(PrimaryUniverse{Operands: ops[:100], N: 100})
	if short.AllocationFeasible {
		t.Error("100 operands should be infeasible for 144 demand")
	}
}

// Fresh freeze: TONAL_T1_FRESH_CORPUS_FROZEN true only when all invariants
// hold, >=1 qualified morphology, N>=144, allocation feasible.
func TestFresh_FreezeGate(t *testing.T) {
	var ops []Candidate
	for p := 1; p <= 160; p++ {
		ops = append(ops, mkOperand("a"+itoa(p), p, "text-1", "512", MorphMultiDigitInteger))
	}
	universe := PrimaryUniverse{
		Operands: ops, N: 160, QualifiedMorphologies: []MorphologyFamily{MorphMultiDigitInteger},
		ByMorphology: map[MorphologyFamily]int{MorphMultiDigitInteger: 160}, DistinctPages: 160,
	}
	capacity := CheckAllocationFeasible(universe)
	man := FreshCorpusFreeze(SourceDoc{SourceSHA256: "fresh"}, StoreIdentity{}, ScanResult{Candidates: ops},
		PagePartition{FrozenBeforeInference: true, ZeroPageOverlap: true},
		BridgeSpec{}, []BridgeMorphologyResult{{Morphology: MorphMultiDigitInteger, Qualified: true}},
		universe, capacity)
	if !man.TONALT1FreshCorpusFrozen || !man.T1D4CanProceed {
		t.Fatalf("freeze gate should pass: invariants=%+v", man.HardInvariants)
	}

	// Break it: N below demand.
	small := universe
	small.Operands = ops[:100]
	small.N = 100
	cap2 := CheckAllocationFeasible(small)
	man2 := FreshCorpusFreeze(SourceDoc{}, StoreIdentity{}, ScanResult{}, PagePartition{FrozenBeforeInference: true, ZeroPageOverlap: true},
		BridgeSpec{}, nil, small, cap2)
	if man2.TONALT1FreshCorpusFrozen {
		t.Error("freeze gate must fail when N < 144")
	}
}
