package learningpolicy

import "tlaloc.local/behaviorlab/internal/promptgenome"

func ApplyGenomeProtection(p Policy, g promptgenome.Genome) Policy {
	for _,m:=range g.Modules{
		if !m.Protected || m.ID==p.Target { continue }
		p.Rules=append(p.Rules,Rule{Kind:RulePreserve,Target:m.ID,Reason:"protected by prompt genome",Confidence:m.Maturity})
		p.Invariants=append(p.Invariants,LearnedInvariant{ID:"genome-"+sanitize(m.ID),Scope:"prompt-module",Maturity:m.Maturity,Preserve:[]string{m.ID},Reason:"prompt genome marks module protected",EvidenceIDs:append([]string(nil),m.EvidenceIDs...),Protected:true})
	}
	return dedupe(p)
}
