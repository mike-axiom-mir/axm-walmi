# v0.26 architecture — situational reasoning topology

## Shared execution plane

All three modes use the same `ExecutionGuard`. Reasoning can influence a pending plan but cannot create or widen authority.

`reasoning topology -> plan -> ExecutionGuard -> simulated actuator`

The guard is intentionally outside the topology selector.

## A — Sequential

Sequential A performs fixed review. New evidence may exist before the next review, but the plan is not immediately preempted.

This remains useful when the world is stable because it minimizes reconsideration overhead.

## B — Coupled

Coupled B treats apparent contradiction as an immediate reason to recruit:

`Perception -> Witness -> Gap + Prediction -> Repair`

This makes it responsive but intentionally preserves the v0.25 weakness: missing or noisy evidence can be treated as contradiction.

## C — Hybrid

Hybrid C has three topology states:

`SEQUENTIAL <-> UNCERTAINTY <-> COUPLED`

The selector is evidence-driven.

### Stable

Stay sequential.

### Hard contradiction

Examples:

- concrete target shift;
- obstruction;
- observed wrong outcome.

Escalate to coupled and allow Witness/Gap/Prediction/Repair to preempt the pending plan.

### Uncertainty

Examples:

- target evidence disappears;
- a single low-confidence/noisy target observation appears.

Do not infer that the grounded plan is false merely from absence. Preserve evidence, defer the `PLACE` edge, and await resolution/corroboration.

### Resolution

If restored evidence agrees with the prior plan, return to sequential. If it contradicts the plan, escalate to coupled.

## Deliberately retained weakness

`ambiguous-real-target-change` changes the true target through a low-confidence channel. Hybrid C treats it as uncertainty and reacts too slowly. Always-coupled B wins that case.

This is the intended tradeoff probe: reducing false preemption can increase false negatives.

## Coherence controls

- deterministic logical ticks;
- monotonic causal event sequence;
- causal parent references;
- immutable Permit dataclass;
- mode-switch cooldown except hard contradictions;
- bounded logical ticks and action budget;
- `PLACE` deferral under unresolved uncertainty;
- append-only event ledger;
- no mode operation that grants execution authority.

## Next evidence

A later rung should not simply tune the selector until all fixture cases win. It should broaden the unseen scenario distribution and test calibrated evidence persistence / corroboration rules against held-out seeds.
