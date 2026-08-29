# ADR 9034: Multiple specialist views are not independent evidence

## Status

Experimental research decision only.

## Context

The tested Platform Code Capability Fabric can surface the same caller-supplied
observation through grammar eyes, discipline lenses, machine heuristics, and
template routes. These are useful perspectives, but they can share one
provenance root.

Counting those derived views as independent corroboration would amplify one
observation into artificial confidence.

## Decision

1. Preserve coupled reasoning as the default topology.
2. Preserve every specialist/direction layer as authority `NONE`.
3. Bind each derived view to a `provenanceRoot`.
4. Corroboration counts distinct provenance roots, not number of views.
5. Weak views do not become corroboration merely through quantity.
6. One-root uncertainty may remain live and use a bounded persistence fallback.
7. A provenance-bound native verifier may excite coupling immediately, but
   verification is still not truth.
8. High-consequence unresolved one-root ambiguity may hold a simulated commit.
9. Execution authority remains outside the grounding/reasoning topology.
10. Retain cases where root-aware grounding loses.

## Held-out evidence

After a 48-seed development freeze, 160 disjoint held-out seeds were introduced.

ROOT_AWARE_MEMBRANE vs RAW_COUPLED:

- final correctness: 94 vs 93;
- false preempts: 40 vs 80;
- divergence ticks: 265 vs 361;
- bad commits: 213 vs 361;
- bad HIGH: 67 vs 147;
- holds: 92 vs 0;
- fallback activations: 26 vs 0.

Retained failures include persistent single-source falsehood, two independent
wrong sources, verified-but-wrong evidence, brief single-source real change,
and rapid regime flaps.

## Consequence

The experiment supports a narrow rule:

`many perspectives from one provenance root != independent corroboration`

It does not support:

`independent sources == truth`

or:

`verified source == truth`

## Boundary

Deterministic simulator only. No provider-backed live neural coupling,
physical-world safety, general intelligence, consciousness, emergence,
free thought, quantum behavior, or general speedup claim.
