# v0.28 change-point evidence invalidation benchmark

Challenge: `STALE_EVIDENCE_IS_NOT_CURRENT_STATE`

Primary table is the deterministic HELD_OUT distribution for the modes frozen before the fresh held-out seeds were inspected. It is simulator evidence, not a claim of general superiority.

| Mode | correct final plans | false-positive preempts | false-negative delay | bad commits | held commits | CP true | CP spurious | CP missed | CP delay | stale invalidated | reasoning events |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| SEQUENTIAL_A | 64/64 | 22 | 164 | 44 | 0 | 0 | 0 | 0 | 0 | 0 | 91 |
| COUPLED_B | 64/64 | 53 | 122 | 21 | 0 | 0 | 0 | 0 | 0 | 0 | 134 |
| HYBRID_C | 40/64 | 7 | 366 | 68 | 0 | 0 | 0 | 0 | 0 | 0 | 32 |
| CALIBRATED_D | 46/64 | 6 | 307 | 39 | 44 | 0 | 0 | 0 | 0 | 0 | 41 |
| CHANGEPOINT_E | 46/64 | 6 | 307 | 39 | 37 | 22 | 7 | 25 | 14 | 24 | 41 |

## Post-hoc ablation

`CHANGEPOINT_HOLD_F` was added after the fresh held-out distribution had been inspected. Its numbers are shown only as a post-hoc mechanism probe and are **not** independent held-out evidence.

| Mode | correct final plans | bad commits | held commits | CP true | CP spurious | CP missed |
|---|---:|---:|---:|---:|---:|---:|
| CHANGEPOINT_HOLD_F | 46/64 | 35 | 56 | 22 | 7 | 25 |

## Confusion matrices

- SEQUENTIAL_A: TP=0 FP=0 FN=88 TN=115 ; decision-error proxy=0.433498
- COUPLED_B: TP=81 FP=53 FN=0 TN=69 ; decision-error proxy=0.261084
- HYBRID_C: TP=25 FP=7 FN=49 TN=122 ; decision-error proxy=0.275862
- CALIBRATED_D: TP=35 FP=6 FN=50 TN=112 ; decision-error proxy=0.275862
- CHANGEPOINT_E: TP=35 FP=6 FN=50 TN=112 ; decision-error proxy=0.275862
- CHANGEPOINT_HOLD_F: TP=35 FP=6 FN=50 TN=112 ; decision-error proxy=0.275862

## Boundary

- Reasoning-mode selector authority is `NONE`.
- Change-point detection and source reliability are evidence-processing signals only.
- Every simulated world-facing commit crosses the same `ExecutionGuard`.
- No result installs, merges, promotes, or changes CANON.
- Negative cases and families are retained.

AXM pokes and logs.
