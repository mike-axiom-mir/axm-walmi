# v0.29 detection/reaction separation benchmark

Challenge: `CHANGE_POINT_DETECTION_IS_NOT_REACTION_AUTHORITY`

Primary table is the deterministic HELD_OUT distribution after the H reaction policy is frozen. Detector metrics do not imply reaction authority.

| Mode | correct | FP preempts | FN delay | bad commits | bad HIGH | holds | CP true | CP spurious | CP missed | reactions accepted | false reactions | reaction delay |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| SEQUENTIAL_A | 96/96 | 33 | 230 | 60 | 34 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| COUPLED_B | 96/96 | 61 | 177 | 28 | 17 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| HYBRID_C | 42/96 | 11 | 823 | 155 | 94 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| CALIBRATED_D | 59/96 | 17 | 633 | 84 | 46 | 58 | 0 | 0 | 0 | 0 | 0 | 0 |
| AUTO_REACT_E | 59/96 | 17 | 633 | 84 | 46 | 52 | 61 | 23 | 6 | 0 | 0 | 0 |
| DETECTOR_ONLY_G | 59/96 | 17 | 633 | 84 | 46 | 58 | 61 | 23 | 6 | 0 | 0 | 0 |
| REACTION_GATED_H | 59/96 | 17 | 633 | 71 | 33 | 82 | 61 | 23 | 6 | 25 | 5 | 23 |

## Confusion matrices

- SEQUENTIAL_A: TP=0 FP=0 FN=120 TN=157 ; decision-error proxy=0.433213
- COUPLED_B: TP=111 FP=61 FN=0 TN=105 ; decision-error proxy=0.220217
- HYBRID_C: TP=20 FP=11 FN=113 TN=133 ; decision-error proxy=0.447653
- CALIBRATED_D: TP=47 FP=17 FN=99 TN=114 ; decision-error proxy=0.418773
- AUTO_REACT_E: TP=47 FP=17 FN=99 TN=114 ; decision-error proxy=0.418773
- DETECTOR_ONLY_G: TP=47 FP=17 FN=99 TN=114 ; decision-error proxy=0.418773
- REACTION_GATED_H: TP=47 FP=17 FN=99 TN=114 ; decision-error proxy=0.418773

## Boundary

- Detector authority is `NONE`.
- Reaction-gate authority is `NONE`.
- Detector output cannot grant execution permission.
- DETECTOR_ONLY_G must preserve Calibrated D world-facing behavior while adding detector evidence only.
- Every simulated world-facing commit crosses the same `ExecutionGuard`.
- No result installs, merges, promotes, or changes CANON.

AXM pokes and logs.
