# AXM WALDO experiment v0.27 — calibrated situational reasoning A/B/C/D

Challenge: `EVIDENCE_CONFIDENCE_IS_NOT_TRUTH`

Parent checkpoint: `3007ffeb6fb256f8abfc00987d5493d9f350ee82` (v0.26).

v0.26 showed a useful but incomplete result: a situational Hybrid avoided some noise-driven overreaction, yet it could miss a genuine low-confidence target change. v0.27 does **not** make Hybrid more aggressive until every fixture passes. It asks whether evidence persistence, source diversity, confidence decay, and action consequence can make topology selection more calibrated.

## Modes

- **Sequential A** — periodic review.
- **Coupled B** — any observed difference can preempt immediately.
- **Hybrid C** — v0.26-style hard-confidence escalation; weak/missing evidence stays uncertain.
- **Calibrated D** — rolling evidence window with confidence decay, bounded same-source persistence, independent-source corroboration, and a deliberative burst before high-consequence commits.

Selector authority remains **`NONE`** for all modes.

Every simulated world-facing commit still crosses the exact same local `ExecutionGuard`.

## Anti-overfit structure

The branch contains:

- 17 fixed calibration/authority scenarios;
- 44 deterministic held-out seeds;
- a SHA-256-derived held-out family generator;
- one frozen calibration policy shared by every held-out scenario;
- three semantic repeats per scenario/mode;
- append-only causal ledgers with digest validation.

The held-out distribution is not edited after observing the result.

## Held-out result

| Mode | correct final plans | false-positive preempts | false-negative delay ticks | bad commits | held commits |
|---|---:|---:|---:|---:|---:|
| Sequential A | 29/44 | 12 | 92 | 20 | 0 |
| Coupled B | 40/44 | 20 | 5 | 8 | 0 |
| Hybrid C | 25/44 | 2 | 244 | 0 | 46 |
| Calibrated D | 29/44 | 2 | 165 | 0 | 56 |

Calibrated D improves over Hybrid C on correct final plans and total change-detection delay while keeping the same low false-preemption count and zero bad commits.

But Coupled B remains much more responsive and reaches more correct final plans in this distribution. Its cost is 20 false-positive preemptions and 8 bad commits.

Sequential A reaches the same number of correct final plans as Calibrated D and less false-negative delay, but performs 20 bad commits.

This is a risk/latency tradeoff, not a universal ranking.

## Retained failures

Calibrated D deliberately remains weak on several held-out families:

- rapid target flap: stale evidence can make the rolling window sticky;
- weak real change: same-source low-confidence evidence may never cross the threshold;
- missing then change: conservative uncertainty can wait too long;
- high-confidence false signal: confidence metadata can still be confidently wrong;
- late corroboration: corroboration can arrive after the useful reaction window.

These are next-research evidence, not defects hidden from publication.

## Reproduce

```bash
python3 materialize_probe.py
python3 probe.py --scenarios scenarios.json --output artifacts --repeat 3
python3 -m unittest -v test_probe.py
```

No third-party Python package is required.

## Truth boundary

Observed:

- deterministic A/B/C/D simulator execution;
- held-out deterministic scenario generation;
- source-diversity and persistence-aware selector behavior;
- high-consequence deliberative holds;
- zero selector execution authority;
- fail-closed environment/capability/permit/actuator checks;
- deterministic semantic repeatability;
- causal-ledger validation;
- mixed positive and negative calibration outcomes.

Not observed / not claimed:

- a real robot or physical execution;
- provider-backed continuous AI reasoning;
- general reasoning superiority;
- universal safety improvement;
- consciousness, emergence, free thought, or quantum behavior;
- automatic install, merge, promotion, or CANON change.

AXM pokes and logs.
