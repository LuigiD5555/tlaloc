package groundingautomaton

import "testing"

func TestHasAuthorityOnlyForClosedVerdicts(t *testing.T) {
	if !HasAuthority(VerifyOutput{Verdict: VerdictSupported}) || !HasAuthority(VerifyOutput{Verdict: VerdictContradicted}) {
		t.Fatal("supported and contradicted must have deterministic authority")
	}
	if HasAuthority(VerifyOutput{Verdict: VerdictUnknown}) || HasAuthority(VerifyOutput{Verdict: VerdictInsufficient}) {
		t.Fatal("unknown and insufficient must defer to fallback")
	}
}
