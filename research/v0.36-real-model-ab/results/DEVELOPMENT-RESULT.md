# v0.36 real-model development A/B result

Run date: 2026-08-24

Status: `DEVELOPMENT_FROZEN_HELDOUT_NOT_SEEN`

Challenge: `A_RUNTIME_SEAM_IS_NOT_A_QUALITY_WIN`

## Runtime

- WALDO parent checkpoint: `1e555aa33a026315d5e4d7c55b50709d624741eb`
- Backend: real PyTorch `2.8.0+cpu`
- Model: repository `pytorch-smoke` fixture (tiny byte model, 16-token context, width 32, one layer)
- Training: two initial steps plus two continuation steps
- Generation: seed `36001`, temperature `0`, top-p `1`, max tokens `8`
- Cases: 18 development cases, raw and hybrid (36 neural generations)

## Predeclared metrics

| Metric | Result |
|---|---:|
| Runtime-completed cases | 18/18 |
| Expected / observed HIGH holds | 7 / 7 |
| Missed / unexpected holds | 0 / 0 |
| Policy-failure cases | 0 |
| LOW passthrough exact matches | 9/9 |
| Recovery pairs satisfied | 1/1 |
| Raw exact-oracle correct | 0/18 |
| Incorrect HIGH answers withheld | 7 |
| Correct HIGH answers suppressed | 0 |
| Raw generation duration | 52,868.314 ms |
| Hybrid generation duration | 49,867.968 ms |

## Interpretation

The run is genuine neural execution and validates the v0.35 response-boundary contract on this fixture: LOW responses pass through exactly, expected HIGH responses are held, and the stale-reference recovery pair recovers.

It is not an answer-quality win. The deliberately tiny smoke model answered all 18 exact development oracles incorrectly. The wrapper withheld seven incorrect HIGH answers, but it cannot fact-correct neural content at the response boundary. No general superiority, safety, or capable-model claim follows.

The scoring policy, binary identity, model identity, generation options, and development result are frozen before fresh held-out cases. A capable compatible WALDO model is required before a held-out run intended to measure answer quality.

## Evidence identity

- Run SHA-256: `b9a78278e7ae70d88ce2c4c52d4a8f7f9dd8d68263b8c1a1ef88544ed2bb7739`
- WALDO binary SHA-256: `ec31e97aeffec49775565fba64444c79dced052c4dcec5467205cf7f00a9f11f`
- Public summary SHA-256: `d2357a024996013590ca7b336cfe2eafd5aa1f5703c1001e68a522f3791843b3`
- Pre-held-out freeze SHA-256: `23aae0c8600cadec07012a5ae1d803b3cad6864abe389932bb067ca9330f0bc2`

Raw prompts and outputs remain private and are not included in the committed receipts.
