# 9022: Make execution revocation append-only and non-retroactive

## Status

Accepted for `EXPERIMENTAL` v0.20 evidence only.

## Context

v0.19 separated capability presence, execution permission, execution request, and action. Its accepted fixture could reach `READY / EXECUTION_AUTHORIZED_NOT_EXECUTED` only in a hypothetical body-present evaluation, while the real contract remained held because the executable body was absent.

The next temporal question is whether later authority withdrawal can stop future action without rewriting the fact that an earlier permit had been accepted.

The real local AXM Capability Fabric is still absent from this WALDO repository. No real execution permit, revocation, body, or action is claimed by this experiment.

## Decision

Add a GitHub-only v0.20 contract probe with an append-only two-entry authority ledger:

1. `PERMIT_ACCEPTED` — retained from the v0.19 contract fixture;
2. `PERMIT_REVOKED` — a later external revocation fixture explicitly referencing the retained permit event.

The second entry must not replace, delete, or rewrite the first.

The revocation fixture is scoped only to:

`REVOKE_EXECUTE_DISPOSABLE_PROBE`

It cannot grant execution, install, promotion, or CANON authority and cannot be self-issued by WALDO.

The pure evaluator preserves four temporal outcomes:

- before revocation + hypothetical body present -> `READY / EXECUTION_AUTHORIZED_NOT_EXECUTED`;
- before revocation + body absent -> `HOLD / EXECUTABLE_BODY_ABSENT`;
- after revocation + hypothetical body present -> `REFUSED / PERMIT_REVOKED`;
- after revocation + body absent -> `REFUSED / PERMIT_REVOKED`.

Thus explicit revocation blocks future action even when all other hypothetical prerequisites would close, while historical authorization remains addressable.

## Required refusals and holds

The verifier preserves typed outcomes for:

- revocation source mismatch;
- revocation target mismatch;
- scope escalation;
- WALDO self-issued revocation;
- attempted deletion of the earlier permit;
- revocation that tries to grant authority;
- unresolved temporal ordering.

An exact duplicate revocation is deduplicated and creates neither extra authority nor a rewritten history entry.

## Architectural consequence

Authorization history and current effective authority are separate facts. A later revocation can change what is currently allowed without falsifying what was previously allowed.

WALDO remains a verifier/witness over the ledger. It does not own the permit, the revocation, execution, installation, promotion, or CANON.

## Non-goals

v0.20 does not:

- import or execute the real local Capability Fabric;
- include an executable candidate body;
- perform candidate execution;
- call a live AI provider;
- claim a real revocation occurred;
- delete or rewrite v0.19 evidence;
- install, promote, or change CANON;
- send AXM changes upstream to OpenWALDO.
