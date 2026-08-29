# AXM WALDO experiment v0.29 — detection is not reaction authority

Challenge: `CHANGE_POINT_DETECTION_IS_NOT_REACTION_AUTHORITY`

Parent: v0.28 checkpoint `9b9a69eea1cdd475fb6a9ca4ecc6a838beb976a9` (`STALE_EVIDENCE_IS_NOT_CURRENT_STATE`).

v0.28 demonstrated that change-point detection can invalidate stale evidence, but it did not improve aggregate plan outcomes over Calibrated D. v0.29 separates **detecting a possible regime change** from **deciding whether reasoning should react to it**.

## Boundary first

`evidence -> detector(authority NONE) -> reaction gate(authority NONE) -> reasoning state -> ExecutionGuard -> simulated commit`

Neither detector output nor reaction-gate acceptance grants execution permission. Every simulated world-facing commit crosses the same environment/capability/permit/actuator `ExecutionGuard`.

## Modes

- **Sequential A** — fixed review cadence.
- **Coupled B** — observed differences can preempt immediately.
- **Hybrid C** — hard-confidence escalation; weaker evidence stays uncertain.
- **Calibrated D** — rolling evidence, persistence/corroboration, decay and ambiguity holds.
- **AutoReact E** — sensitive change detector can directly open a new reasoning evidence epoch; comparator for v0.28-style detector→reaction coupling.
- **DetectorOnly G** — runs the same detector but detector output is evidence only; world-facing behavior must remain identical to D.
- **ReactionGated H** — detector output becomes a proposal. A separate consequence/evidence gate may accept, defer or reject the reasoning reaction. High-consequence commits can be held while a proposal remains unresolved.

## Anti-overfit split

The H policy was developed only against 96 DEVELOPMENT seeds (`29001..29096`) plus fixed calibration/authority fixtures. A pre-held-out freeze receipt binds the exact source and policy before any held-out seeds were introduced.

Then 96 fresh HELD_OUT seeds (`29101..29196`) were fixed in a separate receipt. The sets are disjoint. H was not retuned after the held-out result.

Policy digest: `e128820127db37a044381f7a3d4a79a9d46a14a5a50c16a1e115b56eb7d9b3f3`

## Primary held-out result

| Mode | correct final plans | FP preempts | FN delay | bad commits | bad HIGH-consequence commits | held commits |
|---|---:|---:|---:|---:|---:|---:|
| Sequential A | 96/96 | 33 | 230 | 60 | 34 | 0 |
| Coupled B | 96/96 | 61 | 177 | 28 | 17 | 0 |
| Hybrid C | 42/96 | 11 | 823 | 155 | 94 | 0 |
| Calibrated D | 59/96 | 17 | 633 | 84 | 46 | 58 |
| AutoReact E | 59/96 | 17 | 633 | 84 | 46 | 52 |
| DetectorOnly G | 59/96 | 17 | 633 | 84 | 46 | 58 |
| ReactionGated H | 59/96 | 17 | 633 | 71 | 33 | 82 |

### What actually improved

H does **not** improve final-plan correctness, false-positive preemption count, or false-negative delay over D. Those values remain exactly the same.

What changes is consequence handling:

- bad commits: **84 -> 71** (15.5% reduction);
- bad HIGH-consequence commits: **46 -> 33** (28.3% reduction);
- held commits: **58 -> 82** (41.4% increase).

So the held-out gain is **risk gating by holding unresolved high-consequence actions**, not smarter plans.

DetectorOnly G exactly preserves D's world-facing behavioral metrics while adding detector evidence. This is the key separation proof: detection itself does not need to become reaction.

## Detector / reaction evidence

On the 96 held-out scenarios, H's detector reports:

- true change points detected: **61**
- spurious change points: **23**
- missed real changes: **6**
- detector precision in this simulator: **72.6%**
- detector recall in this simulator: **91.0%**

The H reaction gate accepts **25** proposals: **20 true / 5 false**, or **80.0%** accepted-reaction precision in this exact envelope.

## Retained failures

Do not tune these away merely to make H win:

- H still reaches only **59/96** correct final plans, exactly matching D;
- weak real changes remain poor: held-out `real-change-weak` stays 0/10 correct;
- late corroboration remains 0/5 correct;
- synchronized false corroboration still fools the reaction gate;
- high-confidence false evidence remains a serious failure family;
- detector emits 23 spurious change points and misses 6;
- the bad-commit reduction is purchased with 24 additional holds.

This makes the next question clearer: **a hold is not a resolution**. A future rung should test how unresolved high-consequence holds are resolved without converting the reaction gate into execution authority or silently waiting forever.

Possible next challenge: `HOLDING_IS_NOT_RESOLUTION`.

## Reproduce

```bash
python3 materialize_probe.py
python3 probe.py --scenarios scenarios.json --output artifacts --repeat 3
python3 -m unittest -v test_probe.py
```

No third-party Python package is required.

## Truth boundary

Observed: deterministic simulator behavior, disjoint development/held-out evaluation, detector/reaction separation, fail-closed execution boundary, semantic repeatability, causal-ledger validation, mixed positive/negative outcomes.

Not observed or claimed: physical-world safety, provider-backed continuous AI reasoning, general reasoning superiority, consciousness, emergence, free thought, quantum behavior, automatic install/merge/promotion/CANON.

AXM pokes and logs.
