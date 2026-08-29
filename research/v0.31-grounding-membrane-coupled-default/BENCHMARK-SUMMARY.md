# v0.31 grounding membrane benchmark

Challenge: `GROUNDING_SHOULD_DAMP_NOISE_WITHOUT_DAMPING_REALITY`

Primary table: deterministic 128-scenario HELD_OUT distribution after policy freeze.

| Mode | correct | false preempts | plan/world divergence ticks | bad commits | bad HIGH | holds | fallback activations |
|---|---:|---:|---:|---:|---:|---:|---:|
| SEQUENTIAL_A | 119/128 | 45 | 367 | 62 | 35 | 0 | 0 |
| COUPLED_B | 119/128 | 61 | 224 | 35 | 18 | 0 | 0 |
| COUPLED_MEMBRANE_C | 106/128 | 20 | 385 | 88 | 38 | 0 | 0 |
| **COUPLED_MEMBRANE_FALLBACK_D** | **127/128** | **23** | **205** | **27** | **7** | **50** | **213** |

## Coupled B -> proposed D

- final correctness: 119 -> 127
- false preempts: 61 -> 23
- plan/world divergence ticks: 224 -> 205
- bad commits: 35 -> 27
- bad HIGH: 18 -> 7
- holds: 0 -> 50

## Why C matters

`COUPLED_MEMBRANE_C` is the anti-cheat control for the membrane itself.

It shows that damping without a recovery path can suppress reality:

- final correctness falls to 106/128;
- bad commits rise to 88;
- plan/world divergence rises to 385.

The held-out advantage appears only when coupling stays primary and sequential review is available as a bounded fallback.

## Fallback usage

- no fallback in 77/128 scenarios
- fallback recruited in 51/128 scenarios
- at least one hold in 26/128 scenarios
- total fallback activations: 213

## Retained family failures

D must not be described as uniformly better than Coupled B.

| family | n | B bad | D bad | B divergence | D divergence | D holds |
|---|---:|---:|---:|---:|---:|---:|
| single-high-false | 8 | 1 | 8 | 24 | 52 | 0 |
| synchronized-false | 12 | 6 | 11 | 48 | 80 | 18 |
| rapid-flap | 5 | 0 | 5 | 0 | 5 | 0 |

Positive families are also retained rather than averaged away:

- `stable-noise`: B 0/9 correct vs D 8/9; B 18 false preempts vs D 1.
- `conflict-resolves`: B 16 false preempts vs D 0.
- `specialist-weak-signal`: B 7 bad commits vs D 3.

## Boundary

- membrane authority: `NONE`
- coupled reasoning authority: `NONE`
- fallback authority: `NONE`
- same `ExecutionGuard` for every mode
- no install, merge, promotion or CANON change
- no provider-backed neural process in this probe

AXM pokes and logs.
