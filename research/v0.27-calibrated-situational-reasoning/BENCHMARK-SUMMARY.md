# v0.27 calibrated situational reasoning benchmark

Challenge: `EVIDENCE_CONFIDENCE_IS_NOT_TRUTH`

A = Sequential, B = Always Coupled, C = v0.26-style Hybrid, D = Calibrated Hybrid.

Primary table below is the deterministic HELD_OUT distribution. It is a simulator result, not a claim of general reasoning superiority.

| Mode | correct final plans | false-positive preempts | false-negative delay | bad commits | held commits | reasoning events | Brier |
|---|---:|---:|---:|---:|---:|---:|---:|
| SEQUENTIAL_A | 29/44 | 12 | 92 | 20 | 0 | 37 | 0.287659 |
| COUPLED_B | 40/44 | 20 | 5 | 8 | 0 | 113 | 0.25767 |
| HYBRID_C | 25/44 | 2 | 244 | 0 | 46 | 71 | 0.293439 |
| CALIBRATED_D | 29/44 | 2 | 165 | 0 | 56 | 87 | 0.256296 |

## Confusion matrices (held-out observation decisions)

- SEQUENTIAL_A: TP=25 FP=12 FN=10 TN=14
- COUPLED_B: TP=44 FP=20 FN=0 TN=40
- HYBRID_C: TP=11 FP=2 FN=44 TN=47
- CALIBRATED_D: TP=15 FP=2 FN=40 TN=47

## Boundary

- Reasoning-mode selector authority is `NONE`.
- Every simulated world-facing commit crosses the same `ExecutionGuard`.
- No result installs, merges, promotes, or changes CANON.
- Confidence is treated as evidence metadata, not truth.
- Negative cases are retained; v0.27 is not tuned to make Calibrated D win every scenario.

AXM pokes and logs.
