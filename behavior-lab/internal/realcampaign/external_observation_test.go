package realcampaign

import (
	"encoding/json"
	"os"
	"testing"
)

func TestExternalObservationIsSeparateFromWorkingConfiguration(t *testing.T) {
	root := t.TempDir()
	obs, err := BuildExternalObservation(
		"deepseek-vl2",
		"platform",
		TransportPlatformMediated,
		"platform://deepseek-web",
		"MANUAL_PLATFORM_OBSERVATION",
		"PARTIAL",
		"visible boot recovered, exact payload unavailable",
		"platform may transform the image before model inference",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	path, err := RecordExternalObservation(root, obs)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var reg ExternalObservationRegistry
	if err := json.Unmarshal(body, &reg); err != nil {
		t.Fatal(err)
	}
	if reg.ModelID != "deepseek-vl2" || reg.Family != ModelFamilyDeepSeek {
		t.Fatalf("bad registry identity: %#v", reg)
	}
	if len(reg.Observations) != 1 {
		t.Fatalf("observations=%d want 1", len(reg.Observations))
	}
	if reg.Observations[0].Outcome != "PARTIAL" {
		t.Fatalf("outcome=%q", reg.Observations[0].Outcome)
	}
}
