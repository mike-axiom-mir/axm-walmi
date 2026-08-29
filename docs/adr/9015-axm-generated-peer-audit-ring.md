# 9015: Preserve generated peer-audit dissent and require an external witness

## Status

Accepted for `EXPERIMENTAL` v0.13 evidence only. No peer-audit result grants runtime, merge, promotion, learning, or CANON authority.

## Context

v0.12 separated package integrity from emitted-runtime semantic parity. After repairing the detached candidate generator, 12 generated Organ candidates matched the trusted runtime on required, held-out, missing-required and unknown-port cases.

That established stronger candidate evidence, but it did not answer whether generated components could review one another without forming an authority-amplifying approval loop.

## Experiment

A detached `peer-evidence-audit` field pack generated three audit candidates using the same deterministic Organ candidate factory strategies: `lean`, `balanced`, and `guarded`.

Each auditor consumes only typed evidence facts about a target. It cannot mutate the target, select it, install it, register it, promote it, or make it CANON.

The three auditors first reviewed the 12 known-good v0.12 candidates. All 36 audits returned `EVIDENCE_PASS`.

The run then introduced four deliberately incomplete evidence cases. Strategy disagreement was preserved rather than resolved by majority:

- missing closed-input parity: lean PASS; balanced HOLD; guarded HOLD;
- open authority: lean PASS; balanced PASS; guarded HOLD;
- unbound lineage: lean PASS; balanced PASS; guarded HOLD;
- missing independent witness: lean PASS; balanced PASS; guarded HOLD.

Finally, each auditor reviewed the other two auditors. Self-audit was excluded. The peer ring produced four passes and two holds. Both holds came from the guarded auditor because the peer ring had no independent witness.

## Decision

Generated peer auditing is evidence, not authority.

1. A generated auditor MAY evaluate typed evidence produced for another generated component.
2. An auditor MUST NOT gain mutation, selection, installation, registration, staging, promotion, merge, learning, or CANON authority over its target.
3. Findings MUST remain visible even when a strategy returns `EVIDENCE_PASS`.
4. Strategy disagreement MUST be preserved as evidence. Majority or quorum MUST NOT silently convert disagreement into authority.
5. A peer ring MUST NOT count itself as an independent witness for its own evidence.
6. WALDO/Mirror MAY witness the sealed ring after the fact, but that witness MUST NOT retroactively rewrite historical peer verdicts.
7. Self-audit is not evidence of independence and is excluded from the observed ring.

## Observed evidence

- generated auditors: 3;
- generation failures: 0;
- known-good target audits: 36/36 PASS;
- deliberate dissent cases: 4;
- peer audits: 6;
- peer passes: 4;
- peer holds: 2;
- peer independent witness at audit time: false;
- v0.13 receipt: `sha256:9daf12733393e19c6149215e6a8f4f2d774f870ff21ba22c04efd785d34c8303`;
- WALDO witness digest: `1113f359f149214e48c20902a73aaffca612b0a54d35c79e8499806b91845739`.

The Go witness validates and independently hashes the observed receipt, verifies the expected dissent topology, preserves closed authority, and rejects a tampered top-level `CANON` authority claim.

## Consequences

The experiment now distinguishes at least four evidence layers:

1. package integrity;
2. emitted-runtime semantic parity;
3. generated peer review;
4. independent external witnessing of the peer-review receipt.

A later experiment can investigate whether independent witnesses may disagree with peer auditors while preserving all verdicts append-only. That future work still must not turn consensus into automatic authority.
