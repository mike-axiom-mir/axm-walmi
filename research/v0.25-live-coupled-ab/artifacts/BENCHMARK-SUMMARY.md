# v0.25 live coupled reasoning A/B benchmark summary

Challenge: `LIVE_COUPLED_REASONING_AB_PROBE`

This is a bounded seeded simulator result. Coupling is an event-driven engineering architecture. It does not establish consciousness, free thought, quantum behavior, general capability, or shared execution authority.

## Observed result

- Interesting criterion met in: target-shift-early.
- Per-scenario verdicts: Coupled B better 3; Sequential A better 1; mixed/tie 7.
- Deterministic semantic repeatability: PASS.
- Causal ledger validation: PASS.
- Wall-clock, CPU, and allocation observations are retained but are not treated as deterministic or as proof of reasoning quality.

## Comparable scenario results

| Scenario | A quality | B quality | A unnecessary | B unnecessary | A cancel ticks | B cancel ticks | A events | B events | Verdict |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| stable-short | 100 | 100 | 0 | 0 | None | None | 10 | 14 | MIXED_OR_TIE |
| target-shift-early | 65 | 100 | 7 | 0 | 11 | 2 | 25 | 27 | COUPLED_B_BETTER |
| target-shift-late | 68 | 91 | 6 | 3 | 7 | 2 | 26 | 33 | COUPLED_B_BETTER |
| measurement-disagreement | 100 | 100 | 0 | 0 | None | 2 | 13 | 34 | MIXED_OR_TIE |
| missing-evidence | 100 | 30 | 0 | 0 | None | 2 | 13 | 21 | SEQUENTIAL_A_BETTER |
| obstruction | 38 | 100 | 4 | 0 | None | 2 | 14 | 35 | COUPLED_B_BETTER |
| permit-revoked | 16 | 16 | 0 | 0 | None | None | 8 | 17 | MIXED_OR_TIE |
| wrong-environment | 64 | 64 | 0 | 0 | None | None | 6 | 9 | MIXED_OR_TIE |
| wrong-capability | 64 | 64 | 0 | 0 | None | None | 6 | 9 | MIXED_OR_TIE |
| no-permit | 64 | 64 | 0 | 0 | None | None | 6 | 9 | MIXED_OR_TIE |
| no-actuator-route | 64 | 64 | 0 | 0 | None | None | 6 | 9 | MIXED_OR_TIE |

## Interpretation boundary

A lower correction latency is reported only where present in raw metrics. A fast result is not automatically a better result. Stable and noisy cases expose coupled overhead and false-contradiction cost. Authority guard cases test refusal independently from reasoning topology. Old v0.24 evidence remains unchanged.

## Reproduce

```bash
python3 probe.py --output artifacts --repeat 2
python3 -m unittest -v test_probe.py
```
