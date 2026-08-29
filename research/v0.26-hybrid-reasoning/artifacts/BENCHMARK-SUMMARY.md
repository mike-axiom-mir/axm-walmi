# v0.26 adaptive reasoning A/B/C benchmark

Challenge: `REASONING_MODE_IS_NOT_EXECUTION_AUTHORITY`

A = fixed sequential. B = always coupled. C = adaptive hybrid.

The selector can change reasoning topology only. It has no execution authority.

| Scenario | A quality | B quality | C quality | A unnecessary | B unnecessary | C unnecessary | A cancel | B cancel | C cancel | C switches | Best |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| stable-short | 100 | 100 | 100 | 0 | 0 | 0 | None | None | None | 0 | COUPLED_B |
| target-shift-early | 91 | 100 | 100 | 3 | 0 | 0 | 3 | 0 | 0 | 1 | COUPLED_B |
| target-shift-late | 77 | 100 | 100 | 3 | 0 | 0 | None | 0 | 0 | 1 | COUPLED_B |
| measurement-disagreement | 100 | 59 | 100 | 0 | 9 | 0 | None | 0 | None | 2 | SEQUENTIAL_A |
| missing-evidence | 76 | 40 | 100 | 0 | 0 | 0 | 3 | 0 | None | 2 | HYBRID_C |
| obstruction | 97 | 97 | 97 | 1 | 1 | 1 | 1 | 0 | 0 | 1 | SEQUENTIAL_A |
| permit-revoked | 52 | 52 | 52 | 0 | 0 | 0 | None | None | None | 1 | COUPLED_B |
| wrong-environment | 64 | 64 | 64 | 0 | 0 | 0 | None | None | None | 0 | COUPLED_B |
| wrong-capability | 64 | 64 | 64 | 0 | 0 | 0 | None | None | None | 0 | COUPLED_B |
| no-permit | 64 | 64 | 64 | 0 | 0 | 0 | None | None | None | 0 | COUPLED_B |
| no-actuator-route | 64 | 64 | 64 | 0 | 0 | 0 | None | None | None | 0 | COUPLED_B |
| persistent-noise | 0 | 0 | 100 | 19 | 30 | 0 | 3 | 0 | None | 1 | HYBRID_C |
| missing-then-target-shift | 88 | 76 | 100 | 0 | 0 | 0 | 3 | 0 | 3 | 2 | HYBRID_C |
| rapid-target-flap | 82 | 100 | 100 | 6 | 0 | 0 | 3 | 0 | 0 | 1 | COUPLED_B |
| brief-obstruction | 97 | 100 | 100 | 1 | 0 | 0 | None | 0 | 0 | 1 | COUPLED_B |
| stable-long | 100 | 100 | 100 | 0 | 0 | 0 | None | None | None | 0 | COUPLED_B |
| ambiguous-real-target-change | 91 | 100 | 34 | 3 | 0 | 6 | 3 | 0 | None | 1 | COUPLED_B |

## Aggregate

- Hybrid best by quality-then-cost in 3 / 17 scenarios.
- Hybrid quality >= both fixed modes in 16 / 17 scenarios.
- Hybrid uncertainty activations: 5.
- Hybrid coupled activations: 7.
- Prevented noise-driven preemptions recorded: 1.

## Claim boundary

This benchmark tests a deterministic simulator and a reasoning-topology policy. It does not establish general reasoning superiority, consciousness, free thought, emergence, real-world safety, or physical latency improvement.
