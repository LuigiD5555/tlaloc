package groundingautomaton

import "testing"

func verdict(t *testing.T, name, answer, evidence string, want Verdict) {
	t.Helper()
	got := Verify(VerifyInput{ModelAnswer: answer, PageContent: evidence})
	if got.Verdict != want {
		t.Errorf("%s: got %s, want %s (%+v)", name, got.Verdict, want, got)
	}
}

// R1: a lemmatised paraphrase clears the support threshold that a
// surface-form comparison would miss.
func TestR1_ParaphraseSupported(t *testing.T) {
	verdict(t, "paraphrase support",
		"Losing a few agents rarely changes what the group achieves.",
		"Robustness comes from redundancy, since the loss of individual agents rarely alters the collective outcome of the swarm.",
		VerdictSupported)
}

// R1: a contradiction the claim carries against a SHORT evidence sentence is
// still caught even though the whole-claim lexical coverage is below the
// candidate threshold (strong-signal bypass), and it wins the tie-break
// against a longer, non-conflicting sentence.
func TestR1_MultiClaimContradictionBelowThreshold(t *testing.T) {
	verdict(t, "multi-claim, claim 2 conflicts a short sentence",
		"The swarm has 12 nodes. Coordination is centralised through a leader node.",
		"The swarm has 12 nodes distributed across two racks. Coordination is decentralised; there is no leader node.",
		VerdictContradicted)
	verdict(t, "long claim vs short negated sentence",
		"A thinking swarm is a group of autonomous agents. The system requires a central scheduler to assign every task.",
		"A thinking swarm is a group of autonomous agents whose local interactions produce global behaviour. There is no central scheduler; tasks are claimed opportunistically.",
		VerdictContradicted)
}

// R1: the pre-existing R0 failure — claim 2 aligns to the wrong sentence and
// misses the antonym — is fixed by the contradiction-signal tie-break.
func TestR1_ContradictionPriorityAcrossClaims_Fixed(t *testing.T) {
	verdict(t, "disabled vs enabled across claims",
		"The model has 27 million parameters. The model is disabled.",
		"The model has 27 million parameters. The model is enabled.",
		VerdictContradicted)
}

// R1: a numeric contradiction where the evidence sentence carries an extra,
// unrelated number (which defeated R0's equal-cardinality rule) fires via
// shared adjacent context ("epochs").
func TestR1_NumericContradictionWithBuriedNumber(t *testing.T) {
	verdict(t, "40 vs 20 epochs, evidence also has an accuracy figure",
		"The head was trained for roughly 40 epochs on the synthetic corpus.",
		"It was trained for 20 epochs on the synthetic corpus and reached 0.89 accuracy on the held-out set.",
		VerdictContradicted)
}

// R1: a claim stating a figure the evidence never states cannot be SUPPORTED
// even at high lexical alignment.
func TestR1_ClaimNumberBlocksSupport(t *testing.T) {
	got := Verify(VerifyInput{
		ModelAnswer: "The system was tested on 500 documents.",
		PageContent: "The system was tested on 300 documents from the corpus.",
	})
	if got.Verdict == VerdictSupported {
		t.Errorf("a claim number absent from the evidence must not be SUPPORTED, got %+v", got)
	}
}

// R1: a bare keyword dump is INSUFFICIENT, not scored claim-by-claim.
func TestR1_KeywordDumpIsInsufficient(t *testing.T) {
	verdict(t, "space-joined term list",
		"swarm agents autonomous local global emergent decentralised robust redundancy interactions",
		"A swarm is a collection of autonomous agents whose local interactions produce global behaviour.",
		VerdictInsufficient)
	verdict(t, "comma-joined term list",
		"consensus, quorum, majority, acknowledge, proposal, commit, latency, round trip",
		"Quorum-based consensus requires a majority to acknowledge a proposal before it is committed.",
		VerdictInsufficient)
}

// R1: quantifier scale now includes frequency adverbs.
func TestR1_FrequencyQuantifierContradiction(t *testing.T) {
	verdict(t, "usually vs rarely",
		"Losing even one agent usually breaks the whole swarm.",
		"The loss of individual agents rarely alters the collective outcome.",
		VerdictContradicted)
}

// R1 SAFETY: the tie-break and below-threshold bypass must never produce a
// SUPPORTED on a real contradiction. This mirrors the eval gate.
func TestR1_NoFalseSupportedOnContradiction(t *testing.T) {
	cases := [][2]string{
		{"The swarm uses a central controller.", "The swarm has no central controller."},
		{"Replication is synchronous.", "Replication between regions is asynchronous."},
		{"The retry count is at least five.", "The retry count is capped at five, i.e. at most five."},
		{"Coverage was 90%.", "Deterministic coverage on that corpus was 63%."},
	}
	for _, pair := range cases {
		got := Verify(VerifyInput{ModelAnswer: pair[0], PageContent: pair[1]})
		if got.Verdict == VerdictSupported {
			t.Errorf("false SUPPORTED on contradiction: %q vs %q -> %+v", pair[0], pair[1], got)
		}
	}
}

// R1: lemma groups must never merge an antonym pair.
func TestR1_LemmaDoesNotMergeAntonyms(t *testing.T) {
	for _, pair := range antonymPairs {
		if lemma(stem(pair[0])) == lemma(stem(pair[1])) && lemma(stem(pair[0])) != stem(pair[0]) {
			t.Errorf("lemma merges antonym pair %q/%q into %q", pair[0], pair[1], lemma(stem(pair[0])))
		}
	}
}
