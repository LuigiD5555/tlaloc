# Local preflight

Run from `behavior-lab/` after pulling `main`:

```bash
go test ./internal/tlaloque/groundingautomaton/...
go test ./internal/foldtest/swarmask/...
go run ./cmd/tlaloc-grounding-eval -input testdata/grounding/core-r0.jsonl
go run ./cmd/tlaloc-grounding-eval -input testdata/grounding/metamorphic-r0.jsonl
go test -bench Verify -benchmem ./internal/tlaloque/groundingautomaton/...
go test ./...
```

Do not tune against `ood-r0.jsonl` before the locked OOD set is restored. After the baseline runs are captured, add the 24 OOD triplets unchanged and compare deterministic automaton, keyword, embedding, semantic judge, and any recovered distilled model using the same cases.
