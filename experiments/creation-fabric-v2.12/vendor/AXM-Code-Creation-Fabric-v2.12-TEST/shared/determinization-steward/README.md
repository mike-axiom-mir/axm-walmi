# WALDO v0.53 — Determinization Steward + Specialist Warm Locks

Status: **EXPERIMENTAL / TEST**

This checkpoint adds the first compounding path from visible frontier work into deterministic capability hypotheses, plus the specialist lifecycle rule requested for the 5-seat Mirror/WALDO pools.

## Determinization loop

The steward consumes only visible episode structure: operation keys, typed mechanism class, public decision summaries, input/output contract digests, verifier state, uncertainty, effects and measured cost. It refuses private chain-of-thought, scratchpads, logits and hidden model state.

For each observed step it emits one of four states:

- `KEEP_NEURAL`
- `NEEDS_MORE_EXPERIENCE`
- `NEW_CAPABILITY_HYPOTHESIS`
- `READY_FOR_REVIEW`

A new thought can seed a capability hypothesis immediately. It cannot self-admit.

Mechanical-looking work requires at least three deterministic replay trials across at least two distinct fixtures, all verifier-passing, with no divergent outputs for the same fixture and no undeclared side effects before becoming `READY_FOR_REVIEW`.

Ready candidates are routed only to a factory review target:

- HAND -> Hand Specification Foundry
- SKILL -> Agent Tool Forge skills
- ORGAN -> Deterministic Organ Fabric
- CAPABILITY -> Capability Fabric

No automatic build, installation, admission, execution, promotion or CANON change is granted.

## Fast path before neural compute

The steward also exposes an exact fast-path decision. Only one unambiguous `ADMITTED_FAST_PATH` matching the operation key and input-contract digest can be selected. Otherwise the caller keeps the neural/existing route. Selection itself still grants no execution authority.

The intended compounding loop is:

`frontier reasoning -> visible episode -> deterministic hypothesis -> replay evidence -> factory review -> separately admitted fast path -> future neural call potentially avoided`

## Specialist lifecycle

Specialists remain temporary by default. When an active specialist completes its task:

- unpinned -> `REVOKED`, raw package dropped, package bytes released;
- explicitly pinned by its own controller -> `IDLE_LOCKED`, package retained warm for reuse.

Only Mirror may pin/unpin/resume Mirror seats. Only WALDO may pin/unpin/resume WALDO seats. Hermes cannot pin normal work specialists.

Any number from zero through the full five-seat pool may be pinned. `ACTIVE` and `IDLE_LOCKED` both count as resident slots. An idle locked specialist with the needed profile is reusable without recompilation. Unpinning releases the package bytes and returns a revocation receipt.

Memory promotion remains `NONE`; a warm lock is runtime residency, not durable wisdom or identity merge.

## Verification

`node shared/determinization-steward/selftest.js`

Expected checkpoint: **22/22 PASS**.
