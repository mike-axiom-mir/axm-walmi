# AXM / WALDO v0.12 — generated candidate execution parity

Status: **EXPERIMENTAL / DETACHED / NOT INSTALLED**

This checkpoint continues the exact v0.11 `ONE_CONCEPT_MANY_BODIES` experiment. v0.11 proved that the supplied deterministic Organ Fabric could accept four bounded proposals, generate 12 verified candidate packages, archive them, deduplicate them, and round-trip the archive without selecting or installing anything.

v0.12 asks a stricter question: **do the emitted candidate packages actually behave like the trusted Organ Fabric runtime that generated them?**

## Baseline execution result

All 12 v0.11 candidate packages were materialized into disposable roots and their emitted `organ.js` bodies were executed. Each package selftest passed. Across 48 declared required/held-out positive cases, packaged execution matched the trusted runtime 48/48. Missing-required-input behavior also matched 12/12.

The adversarial closed-port check found a real generator parity gap: the trusted runtime rejected an undeclared extra input field for every candidate, while all 12 emitted `organ.js` bodies accepted that same extra field.

Baseline receipt:

`sha256:a1a3e0885e5aac5767a71c198dff84345206c05d3c67314b90e2e91d5a5e45b0`

Observed baseline totals:

- candidate packages: 12
- package selftests passed: 12
- declared/held-out positive cases: 48/48 parity
- missing-required refusals: 12/12 parity
- unknown-port refusals: trusted 12/12, emitted package 0/12
- parity failures: 12
- static source violations: 0

## Detached repair experiment

Only a detached copy of the donor `shared/deterministic-organ-fabric/core.js` was modified. The canonical donor snapshot was not rewritten or installed.

Candidate factory version: `1.0.1-v012-parity`

Patched core SHA-256:

`sha256:7f0cda3d2ab8e9844ff2132c7c85e84419a92d87fea975420f5f4f5e37854974`

The emitted standalone `organ.js` gained the same closed typed input/output boundary expected by the trusted runtime: plain-object checks, declared-key enforcement, required fields, primitive/object/array type checks, enum/range constraints, array item checks, input validation before strategy guards, and output validation after graph execution.

Before regenerating candidates the donor's focused Organ Fabric test lane was rerun and remained **151/151 PASS**.

The same four proposals were then regenerated into the same three strategy classes (`lean`, `balanced`, `guarded`) and executed again.

Patched receipt:

`sha256:5fcf0530cb97e6568793baaae9c0a9ef4cc4285c235a590665986b27460799a6`

Observed patched totals:

- proposals: 4
- candidates: 12
- generation failures: 0
- package selftests: 12/12 PASS
- positive execution parity: 48/48
- unknown-port refusal parity: 12/12
- missing-required refusal parity: 12/12
- total parity failures: **0**

## Truth boundary

Candidate code was executed only in disposable experiment roots. The harness requested no network access and called no live AI provider. `AI` remains the declared source class of one supplied proposal, not a live dependency of the run.

Nothing in this checkpoint was selected, installed, registered, staged, promoted, admitted to persistent learning, merged into AXM platform `main`, or made CANON.

The patch in this directory is a **candidate donor repair**, not a statement that the local or public Organ Fabric has already adopted it.
