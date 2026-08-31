package realcampaign

import (
	"encoding/json"
	"os"
	"testing"
)

func TestRecordWorkingConfigurationKeepsPerModelSuccessHistory(t *testing.T) {
	root := t.TempDir()
	spec := Spec{
		Endpoint: "http://127.0.0.1:1234/v1",
		Compatibility: "lm-studio",
		TransportCondition: TransportDirectImageAPI,
		Temperature: 0,
		TimeoutSeconds: 180,
		TransportRetries: 1,
		Conditions: []string{"NATIVE_PNG_ONLY"},
	}
	profile := BuildModelInteropProfile("lfm2-vl-1.6b", "lm-studio", TransportDirectImageAPI)
	first := BuildWorkingConfiguration(spec, profile, "DOCTOR_TRANSPORT", "program-a", "carrier-a", "probe-a")
	path, err := RecordWorkingConfiguration(root, first)
	if err != nil { t.Fatal(err) }
	second := BuildWorkingConfiguration(spec, profile, "CAMPAIGN_RUN", "program-a", "carrier-a", "")
	second.Evidence[0].MeanNativeScore = 0.75
	second.Evidence[0].MeanOverallScore = 0.80
	if _, err := RecordWorkingConfiguration(root, second); err != nil { t.Fatal(err) }

	body, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	var reg WorkingConfigurationRegistry
	if err := json.Unmarshal(body, &reg); err != nil { t.Fatal(err) }
	if reg.ModelID != "lfm2-vl-1.6b" || reg.Family != ModelFamilyLiquidAI { t.Fatalf("bad registry identity: %#v", reg) }
	if len(reg.Configurations) != 1 { t.Fatalf("configurations=%d want 1", len(reg.Configurations)) }
	cfg := reg.Configurations[0]
	if cfg.SuccessCount != 2 { t.Fatalf("success_count=%d want 2", cfg.SuccessCount) }
	if len(cfg.Evidence) != 2 { t.Fatalf("evidence=%d want 2", len(cfg.Evidence)) }
	if cfg.Evidence[1].MeanNativeScore != 0.75 { t.Fatalf("native score=%v", cfg.Evidence[1].MeanNativeScore) }
}

func TestRecordWorkingConfigurationSeparatesExactModels(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Endpoint:"https://example.invalid/v1",Compatibility:"openai",TransportCondition:TransportDirectImageAPI}
	q3 := BuildWorkingConfiguration(spec, BuildModelInteropProfile("qwen2.5-vl-3b", "openai", TransportDirectImageAPI), "DOCTOR_TRANSPORT", "p", "c", "r")
	q7 := BuildWorkingConfiguration(spec, BuildModelInteropProfile("qwen2.5-vl-7b", "openai", TransportDirectImageAPI), "DOCTOR_TRANSPORT", "p", "c", "r")
	p3, err := RecordWorkingConfiguration(root, q3); if err != nil { t.Fatal(err) }
	p7, err := RecordWorkingConfiguration(root, q7); if err != nil { t.Fatal(err) }
	if p3 == p7 { t.Fatal("different exact model ids must have separate registries") }
}
