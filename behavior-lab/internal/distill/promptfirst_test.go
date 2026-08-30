package distill

import "testing"

func validCandidate(id string, level DeploymentLevel, fidelity float64) ArtifactCandidate {
	return ArtifactCandidate{
		ID: id, Level: level, Prompt: "follow the demonstrated behavior",
		BehavioralFidelity: fidelity, PassRate: .98, RegressionRate: 0,
		PromptTokens: 900, CleanTargetTrials: 9,
	}
}

func TestPromptOnlyWinsOverSlightlyBetterRuntime(t *testing.T) {
	prompt := validCandidate("portable", LevelPromptOnly, .96)
	runtime := validCandidate("runtime", LevelPromptRuntime, 1.0)
	runtime.Dependencies = []string{"go-runtime", "tool-server"}
	decision, err := SelectPortableArtifact([]ArtifactCandidate{runtime, prompt}, DefaultPromptFirstPolicy())
	if err != nil { t.Fatal(err) }
	if decision.SelectedID != "portable" || decision.SelectedLevel != LevelPromptOnly {
		t.Fatalf("expected prompt-first selection, got %+v", decision)
	}
}

func TestFallsBackWhenPromptCannotPreserveBehavior(t *testing.T) {
	prompt := validCandidate("weak-prompt", LevelPromptOnly, .80)
	tools := validCandidate("tools", LevelPromptTools, .98)
	tools.Dependencies = []string{"calculator"}
	decision, err := SelectPortableArtifact([]ArtifactCandidate{prompt, tools}, DefaultPromptFirstPolicy())
	if err != nil { t.Fatal(err) }
	if decision.SelectedID != "tools" || decision.SelectedLevel != LevelPromptTools {
		t.Fatalf("expected explicit fallback, got %+v", decision)
	}
}

func TestPromptOnlyCannotHideDependencies(t *testing.T) {
	prompt := validCandidate("fake-l0", LevelPromptOnly, .99)
	prompt.Dependencies = []string{"sandbox"}
	decision, err := SelectPortableArtifact([]ArtifactCandidate{prompt}, DefaultPromptFirstPolicy())
	if err != nil { t.Fatal(err) }
	if decision.SelectedID != "" || decision.Evaluations[0].Eligible {
		t.Fatalf("L0 with hidden dependencies should fail: %+v", decision)
	}
}

func TestCleanTargetEvidenceIsRequired(t *testing.T) {
	prompt := validCandidate("untested", LevelPromptOnly, .99)
	prompt.CleanTargetTrials = 0
	decision, err := SelectPortableArtifact([]ArtifactCandidate{prompt}, DefaultPromptFirstPolicy())
	if err != nil { t.Fatal(err) }
	if decision.SelectedID != "" {
		t.Fatalf("untested prompt should not be selected: %+v", decision)
	}
}

func TestSmallerPromptWinsWithinSameDeploymentLevel(t *testing.T) {
	a := validCandidate("a", LevelPromptOnly, .99); a.PromptTokens = 1200
	b := validCandidate("b", LevelPromptOnly, .97); b.PromptTokens = 700
	decision, err := SelectPortableArtifact([]ArtifactCandidate{a,b}, DefaultPromptFirstPolicy())
	if err != nil { t.Fatal(err) }
	if decision.SelectedID != "b" {
		t.Fatalf("expected smaller valid prompt within same level, got %+v", decision)
	}
}
