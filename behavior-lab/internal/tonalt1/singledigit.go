package tonalt1

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"tlaloc.local/behaviorlab/internal/perceptenvelope"
)

// SINGLE_DIGIT capability qualification support (protocol sections 8-15).
//
// R1-C classified SINGLE_DIGIT as FRAGILE (real n=12, value 0.92, CI
// 0.65-0.99 — the CI lower bound < 0.70 fails USABLE_WITH_CONSTRAINTS).
// This enumerates a fresh, deterministic, physically held-out SINGLE_DIGIT
// sample from the same canonical store, under the same D3 v2 presentation
// integrity + domain rules, so a larger-n re-measurement can decide
// whether SINGLE_DIGIT promotes.
//
// Nothing here calls a model, a scorer, or looks at any inference output.
// The frozen promotion threshold lives in perceptenvelope.AggregateR1C /
// verdictFor and is applied AFTER inference by the qualification command.

// SingleDigitBase is one frozen held-out SINGLE_DIGIT stimulus, carrying
// enough to drive perceptenvelope.RunR1C.
type SingleDigitBase struct {
	BaseID      string                         `json:"base_id"`
	CandidateID string                         `json:"candidate_id"`
	Digit       string                         `json:"digit"`
	Page        int                            `json:"page"`
	RegionID    string                         `json:"region_id"`
	LineText    string                         `json:"line_text"`
	RankKey     string                         `json:"rank_key"`
	Candidate   perceptenvelope.MorphCandidate `json:"candidate"`
}

// SingleDigitHeldOut is the frozen qualification dataset.
type SingleDigitHeldOut struct {
	Schema             string            `json:"schema"`
	ExperimentID       string            `json:"experiment_id"`
	SelectorVersion    string            `json:"selector_version"`
	Seed               string            `json:"seed"`
	StoreRootSHA256    string            `json:"store_root_sha256"`
	SourcePDFSHA256    string            `json:"source_pdf_sha256"`
	RequestedN         int               `json:"requested_n"`
	AvailableN         int               `json:"available_held_out_n"`
	PerDigitAvailable  map[string]int    `json:"per_digit_available"`
	PerDigitSelected   map[string]int    `json:"per_digit_selected"`
	HeldOutExclusionOK bool              `json:"held_out_exclusion_verified"`
	Bases              []SingleDigitBase `json:"bases"`
}

const singleDigitSchema = "tonal.t1.singledigit-qual.heldout.r1"

// EnumerateSingleDigitHeldOut runs the D3 v2 scanner with SINGLE_DIGIT
// admitted, keeps only candidates that pass every blocking rule (capability
// morphology aside), excludes every prior-used physical instance
// (including the 12 R1-C SINGLE_DIGIT bases), deduplicates by physical
// identity, and deterministically selects up to requestedN, stratified as
// evenly as possible across digit values 0-9.
func EnumerateSingleDigitHeldOut(root, storeDir string, requestedN int) (SingleDigitHeldOut, error) {
	priorIndex, err := LoadPriorUseIndex(root)
	if err != nil {
		return SingleDigitHeldOut{}, err
	}

	admitSingleDigit = true
	defer func() { admitSingleDigit = false }()

	scan, err := Scan(storeDir, priorIndex)
	if err != nil {
		return SingleDigitHeldOut{}, err
	}

	seen := map[string]bool{}
	byDigit := map[string][]Candidate{}
	for _, cand := range scan.Candidates {
		if cand.Presentation.MorphologyFamily != MorphSingleDigit {
			continue
		}
		// Eligible under every blocking rule (the scanner already applied
		// prior-use, presentation integrity and domain checks; SINGLE_DIGIT
		// is admitted here so REJECT_UNSUPPORTED_MORPHOLOGY is not raised).
		if !cand.Eligibility.Eligible {
			continue
		}
		if cand.PriorUse.Excluded {
			continue
		}
		key := cand.Identity.NormalizedSpanHash
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		byDigit[cand.Source.NumericNormalized] = append(byDigit[cand.Source.NumericNormalized], cand)
	}

	digits := make([]string, 0, len(byDigit))
	total := 0
	perAvail := map[string]int{}
	for digit, list := range byDigit {
		digits = append(digits, digit)
		perAvail[digit] = len(list)
		total += len(list)
		sort.Slice(list, func(i, j int) bool { return rankKey(list[i]) < rankKey(list[j]) })
		byDigit[digit] = list
	}
	sort.Strings(digits)

	// Round-robin across digit strata by rank order until requestedN.
	perSel := map[string]int{}
	var picked []Candidate
	cursor := map[string]int{}
	for len(picked) < requestedN {
		progressed := false
		for _, digit := range digits {
			if len(picked) >= requestedN {
				break
			}
			idx := cursor[digit]
			if idx >= len(byDigit[digit]) {
				continue
			}
			picked = append(picked, byDigit[digit][idx])
			cursor[digit] = idx + 1
			perSel[digit]++
			progressed = true
		}
		if !progressed {
			break
		}
	}

	sort.Slice(picked, func(i, j int) bool { return rankKey(picked[i]) < rankKey(picked[j]) })

	result := SingleDigitHeldOut{
		Schema:            singleDigitSchema,
		ExperimentID:      "tonal-t1-singledigit-qualification",
		SelectorVersion:   SelectorVersion,
		Seed:              Seed,
		StoreRootSHA256:   scan.StoreRootSHA256,
		SourcePDFSHA256:   scan.SourcePDFSHA256,
		RequestedN:        requestedN,
		AvailableN:        total,
		PerDigitAvailable: perAvail,
		PerDigitSelected:  perSel,
	}

	leaked := 0
	for _, cand := range picked {
		if len(priorIndex.Match(cand)) > 0 {
			leaked++
		}
		result.Bases = append(result.Bases, SingleDigitBase{
			BaseID:      "sd-" + cand.CandidateID[:12],
			CandidateID: cand.CandidateID,
			Digit:       cand.Source.NumericNormalized,
			Page:        cand.Corpus.Page,
			RegionID:    cand.Identity.RegionID,
			LineText:    cand.Source.ContainingLineText,
			RankKey:     rankKey(cand),
			Candidate: perceptenvelope.MorphCandidate{
				CandidateID: cand.CandidateID,
				Family:      string(MorphSingleDigit),
				Page:        cand.Corpus.Page,
				RegionID:    cand.Identity.RegionID,
				RegionKind:  "text_line",
				Token:       cand.Source.NumericRaw,
				LineText:    cand.Source.ContainingLineText,
				LineBBox:    cand.Geometry.ContainingLineBBox,
				PageWidth:   cand.Corpus.PageWidth,
				PageHeight:  cand.Corpus.PageHeight,
			},
		})
	}
	result.HeldOutExclusionOK = leaked == 0
	return result, nil
}

// rankKey is the deterministic selection order: sha256(seed | candidate_id).
func rankKey(cand Candidate) string {
	sum := sha256.Sum256([]byte(Seed + "|" + cand.CandidateID))
	return hex.EncodeToString(sum[:])
}
