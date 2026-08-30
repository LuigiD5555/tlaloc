package pdfmemory

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/candidateflow"
	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/canonicalstate"
)

func BuildPDF(pdfPath, outDir, carrierID string) (BuildResult, error) {
	return BuildCorpus([]SourceSpec{{Path: pdfPath, ID: "primary"}}, outDir, carrierID)
}

func BuildCorpus(sources []SourceSpec, outDir, carrierID string) (BuildResult, error) {
	debugf("build start sources=%d out=%s", len(sources), outDir)
	if carrierID == "" {
		carrierID = "document"
	}
	if len(sources) == 0 {
		return BuildResult{}, fmt.Errorf("at least one source PDF is required")
	}
	if err := os.MkdirAll(filepath.Join(outDir, "objects"), 0755); err != nil {
		return BuildResult{}, err
	}
	manifest := Manifest{Schema: Schema, AddressSchema: AddressSchema, ToolProtocol: ToolProtocol, CarrierID: carrierID, DocumentCount: len(sources), ClassificationNote: "block kind is a deterministic layout heuristic; exact bytes/CIDs do not depend on classification"}
	index := Index{Schema: Schema + ".block-index", Postings: map[string][]int{}}
	sketch := newMinHashSketch()
	pageTerms := map[string]map[string]struct{}{}
	usedDocIDs := map[string]int{}
	canonicalDocs := map[string]canonicaldoc.Document{}
	canonicalDirs := map[string]string{}
	var aggregate bytes.Buffer

	for si, src := range sources {
		debugf("source %d read %s", si+1, src.Path)
		if src.Path == "" {
			return BuildResult{}, fmt.Errorf("source %d has empty path", si)
		}
		docID := src.ID
		if docID == "" || (len(sources) == 1 && docID == "primary") {
			docID = slugID(strings.TrimSuffix(filepath.Base(src.Path), filepath.Ext(src.Path)))
		}
		if docID == "" {
			docID = fmt.Sprintf("doc-%03d", si+1)
		}
		if n := usedDocIDs[docID]; n > 0 {
			usedDocIDs[docID] = n + 1
			docID = fmt.Sprintf("%s-%d", docID, n+1)
		} else {
			usedDocIDs[docID] = 1
		}
		sourceBytes, err := os.ReadFile(src.Path)
		if err != nil {
			return BuildResult{}, err
		}
		sourceSHA := hash(sourceBytes)
		pageCount, err := pdfPageCount(src.Path)
		if err != nil {
			return BuildResult{}, err
		}
		canonicalRel := filepath.ToSlash(filepath.Join("canonical", docID))
		canonicalDir := filepath.Join(outDir, filepath.FromSlash(canonicalRel))
		debugf("source %d canonical layout start pages=%d", si+1, pageCount)
		cdoc, err := canonicaldoc.BuildPDF(src.Path, canonicalDir, canonicaldoc.BuildOptions{CarrierID: carrierID, DocumentID: docID, SingleDocument: len(sources) == 1})
		if err != nil {
			return BuildResult{}, fmt.Errorf("canonical document %s: %w", docID, err)
		}
		if cdoc.PageCount != pageCount {
			return BuildResult{}, fmt.Errorf("canonical page mismatch for %s: %d != %d", docID, cdoc.PageCount, pageCount)
		}
		canonicalDocs[docID] = cdoc
		canonicalDirs[docID] = canonicalDir
		debugf("source %d canonical layout done regions=%d figures=%d ocr_pages=%d", si+1, cdoc.RegionCount, cdoc.FigureCount, cdoc.OCRPages)
		rawPages := make([][]byte, cdoc.PageCount)
		for i := 1; i <= cdoc.PageCount; i++ {
			cp, err := canonicaldoc.LoadPage(canonicalDir, i)
			if err != nil {
				return BuildResult{}, err
			}
			rawPages[i-1] = canonicaldoc.CanonicalText(cp)
		}
		debugf("source %d extracted, writing source object", si+1)
		sourcePath, err := writeObject(outDir, sourceSHA, "pdf", sourceBytes)
		if err != nil {
			return BuildResult{}, err
		}
		sourceAddress := fmt.Sprintf("ohf://%s/source/%s", carrierID, docID)
		doc := DocumentRef{ID: docID, Name: filepath.Base(src.Path), SourceSHA256: sourceSHA, SourcePath: sourcePath, SourceAddress: sourceAddress, PageCount: pageCount, CanonicalPath: filepath.ToSlash(filepath.Join(canonicalRel, "document.json")), DigitalPages: cdoc.DigitalPages, OCRPages: cdoc.OCRPages, RegionCount: cdoc.RegionCount, FigureCount: cdoc.FigureCount}
		manifest.Documents = append(manifest.Documents, doc)
		aggregate.WriteString(docID)
		aggregate.WriteByte(0)
		aggregate.WriteString(sourceSHA)
		aggregate.WriteByte(0)
		debugf("source %d writing pages and blocks", si+1)
		for i, p := range rawPages {
			n := i + 1
			cid := hash(p)
			pagePath, err := writeObject(outDir, cid, "txt", p)
			if err != nil {
				return BuildResult{}, err
			}
			addr := pageAddress(carrierID, docID, n, len(sources) == 1)
			layoutRel := filepath.ToSlash(filepath.Join(canonicalRel, cdoc.Pages[i]))
			layoutBytes, err := os.ReadFile(filepath.Join(outDir, filepath.FromSlash(layoutRel)))
			if err != nil {
				return BuildResult{}, err
			}
			layoutPage, err := canonicaldoc.LoadPage(canonicalDirs[docID], n)
			if err != nil {
				return BuildResult{}, err
			}
			pref := PageRef{DocID: docID, Number: n, Address: addr, CID: cid, Bytes: len(p), TokenEq: estimateTokens(p), Path: pagePath, LayoutPath: layoutRel, LayoutCID: hash(layoutBytes), ExtractionMode: layoutPage.ExtractionMode, RegionCount: len(layoutPage.Regions)}
			segments := segmentBlocks(p)
			for bi, seg := range segments {
				bcid := hash(seg.Data)
				// Blocks are content-addressed logical views into the exact page object.
				// Materializing every block as a separate file duplicates bytes and makes
				// verification filesystem-bound without adding exactness.
				bpath := pagePath
				baddr := blockAddress(addr, bi+1)
				br := BlockRef{DocID: docID, Page: n, Number: bi + 1, Address: baddr, CID: bcid, Bytes: len(seg.Data), TokenEq: estimateTokens(seg.Data), Path: bpath, Kind: classifyBlock(seg.Data), StartByte: seg.Start, EndByte: seg.End}
				blockIndex := len(manifest.Blocks)
				manifest.Blocks = append(manifest.Blocks, br)
				pref.Blocks = append(pref.Blocks, baddr)
				terms := tokenize(string(seg.Data))
				for _, term := range terms {
					index.Postings[term] = append(index.Postings[term], blockIndex)
					sketch.Add(term)
				}
				key := fmt.Sprintf("%s:%06d", docID, n)
				set := pageTerms[key]
				if set == nil {
					set = map[string]struct{}{}
					pageTerms[key] = set
				}
				for _, term := range terms {
					set[term] = struct{}{}
				}
			}
			manifest.Pages = append(manifest.Pages, pref)
		}
	}
	debugf("objects built pages=%d blocks=%d", len(manifest.Pages), len(manifest.Blocks))
	manifest.PageCount = len(manifest.Pages)
	manifest.BlockCount = len(manifest.Blocks)
	for _, d := range manifest.Documents {
		manifest.RegionCount += d.RegionCount
	}
	manifest.SourceSHA256 = hash(aggregate.Bytes())
	if len(manifest.Documents) == 1 {
		manifest.SourceName = manifest.Documents[0].Name
		manifest.SourceSHA256 = manifest.Documents[0].SourceSHA256
		manifest.SourcePath = manifest.Documents[0].SourcePath
	}
	debugf("building graph terms=%d", len(index.Postings))
	graph := buildGraph(index, manifest.Blocks, pageTerms, 512, 8)
	debugf("graph built nodes=%d", len(graph.Nodes))
	// Compilation swarm proposes interpretations; only the Go reducer may create CanonicalState.
	debugf("compilation swarm start")
	var candidates []canonicalstate.Candidate
	ledger := canonicalstate.Ledger{Schema: canonicalstate.LedgerSchema}
	for _, d := range manifest.Documents {
		cdoc := canonicalDocs[d.ID]
		cdir := canonicalDirs[d.ID]
		trace, err := candidateflow.ProposeFromCanonical(cdoc, func(n int) (canonicaldoc.Page, error) { return canonicaldoc.LoadPage(cdir, n) })
		if err != nil {
			return BuildResult{}, err
		}
		candidates = append(candidates, trace.Candidates...)
	}
	// Semantic Tlaloque reference: graph landmarks become evidence-anchored
	// concept candidates. Rich model agents can add candidates, but cannot bypass
	// the same reducer/evidence protocol.
	graphTerms := make([]string, 0, len(graph.Nodes))
	for term := range graph.Nodes {
		graphTerms = append(graphTerms, term)
	}
	sort.Strings(graphTerms)
	for _, term := range graphTerms {
		posts := index.Postings[term]
		if len(posts) == 0 {
			continue
		}
		bi := posts[0]
		if bi < 0 || bi >= len(manifest.Blocks) {
			continue
		}
		br := manifest.Blocks[bi]
		candidates = append(candidates, candidateflow.NewCandidate("semantic", br.DocID, "contains_concept", term, true, []canonicalstate.EvidenceRef{{Address: br.Address, CID: br.CID, Kind: "block"}}, .90, "semantic-graph-reference"))
		node := graph.Nodes[term]
		for i, e := range node.Neighbors {
			if i >= 2 {
				break
			}
			candidates = append(candidates, candidateflow.NewCandidate("relation", term, "co_occurs", e.Term, true, []canonicalstate.EvidenceRef{{Address: br.Address, CID: br.CID, Kind: "block"}}, .80, "relation-graph-reference"))
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	debugf("compilation swarm candidates=%d", len(candidates))
	manifest.CandidateCount = len(candidates)
	manifest.CandidatePath = "candidates.json"
	if err := writeJSON(filepath.Join(outDir, manifest.CandidatePath), candidates); err != nil {
		return BuildResult{}, err
	}
	candHash := hashJSONValue(candidates)
	ledger.Append("PROPOSE", "tlaloc.compilation-swarm.r0", manifest.SourceSHA256, candHash, nil)
	debugf("reducer start")
	red := canonicalstate.Reducer{Verifier: preMerkleEvidenceVerifier{StoreDir: outDir, Manifest: &manifest}}
	state, err := red.Reduce(candidates)
	if err != nil {
		return BuildResult{}, err
	}
	debugf("reducer done claims=%d conflicts=%d", len(state.Claims), len(state.Conflicts))
	manifest.CanonicalClaimCount = len(state.Claims)
	manifest.ConflictCount = len(state.Conflicts)
	manifest.CanonicalStateHash = state.StateHash
	manifest.CanonicalStatePath = "canonical_state.json"
	if err := writeJSON(filepath.Join(outDir, manifest.CanonicalStatePath), state); err != nil {
		return BuildResult{}, err
	}
	plan := canonicalstate.BuildVerificationPlan(state, DefaultBudget)
	manifest.VerificationPlanPath = "verification_plan.json"
	if err := writeJSON(filepath.Join(outDir, manifest.VerificationPlanPath), plan); err != nil {
		return BuildResult{}, err
	}
	ledger.Append("REDUCE", canonicalstate.ReducerVersion, state.InputHash, state.StateHash, canonicalEvidenceAddresses(state))
	ledger.Append("UNCERTAINTY_PLAN", canonicalstate.ControllerVersion, state.StateHash, hashJSONValue(plan), nil)
	manifest.LedgerPath = "state_ledger.json"
	if err := writeJSON(filepath.Join(outDir, manifest.LedgerPath), ledger); err != nil {
		return BuildResult{}, err
	}
	manifest.ObjectCount = len(manifest.Documents) + manifest.PageCount + manifest.BlockCount + manifest.PageCount + 1
	debugf("building merkle objects=%d", manifest.ObjectCount)
	merkle := buildMerkleIndex(manifest)
	debugf("merkle built")
	manifest.StoreRootSHA256 = merkle.Root
	if err := writeJSON(filepath.Join(outDir, "manifest.json"), manifest); err != nil {
		return BuildResult{}, err
	}
	if err := writeJSON(filepath.Join(outDir, "index.json"), index); err != nil {
		return BuildResult{}, err
	}
	if err := writeJSON(filepath.Join(outDir, "graph.json"), graph); err != nil {
		return BuildResult{}, err
	}
	if err := writeJSON(filepath.Join(outDir, "merkle.json"), merkle); err != nil {
		return BuildResult{}, err
	}
	meta := FixedCarrierMetadata{CarrierID: carrierID, StoreRoot: manifest.StoreRootSHA256, SourceSHA256: manifest.SourceSHA256, PageCount: uint32(manifest.PageCount), BlockCount: uint32(manifest.BlockCount), DocumentCount: uint32(manifest.DocumentCount), ObjectCount: uint32(manifest.ObjectCount), GraphSignature: sketch.Bytes(), Flags: 0x0007}
	if err := writeJSON(filepath.Join(outDir, "fixed_carrier_meta.json"), meta); err != nil {
		return BuildResult{}, err
	}
	return BuildResult{Manifest: manifest, FixedCarrierMetadata: meta, StoreDir: outDir}, nil
}
