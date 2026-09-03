package perceptenvelope

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// WriteJSON marshals v (indented, deterministic) to path and returns its
// sha256.
func WriteJSON(path string, v any) (string, error) {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return sha256Hex(body), nil
}

// SHA256OfFile returns the sha256 hex of a file's bytes.
func SHA256OfFile(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sha256Hex(body), nil
}

// DoctorReport is the pre-inference readiness gate for R1-A.
type DoctorReport struct {
	Schema              string   `json:"schema"`
	ExperimentID        string   `json:"experiment_id"`
	ModelIdentityOK     bool     `json:"model_identity_ok"`
	EndpointReachable   bool     `json:"endpoint_reachable"`
	EndpointModelListed bool     `json:"endpoint_model_listed"`
	SourcePoolSHA256    string   `json:"source_pool_sha256"`
	R1ABasesSHA256      string   `json:"r1a_bases_sha256"`
	R1BBasesSHA256      string   `json:"r1b_bases_sha256"`
	R1ABaseCount        int      `json:"r1a_base_count"`
	R1BBaseCount        int      `json:"r1b_base_count"`
	Disjoint            bool     `json:"r1a_r1b_disjoint"`
	ExpectedRecords     int      `json:"expected_r1a_records"`
	ContextLevels       int      `json:"context_levels_per_base"`
	Opcode              string   `json:"model_facing_opcode"`
	CueGeometryValid    bool     `json:"target_cues_valid"`
	LeakageAbsent       bool     `json:"expected_answer_leakage_absent"`
	Problems            []string `json:"problems"`
	ReadyR1A            bool     `json:"READY_R1A"`
}

const doctorSchema = "tlaloc.parrot-perceptual-envelope-r1.doctor.r1"

// DoctorInput carries what Doctor needs to check.
type DoctorInput struct {
	ModelIdentityPath string
	Endpoint          string
	Model             string
	SourcePool        SourcePool
	SourcePoolSHA     string
	R1A               Allocation
	R1B               Allocation
	R1ASHA            string
	R1BSHA            string
}

// Doctor runs every pre-inference check and returns the readiness report.
func Doctor(ctx context.Context, in DoctorInput) DoctorReport {
	r := DoctorReport{
		Schema: doctorSchema, ExperimentID: ExperimentID,
		SourcePoolSHA256: in.SourcePoolSHA, R1ABasesSHA256: in.R1ASHA, R1BBasesSHA256: in.R1BSHA,
		R1ABaseCount: len(in.R1A.Bases), R1BBaseCount: len(in.R1B.Bases),
		ContextLevels: len(AllContextLevels), Opcode: FrozenOpcode,
	}
	r.ExpectedRecords = r.R1ABaseCount * r.ContextLevels

	// model identity
	if body, err := os.ReadFile(in.ModelIdentityPath); err == nil {
		var mi map[string]any
		if json.Unmarshal(body, &mi) == nil {
			if ok, _ := mi["STAGE_1_MODEL_IDENTITY_OK"].(bool); ok {
				r.ModelIdentityOK = true
			}
		}
	}
	if !r.ModelIdentityOK {
		r.Problems = append(r.Problems, "MODEL_IDENTITY.json missing or STAGE_1_MODEL_IDENTITY_OK != true")
	}

	// endpoint
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, http.MethodGet, in.Endpoint+"/v1/models", nil)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		defer resp.Body.Close()
		r.EndpointReachable = resp.StatusCode == 200
		var ml struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if json.NewDecoder(resp.Body).Decode(&ml) == nil {
			for _, m := range ml.Data {
				if m.ID == in.Model {
					r.EndpointModelListed = true
				}
			}
		}
	}
	if !r.EndpointReachable {
		r.Problems = append(r.Problems, "endpoint /v1/models not reachable")
	}
	if !r.EndpointModelListed {
		r.Problems = append(r.Problems, fmt.Sprintf("model %q not listed at endpoint", in.Model))
	}

	// counts
	if r.R1ABaseCount != R1ASize {
		r.Problems = append(r.Problems, fmt.Sprintf("R1-A base count %d != %d", r.R1ABaseCount, R1ASize))
	}
	if r.R1BBaseCount != R1BSize {
		r.Problems = append(r.Problems, fmt.Sprintf("R1-B base count %d != %d", r.R1BBaseCount, R1BSize))
	}

	// disjoint
	aIDs := map[string]struct{}{}
	for _, b := range in.R1A.Bases {
		aIDs[b.Candidate.CandidateID] = struct{}{}
	}
	overlap := 0
	for _, b := range in.R1B.Bases {
		if _, ok := aIDs[b.Candidate.CandidateID]; ok {
			overlap++
		}
	}
	r.Disjoint = overlap == 0
	if !r.Disjoint {
		r.Problems = append(r.Problems, fmt.Sprintf("R1-A and R1-B share %d candidate_ids", overlap))
	}

	// cue geometry + leakage
	r.CueGeometryValid = true
	r.LeakageAbsent = true
	for _, b := range in.R1A.Bases {
		c := b.Candidate
		if !primaryTarget.MatchString(c.NormalizedTarget) {
			r.CueGeometryValid = false
			r.Problems = append(r.Problems, fmt.Sprintf("%s: gold %q not ^[0-9]{2,4}$", b.BaseID, c.NormalizedTarget))
		}
		tb, ln := c.TokenBBoxStore, c.Line.BBox
		if !(tb.X1 > 0 && tb.Y1 > 0 && tb.X2 < c.PageWidth && tb.Y2 < c.PageHeight && tb.X1 < tb.X2 && tb.Y1 < tb.Y2) {
			r.CueGeometryValid = false
			r.Problems = append(r.Problems, fmt.Sprintf("%s: token bbox clipped/degenerate", b.BaseID))
		}
		// token x-interval must sit within the line's x-span (pre-pad tolerance = one pad)
		pad := 0.5 * (ln.Y2 - ln.Y1)
		if tb.X1 < ln.X1-pad-1 || tb.X2 > ln.X2+pad+1 {
			r.CueGeometryValid = false
			r.Problems = append(r.Problems, fmt.Sprintf("%s: token x-interval outside line", b.BaseID))
		}
		if fields := len(splitFields(c.Line.Text)); fields > 1 {
			// one-digit-token rule already applied at scan; re-assert
			if countDigitTokens(c.Line.Text) != 1 {
				r.LeakageAbsent = false
				r.Problems = append(r.Problems, fmt.Sprintf("%s: line has %d digit tokens (want 1)", b.BaseID, countDigitTokens(c.Line.Text)))
			}
		}
	}

	r.ReadyR1A = r.ModelIdentityOK && r.EndpointReachable && r.EndpointModelListed &&
		r.R1ABaseCount == R1ASize && r.R1BBaseCount == R1BSize && r.Disjoint &&
		r.CueGeometryValid && r.LeakageAbsent && r.ExpectedRecords == R1ASize*len(AllContextLevels)
	return r
}

func splitFields(s string) []string { return fieldsFunc(s) }

func fieldsFunc(s string) []string {
	var out []string
	cur := ""
	for _, ch := range s {
		if ch == ' ' || ch == '\t' || ch == '\n' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func countDigitTokens(s string) int {
	n := 0
	for _, f := range fieldsFunc(s) {
		if anyDigit.MatchString(f) {
			n++
		}
	}
	return n
}
