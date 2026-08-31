package candidateprepare

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/experimentpolicy"
)

const ReportSchemaR1 = "tlaloc.candidate-preparation.r1"

type Request struct {
	Builder             string
	ParentPNG           string
	InheritedMutations  []experimentpolicy.OrigamiMutation
	Candidate           experimentpolicy.CandidateManifest
	OutputDir           string
}

type Report struct {
	Schema              string                                     `json:"schema"`
	CandidateID         string                                     `json:"candidate_id"`
	EligibleForVLM      bool                                       `json:"eligible_for_vlm"`
	ParentSHA256        string                                     `json:"parent_sha256"`
	CandidatePNG        string                                     `json:"candidate_png"`
	OrigamiSpec         string                                     `json:"origami_spec"`
	ExpectedSemantics   string                                     `json:"expected_semantics"`
	BuildManifest       string                                     `json:"build_manifest"`
	VisibleTextManifest string                                     `json:"visible_text_manifest"`
	BuildValid          bool                                       `json:"build_valid"`
	Parity              experimentpolicy.ParityReport              `json:"parity"`
	VisibleTextFidelity experimentpolicy.VisibleTextFidelityReport `json:"visible_text_fidelity"`
	RegressionPrecheck  experimentpolicy.RegressionReport          `json:"regression_precheck"`
}

func Prepare(req Request) (Report,error) {
	if req.Builder==""{req.Builder="origami-candidate-build"}
	if req.ParentPNG==""||req.Candidate.ID==""{return Report{},fmt.Errorf("parent PNG and candidate are required")}
	if req.OutputDir==""{req.OutputDir=filepath.Join(".","tlaloc-candidates",req.Candidate.ID)}
	if err:=os.MkdirAll(req.OutputDir,0o700);err!=nil{return Report{},err}
	parentBody,err:=os.ReadFile(req.ParentPNG);if err!=nil{return Report{},err};sum:=sha256.Sum256(parentBody);parentSHA:=hex.EncodeToString(sum[:])
	origamiSpec,err:=experimentpolicy.ToOrigamiSpec(req.Candidate,parentSHA);if err!=nil{return Report{},err}
	specPath:=filepath.Join(req.OutputDir,"origami-candidate-spec.json")
	expectedPath:=filepath.Join(req.OutputDir,"expected-semantics.json")
	buildPath:=filepath.Join(req.OutputDir,"build-manifest.json")
	visibleTextPath:=filepath.Join(req.OutputDir,"visible-text-manifest.json")
	pngPath:=filepath.Join(req.OutputDir,req.Candidate.ID+".png")
	if err:=writeJSON(specPath,origamiSpec);err!=nil{return Report{},err}
	inherited,err:=json.Marshal(req.InheritedMutations);if err!=nil{return Report{},err}
	if err:=run(req.Builder,"semantic-manifest","-in",req.ParentPNG,"-mutations-json",string(inherited),"-out",expectedPath);err!=nil{return Report{},fmt.Errorf("expected semantic manifest: %w",err)}
	if err:=run(req.Builder,"build","-parent",req.ParentPNG,"-out",pngPath,"-spec",specPath,"-inherited-mutations-json",string(inherited),"-interop-report",buildPath);err!=nil{return Report{},fmt.Errorf("candidate build: %w",err)}
	var expected experimentpolicy.SemanticManifest;if err:=readJSON(expectedPath,&expected);err!=nil{return Report{},err}
	var build experimentpolicy.BuildManifest;if err:=readJSON(buildPath,&build);err!=nil{return Report{},err}

	report:=Report{Schema:ReportSchemaR1,CandidateID:req.Candidate.ID,ParentSHA256:parentSHA,CandidatePNG:pngPath,OrigamiSpec:specPath,ExpectedSemantics:expectedPath,BuildManifest:buildPath,VisibleTextManifest:visibleTextPath}
	if err:=experimentpolicy.ValidateBuild(req.Candidate,build);err!=nil{
		report.BuildValid=false
		_ = writeJSON(filepath.Join(req.OutputDir,"preparation-report.json"),report)
		return report,fmt.Errorf("build validation: %w",err)
	}
	report.BuildValid=true
	if err:=writeJSON(visibleTextPath,build.VisibleText);err!=nil{return Report{},err}
	report.Parity=experimentpolicy.CheckParity(req.Candidate,expected,build.VisibleSemantics)
	report.VisibleTextFidelity=experimentpolicy.CheckVisibleTextFidelity(req.Candidate,build.VisibleSemantics,build.VisibleText)
	report.RegressionPrecheck=experimentpolicy.CheckRegressionPreconditions(req.Candidate,expected,build)
	report.EligibleForVLM=report.BuildValid&&report.Parity.Pass&&report.VisibleTextFidelity.Pass&&report.RegressionPrecheck.Pass
	if err:=writeJSON(filepath.Join(req.OutputDir,"preparation-report.json"),report);err!=nil{return Report{},err}
	if !report.Parity.Pass{return report,fmt.Errorf("semantic parity failed: %s",report.Parity.FailureCode)}
	if !report.VisibleTextFidelity.Pass{return report,fmt.Errorf("visible text fidelity failed: %s",report.VisibleTextFidelity.FailureCode)}
	if !report.RegressionPrecheck.Pass{return report,fmt.Errorf("regression precheck failed")}
	return report,nil
}

func run(name string,args ...string)error{cmd:=exec.Command(name,args...);out,err:=cmd.CombinedOutput();if err!=nil{return fmt.Errorf("%v: %s",err,bytes.TrimSpace(out))};return nil}
func writeJSON(path string,v any)error{b,err:=json.MarshalIndent(v,"","  ");if err!=nil{return err};return os.WriteFile(path,append(b,'\n'),0o600)}
func readJSON(path string,v any)error{b,err:=os.ReadFile(path);if err!=nil{return err};dec:=json.NewDecoder(bytes.NewReader(b));dec.DisallowUnknownFields();return dec.Decode(v)}
