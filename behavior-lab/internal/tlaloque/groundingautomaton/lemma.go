package groundingautomaton

import "strings"

// R1 alignment aids. All deterministic, curated, and small. The design rule:
// a lemma group may only contain near-exact synonyms in this domain, and two
// members of an antonym pair (see antonymPairs) must NEVER share a group.

// lemmaGroups maps a canonical form to its surface variants (post-stem).
// Applied inside contentTerms so "losing"/"loss", "requires"/"needs" etc.
// align. Keep this list conservative — a wrong merge silently inflates
// alignment and can turn an UNKNOWN into a SUPPORTED.
var lemmaGroups = map[string][]string{
	"loss":        {"lose", "losing", "lost", "loss", "losses", "perdida", "perder", "perdiendo"},
	"change":      {"change", "changes", "changed", "changing", "alter", "alters", "altered", "modify", "modifies", "modified", "cambio", "cambia", "modifica"},
	"increase":    {"increase", "increases", "increased", "increasing", "rise", "rises", "risen", "grow", "grows", "grew", "grown", "raise", "raises", "raised", "aumenta", "aumento", "crece", "crecimiento"},
	"decrease":    {"decrease", "decreases", "decreased", "reduce", "reduces", "reduced", "reducing", "shrink", "shrinks", "shrank", "lower", "lowers", "lowered", "drop", "drops", "dropped", "disminuye", "reduce", "reduccion"},
	"require":     {"require", "requires", "required", "requiring", "need", "needs", "needed", "must", "requiere", "necesita", "necesario"},
	"break":       {"break", "breaks", "broke", "broken", "collapse", "collapses", "collapsed", "rompe", "colapsa", "colapso"},
	"central":     {"central", "centralize", "centralized", "centralised", "centralization", "centralisation", "centralized", "centralizado", "centralizada"},
	"decentral":   {"decentralize", "decentralized", "decentralised", "decentralization", "decentralisation", "descentralizado", "descentralizada"},
	"coordinate":  {"coordinate", "coordinates", "coordinated", "coordination", "coordinating", "coordina", "coordinacion"},
	"agent":       {"agent", "agents", "node", "nodes", "worker", "workers", "agente", "agentes", "nodo", "nodos"},
	"leader":      {"leader", "leaders", "controller", "controllers", "coordinator", "coordinators", "scheduler", "schedulers", "supervisor", "lider", "controlador"},
	"local":       {"local", "locally", "locales", "localmente"},
	"global":      {"global", "globally", "collective", "collectively", "aggregate", "overall", "group", "groups", "colectivo", "colectiva", "grupo", "grupos"},
	"emerge":      {"emerge", "emerges", "emerged", "emergent", "emergence", "emerge", "emergente"},
	"redundancy":  {"redundancy", "redundant", "redundante", "redundancia"},
	"robust":      {"robust", "robustness", "resilient", "resilience", "robusto", "robustez"},
	"result":      {"achieve", "achieves", "achieved", "accomplish", "accomplishes", "attain", "attains", "outcome", "outcomes", "result", "results", "logra", "consigue", "alcanza", "resultado", "resultados"},
	"parameter":   {"parameter", "parameters", "param", "params", "parametro", "parametros"},
	"train":       {"train", "trains", "trained", "training", "entrena", "entrenado", "entrenamiento"},
	"epoch":       {"epoch", "epochs", "epoca", "epocas"},
	"communicate": {"communicate", "communicates", "communication", "message", "messages", "messaging", "comunica", "comunicacion", "mensaje", "mensajes"},
}

// lemmaCanonical is the reverse index, built once.
var lemmaCanonical = func() map[string]string {
	out := make(map[string]string)
	for canonical, variants := range lemmaGroups {
		out[canonical] = canonical
		for _, variant := range variants {
			out[variant] = canonical
			out[stem(variant)] = canonical
		}
	}
	return out
}()

func lemma(term string) string {
	if canonical, ok := lemmaCanonical[term]; ok {
		return canonical
	}
	return term
}

// fillerTerms carry little propositional content — degree words, hedges,
// discourse framing. Dropped from a claim's *core* so a long, hedged claim
// is not penalised on alignment for words the evidence has no reason to
// echo. NOT dropped from the evidence side.
var fillerTerms = mustTermSet(`
roughly approximately about around nearly almost good great some several various
certain many much more most generally basically essentially actually really quite
rather fairly relatively simply just even also very often usually typically
according passage text section material states explains indicates says
aproximadamente cerca casi varios varias cierto muchos generalmente basicamente
realmente bastante segun pasaje texto seccion indica dice explica
`)

// metaClaimPrefixes are stripped from the front of a claim before it is
// split into content terms — pure framing.
var metaClaimPrefixes = []string{
	"according to the passage", "according to the text", "according to this",
	"the passage says that", "the passage states that", "the text says that",
	"the text states that", "it says that", "it states that", "note that",
	"in summary", "in short", "overall", "importantly", "actually",
	"segun el pasaje", "segun el texto", "el pasaje dice que", "el texto dice que",
	"de hecho", "en resumen", "en definitiva",
}

func mustTermSet(blob string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, w := range strings.Fields(blob) {
		out[w] = struct{}{}
		out[stem(w)] = struct{}{}
	}
	return out
}

// grammarWords is a broad EN+ES closed-class set used ONLY by isNonAnswer
// (the canonical `stopwords` map is deliberately tiny and tuned for content
// matching, not for prose detection).
var grammarWords = mustTermSet(`
a an the this that these those of in on at to from by for with without about into
over under between and or but nor so as is are was were be been being it its they
them their his her our your my we you he she i not no if then than there here
what which who whom whose when where why how do does did has have had will would
can could should may might must all every each some any many much more most other
such only own same too very one two three then also both either neither whole
through per via across along around also yet still just even because while during
after before against above below up down out off then once
un una unos unas el la los las de del al en a por para con sin sobre entre y o
pero como que se lo le les su sus mi tu nuestro es son era eran ser este esta esto
estos estas ese esa eso no ni tan mas menos muy solo cada uno dos tres tambien
porque aunque cuando donde
`)

// isNonAnswer detects an answer that carries no verifiable predication: a
// bare list of terms with essentially no grammatical glue, or an all-hedge
// fragment. Such an answer is INSUFFICIENT — it cannot be grounded or
// contradicted, only rejected.
func isNonAnswer(answer string) bool {
	fields := strings.Fields(normalize(answer))
	if len(fields) < 5 {
		return false // too short to classify; let the claim path handle it
	}
	grammar, hedge, content := 0, 0, 0
	for _, field := range fields {
		s := stem(field)
		if _, ok := grammarWords[field]; ok {
			grammar++
			continue
		}
		if _, ok := grammarWords[s]; ok {
			grammar++
			continue
		}
		if _, ok := fillerTerms[s]; ok {
			hedge++
			continue
		}
		content++
	}
	commas := strings.Count(answer, ",")
	// A keyword dump: almost no grammar words and either 4+ commas or no
	// commas at all (space-joined term list).
	if grammar <= 1 && content >= 5 && (commas >= 4 || commas == 0) {
		return true
	}
	// Hedge-dominated: most non-grammar tokens are hedges/degree words.
	if content > 0 && float64(hedge)/float64(hedge+content) >= 0.6 {
		return true
	}
	return false
}

func stripMetaPrefix(claim string) string {
	lowered := strings.ToLower(strings.TrimSpace(claim))
	for _, prefix := range metaClaimPrefixes {
		if strings.HasPrefix(lowered, prefix) {
			rest := strings.TrimSpace(claim[len(prefix):])
			return strings.TrimPrefix(rest, ",")
		}
	}
	return claim
}
