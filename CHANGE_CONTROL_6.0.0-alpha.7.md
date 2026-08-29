# Change control — Tlaloc 6.0.0-alpha.7

Date: 2026-08-28
Status: `PROMOTED_INTEGRATION_METADATA`

## Component changed

Origami integration metadata, project skill and capability documentation.

## Before

Tlaloc `6.0.0-alpha.6` executed the coherent-state Origami profile but did not describe the newer Origami `6.0.0-alpha.2` perceptual-channel contract.

## After

Tlaloc recognizes upstream contract `origami.perceptual-channels.r0` and its operation vocabulary while explicitly reporting runtime support as not implemented. No coherent-state reference logic, prompt compiler semantics or generated prompt was changed.

## Evidence

- `VERSION` -> `6.0.0-alpha.7`;
- project skill updated to `0.2.0`;
- capability table distinguishes `contract-known` from `runtime not implemented`;
- integration contract lists all seven upstream perceptual operations;
- `go test ./...`, `go vet ./...`, and `go test -race ./...` pass in `behavior-lab`;
- generated coherent-state prompt SHA-256 remains `5f46e56e72f793d214053d87b30906cfe924a5fcb652450311e258519d981504`.

## Regression risk

Low. This revision changes integration metadata/guidance only. It must not cause the current evaluator to claim support for perceptual operations.

## Downstream impact

Future Tlaloc work can add executable fixtures/Tlaloque for Origami perceptual channels without redefining the Origami-owned semantics.
