# AXM WALDO experiment v0.28 — change-point evidence invalidation

Challenge: `STALE_EVIDENCE_IS_NOT_CURRENT_STATE`

Parent: v0.27 `EVIDENCE_CONFIDENCE_IS_NOT_TRUTH`.

v0.27 showed that rolling calibrated evidence can become stale/sticky across real target changes. v0.28 asks a narrower question: **can a reasoning selector detect a regime change, invalidate evidence from the old regime, and improve decisions without turning the detector into authority?**

## Boundary first

`evidence -> reasoning selector (authority NONE) -> plan -> ExecutionGuard -> simulated commit`

Change-point detection, source reliability, evidence invalidation, and reasoning-mode choice all have authority **`NONE`**. Every simulated world-facing commit crosses the same `ExecutionGuard` used by every mode.

## Primary comparison modes

- **Sequential A** — fixed-cadence review.
- **Coupled B** — any observed difference may preempt immediately.
- **Hybrid C** — hard-confidence contradiction escalates; weaker evidence stays uncertain.
- **Calibrated D** — v0.27-style rolling evidence, persistence, corroboration, decay, and consequence-aware holds.
- **ChangePoint E** — D plus bounded change-point detection, dynamic source-reliability metadata, and epoch-scoped stale-evidence invalidation.

A later **ChangePoint+Hold F** ablation is included, but it was added after the fresh held-out distribution had been inspected. Its numbers are explicitly **post-hoc**, not independent held-out evidence.

## Anti-overfit split

- fixed calibration/authority fixtures: 16
- development seeds already seen while shaping E: 64 (`28001..28064`)
- fresh held-out seeds reserved after E policy freeze: 64 (`28101..28164`)
- the development and held-out seed sets are disjoint
- one frozen generator family definition is used for both seed sets
- three semantic repeats per scenario/mode
- append-only causal ledgers with deterministic digest validation

After the fresh held-out result was inspected, E's decision policy was not retuned. Reporting/metrics and the explicitly post-hoc F ablation were added without changing E's reaction policy.

## Primary held-out result

| Mode | correct final plans | false-positive preempts | false-negative delay | bad commits | held commits |
|---|---:|---:|---:|---:|---:|
| Sequential A | 64/64 | 22 | 164 | 44 | 0 |
| Coupled B | 64/64 | 53 | 122 | 21 | 0 |
| Hybrid C | 40/64 | 7 | 366 | 68 | 0 |
| Calibrated D | 46/64 | 6 | 307 | 39 | 44 |
| ChangePoint E | 46/64 | 6 | 307 | 39 | 37 |

**Primary result:** E successfully performs stale-evidence invalidation, but in the fresh held-out distribution it does **not** improve final-plan correctness, false-positive preemption, false-negative delay, or bad commits over D. It changes internal evidence state without producing an aggregate decision win.

That is retained as a real negative result.

## Change-point detector evidence

On the 64 fresh held-out scenarios, E reports:

- real world changes detected: **22**
- real world changes missed: **25**
- spurious change points: **7**
- total detector firings: **29**
- detector precision within this simulator: **75.9%**
- detector recall within this simulator: **46.8%**
- summed true-detection latency: **14 ticks**
- average latency for detected real changes: **0.64 ticks**
- stale evidence records invalidated: **24**

A calibration fixture named `stale-window-blocks-update` demonstrates that the mechanism itself can matter: D remains on the stale target while E invalidates the prior epoch and reaches the current target. That mechanism-level success did **not** generalize into a held-out aggregate advantage.

## Retained failures

Do not tune these away merely to make E win:

- high-confidence false evidence can create a spurious change point;
- two synchronized false sources can look like corroboration;
- weak real changes are frequently missed;
- late corroboration can arrive after useful reaction time;
- source reliability metadata can lag a sensor regime shift;
- detector recall is low in this exact held-out envelope;
- invalidating stale evidence can reduce ambiguity/holds without improving the underlying plan.

## Post-hoc consequence-hold ablation

`CHANGEPOINT_HOLD_F` uses E's same detector and plan logic, then briefly holds commits after a **detected** change point. It does not use the hidden world-change event as a control signal.

Observed on the already-inspected distribution:

- correct final plans: 46/64
- bad commits: 35 (E: 39)
- held commits: 56 (E: 37)

This suggests a possible consequence-gating tradeoff, but because F is post-hoc these numbers are hypothesis-generating only.

## Reproduce

```bash
python3 materialize_probe.py
python3 probe.py --scenarios scenarios.json --output artifacts --repeat 3
python3 -m unittest -v test_probe.py
```

No third-party Python package is required.

## Truth boundary

Observed:

- deterministic six-mode simulator execution (five primary + one post-hoc ablation);
- disjoint development and fresh held-out seed sets;
- change-point detection and stale-evidence invalidation;
- source-reliability metadata updates without authority;
- deterministic semantic repeatability;
- causal-ledger validation;
- exact execution-boundary fail-closed tests;
- mixed positive and negative outcomes.

Not observed / not claimed:

- real robot or physical execution;
- provider-backed continuous AI reasoning;
- production safety;
- general reasoning superiority or speedup;
- consciousness, emergence, free thought, or quantum behavior;
- automatic install, merge, promotion, or CANON change.

v0.28 is a fresh deterministic simulator/distribution. Its A/B/C/D numbers are **not** presented as a direct apples-to-apples continuation of v0.27's published numbers.

AXM pokes and logs.
