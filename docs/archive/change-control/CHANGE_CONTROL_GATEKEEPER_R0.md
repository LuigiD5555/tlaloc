# Change Control — Project Gatekeeper R0

Tlaloc now mirrors Tonal's project-wide provenance policy through `gatekeeper.json`, documents it in `GATEKEEPER.md`, and executes `.github/workflows/gatekeeper.yml` on PRs/reviews.

No Tlaloc runtime, BehaviorSpec, PromptIR, Tlaloque or semantic behavior changed. Existing verification remains authoritative for technical correctness. Gatekeeper adds promotion-authority classification only.

Tonal remains the policy and project-agnostic skill authority.
