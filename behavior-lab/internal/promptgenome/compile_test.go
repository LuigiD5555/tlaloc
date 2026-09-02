package promptgenome

import "testing"

func TestCompilePreservesRequiredModulesAndUsesMinText(t *testing.T) {
	g := Genome{Schema: GenomeSchemaR1, ID: "g", Version: 1, Modules: []Module{
		{ID: "BOOT", Version: 1, Text: "BOOT FULL", MinText: "BOOT", Priority: 100, Required: true},
		{ID: "TEMPORAL", Version: 2, Text: "EXECUTE RULES UNTIL STABLE", MinText: "EXEC STABLE", Priority: 80, Required: true, Dependencies: []string{"BOOT"}},
		{ID: "OPTIONAL", Version: 1, Text: "OPTIONAL LONG SECTION", Priority: 1},
	}}
	out, err := Compile(CompileRequest{Genome: g, MaxChars: 24})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Modules) != 2 {
		t.Fatalf("modules=%d %#v", len(out.Modules), out.Modules)
	}
	if out.Modules[0].ID != "BOOT" || out.Modules[1].ID != "TEMPORAL" {
		t.Fatalf("order=%#v", out.Modules)
	}
	if out.Chars > 24 {
		t.Fatalf("chars=%d", out.Chars)
	}
}

func TestCompileRejectsMissingDependency(t *testing.T) {
	g := Genome{Schema: GenomeSchemaR1, ID: "g", Version: 1, Modules: []Module{
		{ID: "A", Version: 1, Text: "A", Priority: 10},
		{ID: "B", Version: 1, Text: "B", Priority: 20, Required: true, Dependencies: []string{"A"}},
	}}
	_, err := Compile(CompileRequest{Genome: g, RelevantModules: []string{"B"}})
	if err == nil {
		t.Fatal("expected dependency error")
	}
}
