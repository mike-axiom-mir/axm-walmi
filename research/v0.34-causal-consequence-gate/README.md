# AXM WALDO experiment v0.34 — causal consequence gate

Challenge: `INDEPENDENCE_IS_NOT_TRUTH`

Parent checkpoint: v0.33 `d769aa3c4a208032c071d047c91a3327c576da13`.

Frozen Platform donor lineage remains cumulative PR #50 -> #55 head
`24031c227f42ea179489bfa0f5ee7709d02965be`.

## Why this rung exists

v0.33 established a narrower provenance rule:

`many perspectives from one provenance root != independent corroboration`

It also retained a harder failure: two genuinely independent sources can still agree and still be wrong, and a provenance-bound verifier can still be wrong.

v0.34 therefore does **not** make coupled reasoning slower or replace it with step control.

Instead it asks whether a strong new coupled belief can remain **provisional for HIGH-consequence execution** until a short downstream consequence challenge resolves.

## Topology

`root-aware grounding -> coupled plan reaction -> causal consequence challenge (NONE) -> same ExecutionGuard`

The plan can change immediately under the same root-aware trigger used by the baseline. The causal layer does not veto reasoning.

When a plan change is caused by:
- two distinct provenance roots;
- a provenance-bound verifier; or
- bounded same-root persistence;

v0.34 opens a short challenge:

> if this candidate reflects current reality, a downstream observable consequence should agree within the bounded horizon.

A distinct-root downstream outcome may:
- confirm the provisional plan;
- contradict it and switch reasoning back;
- fail to arrive before timeout.

While unresolved, only HIGH-consequence simulated commits are held. LOW-consequence commits continue through the same `ExecutionGuard`.

A timeout releases the reasoning hold. It is explicitly **not** treated as proof.

## Modes

- `RAW_COUPLED` — immediate reaction to any usable differing observation.
- `ROOT_AWARE` — v0.33-style provenance-root-aware corroboration.
- `ROOT_CAUSAL_GATE` — same root-aware plan reaction, plus a bounded causal consequence challenge before HIGH-consequence commitment.

All reasoning-stage components have authority `NONE`.

## Anti-overfit protocol

Development used 64 deterministic seeds (`34001..34064`).

Before any held-out seed was introduced, `preheldout-freeze.json` bound:
- exact probe SHA-256;
- exact test SHA-256;
- exact donor-lineage receipt SHA-256;
- development seed receipt and metrics;
- the frozen `ROOT_CAUSAL_GATE` policy.

Only after that freeze were 192 disjoint held-out seeds (`34201..34392`) introduced.

The primary policy was not retuned after held-out inspection.

Three semantic repeats are required.

## Fresh held-out result

| Mode | correct final | false preempts | divergence ticks | bad commits | bad HIGH | holds |
|---|---:|---:|---:|---:|---:|---:|
| RAW_COUPLED | 178/192 | 137 | 272 | 272 | 150 | 0 |
| ROOT_AWARE | **192/192** | 108 | 255 | 255 | 148 | 0 |
| ROOT_CAUSAL_GATE | **192/192** | 108 | 255 | **173** | **66** | **137** |

Against the direct `ROOT_AWARE` baseline, the new gate changed **no plan-level metric**:
- final correctness: `192 -> 192`;
- false preempts: `108 -> 108`;
- divergence ticks: `255 -> 255`.

It changed commitment exposure:
- bad commits: `255 -> 173` (**32.2% lower**);
- bad HIGH-consequence commits: `148 -> 66` (**55.4% lower**);
- held HIGH-consequence attempts: `0 -> 137`.

The candidate opened 178 consequence challenges:
- 82 confirmed;
- 68 rejected by contradictory downstream evidence;
- 28 timed out.

## Useful retained family results

The mechanism completely removed committed error in the held-out `independent_false` and `verified_false` families while the challenge was resolved by contradictory downstream evidence.

It also reduced committed exposure in `persistent_single_false`, `stable_noise`, `causal_false_positive`, and `common_cause_false`.

But it is not truth detection.

Retained failures include:
- `single_real_persistent` — a genuine one-source change can be wrong before the root threshold/challenge exists;
- `single_real_brief` — short real changes can still be missed or lagged;
- `rapid_flap` — the causal horizon can be slower than the world;
- `causal_false_positive` — a wrong downstream consequence can confirm a wrong candidate;
- `common_cause_false` — apparently separate source and consequence evidence can share an unobserved failure cause;
- `consequence_missing` and `late_consequence` — challenge holds cost time without adding truth.

The strongest next pressure is therefore:

`CAUSAL_FIT_IS_NOT_TRUTH`

and separately:

`A_HOLD_IS_NOT_A_RESOLUTION`

## Authority / truth boundary

Observed:
- deterministic simulator behavior;
- provenance-root-aware coupled plan changes;
- short consequence challenges;
- separate reasoning vs HIGH-consequence commitment;
- disjoint development/held-out evaluation;
- deterministic semantic repeatability;
- unchanged fail-closed execution guard.

Not observed / not claimed:
- provider-backed live neural coupled reasoning;
- real software execution through the Code Fabric donor;
- physical-world safety;
- causal inference in the general scientific sense;
- universal reasoning superiority;
- consciousness, emergence, free thought, or quantum behavior;
- install, merge, promotion, publication, or CANON authority.

`independence != truth != causal fit != permission != execution`

**AXM pokes and logs.**
