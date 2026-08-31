package closedloop

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/visualsearch"
)

func TestAutoCandidateConfigsAreDeterministicAndCapabilityFiltered(t *testing.T) {
	tmp := t.TempDir()
	parentPNG := filepath.Join(tmp,"parent.png")
	f, err := os.Create(parentPNG); if err != nil { t.Fatal(err) }
	if err := png.Encode(f,image.NewGray(image.Rect(0,0,640,640))); err != nil { t.Fatal(err) }
	if err := f.Close(); err != nil { t.Fatal(err) }
	p := prepared{cfg:Config{
		AutoCandidates:true,
		CandidateBuilder:[]string{"origami-candidate-build"},
		AutoCandidateBaseProfileID:"origami.temporal-carrier.r0.profile-1",
		AutoCandidatesPerGeneration:4,
	}}
	caps := candidateBuilderCapabilities{
		Schema:"origami.experimental-candidate.r0.capabilities",
		ParentProfiles:[]string{"origami.temporal-carrier.r0.profile-1"},
		SupportedKinds:[]string{"LAYOUT","PROMPT"},
		UnsupportedKinds:[]string{"DEPTH_STRUCTURE"},
		ExactPlaneMutation:false,
		MaxMutations:8,
	}
	plan := adaptivesearch.Plan{SuggestedMutations:[]adaptivesearch.SuggestedMutation{
		{Kind:visualsearch.MutationLayout,Target:"T1_TO_T2_ENTRY_ROUTE",Value:"EXPLICIT_DIRECTIONAL_ANCHOR",Experimental:true},
		{Kind:visualsearch.MutationDepthStructure,Target:"T2",Value:"EXPERIMENTAL_DEPTH_SIGNAL",Experimental:true},
		{Kind:visualsearch.MutationPrompt,Target:"ROSETTA.S2.READ_SUPERINDEX",Value:"DECLARE_T2_LOCATION_BEFORE_DECODE",Experimental:true},
	}}
	parent := SpecimenConfig{ID:"parent-r0",PNG:parentPNG}
	first, err := p.autoCandidateConfigs(plan,caps,parent,filepath.Join(tmp,"g1")); if err != nil { t.Fatal(err) }
	second, err := p.autoCandidateConfigs(plan,caps,parent,filepath.Join(tmp,"g1")); if err != nil { t.Fatal(err) }
	if len(first)!=2 || len(second)!=2 { t.Fatalf("expected two supported candidates, got %d/%d",len(first),len(second)) }
	for i := range first {
		if first[i].ID != second[i].ID { t.Fatalf("non-deterministic id: %q != %q",first[i].ID,second[i].ID) }
		if first[i].ParentSpecimenID != parent.ID { t.Fatalf("wrong parent: %#v",first[i]) }
		if len(first[i].Mutations)!=1 { t.Fatalf("auto candidates must isolate one mutation: %#v",first[i]) }
		if len(first[i].BuildCommand)!=1 || first[i].BuildCommand[0]!="origami-candidate-build" { t.Fatalf("wrong build hook: %#v",first[i]) }
		if first[i].Mutations[0].Kind==visualsearch.MutationDepthStructure { t.Fatal("unsupported mutation leaked into candidate bank") }
	}
}

func TestValidateAutoConfigQueriesBuilderCapabilities(t *testing.T) {
	if os.PathSeparator != '/' { t.Skip("shell fixture requires POSIX") }
	tmp := t.TempDir()
	builder := filepath.Join(tmp,"builder")
	script := `#!/bin/sh
if [ "$1" != "capabilities" ]; then exit 9; fi
cat <<'JSON'
{"schema":"origami.experimental-candidate.r0.capabilities","parent_profiles":["origami.temporal-carrier.r0.profile-1"],"supported_kinds":["LAYOUT"],"unsupported_kinds":["DEPTH_STRUCTURE"],"exact_plane_mutation":false,"max_mutations":8}
JSON
`
	if err := os.WriteFile(builder,[]byte(script),0o755); err != nil { t.Fatal(err) }
	p := prepared{cfg:Config{AutoCandidates:true,CandidateBuilder:[]string{builder},AutoCandidateBaseProfileID:"origami.temporal-carrier.r0.profile-1"}}
	caps, err := validateAutoConfig(context.Background(),p); if err != nil { t.Fatal(err) }
	if caps.ExactPlaneMutation || len(caps.SupportedKinds)!=1 || caps.SupportedKinds[0]!="LAYOUT" { t.Fatalf("unexpected caps: %#v",caps) }
}

func TestValidateAutoConfigRejectsExactPlaneBuilder(t *testing.T) {
	if os.PathSeparator != '/' { t.Skip("shell fixture requires POSIX") }
	tmp := t.TempDir()
	builder := filepath.Join(tmp,"builder")
	script := `#!/bin/sh
cat <<'JSON'
{"schema":"origami.experimental-candidate.r0.capabilities","parent_profiles":["origami.temporal-carrier.r0.profile-1"],"supported_kinds":["LAYOUT"],"unsupported_kinds":[],"exact_plane_mutation":true,"max_mutations":8}
JSON
`
	if err := os.WriteFile(builder,[]byte(script),0o755); err != nil { t.Fatal(err) }
	p := prepared{cfg:Config{AutoCandidates:true,CandidateBuilder:[]string{builder},AutoCandidateBaseProfileID:"origami.temporal-carrier.r0.profile-1"}}
	if _,err := validateAutoConfig(context.Background(),p); err==nil { t.Fatal("expected exact-plane builder rejection") }
}
