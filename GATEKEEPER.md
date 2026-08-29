# Project Gatekeeper R0

This repository follows the project-wide provenance policy owned by **Tonal** (`LuigiD5555/tonal`). The local `gatekeeper.json` is a machine-readable mirror for CI; it is not an independent policy authority.

`OWNER` means the PR is authored by `LuigiD5555` from this canonical repository. Tlaloc verification still runs, but the owner retains explicit promotion-override authority.

`EXTERNAL` means any other PR provenance. Tlaloc verification still runs, an `APPROVED` review from `LuigiD5555` is required, and external contributors cannot override or auto-promote.

The distinction controls promotion authority; it does not claim owner code is correct or external code is incorrect.

The workflow is `.github/workflows/gatekeeper.yml`. For the full policy and operational skill, use Tonal's `GATEKEEPER.md` and canonical `gatekeeper` skill.

For hard enforcement when collaborators are added, repository rules must require both the normal Tlaloc verification check and `gatekeeper / provenance`; an administrator can otherwise manually bypass workflow results.
