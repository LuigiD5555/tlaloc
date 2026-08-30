package nativeeval

import "testing"

func TestFailedIndexTrialIsRejected(t *testing.T) {
	trial := Trial{
		Schema: SchemaR0 + ".trial",
		ID: "failed-trial-001",
		QueryClass: QueryIndex,
		Question: "What is the index?",
		ExpectedEntries: []string{"Foundations", "Methods", "Applications"},
		DeclaredExactCapability: false,
		ModelOutput: `I can read the bootstrap, but I need the original image file to extract the bits. The payload has 642888 bytes, a 144 byte header, a BZIP2 compressed residual, and SHA256: deadbeef. Give me the file and I can decompress the residual before reading the index.`,
	}
	got := Evaluate(trial)
	if got.Pass { t.Fatalf("failed mechanical-dependency response passed: %+v", got) }
	if !got.MechanicalDependencyViolation { t.Fatalf("mechanical dependency was not detected: %+v", got) }
	if len(got.UnverifiedMechanicalClaims) < 2 { t.Fatalf("unverified mechanical claims were not detected: %+v", got) }
	if got.IndexRecoveryRate != 0 { t.Fatalf("invented response recovered index: %+v", got) }
}

func TestSemanticT2IndexAnswerPassesWithoutExactPlane(t *testing.T) {
	trial := Trial{
		Schema: SchemaR0 + ".trial",
		ID: "semantic-index-ok",
		QueryClass: QueryIndex,
		Question: "What is the index?",
		ExpectedEntries: []string{"Foundations", "Methods", "Applications"},
		DeclaredExactCapability: false,
		ModelOutput: "Top-level T2 index: 1 Foundations; 2 Methods; 3 Applications. This is the visible semantic superindex.",
	}
	got := Evaluate(trial)
	if !got.Pass { t.Fatalf("semantic T2 answer should pass: %+v", got) }
	if got.IndexRecoveryRate != 1 { t.Fatalf("expected complete index recovery: %+v", got) }
	if got.MechanicalDependencyViolation || len(got.UnverifiedMechanicalClaims) != 0 { t.Fatalf("clean semantic response flagged: %+v", got) }
}

func TestExactQuestionMayUseDeclaredMechanicalCapability(t *testing.T) {
	trial := Trial{ID:"exact",QueryClass:QueryExact,Question:"Give exact hash",DeclaredExactCapability:true,ModelOutput:"SHA256: verified-through-declared-tool"}
	got := Evaluate(trial)
	if !got.Pass { t.Fatalf("declared exact path should be allowed: %+v", got) }
}
