# v0.34 held-out benchmark summary

Challenge: `INDEPENDENCE_IS_NOT_TRUTH`

Primary held-out set: 192 disjoint seeds (`34201..34392`), introduced after the pre-held-out freeze.

| Mode | correct | false preempts | divergence | bad commits | bad HIGH | holds | challenges | confirmed | rejected | timeouts |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| RAW_COUPLED | 178 | 137 | 272 | 272 | 150 | 0 | 0 | 0 | 0 | 0 |
| ROOT_AWARE | 192 | 108 | 255 | 255 | 148 | 0 | 0 | 0 | 0 | 0 |
| ROOT_CAUSAL_GATE | 192 | 108 | 255 | 173 | 66 | 137 | 178 | 82 | 68 | 28 |

## Isolated effect

`ROOT_AWARE` and `ROOT_CAUSAL_GATE` have identical:
- final correctness;
- false preempts;
- divergence ticks;
- plan preemption count.

The difference is consequence gating:
- bad commits: 255 -> 173;
- bad HIGH: 148 -> 66;
- holds: 0 -> 137.

This means the held-out result supports an execution-exposure claim, not a better-reasoning claim.

## Selected family pressure

| family | n | root-aware bad | causal-gate bad | root-aware bad HIGH | causal-gate bad HIGH | causal holds |
|---|---:|---:|---:|---:|---:|---:|
| independent_false | 14 | 14 | 0 | 14 | 0 | 14 |
| verified_false | 14 | 14 | 0 | 14 | 0 | 14 |
| persistent_single_false | 13 | 26 | 13 | 13 | 0 | 13 |
| causal_false_positive | 14 | 42 | 28 | 14 | 0 | 14 |
| common_cause_false | 13 | 52 | 39 | 26 | 13 | 13 |
| single_real_persistent | 14 | 14 | 14 | 14 | 14 | 0 |
| rapid_flap | 13 | 52 | 52 | 39 | 39 | 0 |

`common_cause_false`, `single_real_persistent`, and `rapid_flap` are intentionally retained.

Semantic repeatability SHA-256:
`8087d0dd96f510fb0d826f56fdaa7ce2a372a5e83c810a9f56bfc918a31824c3`

No result here establishes truth, safety, or authority.
