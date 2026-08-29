# v0.33 held-out benchmark summary

Primary held-out set: 160 disjoint seeds (`33201..33360`).

| Mode | correct | false preempts | divergence | bad commits | bad HIGH | holds | fallback |
|---|---:|---:|---:|---:|---:|---:|---:|
| RAW_COUPLED | 93 | 80 | 361 | 361 | 147 | 0 | 0 |
| VIEW_COUNT_CORROBORATION | 93 | 66 | 361 | 361 | 147 | 0 | 0 |
| ROOT_AWARE_MEMBRANE | 94 | 40 | 265 | 213 | 67 | 92 | 26 |

Key family results for ROOT_AWARE_MEMBRANE:

- fanout-false: 14/14 final correct, 0 false preempts.
- alias-false: 13/13 final correct, 0 false preempts.
- mixed-false-then-real: 13/13 final correct, 0 false preempts.
- independent-real: 13/13 final correct.
- verified-real: 14/14 final correct.
- persistent-fanout-false: 0/13 final correct, 13 false preempts.
- independent-false: 0/13 final correct, 13 false preempts.
- verified-false: 0/14 final correct, 14 false preempts.
- single-real-brief: 0/13 final correct.
- rapid-flap: 0/13 final correct.

This is simulator evidence. It does not establish general reasoning superiority.
