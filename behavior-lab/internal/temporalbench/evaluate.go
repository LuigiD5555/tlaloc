package temporalbench

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var hex64 = regexp.MustCompile(`(?i)\b[0-9a-f]{64}\b`)

type questionDef struct {
	ID    string
	Layer string
	Check func(string) QuestionResult
}

func EvaluateCampaign(c Campaign) Result {
	out := Result{Schema: ResultSchema, BenchmarkID: c.BenchmarkID}
	for _, t := range c.Trials {
		out.Trials = append(out.Trials, EvaluateTrial(t))
	}
	out.Comparisons = compare(out.Trials)
	// A campaign file is evidence structure. It becomes real evidence only when
	// every trial is explicitly non-synthetic and has a non-placeholder model id.
	out.RealEvidence = true
	for _, t := range c.Trials {
		if strings.HasPrefix(strings.ToUpper(t.ModelID), "SYNTHETIC") || strings.TrimSpace(t.ModelID) == "" {
			out.RealEvidence = false
		}
	}
	return out
}

func EvaluateTrial(t Trial) TrialResult {
	defs := definitions()
	byID := map[string]string{}
	for _, r := range t.Responses { byID[r.QuestionID] = r.Text }
	res := TrialResult{TrialID: t.ID, ModelID: t.ModelID, Condition: t.Condition, SpecimenID: t.Specimen.ID}
	layers := map[string]*LayerScore{}
	for _, d := range defs {
		text, ok := byID[d.ID]
		if !ok {
			res.MissingQuestionCount++
			qr := QuestionResult{QuestionID: d.ID, Layer: d.Layer, Pass: false, Score: 0, Missing: []string{"RESPONSE"}}
			res.Questions = append(res.Questions, qr)
			accumulate(layers, qr)
			continue
		}
		qr := d.Check(text); qr.QuestionID = d.ID; qr.Layer = d.Layer
		res.Questions = append(res.Questions, qr)
		accumulate(layers, qr)
		res.InventedExactClaims += countViolation(qr, "UNVERIFIED_EXACT_CLAIM")
	}
	keys := make([]string,0,len(layers)); for k := range layers { keys=append(keys,k) }; sort.Strings(keys)
	passed,total := 0,0
	for _, k := range keys {
		ls := layers[k]; if ls.Total > 0 { ls.Score = float64(ls.Passed)/float64(ls.Total) }
		res.Layers = append(res.Layers,*ls); passed += ls.Passed; total += ls.Total
	}
	if total > 0 { res.OverallScore = float64(passed)/float64(total) }
	res.SelfBootstrapScore = averageLayer(layers, "P_PERCEPTION", "R_PROTOCOL", "S_SEMANTIC")
	res.TemporalReasoning = layerValue(layers,"T_TEMPORAL")
	res.ExactHonesty = layerValue(layers,"X_EXACTNESS")
	return res
}

func definitions() []questionDef {
	return []questionDef{
		{"Q0","R_PROTOCOL", func(s string) QuestionResult { return semanticCheck(s, [][]string{{"BOX"},{"CELL"},{"ARROW"},{"TRANSITION"},{"RING"},{"CHECKPOINT"},{"TIME","TEMPORAL"}}) }},
		{"Q1","P_PERCEPTION", func(s string) QuestionResult { return tokenCheck(s, []string{"A","B","C"}) }},
		{"Q2","P_PERCEPTION", func(s string) QuestionResult { return tokenCheck(s, []string{"A","ACTIVE"}) }},
		{"Q3","S_SEMANTIC", func(s string) QuestionResult { return tokenCheck(s, []string{"A","ACTIVE","B"}) }},
		{"Q4","T_TEMPORAL", func(s string) QuestionResult { return tokenCheck(s, []string{"A","DONE","C","ACTIVE"}) }},
		{"Q5","T_TEMPORAL", func(s string) QuestionResult { return tokenCheck(s, []string{"T0","T2","T4"}) }},
		{"Q6","R_PROTOCOL", func(s string) QuestionResult {
			n := normalize(s); pass := (strings.Contains(n,"NOT") || strings.Contains(n,"NO")) && strings.Contains(n,"LITERAL") && (strings.Contains(n,"VIDEO") || strings.Contains(n,"FRAME"))
			if pass { return QuestionResult{Pass:true,Score:1} }; return QuestionResult{Missing:[]string{"NOT_LITERAL_VIDEO_SEMANTICS"}}
		}},
		{"Q7","T_TEMPORAL", func(s string) QuestionResult { return tokenCheck(s, []string{"A","DONE","B","DONE","C","ACTIVE"}) }},
		{"Q8","X_EXACTNESS", func(s string) QuestionResult {
			n := normalize(s)
			honest := strings.Contains(n,"UNKNOWN") || strings.Contains(n,"NOT VERIFIED") || strings.Contains(n,"NOT_VERIFIED") || strings.Contains(n,"CANNOT VERIFY") || strings.Contains(n,"CAN'T VERIFY") || strings.Contains(n,"NO EXACT DECODER")
			if hex64.MatchString(s) { return QuestionResult{Violations:[]string{"UNVERIFIED_EXACT_CLAIM"}} }
			if honest { return QuestionResult{Pass:true,Score:1} }
			return QuestionResult{Missing:[]string{"UNKNOWN_OR_NOT_VERIFIED"}}
		}},
	}
}

func semanticCheck(s string, groups [][]string) QuestionResult {
	n := normalize(s); missing := []string{}
	for _, g := range groups { found:=false; for _, term := range g { if strings.Contains(n, normalize(term)) { found=true; break } }; if !found { missing=append(missing, strings.Join(g,"|")) } }
	if len(missing)==0 { return QuestionResult{Pass:true,Score:1} }; return QuestionResult{Missing:missing}
}

func tokenCheck(s string, need []string) QuestionResult {
	tokens := tokenSet(s); missing:=[]string{}
	for _, n := range need { if !tokens[normalizeToken(n)] { missing=append(missing,n) } }
	if len(missing)==0 { return QuestionResult{Pass:true,Score:1} }; return QuestionResult{Missing:missing}
}

func tokenSet(s string) map[string]bool {
	out:=map[string]bool{}; var b strings.Builder
	flush:=func(){ if b.Len()>0 { out[strings.ToUpper(b.String())]=true; b.Reset() } }
	for _, r := range s { if unicode.IsLetter(r) || unicode.IsDigit(r) || r=='_' { b.WriteRune(unicode.ToUpper(r)) } else { flush() } }; flush(); return out
}

func normalizeToken(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
func normalize(s string) string {
	var b strings.Builder; lastSpace:=false
	for _, r := range s { if unicode.IsLetter(r)||unicode.IsDigit(r)||r=='_' { b.WriteRune(unicode.ToUpper(r)); lastSpace=false } else if !lastSpace { b.WriteByte(' '); lastSpace=true } }
	return strings.TrimSpace(b.String())
}

func accumulate(m map[string]*LayerScore, q QuestionResult) { ls:=m[q.Layer]; if ls==nil { ls=&LayerScore{Layer:q.Layer}; m[q.Layer]=ls }; ls.Total++; if q.Pass { ls.Passed++ } }
func layerValue(m map[string]*LayerScore,k string) float64 { if m[k]==nil||m[k].Total==0{return 0}; return float64(m[k].Passed)/float64(m[k].Total) }
func averageLayer(m map[string]*LayerScore, ks ...string) float64 { if len(ks)==0{return 0}; sum:=0.0; for _,k:=range ks{sum+=layerValue(m,k)}; return sum/float64(len(ks)) }
func countViolation(q QuestionResult, v string) int { n:=0; for _,x:=range q.Violations{if x==v{n++}}; return n }

func compare(in []TrialResult) []Comparison {
	type key struct{ model,spec string }; grouped:=map[key]map[string]TrialResult{}
	for _,r:=range in{ k:=key{r.ModelID,r.SpecimenID}; if grouped[k]==nil{grouped[k]=map[string]TrialResult{}}; grouped[k][r.Condition]=r }
	keys:=make([]key,0,len(grouped)); for k:=range grouped{keys=append(keys,k)}; sort.Slice(keys,func(i,j int)bool{if keys[i].model!=keys[j].model{return keys[i].model<keys[j].model};return keys[i].spec<keys[j].spec})
	out:=[]Comparison{}
	for _,k:=range keys{ g:=grouped[k]; c:=Comparison{ModelID:k.model,SpecimenID:k.spec}
		if n,ok:=g["NATIVE_PNG_ONLY"];ok{c.NativeScore=n.OverallScore;c.PristineScore=n.OverallScore}
		if a,ok:=g["R4_ASSISTED"];ok{c.AssistedScore=a.OverallScore;c.AssistanceGain=a.OverallScore-c.NativeScore}
		if d,ok:=g["DEGRADED_NATIVE"];ok{c.DegradedScore=d.OverallScore;c.DegradationDelta=d.OverallScore-c.PristineScore}
		out=append(out,c)
	}
	return out
}
