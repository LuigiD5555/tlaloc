package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tlaloc.local/behaviorlab/internal/lfm2boundary"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func main() {
	if len(os.Args) != 2 {
		fail(fmt.Errorf("usage: tlaloc-lfm2-worker <rosetta|cells|transitions|timeline|consolidate>"))
	}
	role := strings.ToLower(strings.TrimSpace(os.Args[1]))
	var req tlaloque.CapabilityRequest
	dec := json.NewDecoder(os.Stdin)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		fail(fmt.Errorf("decode request: %w", err))
	}
	ctx := target.WithGenerationPolicy(context.Background(), target.GenerationPolicy{Guard: target.RepetitionGuard{MinRepeatedLines: 16}})
	var resp tlaloque.CapabilityResponse
	var err error
	switch role {
	case "rosetta", "cells", "transitions", "timeline":
		resp, err = lfm2boundary.RunVisualSpecialist(ctx, role, req)
	case "consolidate":
		resp, err = lfm2boundary.RunConsolidator(ctx, req)
	default:
		err = fmt.Errorf("unknown responsibility %q", role)
	}
	if err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		fail(err)
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
