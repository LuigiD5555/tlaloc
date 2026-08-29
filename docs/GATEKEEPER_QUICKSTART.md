# Gatekeeper quickstart

- Your canonical PR: Gatekeeper classifies `OWNER`; run normal Tlaloc verification. Explicit owner override remains a deliberate promotion decision.
- Someone else's PR/fork: Gatekeeper classifies `EXTERNAL`; normal verification plus an `APPROVED` review from `LuigiD5555` is required.
- After Tlaloc is promoted: update Tonal's exact Tlaloc pin and run Tonal full-stack verification before distributing a new stack.

Project-wide policy and the reusable `gatekeeper` skill are owned by Tonal. See `GATEKEEPER.md`.
