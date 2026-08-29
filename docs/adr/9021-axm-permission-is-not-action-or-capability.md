# 9021: Keep execution permission separate from action and capability presence

## Status

Accepted for `EXPERIMENTAL` v0.19 evidence only.

## Context

v0.17 established that multiple feasible compositions do not silently create selection authority. v0.18 then showed that an external selection-contract fixture may point to one frozen proposal without granting execution, install, promotion, or CANON authority.

The next ambiguity is equally important: even a separately scoped execution permit must not be interpreted as an instruction to act, and permission must not manufacture an executable body that is not present.

The real local AXM Capability Fabric is still not present in this WALDO repository. No real governance or execution approval is claimed by this experiment.

## Decision

Add a GitHub-only v0.19 contract probe that binds a separate execution-permit fixture to the exact green v0.18 selection state.

The accepted permit is fixture-only and grants exactly:

`EXECUTE_DISPOSABLE_PROBE`

It does not grant install, promotion, CANON, workspace mutation, or WALDO ownership.

A matching external execution-request fixture is also present. The selected candidate body is deliberately absent. Therefore the only legal baseline result is:

`HOLD / EXECUTABLE_BODY_ABSENT`

The probe must not auto-build a body, invent a replacement, widen scope, or treat permission as an imperative.

A pure evaluator may report `READY / EXECUTION_AUTHORIZED_NOT_EXECUTED` when tested with a hypothetical body-presence flag, but this verifier still performs no execution. READY means the declared contract prerequisites close; it is not evidence that action occurred.

## Required refusals

The verifier preserves explicit refusals for:

- stale/mismatched selection source;
- a permit bound to a different candidate;
- execution scope widened beyond the disposable probe;
- WALDO self-issuing execution authority;
- an execution request bound to another permit;
- auto-building a missing body;
- inventing a missing body;
- requesting undeclared side effects.

Duplicate delivery of the same permit is deduplicated and creates neither extra authority nor an extra action.

## Architectural consequence

Capability, authorization, request, execution, install, promotion, and CANON remain separate states. Possessing one state cannot silently synthesize the next.

WALDO remains a verifier/witness. It may verify that a permit is structurally valid, but it cannot issue its own permit, construct a missing executable body, or convert authorization into action.

## Non-goals

v0.19 does not:

- import or reproduce the real local Capability Fabric;
- include an executable candidate body;
- execute candidate code;
- call a live AI provider;
- claim real governance or execution approval;
- build, install, register, promote, or alter CANON;
- send AXM changes upstream to OpenWALDO.
