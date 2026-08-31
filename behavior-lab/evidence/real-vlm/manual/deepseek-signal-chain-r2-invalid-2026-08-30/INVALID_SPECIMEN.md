# DeepSeek R2 trial — INVALID SPECIMEN

Status: `REAL_MODEL_MANUAL_INVALID_SPECIMEN`

Promotion authority: `NONE`

This trial must not be scored as a failure of DeepSeek temporal execution. The visual artifact used for the test mutated the program semantics while attempting to add `t2-execute-to-stable-r1`.

## Intended unchanged rule set

```text
R1: IF A=ACTIVE => B:IDLE>ACTIVE
R2: IF B=ACTIVE => A:ACTIVE>DONE
R3: IF B=ACTIVE => C:IDLE>ACTIVE
R4: IF C=ACTIVE => B:ACTIVE>DONE
```

## Rule set actually visible in the tested image

```text
R1': IF B=ACTIVE => B:IDLE>ACTIVE   <-- WRONG / SEMANTIC MUTATION
R2 : IF B=ACTIVE => A:ACTIVE>DONE
R3 : IF B=ACTIVE => C:IDLE>ACTIVE
R4 : IF C=ACTIVE => B:ACTIVE>DONE
```

The intended trigger `A=ACTIVE` was replaced by `B=ACTIVE` in R1. Given the declared initial state `A=ACTIVE, B=IDLE, C=IDLE`, no visible rule is enabled. DeepSeek therefore correctly concluded that the system is immediately stable at the initial state.

## Evaluation

```text
DeepSeek visible-rule reading: PASS
DeepSeek synchronous execution: PASS relative to visible artifact
Candidate semantic fidelity: FAIL
Trial validity for intended R2 hypothesis: INVALID
```

No before/after temporal score should be assigned to this trial for the intended `t2-execute-to-stable-r1` hypothesis.

## Failure frontier

```text
ARTIFACT_GENERATION
  -> VISIBLE_SEMANTICS_DRIFT
  -> INVALID_SPECIMEN
```

This failure belongs to candidate materialization, not VLM reasoning.

## Process correction

Future experimental visual carriers must satisfy a semantic-fidelity gate before external VLM testing:

1. derive visible rule text/notation deterministically from the canonical `TemporalProgram`;
2. extract or reconstruct the candidate's intended visible semantic record;
3. compare every rule precondition, target cell, from-state and to-state against the canonical program;
4. verify the embedded exact program SHA remains unchanged;
5. reject the artifact if visible semantics and exact semantics disagree;
6. do not use generative image synthesis as the authoritative renderer for exact protocol/data-carrier specimens.

The next valid R2 specimen should contain the same R1 rule as the R1 candidate (`A=ACTIVE => B:IDLE>ACTIVE`) plus only the execution-to-stable directive.

## Learning lesson

The experiment uncovered an infrastructure requirement that is more fundamental than another VLM prompt tweak:

> A visual carrier cannot be considered valid experimental evidence unless its human/VLM-visible semantic plane is mechanically checked against its canonical exact program.

This trial is retained as immutable process-learning evidence and must never be relabeled as a model failure.
