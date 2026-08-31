# Temporal Native Debug R0

Status: experimental test-only diagnostic workflow.

## Purpose

When an Origami Native benchmark question fails, repeat the same question under the same semantic condition with `diagnostic_mode=true`. The diagnostic repetition does **not** count as primary Native/self-bootstrap evidence. It exists only to locate the failure frontier.

The target model must not provide private reasoning or chain-of-thought. It reports only observable protocol execution state.

## Workflow

1. Run the clean trial (`NATIVE_PNG_ONLY`, `R4_ASSISTED`, or degraded equivalent).
2. Preserve the answer verbatim.
3. If a question fails or is ambiguous, repeat that exact question with the test-only diagnostic suffix printed by:

```bash
tlaloc-temporal-bench -print-debug-instruction
```

4. Preserve the complete response verbatim, including the final `ORIGAMI_DEBUG_R0=` line.
5. Evaluate the campaign with `tlaloc-temporal-bench`.
6. Inspect `debug_reports` and `debug_summary`.

## Observable trace

The model appends one JSON object after `ORIGAMI_DEBUG_R0=` with:

- `status`
- `last_completed_stage`
- `selected_codec`
- `last_instruction`
- `next_instruction`
- `failure_code`
- `evidence_refs`
- `confidence`
- optional `note` (max 160 characters)

The trace reports protocol checkpoints, not hidden reasoning.

## Stages

```text
NONE
  -> BOOT
  -> ROSETTA
  -> CODEC_DISCOVERY
  -> T2_NAVIGATION
  -> SEMANTIC_DECODE
  -> TEMPORAL_ROUTE
  -> TEMPORAL_STEP
  -> EXACT_BOUNDARY
  -> ANSWER
```

## Typical failure interpretation

- `BOOT_NOT_FOUND`: visual bootstrap was not recognized.
- `ROSETTA_NOT_FOUND`: BOOT may be visible, but the model did not enter the declared grammar.
- `CODEC_NOT_FOUND`: grammar was understood but no suitable declared decoder/encoder entrypoint was found.
- `T2_NOT_FOUND`: the model reached ROSETTA but could not locate the semantic/temporal supergraph.
- `SEMANTIC_EVIDENCE_INSUFFICIENT`: T2 was reached, but the required rule/relation could not be recovered.
- `TEMPORAL_RULE_AMBIGUOUS`: semantic rules were found but could not be deterministically applied.
- `CHECKPOINT_NOT_FOUND`: timeline routing failed at checkpoint discovery.
- `EXACT_DECODER_REQUIRED`: correct boundary for an exact query when no exact decoder was executed.
- `CAPABILITY_MISMATCH`: the declared operation exceeds receiver capability.

## Tlaloc summary fields

`debug_summary` exposes:

- `trace_coverage`: fraction of diagnostic responses with a parseable trace.
- `trace_consistency_score`: agreement between answer success/failure and reported trace status.
- `dominant_failure_frontier`: most common last completed stage among non-PASS traces.
- `earliest_failure_frontier`: earliest stage at which any diagnostic response stopped.
- `furthest_completed_stage`: furthest observable stage reached in the diagnostic trial.
- `most_common_failure_code`: dominant declared stop reason.
- missing/invalid trace counts and answer/trace mismatch count.

## Scientific boundary

Diagnostic mode can change model behavior because it adds an output-format instruction. Therefore:

```text
DIAGNOSTIC_RESULT != NATIVE_SELF_BOOTSTRAP_EVIDENCE
```

Use it to explain a failure, never to replace the clean score.
