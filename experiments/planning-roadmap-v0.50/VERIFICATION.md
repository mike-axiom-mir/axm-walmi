# v0.50 verification receipt

Local targeted verification before GitHub write:

```text
node shared/waldo-planning-membrane/selftest.js
WALDO visible planning membrane selftest PASS (38 checks)
```

The focused suite covers:

- `DIRECT` skips extra neural planning calls;
- automatic `ROADMAP` and `DEEP` selection;
- explicit planning-mode override;
- deep planning is exactly two public calls, never an unbounded loop;
- hidden-reasoning fields are refused;
- milestone dependency cycles are refused;
- capability gaps hold the roadmap;
- acceptance evidence begins `UNTESTED`;
- creation handoff targets `bounded-creation-program-planner-v1` without authority;
- Hermes queue payloads remain proposal-only;
- dependency-ordered progress and evidence-required `DONE` transitions;
- deep plan revisions preserve both proposal digests and declare `silentPlanRewrite: false`.

These tests establish deterministic planning/roadmap mechanics only. They do not prove that a neural plan is correct, that WALDO is generally intelligent, or that a roadmap is permission to execute.
