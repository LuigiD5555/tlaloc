package candidateprepare

import (
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/experimentpolicy"
)

func TestPrepareRunsBuilderAndParityGate(t *testing.T){
	d:=t.TempDir();parent:=filepath.Join(d,"parent.png");if err:=os.WriteFile(parent,[]byte("parent"),0o600);err!=nil{t.Fatal(err)}
	builder:=filepath.Join(d,"builder.sh")
	script:=`#!/usr/bin/env bash
set -euo pipefail
cmd="$1"; shift
if [[ "$cmd" == semantic-manifest ]]; then
  out=""; while (($#)); do [[ "$1" == -out ]] && { out="$2"; shift 2; continue; }; shift; done
  cat > "$out" <<'JSON'
{"schema":"origami.semantic-manifest.r1","program_sha256":"p","facts":[{"key":"RULE_R1","value":"A=ACTIVE"},{"key":"EXECUTION_POLICY","value":"NONE"}]}
JSON
  exit 0
fi
out=""; interop=""; while (($#)); do
  case "$1" in -out) out="$2"; shift 2;; -interop-report) interop="$2"; shift 2;; *) shift;; esac
done
printf 'png' > "$out"
cat > "$interop" <<'JSON'
{"schema":"origami.candidate-build-manifest.r1","candidate_id":"c","renderer_version":"r","artifact_sha256":"a","artifact_bytes":3,"program_sha256":"p","applied_mutations":[{"kind":"PROMPT","target":"EXECUTION_POLICY","value":"EXECUTE_VISIBLE_RULES_TO_STABLE_R1"}],"visible_semantics":{"schema":"origami.semantic-manifest.r1","program_sha256":"p","facts":[{"key":"RULE_R1","value":"A=ACTIVE"},{"key":"EXECUTION_POLICY","value":"EXECUTE_VISIBLE_RULES_TO_STABLE_R1"}]}}
JSON
`
	if err:=os.WriteFile(builder,[]byte(script),0o700);err!=nil{t.Fatal(err)}
	c:=experimentpolicy.CandidateManifest{Schema:experimentpolicy.CandidateSchemaR1,ID:"c",ProgramSHA256:"p",Mutations:[]experimentpolicy.Mutation{{Kind:"PROMPT",Target:"EXECUTION_POLICY",Value:"EXECUTE_VISIBLE_RULES_TO_STABLE_R1"}},ExpectedSemanticChanges:[]experimentpolicy.SemanticFact{{Key:"EXECUTION_POLICY",Value:"EXECUTE_VISIBLE_RULES_TO_STABLE_R1"}}}
	r,err:=Prepare(Request{Builder:builder,ParentPNG:parent,Candidate:c,OutputDir:filepath.Join(d,"out")});if err!=nil{t.Fatal(err)}
	if !r.EligibleForVLM||!r.Parity.Pass{t.Fatalf("report=%#v",r)}
}
