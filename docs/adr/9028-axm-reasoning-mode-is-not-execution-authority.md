# ADR 9028: Reasoning mode is not execution authority

## Status

Experimental contract/probe only.

## Context

v0.25 observed that always-coupled reasoning could improve correction on some changing-world scenarios while performing worse when evidence was temporarily missing or noisy.

A single permanent reasoning topology therefore appears unnecessarily rigid for the current simulator.

## Decision

1. AXM may choose among sequential, uncertainty, and coupled reasoning topologies from current evidence.
2. A topology change may revise or cancel a pending plan.
3. Missing evidence is not automatically equivalent to contradictory evidence.
4. A single low-confidence observation may be held as uncertainty until corroborated or resolved.
5. Hard contradictions may recruit coupled reasoning immediately.
6. The reasoning-mode selector has authority `NONE`.
7. Selecting `COUPLED` does not grant, widen, inherit, or refresh execution permission.
8. Every world-facing action still crosses the same environment/capability-scoped execution guard.
9. Mode transitions and their reasons are append-only evidence.
10. Negative selector outcomes are preserved; the benchmark must not be tuned solely to make Hybrid C win every fixture.
11. No result promotes itself, changes CANON, installs a runtime, or proves general reasoning superiority.

## Consequences

Reasoning becomes situational rather than uniformly sequential or uniformly coupled. This can reduce unnecessary reconsideration when stable and improve responsiveness when reality invalidates a plan.

It also creates a new calibration problem: uncertainty policy can suppress both false alarms and true low-confidence changes. The v0.26 `ambiguous-real-target-change` scenario deliberately demonstrates that tradeoff.

## Boundary

This ADR does not claim a real robot, provider-backed continuous AI process, consciousness, emergence, free thought, or general speedup. It records a deterministic simulator architecture and its observed fixture results.
