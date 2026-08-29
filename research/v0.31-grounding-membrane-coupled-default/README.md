# AXM WALDO experiment v0.31 — grounding membrane with coupled-default reasoning

Challenge: `GROUNDING_SHOULD_DAMP_NOISE_WITHOUT_DAMPING_REALITY`

Parent evidence checkpoint: v0.29 `466f64b561f23e2c782729f36ae8f1aa4e71ad35`.

A separate branch name for v0.30 (`holding-resolution`) already existed at the v0.29 checkpoint when this experiment began. v0.31 therefore does not reuse or overwrite that reserved rung.

## Hypothesis

Keep **coupled reasoning as the default**. Do not restore step-by-step reasoning as the primary controller.

Instead, place a non-authoritative deterministic **Grounding Membrane** in front of the coupled event plane. The membrane translates raw observations into compact grounding packets:

- `CONFIRM_CURRENT`
- `PASS_CHANGE`
- `DAMP_TO_UNCERTAINTY`
- `MISSING_NOT_CONTRADICTION`

When the membrane cannot resolve ambiguity, a bounded **Sequential Fallback Review** may be recruited. The fallback is not the default topology and has authority `NONE`.

The intended flow is:

`raw evidence -> Grounding Membrane (NONE) -> coupled reasoning (NONE) -> conditional sequential fallback (NONE) -> ExecutionGuard -> simulated commit`

This is a collaboration topology, not step-level command/control. Evidence-processing layers may influence reasoning state but cannot grant execution permission.

## Modes

- **Sequential A** — fixed-cadence review; retained as fallback/control baseline.
- **Coupled B** — raw always-coupled reasoning.
- **Coupled+Membrane C** — coupled reasoning with grounding packets, but no sequential fallback.
- **Coupled+Membrane+Fallback D** — **proposed default**: coupled first, membrane damping, sequential review only when ambiguity persists or grounding confidence collapses.

## Grounding semantics

A verified/reference anchor is treated as a strong historical reference, **not permanent truth**.

The membrane may use:

- reported confidence as metadata, never truth;
- bounded source-consistency metadata;
- independent-source corroboration;
- same-source persistence;
- anchor age/freshness;
- missing-evidence distinction;
- consequence-aware ambiguity.

The membrane may damp a signal, pass a change into the coupled plane, or describe uncertainty. It cannot authorize execution.

A strong/corroborated real change can still cross the membrane immediately. Weak evidence remains available to the coupled system as uncertainty rather than being deleted.

## Anti-overfit protocol

The primary D policy was developed against:

- 15 fixed calibration/authority fixtures;
- 96 deterministic DEVELOPMENT seeds (`31001..31096`).

Then `preheldout-freeze.json` bound the exact source, tests and policy while held-out seed count was zero.

Only after that freeze, 128 disjoint HELD_OUT seeds (`31201..31328`) were fixed in `heldout-seed-receipt.json`.

The primary membrane/fallback policy was not retuned after held-out inspection.

Three semantic repeats are used per scenario/mode.

## Fresh held-out result

| Mode | correct final plans | false preempts | plan/world divergence ticks* | bad commits | bad HIGH | held commits |
|---|---:|---:|---:|---:|---:|---:|
| Sequential A | 119/128 | 45 | 367 | 62 | 35 | 0 |
| Coupled B | 119/128 | 61 | 224 | 35 | 18 | 0 |
| Coupled+Membrane C | 106/128 | 20 | 385 | 88 | 38 | 0 |
| **Coupled+Membrane+Fallback D** | **127/128** | **23** | **205** | **27** | **7** | **50** |

\* The simulator's legacy metric key is `false_negative_delay_ticks`; in this v0.31 probe it increments whenever the active plan differs from simulated world state, so this README calls it **plan/world divergence ticks** rather than pretending it is a pure change-detection latency measurement.

Compared with raw Coupled B, proposed D observed:

- correct final plans: **119 -> 127**;
- false preempts: **61 -> 23** (62.3% lower);
- plan/world divergence ticks: **224 -> 205**;
- bad commits: **35 -> 27** (22.9% lower);
- bad HIGH-consequence commits: **18 -> 7** (61.1% lower);
- held commits: **0 -> 50**.

This is a simulator result, not a general reasoning claim.

## Why the fallback matters

Membrane-only C is a retained failure: it suppresses false preemption but **over-damps reality**. In held-out data it falls to 106/128 correct final plans and 88 bad commits.

D recovers by making coupled reasoning the normal path while allowing bounded sequential review only for unresolved ambiguity.

Across the 128 held-out scenarios:

- **77** scenarios used no fallback at all;
- **51** recruited at least one fallback review;
- **26** experienced at least one held commit;
- total fallback review activations: **213**.

So the fallback participates materially, but it is not the default reasoning cadence.

## Retained failure families

Do not tune these away on the already-inspected held-out distribution.

D is worse than raw Coupled B on several families:

- `single-high-false`: D performs more bad commits and accumulates more divergence after a confidently wrong signal;
- `synchronized-false`: false independent-looking corroboration can cross the membrane and produce worse commit outcomes;
- `rapid-flap`: raw coupling reacts immediately while D can lag and commit from a briefly stale grounded state;
- one `stable-noise` held-out case remains incorrect at the end;
- grounding/fallback reduces noise at the cost of **50 held commits** and **213 fallback activations**.

These are evidence for the next tuning round, not defects hidden from publication.

## Specialist / Code Fabric seam

During design, AXM Collaboration Platform PR #52 was observed as a possible future donor for richer grounding translation. It currently exposes 102 separately addressable language organs, grammar-native profiles, specialist eyes, and a non-authoritative Discovery Seam.

v0.31 **does not import or bind PR #52**, because that PR is still draft/moving. The simulator's `specialist-weak-signal` family is generic and does not claim parity with the real 102-eye fabric.

A later experiment may bind an exact, stable PR #52 checkpoint and test whether specialist output can be compressed into grounding packets without becoming authority.

## Reproduce

```bash
python3 materialize_probe.py
python3 probe.py --scenarios scenarios.json --output artifacts --repeat 3
python3 -m unittest -v test_probe.py
```

No third-party Python dependency is required.

## Truth / authority boundary

Observed:

- deterministic coupled-default simulator behavior;
- a deterministic grounding membrane;
- non-authoritative grounding packets;
- conditional sequential fallback;
- disjoint development/held-out evaluation;
- deterministic semantic repeatability;
- fail-closed execution boundary tests;
- mixed positive and negative outcomes.

Not observed / not claimed:

- provider-backed live neural coupled reasoning;
- physical-world safety;
- universal superiority;
- removal of execution control;
- consciousness, emergence, free thought, or quantum behavior;
- automatic install, merge, promotion, publication, or CANON change.

`capability != exposure != grounding != reasoning mode != fallback != permission != execution != promotion != CANON`

**AXM pokes and logs.**
