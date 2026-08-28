#!/usr/bin/env sh
set -eu
go test ./...
go run ./cmd/behaviorlab compile -spec profiles/origami/quantum-inspired-r0.json -out generated/origami-quantum-inspired-r0.generic.prompt.md
