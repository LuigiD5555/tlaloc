package promotion

import "testing"

func TestParseObservationJSONFence(t *testing.T) {
	content := "```json\n{\"boot_text\":[\"ORIGAMI FIXED CARRIER R2\"],\"probe_top\":\"10101010\",\"probe_bottom\":\"10101010\",\"tool_protocol\":\"tlaloc.origami-tools.r2\",\"address_abi\":\"ohf-address.r2\",\"t3\":null}\n```"
	obs, err := ParseObservation(content)
	if err != nil {
		t.Fatal(err)
	}
	if obs.ProbeTop != "10101010" || obs.ToolProtocol != "tlaloc.origami-tools.r2" || obs.T3 != nil {
		t.Fatalf("unexpected observation %+v", obs)
	}
}

func TestParseObservationRejectsExtraGroundTruthFields(t *testing.T) {
	_, err := ParseObservation(`{"boot_text":[],"probe_top":"","probe_bottom":"","tool_protocol":"","address_abi":"","t3":null,"expected_probe":"10101010"}`)
	if err == nil {
		t.Fatal("private/extra ground-truth-like field should be rejected")
	}
}

func TestObservationQuestionDoesNotContainExpectedProbeOrCarrierTruth(t *testing.T) {
	q := ObservationQuestion(true)
	for _, forbidden := range []string{"10101010", "store root is", "expected probe"} {
		if containsFold(q, forbidden) {
			t.Fatalf("question leaked ground truth: %s", forbidden)
		}
	}
}

func containsFold(a, b string) bool {
	if len(b) > len(a) {
		return false
	}
	for i := 0; i+len(b) <= len(a); i++ {
		same := true
		for j := range b {
			ca := a[i+j]
			cb := b[j]
			if ca >= 'A' && ca <= 'Z' {
				ca += 32
			}
			if cb >= 'A' && cb <= 'Z' {
				cb += 32
			}
			if ca != cb {
				same = false
				break
			}
		}
		if same {
			return true
		}
	}
	return false
}
