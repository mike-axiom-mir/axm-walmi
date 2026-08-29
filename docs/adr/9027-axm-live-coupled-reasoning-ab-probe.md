# ADR 9027: Measure live coupled reasoning against a sequential control

## Status

Experimental, branch-only, measured in a bounded simulator. Not promoted and not CANON.

## Context

ADR 9026 established a contract: reasoning components may share and revise an evolving state, but shared reasoning does not grant execution authority. v0.24 did not implement or measure a live runtime.

The v0.25 question is narrower than general autonomy: when contradictory evidence appears while a plan is pending, can an event-driven reasoning plane recruit relevant roles and revise the plan before execution, and does it do so earlier or with fewer unnecessary actions than an otherwise comparable fixed sequential orchestrator?

"Coupled" is an engineering term for causal shared-state event processing. It is not a quantum, consciousness, free-thought, or general-intelligence claim.

## Decision

1. Implement two runtimes over one seeded fixture, simulator, role set, latency table, plan commit delay, actuator route, and permit envelope.
2. Sequential A uses fixed completed stages. Evidence may be logged while a stage or pending execution window exists, but no completed role is reinvoked until the next fixed observation cycle.
3. Coupled B uses a bounded event queue and role subscriptions. An observation can recruit Witness, Gap, Prediction, Opinion, and Repair without an external `now call Repair` script.
4. Pending proposals carry identities. Contradictions may cancel them; queued writes for superseded proposal identities are rejected and preserved as stale-write evidence.
5. Both runtimes call the same independent execution guard for every simulated micro-action.
6. A permit is immutable after construction except for revocation. The guard exposes no grant or scope-expansion operation to reasoning code.
7. Deterministic logical metrics and semantic digests are separated from wall-clock, process CPU, allocation, and RSS observations. Machine timing is evidence but is excluded from repeatability claims.
8. Preserve unfavorable results, including extra events, false contradiction, HOLD, lower quality, and Sequential A wins.
9. Keep all output candidate-only. Do not open a PR, merge, install, promote, or alter CANON as part of this probe.

## Coherence controls

- monotonic event sequence numbers;
- causal parent references;
- chained state and event digests;
- proposal-identity stale-write rejection;
- bounded queue, event, action, reconsideration, and logical-time budgets;
- duplicate-event suppression;
- append-only dissent and HOLD evidence;
- deterministic shutdown;
- no Builder priority or authority dominance;
- exact environment/capability/scope/route checks at the actuator boundary.

## Consequences

The probe can support only claims directly represented in its raw metrics. One case showing earlier cancellation does not prove a general speedup or better reasoning. A stable case can show pure event overhead, and incomplete/noisy evidence can make coupled reasoning worse.

The simulator does not test real physical safety, provider/model inference latency, networks, tools, continuous operation, or generalization outside its fixtures. Those remain HOLDs for later experiments.

