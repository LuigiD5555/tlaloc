# Tlaloc Learning Memory R0

Status: `EXPERIMENTAL_REFERENCE_IMPLEMENTATION`

Tlaloc Learning Memory preserves development evidence across runs so repeated errors become cumulative knowledge instead of isolated anecdotes.

## Core loop

```text
real trial
  -> deterministic benchmark
  -> optional DEBUG_TRACE_R0
  -> immutable memory observations
  -> derived failure patterns
  -> candidate change
  -> post-change trial
  -> outcome link
  -> next experiment
```

The memory is development state. It is not Origami truth and it cannot promote an Origami profile by itself.

## Three layers

### Evidence Ledger

Every observation is stored as an immutable content-addressed event under:

```text
$XDG_STATE_HOME/tlaloc/learning-memory/events/<event_id>.json
```

or, when `XDG_STATE_HOME` is not set:

```text
~/.local/state/tlaloc/learning-memory/events/<event_id>.json
```

Re-ingesting the same evidence is idempotent. `recorded_at` is deliberately excluded from the content ID so the same benchmark evidence cannot become a second memory merely because it was imported later.

### Pattern Index

Patterns are derived from the ledger and can always be rebuilt. R0 groups failed real-model observations by:

```text
last completed stage
+ failure code
+ benchmark score layer
```

It also tracks affected models, specimens and questions.

Synthetic fixtures remain available for evaluator validation but are excluded from the empirical failure priority.

### Experiment History

A proposed change records the evidence events that motivated it. A later outcome links the change event to post-change evidence and stores before/after scores.

Therefore Tlaloc can remember not only:

```text
T2_NOT_FOUND happened 14 times
```

but eventually:

```text
candidate t2-route-07
motivated by events A,B,C
changed ROSETTA->T2 routing
score .51 -> .83
mean delta +.32
```

Old failures remain in the ledger after a fix so they can continue to act as regression history.

## Automatic benchmark ingestion

`tlaloc-temporal-bench` automatically stores a campaign when the evaluator classifies it as real evidence. Disable this only explicitly:

```bash
tlaloc-temporal-bench ... -no-memory
```

Override the memory root with:

```bash
-memory-store /path/to/memory
```

Useful provenance labels are:

```bash
-origami-version <version/profile>
-tlaloc-version <version>
-candidate-id <candidate>
```

Synthetic campaigns are not auto-ingested.

## Memory CLI

```bash
tlaloc-learning-memory summary

tlaloc-learning-memory events

tlaloc-learning-memory ingest-benchmark \
  -campaign campaign.json \
  -result result.json
```

Synthetic evaluator fixtures may be imported deliberately for testing with `-include-synthetic`; they remain marked `SYNTHETIC`.

Record a change motivated by known evidence:

```bash
tlaloc-learning-memory record-change \
  -candidate-id t2-route-07 \
  -summary 'make the ROSETTA to T2 route visually explicit' \
  -parents <event-id-1>,<event-id-2>
```

After a new trial, link the outcome:

```bash
tlaloc-learning-memory record-outcome \
  -candidate-id t2-route-07 \
  -parents <change-event-id>,<post-change-evidence-id> \
  -before 0.51 \
  -after 0.83
```

## Next debug target

R0 deterministically maps the most frequent unresolved real failure pattern to a development area, for example:

```text
BOOT_NOT_FOUND                 -> BOOT
ROSETTA_NOT_FOUND              -> ROSETTA
CODEC_NOT_FOUND                -> CODEC_REGISTRY
T2_NOT_FOUND                   -> T2_NAVIGATION
SEMANTIC_EVIDENCE_INSUFFICIENT -> SEMANTIC_LAYOUT
TEMPORAL_RULE_AMBIGUOUS        -> TEMPORAL_GRAMMAR
CHECKPOINT_NOT_FOUND           -> TEMPORAL_ROUTING
CAPABILITY_MISMATCH            -> CAPABILITY_FALLBACK
```

This is a priority signal, not an automatic patch generator and not a promotion decision.

## Retention and lifecycle

The managed Tlaloc installer lives under `XDG_DATA_HOME`; learning memory lives under `XDG_STATE_HOME`. Upgrading or uninstalling a managed Tlaloc version preserves the memory.

R0 has no destructive forgetting. Future relevance/recency weighting may change derived ranking, but evidence deletion must never be used to make regressions disappear.

## Hard boundaries

```text
MEMORY EVENT = IMMUTABLE
DERIVED INDEX = REBUILDABLE
SYNTHETIC != REAL EVIDENCE
MEMORY != CANONICAL ORIGAMI TRUTH
MEMORY != AUTOMATIC PROMOTION
SUCCESS + FAILURE BOTH RETAINED
FIXED FAILURE != DELETED FAILURE
CHANGE MUST LINK TO MOTIVATING EVIDENCE
OUTCOME MUST LINK CHANGE + POST-CHANGE EVIDENCE
```
