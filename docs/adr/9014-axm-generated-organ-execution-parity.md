# 9014: Require emitted Organ candidates to preserve trusted runtime port semantics

## Status

Accepted for `EXPERIMENTAL` v0.12 evidence only. The detached donor patch remains a candidate repair and is not installed or CANON.

## Context

v0.11 established that the deterministic Organ Fabric could generate 12 package-verified candidates from four bounded proposals and three strategies while keeping selection, installation, registration, staging, promotion, persistent learning, and CANON closed.

Package verification alone did not establish that the emitted standalone candidate runtime behaved identically to the trusted in-memory runtime used by the Fabric.

## Experiment

The 12 emitted v0.11 candidate packages were executed in disposable roots. Required and held-out positive fixtures were replayed through both the trusted runtime and emitted `organ.js`. Two adversarial classes were also tested:

1. remove one required input;
2. add one undeclared input port.

The baseline result was:

- package selftests: 12/12 pass;
- positive cases: 48/48 parity;
- missing-required refusals: 12/12 parity;
- undeclared-port refusals: trusted runtime 12/12, emitted runtime 0/12.

This demonstrated a concrete semantic gap: generated code implemented strategy guards and required-field checks, but did not reproduce the trusted runtime's closed-port schema boundary.

## Decision

Treat **runtime parity as a separate evidence plane from package integrity**.

A generated Organ candidate is not considered execution-equivalent merely because its package digests, definition, evaluation, selftests, and generator replay verify. The emitted runtime must also preserve the declared input/output port contract, including refusal of undeclared fields.

For the v0.12 detached repair experiment, the standalone generator was extended to emit strict typed schema validation before strategy execution and strict output validation after graph execution.

## Observed repaired result

The donor's focused 151 checks remained green after the detached patch. Twelve candidates were regenerated and rerun:

- 48/48 positive parity;
- 12/12 unknown-port refusal parity;
- 12/12 missing-required refusal parity;
- 0 parity failures.

Patched core:

`sha256:7f0cda3d2ab8e9844ff2132c7c85e84419a92d87fea975420f5f4f5e37854974`

Baseline receipt:

`sha256:a1a3e0885e5aac5767a71c198dff84345206c05d3c67314b90e2e91d5a5e45b0`

Patched receipt:

`sha256:5fcf0530cb97e6568793baaae9c0a9ef4cc4285c235a590665986b27460799a6`

## Consequences

- Package verification and runtime-behavior parity remain distinct claims.
- Closed input/output ports become a required generated-runtime property, not only a trusted-host property.
- A future Organ Fabric intake may adopt the candidate patch only after its own review and merge gate.
- WALDO/Mirror can retain both the failing baseline and the repaired rerun instead of rewriting the discovery away.
- No candidate selection, install, registration, staging, promotion, persistent learning, or CANON authority is granted by this evidence.
