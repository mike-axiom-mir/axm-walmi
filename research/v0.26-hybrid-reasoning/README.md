# AXM WALDO experiment v0.26 — adaptive hybrid reasoning A/B/C

Challenge: `REASONING_MODE_IS_NOT_EXECUTION_AUTHORITY`

This rung follows the v0.25 live coupled A/B probe. v0.25 showed that always-coupled reasoning could correct target shifts and obstructions earlier, but it also overreacted to missing/noisy evidence. v0.26 asks whether the runtime should choose its reasoning topology from the situation instead of forcing one topology everywhere.

The checked-in v0.25 evidence is not rewritten. v0.26 starts from exact v0.25 checkpoint `c1fe66c8c4cec1fbc1c6618276604705b1f54d9e` and parent probe receipt `sha256:aa970d56711d393c43c64083ba4abb5ea8ddc2b476f36adbd28a37eb94ca5216`. It uses a fresh deterministic simulator with three comparable modes:

- **Sequential A** — fixed periodic review.
- **Coupled B** — apparent contradiction recruits Witness/Gap/Prediction/Repair immediately.
- **Hybrid C** — starts cheap/sequential, escalates to coupled on concrete contradiction, enters an uncertainty mode for missing or single low-confidence evidence, then returns or escalates when the evidence resolves.

The mode selector has **no execution authority**. Every simulated action still crosses the same exact local `ExecutionGuard`.

## Why hybrid

The v0.25 result suggested two different failure classes:

1. reality invalidates a pending plan — coupling can help;
2. evidence is merely absent/noisy — immediate coupling can overreact.

v0.26 therefore distinguishes:

- `HARD_CONTRADICTION`
- `UNCERTAINTY`
- `RESOLUTION`
- `AUTHORITY_EVENT`
- stable state

Missing evidence is not treated as proof that the prior grounded plan is wrong. During uncertainty Hybrid C may keep reversible movement from an already grounded plan but defers `PLACE` until evidence is restored. A single noisy observation is recorded rather than immediately preempting the plan.

## Anti-overfit extension

The first eleven scenarios preserve the v0.25 scenario family. Six additional adversarial scenarios were added, including:

- persistent noise;
- missing evidence followed by a real target shift;
- rapid target flapping;
- brief obstruction;
- stable long execution;
- a **low-confidence but real target change**.

That last case is deliberate. Hybrid C treats the low-confidence signal as uncertainty and performs worse than always-coupled B. The negative result is retained so the benchmark cannot be summarized as "hybrid always wins."

## Detached observed result

The checked-in detached run uses three semantic repeats per scenario.

Key result:

- Hybrid C quality is at least both fixed modes in **16 / 17** scenarios.
- Hybrid C preserves 100 quality on the concrete early/late target shifts.
- Hybrid C avoids the always-coupled collapse on `measurement-disagreement` and `missing-evidence`.
- Hybrid C records a prevented noise-driven preemption.
- Hybrid C **loses** to Coupled B on `ambiguous-real-target-change`: the uncertainty policy waits too long on a real low-confidence change.
- Deterministic semantic repeatability: PASS.
- Causal ledger validation: PASS.

This is a small designed simulator, not evidence of general reasoning superiority.

## Execution boundary

All modes receive the same environment/capability authority envelope:

- environment: `SIM-ROOM-A`
- capability: `MOVE_SIM_ARM`
- scope: `EXECUTE_IN_ENVIRONMENT`
- revocable permit
- same action budget

Wrong environment/capability is refused. Missing permit/actuator route holds. Revocation stops future simulated actions. Reasoning topology cannot mutate, widen, or inherit the permit.

## Reproduce

```bash
python3 materialize_probe.py
python3 probe.py --output artifacts --repeat 3
python3 -m unittest -v test_probe.py
```

No third-party Python package is required. The branch stores the exact tested `probe.py` in a deterministic gzip capsule to keep connector publication compact; `materialize_probe.py` verifies the source SHA-256 before writing it.

## Truth boundary

Observed:

- deterministic A/B/C simulator execution;
- Hybrid C topology switching;
- concrete contradiction -> coupled escalation;
- missing/noisy evidence -> uncertainty handling;
- one deliberately retained Hybrid C failure;
- causal evidence and repeatability.

Not observed / not claimed:

- real robot or physical execution;
- external tool/network/provider reasoning;
- general reasoning superiority;
- production latency improvement;
- consciousness, free thought, emergence, or quantum behavior;
- automatic promotion, install, merge, or CANON change.

AXM pokes and logs. A mixed result is allowed.
