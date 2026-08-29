# ADR 9031: Change-point detection is not reaction authority

## Status

Experimental contract/probe only.

## Context

v0.28 showed that stale-evidence invalidation can work in a constructed case without improving aggregate held-out decisions. The detector itself also produced both misses and spurious change points.

Treating every detector firing as a reasoning command collapses two distinct questions: whether a regime change may have occurred, and whether the system should react now.

## Decision

1. Change-point detection is evidence, not reaction authority.
2. Detector authority is `NONE`.
3. A separate reaction gate may accept, defer or reject a detector proposal; reaction-gate authority is also `NONE`.
4. DetectorOnly G must preserve Calibrated D world-facing behavior while adding detector evidence only.
5. Hidden world-change truth remains evaluation-only and cannot trigger detection, reaction, hold, permission or execution.
6. A reaction may change active reasoning state (for example, evidence epoch) but cannot grant or widen execution permission.
7. HIGH-consequence unresolved proposals may cause bounded reasoning holds without modifying permits.
8. All world-facing simulated actions still cross the same `ExecutionGuard`.
9. Development and held-out seed sets are disjoint; policy is frozen before held-out introduction.
10. Held-out failure families remain publishable even when a mechanism improves one risk metric.
11. No result installs, merges, promotes, changes CANON or proves general reasoning superiority.

## Observed result

On 96 fresh held-out scenarios, DetectorOnly G exactly matches D's world-facing behavioral metrics while recording detector events. ReactionGated H also matches D's final-plan correctness, false-positive preemptions and false-negative delay, but reduces bad commits from 84 to 71 and HIGH-consequence bad commits from 46 to 33. Held commits rise from 58 to 82.

H's detector records 61 true detections, 23 spurious detections and 6 misses. The gate accepts 25 reactions, including 5 false accepts.

## Consequence

The experiment supports the architectural separation between detecting a possible change and reacting to it. It does not show better underlying plan reasoning. The observed risk reduction comes from holding unresolved high-consequence commits.

That exposes the next problem: `HOLDING_IS_NOT_RESOLUTION`.

## Boundary

Deterministic simulator evidence only. No physical-world safety, general reasoning superiority, consciousness, emergence, free thought, quantum behavior or general speedup claim.
