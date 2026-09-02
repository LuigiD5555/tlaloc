package realcampaign

import "testing"

func TestDetectModelFamily(t *testing.T) {
	cases := map[string]string{
		"lfm2-vl-1.6b":     ModelFamilyLiquidAI,
		"LiquidAI/LFM2-VL": ModelFamilyLiquidAI,
		"deepseek-vl2":     ModelFamilyDeepSeek,
		"Qwen2.5-VL-7B":    ModelFamilyQwen,
		"other-model":      ModelFamilyUnknown,
	}
	for id, want := range cases {
		if got := DetectModelFamily(id); got != want {
			t.Fatalf("DetectModelFamily(%q)=%q want %q", id, got, want)
		}
	}
}

func TestInteropProfileSeparatesTransport(t *testing.T) {
	direct := BuildModelInteropProfile("deepseek-vl2", "openai", TransportDirectImageAPI)
	platform := BuildModelInteropProfile("deepseek-vl2", "openai", TransportPlatformMediated)
	if direct.SpecimenKey == platform.SpecimenKey {
		t.Fatal("transport conditions must produce distinct specimen keys")
	}
	if direct.Family != ModelFamilyDeepSeek || platform.Family != ModelFamilyDeepSeek {
		t.Fatal("family should remain DeepSeek across transport conditions")
	}
}

func TestInteropProfileSeparatesExactModels(t *testing.T) {
	a := BuildModelInteropProfile("qwen2.5-vl-3b", "openai", TransportDirectImageAPI)
	b := BuildModelInteropProfile("qwen2.5-vl-7b", "openai", TransportDirectImageAPI)
	if a.SpecimenKey == b.SpecimenKey {
		t.Fatal("different exact model ids must remain distinct specimens")
	}
}

func TestNormalizeTransportConditionDefaultsDirectForCompatAPI(t *testing.T) {
	if got := NormalizeTransportCondition("", "lm-studio"); got != TransportDirectImageAPI {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeTransportCondition("web", "openai"); got != TransportPlatformMediated {
		t.Fatalf("got %q", got)
	}
}
