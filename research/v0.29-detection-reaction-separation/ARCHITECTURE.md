# v0.29 architecture — detector/reaction separation

## Root separation

`evidence -> detector(authority NONE) -> reaction gate(authority NONE) -> reasoning topology/state -> ExecutionGuard -> simulated commit`

Detection is an evidence-processing event. Reaction is a reasoning-policy event. Permission and execution remain separate downstream concerns.

## Detector

The detector may emit a `CHANGE_DETECTED` proposal using only observable evidence:

- high-confidence candidate observation;
- independent-source corroboration;
- bounded same-source persistence;
- bounded source-reliability metadata.

Hidden `WORLD_TARGET_CHANGE` events are evaluation-only. They label detector precision/recall and latency but cannot trigger detector, reaction, hold, plan or execution behavior.

## G — DetectorOnly

G runs the detector and source-reliability updates but sends no detector output into active reasoning state. It then uses the same calibrated reasoning/hold behavior as D.

The test contract requires D and G to match on world-facing behavior (plans, execution/hold outcomes and behavioral metrics) while G may emit additional detector evidence.

This is the architectural control proving that observation of a possible regime change does not itself have reaction authority.

## E — AutoReact comparator

E represents the direct-coupled alternative: a new detector event can immediately open a fresh active evidence epoch before calibrated reasoning continues. Detector authority is still `NONE`, but detector output is wired directly into reasoning reaction.

## H — ReactionGated

H inserts a separate gate. Detector output becomes a `REACTION_PROPOSED` record. The gate may:

- `ACCEPT` a reasoning reaction when evidence is independently corroborated (or a low-consequence persistence rule passes);
- `DEFER` when evidence is insufficient;
- `REJECT` when the current plan is independently reconfirmed, the proposal expires, or another candidate supersedes it.

An accepted reaction may open a new active evidence epoch. It does not set execution permission.

For HIGH-consequence scenarios, an unresolved proposal may temporarily hold a commit. That hold is upstream of `ExecutionGuard` and does not alter the permit.

## Why H can reduce bad commits without better plans

Held-out H and D have identical final-plan correctness, false preemptions and false-negative delay. H nevertheless performs fewer bad HIGH-consequence commits because some uncertain commits are held instead of executed.

That is a consequence-policy effect, not evidence that H reasons more accurately.

## Freeze protocol

1. Fixed calibration/authority fixtures + 96 DEVELOPMENT seeds shape detector/reaction policies.
2. `preheldout-freeze.json` binds source, tests and policy digest while held-out count is zero.
3. A disjoint 96-seed HELD_OUT set is then fixed in `heldout-seed-receipt.json`.
4. H policy is not retuned after inspection.
5. Negative held-out families remain published.

## Execution boundary

All modes share the same guard checks:

- actuator route;
- permit presence;
- revocation;
- environment scope;
- capability scope.

Wrong environment/capability -> refuse. Missing/revoked permit or actuator -> blocked. Detector/reaction state cannot grant or inherit authority.

## Next research problem

H demonstrates that an unresolved proposal can be safer to hold than to execute, but more holding is not automatically better. Indefinite holds can become a different failure mode.

Next candidate: `HOLDING_IS_NOT_RESOLUTION`.

The next rung should measure bounded resolution strategies, timeout/escalation behavior, reversible fallback, and the cost of unresolved work without granting the resolver execution authority.
