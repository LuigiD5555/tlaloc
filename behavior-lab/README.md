# Tlaloc Behavior Lab

Behavior Lab is Tlaloc's R0 experiment for compiling formal behavior contracts into model-facing artifacts, testing them against target models and using bounded **Tlaloque** to propose repairs from structured failures.

The original compiler path is:

```text
BehaviorSpec -> PromptIR -> prompt -> target model -> findings -> Tlaloque repairs
```

The experimental Origami Hybrid Receiver path adds a second stage:

```text
Origami receiver contract
  -> rich swarm/Tlaloque receiver behavior
  -> successful semantic trace
  -> internal/distill
  -> deterministic receiver proposal
       UniversalPrompt
       + BOOT strategy
       + Rosetta constraints
       + MicroAgent rules
  -> fitness tournament
  -> Origami import / validation / storage
```

The point of distillation is not to preserve private chain-of-thought. It preserves externally relevant state transitions, **actions**, effects and provenance so a complex swarm can be replaced at receive time by simpler deterministic local behavior where evidence shows that is safe.

## Terms

- `internal/reference` calculates deterministic expected states for the original coherent-state profile.
- `internal/tlaloque` contains bounded specialist agents that diagnose findings and propose prompt patches.
- `internal/distill` contains the experimental swarm-trace -> deterministic receiver-candidate path and receiver tournament.
- `internal/target` now contains both the original text transport and an experimental OpenAI-compatible multimodal/tool receiver loop.
- `profiles/origami/hybrid-receiver-r0.json` defines the experimental receiver search objective and hard safety gates.
- `tlaloc.origami-hybrid-artifact-set.r0` is the importable proposal contract consumed by Origami.
- Origami remains authoritative for its receiver semantics, carrier-local physical bindings, deterministic runtime and promoted receiver storage.
- Tlaloc retains candidate-search/evaluation/model-loop authority; Tlaloque cannot promote their own proposals.

## Two targets from one discovered behavior

Tlaloc deliberately separates:

```text
UniversalPrompt
  -> external receiver bootstrap

BOOT strategy + Rosetta constraints + MicroProgram
  -> carrier/runtime behavior proposal for Origami
```

The universal prompt should stay small and carrier-agnostic. Concrete glyph meanings belong inside each Origami carrier's own ROSETTA, not in Tlaloc's prompt.

## Current receiver hard gates

A candidate is ineligible when any of these hold:

- contaminated trial;
- false exactness;
- active interface budget violation;
- conflicting/non-deterministic distilled transition.

Correctness alone cannot override these gates.

## Commands

Existing Behavior Lab commands remain:

```bash
go run ./cmd/behaviorlab compile
go run ./cmd/behaviorlab tlaloque
go run ./cmd/behaviorlab train -model <model-name>
```

Hybrid Receiver R0 adds candidate creation/ranking:

```bash
go run ./cmd/behaviorlab receiver-distill \
  -trace testdata/receiver/swarm-trace-r0.json \
  -prompt testdata/receiver/universal-bootstrap-r0.md \
  -window 4000 \
  -out generated/origami-hybrid-receiver-r0.candidate.json \
  -hybrid-out generated/origami-hybrid-receiver-r0.artifact-set.json

go run ./cmd/behaviorlab receiver-rank \
  -in testdata/receiver/scored-candidates-r0.json \
  -window 4000 \
  -out generated/origami-hybrid-receiver-r0.ranking.json
```

`receiver-distill` writes both the internal distilled candidate and the explicit Hybrid artifact set that Origami can import. `receiver-rank` applies the multi-objective fitness function and hard safety gates.

These commands produce Tlaloc **candidates/evidence**; they do not write directly into Origami's promoted receiver registry.

## Automatic model ↔ Origami tool loop

`receiver-run` drives a fresh OpenAI-compatible multimodal conversation. It sends the universal Master Prompt and one Origami carrier image to the model, advertises only declared Origami functions, executes tool calls through the independent `origami-hybrid-tool` process, returns tool results to the model, and repeats until an answer or the configured turn limit.

Example from `tlaloc/behavior-lab/` when `origami/` and `tlaloc/` are sibling repositories:

```bash
go run ./cmd/behaviorlab receiver-run \
  -endpoint http://127.0.0.1:1234/v1 \
  -model <VISION_TOOL_MODEL> \
  -prompt ../../origami/runs/hybrid-carrier-synthetic-r0/public/MASTER_PROMPT.md \
  -carrier ../../origami/runs/hybrid-carrier-synthetic-r0/public/carrier.png \
  -packet ../../origami/runs/hybrid-carrier-synthetic-r0/public/model_packet.json \
  -origami-tool ../../origami/bin/origami-hybrid-tool \
  -question 'What is the value of the second-order depends dependency of K7F91?'
```

The runner records final answer, model/tool turns, tool-call count, tool-output bytes/token-equivalent, and server-reported prompt/completion usage when available.

Tlaloc does **not** read Origami's private source or envelope. `OrigamiCLIExecutor` is a process boundary rather than a Go dependency on Origami; the Origami tool validates the public carrier and packet itself.

The implementation of this loop is testable without an external model through a mock OpenAI-compatible server. That unit/integration test proves request shape and tool-loop mechanics; it does not substitute for a real held-out VLM campaign.

See `HYBRID_RECEIVER_PIPELINE_R0.md` for the complete swarm -> distillation -> Origami import loop and Origami's Hybrid Receiver quickstart for carrier generation.
