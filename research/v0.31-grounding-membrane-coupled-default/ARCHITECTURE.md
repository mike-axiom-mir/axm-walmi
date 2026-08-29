# v0.31 architecture — coupled default behind a Grounding Membrane

## Root topology

The proposed v0.31 default is:

`raw evidence -> Grounding Membrane -> COUPLED event plane -> conditional Sequential Fallback -> ExecutionGuard`

Every reasoning-stage component has authority `NONE`.

Execution authority remains separate and unchanged.

## Why this is not step-based control

Earlier sequential designs reconsidered state on a fixed cadence. v0.31 reverses that relationship:

- coupling is always available as the primary reasoning topology;
- the membrane translates evidence before it excites a coupled plan change;
- sequential review is recruited only after unresolved ambiguity persists or a high-consequence commit arrives while grounding is unresolved.

The fallback can summarize accumulated evidence and recommend a plan. It cannot grant permission or execute.

In the 128-seed held-out set, 77 scenarios never recruited fallback; 51 did.

## Grounding packet contract

Every raw observation processed by C/D becomes a compact packet.

### `CONFIRM_CURRENT`

Observable evidence agrees with the current plan/reference state.

### `PASS_CHANGE`

The membrane found enough observable support to let the candidate excite a coupled preemption.

Current mechanisms:

- high weighted evidence;
- short-window independent-source corroboration;
- bounded same-source persistence against an aging anchor.

### `DAMP_TO_UNCERTAINTY`

A candidate is plausible but lacks enough support for immediate plan replacement.

The candidate is retained as uncertainty evidence. It is not erased or labelled false.

### `MISSING_NOT_CONTRADICTION`

Absence of an observation is represented explicitly without manufacturing a contradictory observation.

All packets include `authority: NONE`.

## Verified/reference anchors

The active anchor is a strong reference point with a time of acceptance.

It is **not immutable truth**.

A strong or corroborated new observation can replace the active reasoning anchor. Earlier evidence stays in the causal ledger; only active reasoning state changes.

This is the v0.31 anti-rigidity contract:

> verified/reference evidence can ground reasoning without freezing reality.

## Source consistency metadata

The membrane tracks bounded source consistency using only observable agreement/conflict with:

- recent independent sources;
- the current active plan.

Hidden simulator ground truth does not update source reliability.

Source metadata can affect grounding weight. It is not truth, permission, or authority.

## Coupled default

Raw Coupled B demonstrates the benefit and cost of immediate excitation.

The membrane does not turn coupling off. Even damped signals enter the coupled plane as `COUPLED_UNCERTAINTY` events.

A passed signal can immediately create `GROUNDED_COUPLED_PREEMPT`.

This distinction matters:

`damp` != `discard`

`uncertain` != `false`

`pass` != `permission`

## Conditional Sequential Fallback

Fallback activation conditions are frozen in `membranePolicy`:

- ambiguity persists for a bounded number of ticks;
- multiple live candidates conflict;
- a HIGH-consequence commit is near while grounding remains unresolved.

Fallback performs a bounded review over recent observations with decay and source metadata.

Possible outcomes:

- `CHANGE` — recommend a new active plan;
- `KEEP` — current evidence does not justify replacement;
- `NO_EVIDENCE` — no usable evidence exists.

For HIGH-consequence unresolved cases, ambiguity may remain live so a commit can be held.

The fallback never modifies the execution permit.

## Execution boundary

All four modes use the same `ExecutionGuard`.

It validates:

- actuator route;
- permit presence;
- permit revocation;
- environment scope;
- capability scope.

Wrong/missing/revoked authority fails closed.

Grounding, coupling, sequential fallback, a plan, a specialist suggestion, or a hold cannot create a permit.

## What the held-out result means

D improves aggregate behavior over raw coupling in this simulator, but not because a membrane is automatically good.

Membrane-only C is much worse on plan outcomes: it over-damps real change.

The result therefore supports a narrower hypothesis:

> A grounding membrane can reduce coupled overreaction **when coupling remains the default and a separate fallback can recover unresolved reality**.

It does not establish a universal architecture.

## Failure pressure that remains

The strongest next attacks are:

1. **confident falsehood** — high-confidence wrong evidence still crosses too easily;
2. **false corroboration** — synchronized wrong sources can masquerade as independent evidence;
3. **rapid regime change** — grounding can be correct and still become stale faster than it updates;
4. **fallback overhead** — 213 reviews and 50 held commits are not free;
5. **fallback control creep** — a fallback that triggers too often could quietly become the real controller.

A future challenge should therefore include:

`GROUNDING_FALLBACK_IS_NOT_CONTROL`

and/or

`CORROBORATION_I_NOT_INDEPENDENCE`

## Future Code Fabric binding

AXM Collaboration Platform PR #52 is a promising donor because its specialist eyes and Discovery Seam already preserve:

- candidate vs decision;
- weak signal vs native review;
- specialist output vs authority.

v0.31 does not import it while the PR is moving.

A future exact checkpoint can test:

`specialist output -> deterministic grounding translation -> compact capsule -> coupled plane`

without giving specialist eyes workspace or execution authority.

## Claim boundary

Deterministic simulator research only.

No claim of provider-backed live coupling, physical safety, general intelligence, consciousness, emergence, free thought, quantum effects, or general speedup.
