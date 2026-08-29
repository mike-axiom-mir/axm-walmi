# ADR 9030: Stale evidence is not current state

## Status

Experimental contract/probe only.

## Context

v0.27 showed that a rolling evidence window can remain sticky after reality changes. Strong evidence for a previous target may remain inside the active window long enough to interfere with a later genuine change.

Simply shortening the window or lowering thresholds would tune the existing symptoms without testing the underlying distinction between historical evidence and evidence that still describes the current state.

## Decision

1. Evidence remains append-only causal history even after it becomes stale.
2. A detected change point may start a new active evidence epoch without deleting or rewriting earlier observations.
3. Change-point detection is a reasoning signal only; detector authority is `NONE`.
4. Source reliability is evidence metadata, not truth and not execution authority.
5. Ground-truth world-change events are evaluation-only and must not influence selector, detector, hold, permit, or execution behavior.
6. Every simulated world-facing commit continues to cross the same environment/capability/permit/actuator `ExecutionGuard`.
7. Development seeds observed while shaping the policy are labeled DEVELOPMENT, never retroactively HELD_OUT.
8. The primary E policy is frozen before inspecting the disjoint held-out seed set.
9. A later consequence-hold variant is explicitly POST_HOC and must not be reported as independent held-out evidence.
10. Negative held-out results are retained even when the mechanism succeeds in a constructed calibration case.
11. No result installs, merges, promotes, changes CANON, or grants execution authority.

## Observed result

In the constructed stale-window calibration case, change-point invalidation prevents old-epoch evidence from blocking the current target and succeeds where Calibrated D remains stale.

On the fresh 64-seed held-out distribution, primary ChangePoint E does not improve aggregate correct-final-plan count, false-positive preemption, false-negative delay, or bad commits over Calibrated D. It detects 22 true change points, misses 25, and produces 7 spurious detections in this exact simulator envelope.

The post-hoc hold ablation reduces bad commits from 39 to 35 while increasing held commits from 37 to 56, but it was added after held-out inspection and is hypothesis-generating only.

## Consequences

Stale-evidence invalidation is demonstrated as a mechanism but not established as an aggregate improvement. The next research problem is to separate change detection from reaction authority and improve detection calibration without hiding misses or false detections.

## Boundary

This ADR records deterministic simulator evidence only. It does not establish production safety, physical-world performance, general reasoning superiority, consciousness, emergence, free thought, quantum behavior, or general speedup.
