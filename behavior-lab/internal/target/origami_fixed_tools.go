package target

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

type FixedOrigamiExecutor struct {
	OrigamiBinary string
	Carrier       string
	StoreDir      string
}

type fixedCarrierDecoded struct {
	Schema        string `json:"schema"`
	Profile       string `json:"profile"`
	ToolProtocol  string `json:"tool_protocol"`
	AddressABI    string `json:"address_abi"`
	CarrierID     string `json:"carrier_id"`
	StoreRoot     string `json:"store_root_sha256"`
	SourceSHA256  string `json:"source_sha256"`
	PageCount     uint32 `json:"page_count"`
	BlockCount    uint32 `json:"block_count"`
	DocumentCount uint32 `json:"document_count"`
	ObjectCount   uint32 `json:"object_count"`
	CarrierDigest string `json:"carrier_digest_sha256"`
	VisualProbe   string `json:"visual_probe"`
}

type HostCapabilities struct {
	Vision        bool `json:"vision"`
	OriginalImage bool `json:"original_image"`
	NativeTools   bool `json:"native_tools"`
	TextBridge    bool `json:"text_bridge"`
	CodeExecution bool `json:"code_execution,omitempty"`
}

func OrigamiFixedTools() []ToolDefinition {
	object := func(properties map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	budget := map[string]any{"type": "integer", "minimum": 1, "maximum": 32768, "description": "Maximum model-facing token-equivalent for this access"}
	caps := object(map[string]any{"vision": map[string]any{"type": "boolean"}, "original_image": map[string]any{"type": "boolean"}, "native_tools": map[string]any{"type": "boolean"}, "text_bridge": map[string]any{"type": "boolean"}, "code_execution": map[string]any{"type": "boolean"}}, "vision", "original_image", "native_tools", "text_bridge")
	bootParams := map[string]any{"type": "object", "properties": map[string]any{
		"visual_probe_top":    map[string]any{"type": "string", "pattern": "^[01]{8}$", "description": "TOP T1 eight-cell challenge, visually read left-to-right; black=1 white=0"},
		"visual_probe_bottom": map[string]any{"type": "string", "pattern": "^[01]{8}$", "description": "BOTTOM duplicated T1 challenge; visually read independently"},
		"visual_probe":        map[string]any{"type": "string", "pattern": "^[01]{8}$", "description": "Deterministic-host compatibility probe only"},
		"capabilities":        caps,
	}, "anyOf": []any{map[string]any{"required": []string{"visual_probe_top", "visual_probe_bottom", "capabilities"}}, map[string]any{"required": []string{"visual_probe"}}}, "additionalProperties": false}
	return []ToolDefinition{
		{Type: "function", Function: ToolFunction{Name: "origami_boot", Description: "Boot Origami Fixed Carrier R2. Validates actual visual presence through duplicated T1 probes, carrier profile, T0 binding, address ABI, Tlaloc tool protocol and Merkle-root/store binding; negotiates host capabilities. OCR is not the memory mechanism.", Parameters: bootParams}},
		{Type: "function", Function: ToolFunction{Name: "origami_query", Description: "Search the bound R2 memory plane by block index plus bounded graph expansion. Returns verified exact block evidence and reopenable addresses without exposing the corpus globally.", Parameters: object(map[string]any{"query": map[string]any{"type": "string"}, "budget": budget, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 20}}, "query")}},
		{Type: "function", Function: ToolFunction{Name: "origami_expand", Description: "Open one page/block address. Exact content is CID and Merkle verified; non-exact page unfolding returns exact verified blocks rather than truncating an exact claim.", Parameters: object(map[string]any{"address": map[string]any{"type": "string"}, "fidelity": map[string]any{"type": "string", "enum": []string{"detail", "evidence", "exact"}}, "budget": budget}, "address")}},
		{Type: "function", Function: ToolFunction{Name: "origami_verify", Description: "Verify every source, page and block CID, recompute the Merkle root, and verify its binding to origami.png.", Parameters: object(map[string]any{})}},
	}
}

func (e FixedOrigamiExecutor) VisualProbe(ctx context.Context) (string, error) {
	d, err := e.decode(ctx)
	if err != nil {
		return "", err
	}
	return d.VisualProbe, nil
}

func (e FixedOrigamiExecutor) Execute(ctx context.Context, name string, arguments json.RawMessage) (string, error) {
	m, idx, err := pdfmemory.Load(e.StoreDir)
	if err != nil {
		return "", err
	}
	switch name {
	case "origami_boot":
		var a struct {
			VisualProbeTop    string           `json:"visual_probe_top"`
			VisualProbeBottom string           `json:"visual_probe_bottom"`
			VisualProbe       string           `json:"visual_probe"`
			Capabilities      HostCapabilities `json:"capabilities"`
		}
		if err := json.Unmarshal(arguments, &a); err != nil {
			return "", err
		}
		d, err := e.decode(ctx)
		if err != nil {
			return "", err
		}
		probeMode := "dual"
		recovered := false
		if a.VisualProbeTop != "" || a.VisualProbeBottom != "" {
			if !a.Capabilities.Vision {
				return "", fmt.Errorf("ORIGAMI_IMAGE_UNAVAILABLE: host declared vision=false")
			}
			if len(a.VisualProbeTop) != 8 || len(a.VisualProbeBottom) != 8 {
				return "", fmt.Errorf("ORIGAMI_IMAGE_PROBE_FAILED: both T1 rows are required")
			}
			top := a.VisualProbeTop == d.VisualProbe
			bottom := a.VisualProbeBottom == d.VisualProbe
			switch {
			case top && bottom:
			case top && hammingBits(a.VisualProbeBottom, d.VisualProbe) <= 1:
				recovered = true
			case bottom && hammingBits(a.VisualProbeTop, d.VisualProbe) <= 1:
				recovered = true
			default:
				return "", fmt.Errorf("ORIGAMI_IMAGE_PROBE_FAILED: duplicated T1 challenge mismatch")
			}
		} else {
			probeMode = "deterministic_host"
			if a.VisualProbe == "" || a.VisualProbe != d.VisualProbe {
				return "", fmt.Errorf("ORIGAMI_IMAGE_PROBE_FAILED: deterministic probe mismatch")
			}
		}
		if err := validateBinding(d, m); err != nil {
			return "", err
		}
		mode := "HYBRID"
		if probeMode == "deterministic_host" {
			mode = "COMPUTATIONAL_TEST"
		} else if !a.Capabilities.NativeTools && !a.Capabilities.TextBridge {
			mode = "NATIVE_CONTROL_ONLY"
		}
		spaces := map[string]string{"page": "ohf://" + m.CarrierID + "/pages/<six-digit-page> (single-document alias)", "document": "ohf://" + m.CarrierID + "/docs/<doc-id>/pages/<six-digit-page>", "source": "ohf://" + m.CarrierID + "/source/<doc-id>", "block": "<page-address>/blocks/<four-digit-block>"}
		out := map[string]any{"status": "BOOT_OK", "schema": d.Schema, "profile": d.Profile, "tool_protocol": d.ToolProtocol, "address_abi": d.AddressABI, "carrier_id": m.CarrierID, "carrier_id_hash": d.CarrierID, "carrier_digest_sha256": d.CarrierDigest, "store_root_sha256": d.StoreRoot, "source_sha256": d.SourceSHA256, "document_count": d.DocumentCount, "page_count": d.PageCount, "block_count": d.BlockCount, "object_count": d.ObjectCount, "region_count": m.RegionCount, "candidate_count": m.CandidateCount, "canonical_claim_count": m.CanonicalClaimCount, "conflict_count": m.ConflictCount, "canonical_state_hash_sha256": m.CanonicalStateHash, "address_spaces": spaces, "ocr_required": false, "t0_plaintext_boot": true, "visual_probe_verified": true, "visual_probe_mode": probeMode, "visual_probe_recovered": recovered, "host_capabilities": a.Capabilities, "mode": mode, "native_without_tools": []string{"read T0", "verify visual presence through T1", "interpret T2 root spaces/graph landmarks", "recognize T3 machine profile"}, "external_memory_requires_tools": true, "false_exact": 0}
		return marshal(out), nil
	case "origami_query":
		var a struct {
			Query  string `json:"query"`
			Budget int    `json:"budget"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(arguments, &a); err != nil {
			return "", err
		}
		if a.Query == "" {
			return "", fmt.Errorf("origami_query requires query")
		}
		if a.Budget == 0 {
			a.Budget = pdfmemory.DefaultBudget
		}
		if err := pdfmemory.ValidateBudget(a.Budget); err != nil {
			return "", err
		}
		p, err := pdfmemory.Query(e.StoreDir, m, idx, a.Query, a.Budget, a.Limit)
		if err != nil {
			return "", err
		}
		return marshal(p), nil
	case "origami_expand":
		var a struct {
			Address  string `json:"address"`
			Fidelity string `json:"fidelity"`
			Budget   int    `json:"budget"`
		}
		if err := json.Unmarshal(arguments, &a); err != nil {
			return "", err
		}
		if a.Address == "" {
			return "", fmt.Errorf("origami_expand requires address")
		}
		if a.Budget == 0 {
			a.Budget = pdfmemory.DefaultBudget
		}
		if err := pdfmemory.ValidateBudget(a.Budget); err != nil {
			return "", err
		}
		p, err := pdfmemory.Expand(e.StoreDir, m, a.Address, a.Fidelity, a.Budget)
		if err != nil {
			return "", err
		}
		return marshal(p), nil
	case "origami_verify":
		d, err := e.decode(ctx)
		if err != nil {
			return "", err
		}
		if err := validateBinding(d, m); err != nil {
			return "", err
		}
		v, err := pdfmemory.VerifyStore(e.StoreDir, m)
		if err != nil {
			return "", err
		}
		v["carrier_digest_sha256"] = d.CarrierDigest
		v["carrier_binding"] = "VERIFIED"
		v["profile"] = d.Profile
		return marshal(v), nil
	default:
		return "", fmt.Errorf("undeclared fixed Origami tool %q", name)
	}
}

func (e FixedOrigamiExecutor) decode(ctx context.Context) (fixedCarrierDecoded, error) {
	bin := e.OrigamiBinary
	if bin == "" {
		bin = "origami-fixed-carrier"
	}
	if e.Carrier == "" {
		return fixedCarrierDecoded{}, fmt.Errorf("carrier path required")
	}
	out, err := exec.CommandContext(ctx, bin, "-mode", "decode", "-in", e.Carrier).CombinedOutput()
	if err != nil {
		return fixedCarrierDecoded{}, fmt.Errorf("decode carrier: %w: %s", err, strings.TrimSpace(string(out)))
	}
	var d fixedCarrierDecoded
	if err := json.Unmarshal(out, &d); err != nil {
		return d, fmt.Errorf("decode carrier output: %w: %s", err, string(out))
	}
	return d, nil
}
func validateBinding(d fixedCarrierDecoded, m pdfmemory.Manifest) error {
	if d.Schema != "origami.fixed-carrier.r2" {
		return fmt.Errorf("unexpected carrier schema %q", d.Schema)
	}
	if d.Profile != "origami.fixed-carrier.r2.profile-1" {
		return fmt.Errorf("unexpected carrier profile %q", d.Profile)
	}
	if d.ToolProtocol != pdfmemory.ToolProtocol {
		return fmt.Errorf("tool protocol mismatch")
	}
	if d.AddressABI != pdfmemory.AddressSchema {
		return fmt.Errorf("address ABI mismatch")
	}
	if d.StoreRoot != m.StoreRootSHA256 {
		return fmt.Errorf("store root does not match carrier")
	}
	if d.SourceSHA256 != m.SourceSHA256 {
		return fmt.Errorf("source hash does not match carrier")
	}
	if int(d.PageCount) != m.PageCount || int(d.BlockCount) != m.BlockCount || int(d.DocumentCount) != m.DocumentCount || int(d.ObjectCount) != m.ObjectCount {
		return fmt.Errorf("store counts do not match carrier")
	}
	sum := sha256.Sum256([]byte(m.CarrierID))
	if d.CarrierID != hex.EncodeToString(sum[:]) {
		return fmt.Errorf("carrier id binding mismatch")
	}
	return nil
}
func hammingBits(a, b string) int {
	if len(a) != len(b) {
		return 99
	}
	d := 0
	for i := range a {
		if a[i] != b[i] {
			d++
		}
	}
	return d
}
func marshal(v any) string { b, _ := json.Marshal(v); return string(b) }
