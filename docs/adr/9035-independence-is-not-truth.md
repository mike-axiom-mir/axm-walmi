# ADR 9035: Independence is not truth

## Status

Experimental research decision only.

## Context

v0.33 reduced artificial corroboration by counting provenance roots instead of specialist-view count. That prevents one caller observation from becoming many fake independent facts.

The retained problem is harder: independent sources can still agree and still be wrong. A provenance-bound verifier can also be wrong.

Reducing reasoning reactivity further would risk undoing the coupled-default architecture. The next experiment therefore separates rapid reasoning reaction from consequential commitment.

## Decision

1. Keep coupled/root-aware plan reaction unchanged.
2. A qualifying new plan may become provisionally active immediately.
3. Open a bounded causal consequence challenge after a qualifying non-outcome plan change.
4. The challenge predicts only a near-term observable consequence of the candidate state.
5. A distinct-root downstream observation may confirm or contradict the challenge.
6. Contradictory consequence evidence may revise the reasoning state.
7. HIGH-consequence simulated commits are held while the challenge is unresolved.
8. LOW-consequence commits are not globally step-gated.
9. Challenge timeout releases the hold but is not proof of truth.
10. Causal confirmation is evidence, not authority.
11. Every commit attempt still crosses the same `ExecutionGuard`.
12. Wrong/missing/revoked permit, environment, capability, or actuator authority fails closed.
13. Preserve causal false-positive, common-cause, missing-consequence, late-consequence, and rapid-change failures.
14. No result installs, merges, promotes, changes CANON, or establishes general reasoning superiority.

## Held-out evidence

After the 64-seed development freeze, 192 disjoint held-out seeds were introduced.

Against `ROOT_AWARE`, `ROOT_CAUSAL_GATE` retained identical final correctness, false preemption, and divergence metrics while:
- reducing bad commits 255 -> 173;
- reducing bad HIGH commits 148 -> 66;
- holding 137 HIGH-consequence attempts.

It opened 178 challenges: 82 confirmed, 68 rejected, and 28 timed out.

## Consequence

The result supports only the narrow claim:

> coupled reasoning can remain aggressive while consequential execution is temporarily conservative around an unresolved causal consequence challenge.

It does not support:

> causal fit proves truth.

The next failure pressure is `CAUSAL_FIT_IS_NOT_TRUTH`.

## Boundary

Deterministic simulator research only. No provider-backed live coupled reasoning, physical safety, general causal inference, consciousness, emergence, free thought, quantum behavior, or general speedup claim.
