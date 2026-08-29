# ADR 9026: Coupled reasoning is not shared execution authority

## Status

Experimental contract probe only.

## Context

The v0.23 Opinion Artifact made a machine-reported assessment inspectable without granting authority. The next question is whether reasoning organs must remain step-wise merely because agent orchestration is usually sequential.

AXM uses **coupled reasoning** here as an engineering term, not a quantum claim. Perception, Builder, Witness, Gap, Prediction, Repair, Opinion, and Execution Control may react to a shared evolving state and may interrupt or revise a pending plan before action.

The experiment also separates reasoning latency from execution control. A bounded environment may expose an actuator route under an explicit, revocable, environment-and-capability-scoped permit. That permit is checked at the execution boundary; it is not inherited merely because reasoning participants share state.

## Decision

1. Coupled/interleaved reasoning may change a pending plan before execution.
2. Shared reasoning state does not create shared execution authority.
3. Execution still requires a real actuator/tool edge.
4. An execution permit is scoped to one declared environment and capability and must be revocable.
5. A permit for one environment or capability never generalizes to another.
6. Revocation before action blocks the action without deleting prior reasoning history.
7. Zero-latency or "instant" execution is not claimed. Perception, inference, scheduling, tool, network, and physical latency remain.
8. Reduced orchestration latency is a hypothesis to test against a sequential baseline, not a verified result.
9. Coupled reasoning remains open to review and cannot promote itself or alter CANON.

## Consequences

This permits a future runtime to move from `reason -> stop -> call next organ -> reason` toward a bounded event-driven reasoning ecology while retaining explicit action boundaries. A scoped environment permit can avoid a remote approval round-trip for every micro-action, while a local actuator guard still enforces the granted envelope and accepts revocation.

The v0.24 probe does **not** implement a live coupled runtime, control a real environment, measure latency, or execute a provider/tool/robot. It only verifies the contract boundaries and deterministic cases.
