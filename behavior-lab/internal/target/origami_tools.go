package target

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type OrigamiCLIExecutor struct {
	Binary  string
	Carrier string
	Packet  string
}

func OrigamiHybridTools() []ToolDefinition {
	object := func(properties map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	query := map[string]any{"type": "string", "description": "Origami key or address"}
	relation := map[string]any{"type": "string", "description": "Declared relation name such as depends"}
	depth := map[string]any{"type": "integer", "minimum": 0, "maximum": 1024, "description": "Maximum relation depth"}
	return []ToolDefinition{
		{Type: "function", Function: ToolFunction{Name: "origami_boot", Description: "Read the carrier BOOT structure before interpreting carrier-local semantics.", Parameters: object(map[string]any{})}},
		{Type: "function", Function: ToolFunction{Name: "origami_lookup", Description: "Lookup one declared Origami key or address without a global semantic scan.", Parameters: object(map[string]any{"query": query}, "query")}},
		{Type: "function", Function: ToolFunction{Name: "origami_follow", Description: "Follow one declared relation from a key/address to bounded depth.", Parameters: object(map[string]any{"query": query, "relation": relation, "depth": depth}, "query", "relation", "depth")}},
		{Type: "function", Function: ToolFunction{Name: "origami_trace", Description: "Trace one declared relation path from a key/address to bounded depth.", Parameters: object(map[string]any{"query": query, "relation": relation, "depth": depth}, "query", "relation", "depth")}},
		{Type: "function", Function: ToolFunction{Name: "origami_verify", Description: "Verify the image-backed carrier memory commitment and return its evidence reference.", Parameters: object(map[string]any{})}},
		{Type: "function", Function: ToolFunction{Name: "origami_stop", Description: "Stop Origami navigation when sufficient evidence has been obtained.", Parameters: object(map[string]any{})}},
	}
}

func (e OrigamiCLIExecutor) Execute(ctx context.Context, name string, arguments json.RawMessage) (string, error) {
	binary := e.Binary
	if binary == "" { binary = "origami-hybrid-tool" }
	if e.Carrier == "" || e.Packet == "" { return "", fmt.Errorf("Origami carrier and packet paths are required") }
	var args struct {
		Query    string `json:"query"`
		Relation string `json:"relation"`
		Depth    int    `json:"depth"`
	}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &args); err != nil { return "", err }
	}

	op := ""
	cmdArgs := []string{"-carrier", e.Carrier, "-packet", e.Packet}
	switch name {
	case "origami_boot":
		op = "BOOT"
	case "origami_lookup":
		if args.Query == "" { return "", fmt.Errorf("origami_lookup requires query") }
		op = "LOOKUP"
		cmdArgs = append(cmdArgs, "-query", args.Query)
	case "origami_follow", "origami_trace":
		if args.Query == "" || args.Relation == "" { return "", fmt.Errorf("%s requires query and relation", name) }
		if args.Depth < 0 || args.Depth > 1024 { return "", fmt.Errorf("depth out of bounds: %d", args.Depth) }
		if name == "origami_follow" { op = "FOLLOW" } else { op = "TRACE" }
		cmdArgs = append(cmdArgs, "-query", args.Query, "-relation", args.Relation, "-depth", strconv.Itoa(args.Depth))
	case "origami_verify":
		op = "VERIFY"
	case "origami_stop":
		op = "STOP"
	default:
		return "", fmt.Errorf("undeclared Origami tool %q", name)
	}
	cmdArgs = append(cmdArgs, "-op", op)
	cmd := exec.CommandContext(ctx, binary, cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil { return "", fmt.Errorf("%s failed: %w: %s", binary, err, strings.TrimSpace(string(out))) }
	return strings.TrimSpace(string(out)), nil
}
