# WALDO v0.51 — Ephemeral Specialist Pools

Status: **EXPERIMENTAL / TEST**

This experiment adds a bounded specialist-team fabric above the existing specialist mask compiler. It does not create new authority and it does not claim specialist-agent quality merely because a package was compiled.

## Shape

- **Mirror:** 0–5 active temporary specialists.
- **WALDO:** 0–5 active temporary specialists.
- **Hermes:** 0–2 triggered reasoning seats only:
  - `OUTER_ANALYST` — high-perspective review of contradictions, blind spots, duplicate effort, roadmap drift, assumptions and systemic risk.
  - `GAP_ANALYST` — missing capability/evidence/perspective review and bounded spot-use recommendations.
- **Hermes Discovery burst:** one temporary `DISCOVERY_SCOUT` seat on every 10th completed collaboration cycle when Hermes reasoning is available, Hermes reasoning consent is on, discovery consent is on, and that cycle has not already received discovery.

The maxima are ceilings, not quotas. Zero specialists is valid.

## Ephemeral specialist package

Each active seat compiles one existing specialist mask into a temporary package with its own inner settings:

- action ceiling;
- tool-call ceiling;
- lease in collaboration cycles;
- material-claim evidence policy;
- `PROPOSE_THEN_AUTHORIZE` action policy;
- `EPHEMERAL` memory policy;
- spot-use request permission;
- authority `NONE`.

The fabric measures the serialized package bytes while the seat is active. Revocation drops the raw package and preserves only a receipt/fingerprint plus separately reviewed lessons if another evidence gate approves them. This creates measurements for a future hardware-aware policy without letting RAM observations resize teams yet.

## Spot-use teamwork

An active Mirror or WALDO specialist may issue one bounded cross-controller spot-use request for a subproblem. The target pool:

1. reuses an already-active matching specialist when possible;
2. otherwise compiles one temporary specialist if a target slot is free;
3. otherwise returns a typed hold.

Spot-use never transfers controller authority and never silently expands a pool above five seats.

Hermes does not own normal work seats. Its outer-eye logs may recommend spot use, but the recommendation is not acceptance or control transfer.

## Hermes trigger boundary

Hermes is not always on. A deterministic trigger assessment can wake:

- the outer analyst for cross-pool disagreement, repeated repair, roadmap drift, unresolved high-risk claims, or sufficiently consequential work;
- the gap analyst for missing capabilities, blocked spot requests, repeated abstention, or degraded/unknown/blocked capability state;
- the discovery scout only on the 10-cycle cadence described above.

Each triggered Hermes seat requests exactly one bounded reasoning call over a public team snapshot. The requested output is a public outer-perspective log, not hidden chain-of-thought. No automatic action follows the log.

The current public Hermes control layer still exposes its reasoning-shell specialist as a scaffold. Therefore this fabric requires the host runtime to explicitly report `hermesReasoningAvailable: true` before any Hermes reasoning seat can trigger. A shell button is not treated as proof of a reasoning runtime.

## Hardware path later

The team packet records caller-supplied resource observations such as available RAM, active systems, latency or other host telemetry, but v0.51 does **not** use those observations to change team size.

A later measured policy can use evidence such as:

- compiled bytes per specialist;
- active bytes per controller;
- bytes released after revocation;
- spawn/compile latency;
- concurrent active systems;
- available RAM / memory pressure;
- task quality and repair rate versus team size.

That later policy may choose 5→4→3→2→1→0 or shorter leases, but only after measurement.

## Truth boundary

This experiment proves bounded composition/lifecycle mechanics under its focused tests. It does **not** prove that a compiled temporary specialist equals the quality of a separately trained specialist agent. That requires A/B evaluation on real tasks.

No install, network, merge, promotion, CANON, model-weight, identity-merge, automatic memory-promotion, or autonomous permission authority is granted.

## Focused verification

`node shared/ephemeral-specialist-team/selftest.js`

Expected checkpoint: **20/20 PASS**.
