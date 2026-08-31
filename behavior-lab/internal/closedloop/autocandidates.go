package closedloop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/visualsearch"
)

type candidateBuilderCapabilities struct {
	Schema             string   `json:"schema"`
	ParentProfiles     []string `json:"parent_profiles"`
	SupportedKinds     []string `json:"supported_kinds"`
	UnsupportedKinds   []string `json:"unsupported_kinds"`
	ExactPlaneMutation bool     `json:"exact_plane_mutation"`
	MaxMutations       int      `json:"max_mutations"`
}

func normalizeAutoConfig(cfg *Config) {
	if cfg.AutoCandidatesPerGeneration <= 0 {
		cfg.AutoCandidatesPerGeneration = 4
	}
	if strings.TrimSpace(cfg.AutoCandidateBaseProfileID) == "" {
		cfg.AutoCandidateBaseProfileID = "origami.temporal-carrier.r0.profile-1"
	}
}

func validateAutoConfig(ctx context.Context, p prepared) (candidateBuilderCapabilities, error) {
	if !p.cfg.AutoCandidates {
		return candidateBuilderCapabilities{}, nil
	}
	if len(p.cfg.CandidateBuilder) == 0 || strings.TrimSpace(p.cfg.CandidateBuilder[0]) == "" {
		return candidateBuilderCapabilities{}, fmt.Errorf("auto_candidates requires candidate_builder")
	}
	caps, err := p.queryCandidateBuilderCapabilities(ctx)
	if err != nil {
		return candidateBuilderCapabilities{}, err
	}
	if caps.Schema != "origami.experimental-candidate.r0.capabilities" {
		return candidateBuilderCapabilities{}, fmt.Errorf("unexpected candidate builder capability schema %q", caps.Schema)
	}
	if caps.ExactPlaneMutation {
		return candidateBuilderCapabilities{}, fmt.Errorf("candidate builder may mutate exact plane; refusing automatic use")
	}
	if !containsString(caps.ParentProfiles, p.cfg.AutoCandidateBaseProfileID) {
		return candidateBuilderCapabilities{}, fmt.Errorf("candidate builder does not support parent profile %q", p.cfg.AutoCandidateBaseProfileID)
	}
	if len(caps.SupportedKinds) == 0 {
		return candidateBuilderCapabilities{}, fmt.Errorf("candidate builder reports no supported mutation kinds")
	}
	return caps, nil
}

func (p prepared) queryCandidateBuilderCapabilities(ctx context.Context) (candidateBuilderCapabilities, error) {
	args := append([]string(nil), p.cfg.CandidateBuilder[1:]...)
	args = append(args, "capabilities")
	cmd := exec.CommandContext(ctx, p.cfg.CandidateBuilder[0], args...)
	body, err := cmd.CombinedOutput()
	if err != nil {
		return candidateBuilderCapabilities{}, fmt.Errorf("candidate builder capabilities: %v: %s", err, strings.TrimSpace(string(body)))
	}
	var caps candidateBuilderCapabilities
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&caps); err != nil {
		return candidateBuilderCapabilities{}, fmt.Errorf("candidate builder capabilities JSON: %w", err)
	}
	return caps, nil
}

func (p prepared) autoCandidateConfigs(plan adaptivesearch.Plan, caps candidateBuilderCapabilities, parent SpecimenConfig, outputDir string) ([]CandidateConfig, error) {
	if !p.cfg.AutoCandidates {
		return nil, nil
	}
	meta, err := readPNGMeta(parent.PNG)
	if err != nil {
		return nil, fmt.Errorf("auto candidate parent: %w", err)
	}
	supported := map[string]bool{}
	for _, kind := range caps.SupportedKinds {
		supported[strings.ToUpper(strings.TrimSpace(kind))] = true
	}
	seen := map[string]bool{}
	out := []CandidateConfig{}
	for _, suggestion := range plan.SuggestedMutations {
		kind := strings.ToUpper(strings.TrimSpace(string(suggestion.Kind)))
		if !supported[kind] {
			continue
		}
		mutation := visualsearch.Mutation{
			Kind:         suggestion.Kind,
			Target:       suggestion.Target,
			Value:        suggestion.Value,
			Rationale:    suggestion.Rationale,
			Experimental: true,
		}
		fingerprint, err := autoCandidateFingerprint(parent.ID, meta.sha, mutation)
		if err != nil {
			return nil, err
		}
		id := "auto-" + strings.ToLower(kind) + "-" + fingerprint[:12]
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, CandidateConfig{
			ID:               id,
			PNG:              filepath.Join(outputDir, "auto-candidates", id+".png"),
			BaseProfileID:    p.cfg.AutoCandidateBaseProfileID,
			ParentSpecimenID: parent.ID,
			Mutations:        []visualsearch.Mutation{mutation},
			BuildCommand:     append([]string(nil), p.cfg.CandidateBuilder...),
		})
		if len(out) >= p.cfg.AutoCandidatesPerGeneration {
			break
		}
	}
	return out, nil
}

func autoCandidateFingerprint(parentID, parentSHA string, mutation visualsearch.Mutation) (string, error) {
	payload := struct {
		ParentID  string                `json:"parent_id"`
		ParentSHA string                `json:"parent_sha256"`
		Mutation  visualsearch.Mutation `json:"mutation"`
	}{ParentID: parentID, ParentSHA: parentSHA, Mutation: mutation}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func mergeCandidateConfigs(manual, automatic []CandidateConfig, tested map[string]bool, incumbentID string) ([]CandidateConfig, error) {
	byID := map[string]CandidateConfig{}
	order := []string{}
	add := func(c CandidateConfig) error {
		if tested[c.ID] {
			return nil
		}
		if c.ParentSpecimenID != "" && c.ParentSpecimenID != incumbentID {
			return nil
		}
		if existing, ok := byID[c.ID]; ok {
			a, _ := json.Marshal(existing)
			b, _ := json.Marshal(c)
			if string(a) != string(b) {
				return fmt.Errorf("candidate id collision %q", c.ID)
			}
			return nil
		}
		byID[c.ID] = c
		order = append(order, c.ID)
		return nil
	}
	for _, c := range manual {
		if err := add(c); err != nil { return nil, err }
	}
	for _, c := range automatic {
		if err := add(c); err != nil { return nil, err }
	}
	sort.Strings(order)
	out := make([]CandidateConfig, 0, len(order))
	for _, id := range order { out = append(out, byID[id]) }
	return out, nil
}

func visualCandidatesFromConfigs(configs []CandidateConfig) ([]visualsearch.Candidate, map[string]CandidateConfig) {
	out := make([]visualsearch.Candidate, 0, len(configs))
	by := map[string]CandidateConfig{}
	for _, c := range configs {
		out = append(out, c.VisualCandidate())
		by[c.ID] = c
	}
	return out, by
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(want)) { return true }
	}
	return false
}
