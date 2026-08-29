# 9019: Separate capability feasibility from selection authority

## Status

Accepted for `EXPERIMENTAL` v0.17 evidence only.

## Context

v0.15 separated a capability need from its possible embodiments. v0.16 then required incomplete multi-capability compositions to return `HOLD / MISSING_LINK` instead of inventing missing bodies.

The next risk is subtler: a system can discover several feasible compositions and accidentally convert an implementation detail—array order, shortest path, candidate count, or “only one left”—into hidden selection authority.

## Decision

Add a contract probe with two complete read-only proposal chains for the same inherited need:

- `direct-read-chain`: `SKILL -> HAND -> ORGAN`;
- `projected-read-chain`: `SKILL -> ORGAN -> HAND -> ORGAN`.

Both may be feasible. WALDO still has `selectionAuthority: NONE`, so multiple feasible candidates return:

`HOLD / MULTIPLE_VALID_COMPOSITIONS`

The witness canonicalizes candidate-set identity independent of serialized order. Advisory priority is explicitly absent.

A single surviving feasible candidate with no selection authority returns:

`HOLD / SELECTION_AUTHORITY_ABSENT`

A cycle returns `HOLD / CYCLE_DETECTED`. A child request exceeding the inherited read-only ceiling returns `REFUSED / CHILD_AUTHORITY_EXCEEDS_ANCESTOR_GAP`.

## Consequence

The experiment distinguishes four separate facts:

- **feasibility**: a composition closes its typed contract;
- **candidate-set identity**: feasible alternatives remain visible independent of order;
- **selection authority**: an external governance permission not held by WALDO;
- **execution/promotion authority**: still absent.

WALDO can therefore become better at describing choices without becoming the chooser.

## Non-goals

v0.17 does not import or execute the real local Capability Fabric, rank or select candidates, infer priority from order or path length, auto-select a lone survivor, repair cycles by invention, install/promote/CANON, make WALDO the orchestration brain, or send AXM work upstream to OpenWALDO.
