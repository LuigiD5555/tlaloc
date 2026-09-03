package decompositionlab

import (
	"os"
	"reflect"
	"testing"

	"tlaloc.local/behaviorlab/internal/exocortex"
)

const (
	testP0Dir = "../../experiments/parrot-capability-r0"
	testP2A   = "../../experiments/parrot-microisa-r0.1/results/PARROT_MICRO_ISA_R0.json"
)

func skipIfNoFrozen(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(testP0Dir); err != nil {
		t.Skipf("frozen P0 experiment not present")
	}
	if _, err := os.Stat(testP2A); err != nil {
		t.Skipf("frozen P2-A artifact not present")
	}
}

func realProfile(t *testing.T) exocortex.CapabilityProfile {
	t.Helper()
	profile, err := exocortex.CompileParrotProfileReal(testP2A, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("compile real profile: %v", err)
	}
	return profile
}

func TestEligibilityAudit_IsDeterministic(t *testing.T) {
	skipIfNoFrozen(t)
	provRecords, _, err := LoadP0Provenance(testP0Dir)
	if err != nil {
		t.Fatal(err)
	}
	records, err := BuildT0BRecords(provRecords, testP0Dir)
	if err != nil {
		t.Fatal(err)
	}
	profile := realProfile(t)

	a1 := RunEligibilityAudit(records, profile, "sha")
	a2 := RunEligibilityAudit(records, profile, "sha")
	if !reflect.DeepEqual(a1, a2) {
		t.Fatalf("eligibility audit is not deterministic across two runs")
	}
}

func TestEligibilityAudit_CoverageDenominatorIsFullP0(t *testing.T) {
	skipIfNoFrozen(t)
	provRecords, _, _ := LoadP0Provenance(testP0Dir)
	records, _ := BuildT0BRecords(provRecords, testP0Dir)
	profile := realProfile(t)
	audit := RunEligibilityAudit(records, profile, "sha")

	if audit.TotalP0Cases != len(records) {
		t.Fatalf("total_p0_cases = %d, want %d", audit.TotalP0Cases, len(records))
	}
	if audit.EligibleR0+audit.NotApplicableR0 != audit.TotalP0Cases {
		t.Fatalf("eligible + not_applicable (%d) != total (%d)", audit.EligibleR0+audit.NotApplicableR0, audit.TotalP0Cases)
	}
	wantCoverage := float64(audit.EligibleR0) / float64(audit.TotalP0Cases)
	if audit.R0TaskCoverage != wantCoverage {
		t.Fatalf("coverage = %v, want eligible/total = %v", audit.R0TaskCoverage, wantCoverage)
	}
	// NOT_APPLICABLE_R0 cases must carry no runnable recipe.
	for _, c := range audit.Cases {
		if c.Eligibility == NotApplicableR0 && len(c.Recipe) != 0 {
			t.Fatalf("NOT_APPLICABLE_R0 case %s has a recipe (would leak into the eligible denominator)", c.BaseID)
		}
	}
}

func TestEligibilityAudit_TracksProfileNotP0Correctness(t *testing.T) {
	skipIfNoFrozen(t)
	provRecords, _, _ := LoadP0Provenance(testP0Dir)
	records, _ := BuildT0BRecords(provRecords, testP0Dir)

	// Externalize EXTRACT_NUMBER in the profile: numeric cases must flip to
	// NOT_APPLICABLE regardless of how P0 actually scored them.
	profile := realProfile(t)
	for i := range profile.Capabilities {
		if profile.Capabilities[i].Opcode == exocortex.OpExtractNumber {
			profile.Capabilities[i].DeploymentRecommendation = exocortex.DeploymentExternalize
		}
	}
	audit := RunEligibilityAudit(records, profile, "sha")
	for _, c := range audit.Cases {
		if c.Category == CategoryNumeric && c.Eligibility == EligibleR0 {
			t.Fatalf("numeric case %s still ELIGIBLE after EXTRACT_NUMBER externalized", c.BaseID)
		}
	}
}
