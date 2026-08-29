# ADR 9029: Evidence confidence is not truth

## Status

Experimental contract/probe only.

## Context

v0.26 showed that uncertainty handling can suppress noise-driven preemption but can also delay a genuine low-confidence world change.

Treating every low-confidence signal as false is not calibrated. Treating every reported confidence as truth is also not calibrated.

## Decision

1. Evidence confidence is metadata, not ground truth.
2. Reasoning mode selection may use confidence, persistence, source diversity, decay, and action consequence.
3. Repeated evidence from one source must not automatically count as independent corroboration.
4. Missing evidence is not contradiction.
5. High-confidence evidence may still be false and negative cases must be retained.
6. Stale evidence must remain observable in the ledger so later experiments can measure stickiness.
7. A high-consequence action may trigger a deliberative hold without widening execution authority.
8. The reasoning-mode selector has authority `NONE`.
9. Every simulated world-facing action crosses the same environment/capability-scoped `ExecutionGuard`.
10. Held-out/adversarial results must remain publishable when Sequential or Always-Coupled wins.
11. No result promotes itself, installs a runtime, changes CANON, or proves general reasoning superiority.

## Consequences

Calibration can reduce false preemption and bad irreversible commits, but can increase false-negative delay.

v0.27 deliberately preserves both sides of that tradeoff.

## Boundary

This ADR does not claim a real robot, provider-backed continuous AI process, consciousness, emergence, free thought, general speedup, or universal safety improvement.
