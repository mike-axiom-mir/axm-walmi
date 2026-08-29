# v0.27 architecture — calibrated situational reasoning

## Boundary first

Reasoning topology is upstream of a fixed execution boundary:

`evidence -> selector(authority NONE) -> plan -> ExecutionGuard -> simulated commit`

The selector can recommend, defer, preempt, or change reasoning topology. It cannot grant, widen, refresh, inherit, or revoke execution permission.

## Four comparison modes

### A — Sequential

A bounded evidence window is reviewed on a fixed cadence. This limits reconsideration overhead but can delay genuine changes and can commit from stale plans.

### B — Always Coupled

Any observed difference can preempt immediately. Missing evidence also recruits coupled reasoning. This is deliberately responsive and deliberately vulnerable to noisy/false evidence.

### C — v0.26-style Hybrid

Hard-confidence contradiction escalates to coupled. Missing evidence and lower-confidence differences enter uncertainty. This suppresses false preemption but can miss genuine weak changes.

### D — Calibrated

D adds a rolling evidence accumulator.

Topology states:

`SEQUENTIAL <-> UNCERTAINTY <-> DELIBERATIVE_BURST <-> COUPLED`

Evidence features:

- confidence decay over a bounded time window;
- independent source count;
- bounded persistence credit for repeated same-source evidence;
- candidate-vs-current evidence margin;
- immediate hard-contradiction threshold;
- corroboration threshold;
- persistence threshold;
- high-consequence review before irreversible commit.

A confidence value is treated only as metadata supplied by an observation. It is not treated as truth.

## Independent-source combination

For each candidate target, D keeps the strongest decayed contribution per source and combines independent sources multiplicatively.

Repeated observations from the same source do not masquerade as new independent witnesses. They receive only a bounded persistence bonus.

This creates an intentional tradeoff:

- false one-source persistence is harder to promote;
- true one-source weak change may also be missed.

## High-consequence boundary

When a simulated irreversible `COMMIT` is near and evidence remains ambiguous, D may enter `DELIBERATIVE_BURST` and hold the commit.

The hold is a reasoning outcome, not an authority grant.

If a commit is requested, the same `ExecutionGuard` used by A/B/C checks:

- actuator route;
- permit presence;
- revocation;
- environment scope;
- capability scope;
- execution scope.

## Held-out generation

Held-out scenarios are generated from a frozen seed list. SHA-256-derived values choose the family and parameters. The generator covers:

- stable noise;
- weak real change;
- corroborated real change;
- strong real change;
- source conflict;
- missing evidence followed by change;
- rapid target flap;
- persistent false evidence from one source;
- high-confidence false evidence;
- late corroboration;
- irreversible ambiguity.

The policy constants are shared across all generated cases.

## Why the negative result matters

D is safer about commits in this simulator, but its rolling evidence can become stale/sticky.

The strongest observed failure is rapid target flap: strong evidence for the previous target remains in-window long enough to interfere with the next real change.

Other retained weaknesses are weak one-source changes and changes that arrive after a missing-evidence period.

A next rung should therefore study stale-evidence invalidation and change-point detection rather than simply lowering thresholds.

## Claim boundary

This is deterministic simulator research. It does not establish a production architecture, physical safety, general reasoning superiority, consciousness, emergence, free thought, or speedup.
