# Local evaluation instances

This directory is the local workspace for concrete model and campaign runs.
Every direct subdirectory is ignored by Git because it may contain machine-specific
endpoints, installed-model paths, generated datasets, images, freeze ledgers, run
records, and results.

Use one descriptive directory per evaluation instance, for example:

```text
experiments/
  model-a-capability-r0/
  model-a-microisa-r1/
  model-b-capability-r0/
```

Reusable generators, schemas, runners, and aggregators belong in `cmd/`, `internal/`,
`tools/`, or another tracked source directory. Do not depend on a concrete directory
under `experiments/` from tests or production code.
