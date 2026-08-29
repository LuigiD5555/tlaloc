# Hybrid Receiver Pipeline R0

This pipeline captures the intended Tlaloc -> Origami receiver-development loop.

## 1. Rich behavior discovery

Tlaloc may use Behavior Compiler, Tlaloque, model trials and richer swarm coordination to solve a receiver task. The discovery system is allowed to be more capable than the final carrier micro-machine.

The evidence product is a semantic trace of externally relevant transitions, not private chain-of-thought.

## 2. Distillation

```text
rich swarm/Tlaloque behavior
  -> successful semantic trace
  -> receiver-distill
  -> deterministic MicroRule set
```

Repeated identical transitions collapse. Conflicting transitions fail rather than being hidden behind ordering or model judgment.

The distilled rule preserves:

```text
state + token + action + next_state + emit
```

## 3. Two output targets

The same successful receiver behavior produces two logically different targets:

```text
A. UniversalPrompt
   external model bootstrap; minimal and carrier-agnostic

B. Carrier behavior
   BOOT strategy + Rosetta constraints + MicroProgram
   imported by Origami and embedded/compiled into a concrete carrier/runtime
```

Tlaloc must not assign universal meaning to physical glyphs. Concrete physical-to-semantic bindings belong to each Origami carrier.

## 4. Generate the importable artifact set

```bash
cd behavior-lab

go run ./cmd/behaviorlab receiver-distill \
  -trace testdata/receiver/swarm-trace-r0.json \
  -prompt testdata/receiver/universal-bootstrap-r0.md \
  -window 4000 \
  -out generated/origami-hybrid-receiver-r0.candidate.json \
  -hybrid-out generated/origami-hybrid-receiver-r0.artifact-set.json
```

The `artifact-set.json` contract is:

`tlaloc.origami-hybrid-artifact-set.r0`

It is a proposal, not a promoted Origami artifact.

## 5. Candidate improvement

Receiver improvement is a tournament/search loop, not manual prompt inflation:

```text
candidate
  -> isolated model/carrier trials
  -> structured failures
  -> Tlaloque/swarm repairs
  -> new candidate
  -> rank
  -> repeat
```

Hard-ineligible candidates include contaminated trials, false exactness, active-window violations and non-deterministic distilled behavior.

Optimization should prefer the smallest reliable external prompt and simplest deterministic carrier behavior that preserve correctness, evidence, UNKNOWN behavior and bootstrap/navigation success.

## 6. Origami boundary

Tlaloc stops at candidate/evidence production. Origami imports the artifact set, binds carrier-local symbols, constructs concrete BOOT/ROSETTA/PROGRAM/INDEX/MEMORY/VERIFICATION sections, runs its semantic gates and stores candidate/promoted artifacts.

Tlaloc cannot mark its own proposal as `PROMOTED`.
