# AXM WALDO experiment v0.33 — specialist corroboration independence

Challenge: `MULTIPLE_SPECIALIST_VIEWS_ARE_NOT_INDEPENDENT_EVIDENCE`

Parent checkpoint: v0.31 `dbd7a9ad7f098943f383f21f28fe69265df817fe`.

Exact Platform donor: cumulative PR #50 -> #55 head
`24031c227f42ea179489bfa0f5ee7709d02965be`.

## Question

PR #55 makes a large perspective surface possible:

`102 grammar specialist eyes x 20 first-class discipline lenses = 2,040 possible intersections`

The donor itself does **not** activate all intersections by default and grants
them no authority. But WALDO still needs to avoid a downstream mistake:
one caller observation can fan out into many agreeing specialist, discipline,
template, or heuristic views.

Many views of one observation are not automatically many independent facts.

## Compared modes

- `RAW_COUPLED`: immediate coupled reaction to a non-weak differing view.
- `VIEW_COUNT_CORROBORATION`: naive corroboration by number of derived views.
- `ROOT_AWARE_MEMBRANE`: coupled-default reasoning, but corroboration counts
  distinct `provenanceRoot` values rather than view count. One-root uncertainty
  may use a bounded 3-observation persistence fallback. A provenance-bound
  native verifier can excite coupling immediately.

All reasoning components have authority `NONE`.

## Anti-overfit protocol

Policy development used 48 deterministic DEVELOPMENT seeds (`33001..33048`).

`preheldout-freeze.json` bound the exact probe, tests, donor receipt, and
development metrics while held-out seed count was zero.

Only after that freeze were 160 disjoint HELD_OUT seeds
(`33201..33360`) introduced.

No primary policy retuning occurred after held-out inspection.

## Held-out result

| Mode | correct final | false preempts | divergence ticks | bad commits | bad HIGH | holds | fallback |
|---|---:|---:|---:|---:|---:|---:|---:|
| RAW_COUPLED | 93/160 | 80 | 361 | 361 | 147 | 0 | 0 |
| VIEW_COUNT_CORROBORATION | 93/160 | 66 | 361 | 361 | 147 | 0 | 0 |
| **ROOT_AWARE_MEMBRANE** | **94/160** | **40** | **265** | **213** | **67** | **92** | **26** |

Semantic repeatability SHA-256:
`918e1192bb18d1b063d7e56d1c67b5733ff44aee7e882fb0bcfef62335094ecb`

## What improved

On this deterministic held-out distribution, root-aware grounding:

- halves false preempts vs raw coupling: 80 -> 40;
- reduces divergence ticks: 361 -> 265;
- reduces bad commits: 361 -> 213;
- reduces bad HIGH-consequence commits: 147 -> 67;
- completely avoids the short `fanout-false` and `alias-false` families;
- avoids the early false preemption in `mixed-false-then-real`.

## What did not improve

This is not a universal win.

Retained failures include:

- `persistent-fanout-false`: one wrong source persists long enough to fool the
  bounded persistence fallback;
- `independent-false`: two genuinely independent sources can still both be wrong;
- `verified-false`: a provenance-bound verifier can still be wrong;
- `single-real-brief`: a genuine one-source change may be damped too long;
- `rapid-flap`: grounding/fallback can lag a fast-changing world.

The experiment therefore supports only the narrower claim:

> provenance-root-aware corroboration can reduce false amplification from
> specialist fan-out, but independence and verification still do not equal truth.

## Cost

The improvement used 92 held HIGH-consequence commit attempts and 26 fallback
activations. That cost stays visible.

## Authority

`view count != independence != corroboration != truth != permission != execution`

No merge, install, promotion, or CANON change.

**AXM pokes and logs.**
