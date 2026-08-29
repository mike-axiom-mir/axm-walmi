# ADR 9032: Grounding should damp noise without damping reality

## Status

Experimental contract/probe only.

## Context

Raw coupled reasoning reacts quickly, but earlier WALDO probes showed that immediate coupling can overreact to missing/noisy evidence.

A rigid grounding wall creates the opposite failure: it can suppress a genuine world change because unverified or weak evidence is treated as if it were false.

v0.31 tests a third shape: keep coupling as the default reasoning topology, place a deterministic non-authoritative membrane in front of coupled preemption, and retain sequential review only as a fallback when ambiguity remains unresolved.

## Decision

1. Coupled reasoning is the default topology in the proposed v0.31 architecture.
2. Sequential reasoning is a conditional fallback, not the primary controller.
3. The Grounding Membrane translates evidence; it does not grant authority.
4. `DAMP_TO_UNCERTAINTY` must preserve a candidate as uncertainty rather than erase or label it false.
5. `MISSING_NOT_CONTRADICTION` must not manufacture a contradictory observation.
6. Verified/reference anchors are historical grounding points, not immutable truth.
7. Strong or corroborated current evidence may supersede an older anchor.
8. Reported confidence and source-consistency metadata are evidence metadata, not truth.
9. A fallback review may recommend `CHANGE`, `KEEP`, or `NO_EVIDENCE`; it has authority `NONE`.
10. HIGH-consequence unresolved ambiguity may produce a bounded reasoning hold without modifying permits.
11. Every simulated world-facing action crosses the same `ExecutionGuard`.
12. Development and held-out seeds remain disjoint; the primary policy is frozen before held-out introduction.
13. Held-out failure families remain visible even when aggregate metrics improve.
14. No result installs, merges, promotes, changes CANON, or establishes general reasoning superiority.

## Observed result

On 128 fresh held-out scenarios:

- raw Coupled B: 119/128 correct, 61 false preempts, 35 bad commits, 18 bad HIGH commits;
- Coupled+Membrane C: 106/128 correct, showing that membrane-only damping can over-suppress reality;
- proposed Coupled+Membrane+Fallback D: 127/128 correct, 23 false preempts, 27 bad commits, 7 bad HIGH commits, with 50 holds.

D used no fallback in 77/128 held-out scenarios and recruited fallback in 51/128.

D is not uniformly better. It is worse than raw coupling on high-confidence false signals, synchronized false corroboration, and rapid target flaps.

## Consequence

The observed result supports a narrow architectural hypothesis:

> coupled-default reasoning can be damped against noise without becoming step-controlled, but only if uncertainty remains recoverable through a separate bounded fallback.

It does **not** support the claim that grounding is truth or that more damping is always safer.

The next tuning pressure is to reduce false corroboration, rapid-change lag, repeated fallback overhead, and the risk that a fallback quietly becomes the real controller.

## Boundary

Deterministic simulator evidence only. No provider-backed continuous neural coupling, physical-world safety, consciousness, emergence, free thought, quantum behavior, or general speedup claim.
