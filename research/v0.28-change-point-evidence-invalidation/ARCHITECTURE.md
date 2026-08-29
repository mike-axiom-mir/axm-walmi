# v0.28 architecture — change-point evidence invalidation

## Authority boundary

The architecture remains:

`evidence -> selector(authority NONE) -> plan -> ExecutionGuard -> simulated commit`

The selector may classify evidence, open uncertainty, detect a change point, invalidate stale evidence, change reasoning topology, recommend a plan, or hold a commit for deliberation. None of those operations can grant, widen, refresh, inherit, or revoke execution permission.

Execution remains environment + capability + actuator + permit scoped and revocable.

## Why v0.28 exists

v0.27's calibrated D mode used a bounded rolling window. That reduced some noise-driven preemption but created a new risk: strong evidence for a previous real target can remain in the window after reality changes again. The old evidence is not false historically; it is **stale for the current regime**.

v0.28 therefore separates two questions:

1. Is there evidence that the observation regime changed?
2. If yes, which prior evidence should stop participating in current-plan selection?

`STALE_EVIDENCE_IS_NOT_CURRENT_STATE` does not mean stale evidence should be deleted from history. It remains in the causal ledger; it is only excluded from the active evidence epoch.

## E — ChangePoint

E keeps D's calibrated plan-selection logic but adds an upstream detector.

Detector features:

- hard high-confidence change signal;
- short-window independent-source corroboration;
- bounded same-source persistence;
- bounded source-reliability metadata;
- no use of hidden simulator ground truth for behavior;
- an evidence epoch boundary when a change point is detected.

When E opens a new epoch, evidence older than the candidate evidence that justified the detector firing is no longer used for current-plan scoring. Historical events are not erased.

Source reliability is deliberately weak metadata. Agreement nudges it upward, conflict nudges it downward, and it is bounded around a neutral prior. Reliability changes detector weighting only; it is not permission, truth, or execution authority.

## F — post-hoc hold ablation

After the fresh held-out E result was observed, a separate F ablation was added. F reuses E's detector and plan behavior, but a detected change point may temporarily hold a commit.

Crucially, F keys the hold to `last_detected_cp_tick`, not the simulator's hidden `WORLD_TARGET_CHANGE` truth event. A false detector firing can therefore also cause a hold. This keeps the ablation honest about what the reasoning system actually knows.

F is not independent held-out evidence.

## Ground-truth separation

Simulator truth is used only for evaluation metrics such as:

- whether a plan is currently correct;
- false-positive preemption count;
- false-negative delay;
- bad commit count;
- true/spurious/missed change-point labels;
- detector latency.

It is not an input to the reasoning selector or ExecutionGuard decision.

## Evidence split

The first generated seed set (`28001..28064`) was used as development evidence and is labelled DEVELOPMENT because it was inspected while shaping E.

A disjoint seed set (`28101..28164`) was reserved for the fresh held-out evaluation after E's reaction policy was frozen. No E threshold/reaction retuning was performed after that result was inspected.

The later F ablation is explicitly post-hoc.

## Interpreting the negative result

E can solve the designed stale-window fixture, and it invalidates stale evidence in held-out runs, but it does not outperform D on aggregate plan outcomes in the fresh 64-seed distribution.

That separates **mechanism works in a constructed case** from **mechanism improves the sampled distribution**. The latter was not observed.

The detector also misses many real changes and emits some spurious change points. A future rung should therefore distinguish detector quality from reaction policy rather than simply making E more aggressive.

Possible next challenge:

`DETECTED_CHANGE_IS_NOT_TRUTH`

or

`CHANGE_POINT_DETECTION_IS_NOT_REACTION_AUTHORITY`

## Claim boundary

This is deterministic simulator research. No production or physical-safety claim is made.
