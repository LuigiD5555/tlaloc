# Tlaloque Swarm R0

Tlaloc can execute swarms of bounded, ultra-specialized workers without assuming that every worker is an LLM.

## Design

A runtime Tlaloque declares a capability contract:

- `capability`: operation it performs (`DETECT_INTENT`, `EXTRACT_ENTITY`, `ROUTE`, etc.).
- `scope`: `GENERAL` or `SPECIFIC`.
- `domain`: required for `SPECIFIC` workers.
- `engine`: model, deterministic worker, or other bounded implementation.
- `input_schema` / `output_schema`: explicit interface boundary.
- `parameter_count`: optional declared model size.
- `max_concurrency`: per-worker concurrency limit.

Tlaloc selects by capability, never by model brand. Without domain evidence, a specific worker is not eligible. A plan may pin an exact worker when reproducibility requires it.

## Transports

### PROCESS

For cheap deterministic workers or programs that can start per invocation. The child receives one `CapabilityRequest` JSON document on stdin and returns one `CapabilityResponse` JSON document on stdout.

### HTTP_JSON

For resident micro-models. A Python/Rust/Go service loads BERT, BART, an embedding model, or another specialist once and exposes one local HTTP POST endpoint. Tlaloc sends the same bounded request/response contract without reloading model weights.

## Composition

A `SwarmPlan` is a DAG. Independent nodes may execute concurrently up to `max_parallel`. A node receives:

- the original task input;
- the named outputs of nodes in `depends_on`.

A complete sub-swarm can be exposed as one `CompositeWorker`, allowing hierarchical composition:

```text
intent classifier ----\
                       -> document router -> downstream capability
entity extractor -----/
```

The composite itself can then participate as one Tlaloque inside another swarm.

## CLI

After installing Tlaloc:

```bash
tlaloc swarm example

tlaloc swarm validate --manifest swarm.json

tlaloc swarm catalog --manifest swarm.json

tlaloc swarm run --manifest swarm.json --input input.json --task demo-001
```

The run report records registered workers, executed nodes, peak parallelism, per-node worker identity, latency, confidence, output and errors.

## Worker response

```json
{
  "worker_id": "intent-general-r0",
  "output": {"intent": "SEARCH"},
  "confidence": 0.98
}
```

`output` must be valid JSON. The worker has no orchestration or promotion authority.

## Intended model topology

Typical resident swarm:

```text
Tlaloc
  |
  +-- BERT intent classifier (GENERAL)
  +-- BERT NER extractor (GENERAL)
  +-- tiny CFDI classifier (SPECIFIC / CFDI)
  +-- small BART normalizer (GENERAL)
  +-- Go date resolver (DETERMINISTIC)
  +-- Origami search worker (DETERMINISTIC)
  +-- composite DOCUMENT_ANALYST
```

The goal is not to make each worker broadly intelligent. The goal is to make each contract narrow enough that small models or deterministic programs can be reliable, and let complexity emerge from verified composition.
