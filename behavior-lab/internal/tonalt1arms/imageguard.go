package tonalt1arms

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// OperandImageRecord mirrors one entry of TONAL_T1_IMAGE_MANIFEST_FINAL.json's
// "operands" array -- the frozen per-(workflow,role) operand image hash.
type OperandImageRecord struct {
	WorkflowID         string `json:"workflow_id"`
	Role               string `json:"role"`
	CandidateID        string `json:"candidate_id"`
	Page               int    `json:"page"`
	RegionID           string `json:"region_id"`
	PageSHA256         string `json:"page_sha256"`
	Run1PreparedSHA256 string `json:"run1_prepared_sha256"`
	Run2PreparedSHA256 string `json:"run2_prepared_sha256"`
	Equal              bool   `json:"equal"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	PreparedBytes      int    `json:"prepared_bytes"`
}

// CompositeImageRecord mirrors one entry of
// TONAL_T1_IMAGE_MANIFEST_FINAL.json's "composites" array (equivalently,
// TONAL_T1_ARM_A_COMPOSITE_MANIFEST.json's "records" array) -- the frozen
// per-workflow Arm-A composite hash.
type CompositeImageRecord struct {
	WorkflowID          string   `json:"workflow_id"`
	OrderedRoles        []string `json:"ordered_roles"`
	OrderedCandidateIDs []string `json:"ordered_candidate_ids"`
	OperandSHA256       []string `json:"operand_sha256"`
	Width               int      `json:"width"`
	Height              int      `json:"height"`
	PaddingPx           int      `json:"padding_px"`
	Run1CompositeSHA256 string   `json:"run1_composite_sha256"`
	Run2CompositeSHA256 string   `json:"run2_composite_sha256"`
	Equal               bool     `json:"equal"`
}

// ImageManifest is the frozen TONAL_T1_IMAGE_MANIFEST_FINAL.json artifact:
// the single authoritative source for both the 144 operand-image hashes and
// the 60 Arm-A composite hashes. This is a READ-ONLY frozen artifact --
// nothing in this package ever writes to it.
type ImageManifest struct {
	Schema             string                 `json:"schema"`
	ReadyTonalT1Images bool                   `json:"READY_TONAL_T1_IMAGES"`
	WorkflowCount      int                    `json:"workflows"`
	OperandAssignments int                    `json:"operand_assignments"`
	Operands           []OperandImageRecord   `json:"operands"`
	Composites         []CompositeImageRecord `json:"composites"`
}

// LoadImageManifest loads and parses the frozen image manifest. It performs
// no verification of its own beyond JSON parsing -- StartupImageSweep and
// Verify* are what actually check bytes against it.
func LoadImageManifest(path string) (*ImageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest ImageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// operandKey/compositeKey give the manifest's lookup convention:
// (workflow_id, role) for operands, workflow_id alone for composites --
// matching internal/tonalt1.GenerateOperandPresentations' own
// workflow_id+"|"+role map-key convention.
func operandKey(workflowID, role string) string { return workflowID + "|" + role }

// operandIndex/compositeIndex build lookup maps once, since a manifest is
// loaded once per run and then consulted per-call.
func (m *ImageManifest) operandIndex() map[string]OperandImageRecord {
	idx := make(map[string]OperandImageRecord, len(m.Operands))
	for _, rec := range m.Operands {
		idx[operandKey(rec.WorkflowID, rec.Role)] = rec
	}
	return idx
}

func (m *ImageManifest) compositeIndex() map[string]CompositeImageRecord {
	idx := make(map[string]CompositeImageRecord, len(m.Composites))
	for _, rec := range m.Composites {
		idx[rec.WorkflowID] = rec
	}
	return idx
}

// sha256Hex computes the lowercase hex SHA-256 of data.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ImageHashMismatchError is the required typed fail-closed error for any
// operand/composite byte mismatch against the frozen manifest -- never a
// warning, always a hard error a caller must handle by refusing to proceed.
type ImageHashMismatchError struct {
	Kind       string // "OPERAND" | "COMPOSITE"
	WorkflowID string
	Role       string // empty for composites
	Reason     string // "MISSING_MANIFEST_RECORD" | "HASH_MISMATCH"
	Want       string
	Got        string
}

func (e *ImageHashMismatchError) Error() string {
	if e.Role != "" {
		return fmt.Sprintf("IMAGE_HASH_MISMATCH: operand (workflow=%s role=%s): %s (want %s, got %s)", e.WorkflowID, e.Role, e.Reason, e.Want, e.Got)
	}
	return fmt.Sprintf("IMAGE_HASH_MISMATCH: composite (workflow=%s): %s (want %s, got %s)", e.WorkflowID, e.Reason, e.Want, e.Got)
}

// VerifyOperandImage computes SHA-256 of bytes and compares it against the
// frozen manifest's record for (workflowID, role). A missing record or hash
// mismatch is a hard, typed IMAGE_HASH_MISMATCH error -- never a warning.
// The manifest records two independent generation runs' hashes
// (run1_prepared_sha256/run2_prepared_sha256, already proven equal when the
// manifest was frozen); this checks against run1's hash, since both are
// identical whenever Equal is true, and Equal being false would itself be a
// frozen-manifest integrity problem this function also refuses to trust.
func VerifyOperandImage(manifest *ImageManifest, workflowID, role string, data []byte) error {
	idx := manifest.operandIndex()
	rec, ok := idx[operandKey(workflowID, role)]
	if !ok {
		return &ImageHashMismatchError{Kind: "OPERAND", WorkflowID: workflowID, Role: role, Reason: "MISSING_MANIFEST_RECORD"}
	}
	if !rec.Equal || rec.Run1PreparedSHA256 != rec.Run2PreparedSHA256 {
		return &ImageHashMismatchError{Kind: "OPERAND", WorkflowID: workflowID, Role: role, Reason: "MANIFEST_INTERNALLY_INCONSISTENT", Want: rec.Run1PreparedSHA256, Got: rec.Run2PreparedSHA256}
	}
	got := sha256Hex(data)
	if got != rec.Run1PreparedSHA256 {
		return &ImageHashMismatchError{Kind: "OPERAND", WorkflowID: workflowID, Role: role, Reason: "HASH_MISMATCH", Want: rec.Run1PreparedSHA256, Got: got}
	}
	return nil
}

// VerifyComposite is VerifyOperandImage's composite-image counterpart,
// keyed by workflowID alone.
func VerifyComposite(manifest *ImageManifest, workflowID string, data []byte) error {
	idx := manifest.compositeIndex()
	rec, ok := idx[workflowID]
	if !ok {
		return &ImageHashMismatchError{Kind: "COMPOSITE", WorkflowID: workflowID, Reason: "MISSING_MANIFEST_RECORD"}
	}
	if !rec.Equal || rec.Run1CompositeSHA256 != rec.Run2CompositeSHA256 {
		return &ImageHashMismatchError{Kind: "COMPOSITE", WorkflowID: workflowID, Reason: "MANIFEST_INTERNALLY_INCONSISTENT", Want: rec.Run1CompositeSHA256, Got: rec.Run2CompositeSHA256}
	}
	got := sha256Hex(data)
	if got != rec.Run1CompositeSHA256 {
		return &ImageHashMismatchError{Kind: "COMPOSITE", WorkflowID: workflowID, Reason: "HASH_MISMATCH", Want: rec.Run1CompositeSHA256, Got: got}
	}
	return nil
}

// StartupSweepResult is the eager, complete, front-loaded outcome of
// verifying every one of the 144 operand images and 60 Arm-A composites
// before any model-identity preflight or inference is attempted (task
// correction E). OperandImages/CompositeImages carry the EXACT verified
// byte bundle the sweep hashed -- executors are constructed from this
// bundle directly (correction I) and never re-materialize or independently
// resolve an image mid-run; VerifyOperandImage/VerifyComposite are then
// used only to re-hash these SAME bytes immediately before each adapter
// call, as defense against in-process mutation between startup and call.
type StartupSweepResult struct {
	OperandHashesValid   string // "144/144" style summary
	CompositeHashesValid string
	AllValid             bool
	Failures             []string
	OperandImages        map[string][]byte // keyed by workflowID+"|"+role
	CompositeImages      map[string][]byte // keyed by workflowID
}

// StartupImageSweep materializes every operand presentation and Arm-A
// composite via the caller-supplied deterministic materializer functions
// (the real internal/tonalt1.RasterizePages -> GenerateOperandPresentations
// -> GenerateArmAComposites pipeline in the live CLI; a fake in offline
// tests) and verifies all resulting hashes against the frozen manifest
// BEFORE returning success. Any single mismatch fails the whole sweep
// closed (AllValid=false, Failures populated) -- there is no partial-success
// return; a live caller must refuse to proceed to model-identity preflight
// or inference if AllValid is false.
func StartupImageSweep(
	manifest *ImageManifest,
	operandImages map[string][]byte, // keyed by workflowID+"|"+role, already materialized by the caller
	compositeImages map[string][]byte, // keyed by workflowID, already materialized by the caller
) (StartupSweepResult, error) {
	result := StartupSweepResult{
		OperandImages:   operandImages,
		CompositeImages: compositeImages,
	}

	operandChecked, operandTotal := 0, len(manifest.Operands)
	for _, rec := range manifest.Operands {
		key := operandKey(rec.WorkflowID, rec.Role)
		data, ok := operandImages[key]
		if !ok {
			result.Failures = append(result.Failures, fmt.Sprintf("operand %s: no materialized image supplied", key))
			continue
		}
		if err := VerifyOperandImage(manifest, rec.WorkflowID, rec.Role, data); err != nil {
			result.Failures = append(result.Failures, err.Error())
			continue
		}
		operandChecked++
	}

	compositeChecked, compositeTotal := 0, len(manifest.Composites)
	for _, rec := range manifest.Composites {
		data, ok := compositeImages[rec.WorkflowID]
		if !ok {
			result.Failures = append(result.Failures, fmt.Sprintf("composite %s: no materialized image supplied", rec.WorkflowID))
			continue
		}
		if err := VerifyComposite(manifest, rec.WorkflowID, data); err != nil {
			result.Failures = append(result.Failures, err.Error())
			continue
		}
		compositeChecked++
	}

	result.OperandHashesValid = fmt.Sprintf("%d/%d", operandChecked, operandTotal)
	result.CompositeHashesValid = fmt.Sprintf("%d/%d", compositeChecked, compositeTotal)
	result.AllValid = operandChecked == operandTotal && compositeChecked == compositeTotal && len(result.Failures) == 0

	if !result.AllValid {
		return result, fmt.Errorf("tonalt1arms: StartupImageSweep: FAILED (%s operands, %s composites) -- %d failure(s), first: %s",
			result.OperandHashesValid, result.CompositeHashesValid, len(result.Failures), firstOrEmpty(result.Failures))
	}
	return result, nil
}

func firstOrEmpty(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
