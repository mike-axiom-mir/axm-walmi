# AXM/WALDO v0.13 — Generated Peer-Audit Ring

Status: **EXPERIMENTAL / OBSERVED / DISSENT PRESERVED / NO AUTHORITY**

v0.13 keeps the v0.12 generated-organ lineage fixed and asks a different question:

> Can generated organs audit other generated organs from typed evidence without gaining authority over their targets or silently collapsing disagreement into a majority decision?

## Frozen source

- v0.12 patched parity receipt: `sha256:5fcf0530cb97e6568793baaae9c0a9ef4cc4285c235a590665986b27460799a6`
- donor packet: `sha256:8f4688db3abcba988011f2a2f36f172a09c3d848e57a2da54db90abd2669e634`
- detached patched Organ runtime: `1.0.1-v012-parity`
- peer-audit field pack: `sha256:5ebd8a83f405243f501dcd4930e5279b86c4e39dbc082f66ffc6ccab26aaaaf7`

The peer-audit pack is an experimental field pack for this research branch. It is not a claim that the canonical Organ Fabric has adopted it.

## Generated auditors

The same deterministic candidate factory produced three audit candidates: `lean`, `balanced`, and `guarded`.

Generation produced 3 candidates and 0 failures. Each auditor package was independently required to remain package-verified, selftest-passing, behavior-parity checked, closed-input checked, authority-closed and lineage-bound before entering the ring.

The auditors consume **typed evidence summaries**. They do not receive target mutation, install, registration, selection, staging, promotion, merge or CANON authority.

## Observed run

Against the 12 known-good v0.12 generated candidates:

- target candidates: **12**
- auditor strategies: **3**
- audit executions: **36**
- `EVIDENCE_PASS`: **36**
- `HOLD`: **0**

Deliberately weakened evidence then produced strategy-specific dissent:

| Evidence gap | lean | balanced | guarded |
| --- | --- | --- | --- |
| closed-input parity missing | PASS | HOLD | HOLD |
| authority not closed | PASS | PASS | HOLD |
| lineage unbound | PASS | PASS | HOLD |
| independent witness missing | PASS | PASS | HOLD |

A pass from a narrower strategy does **not** erase its findings and does not overrule a stricter hold.

## Peer ring

Each generated auditor then audited the other two generated auditors. Self-audit was excluded.

- peer audit executions: **6**
- `EVIDENCE_PASS`: **4**
- `HOLD`: **2**
- independent witness supplied to the ring: **false**

The two holds are the guarded auditor reviewing the lean and balanced auditors. It refused to fully pass either peer because the ring itself did not contain an independent witness. Lean and balanced accepted the narrower evidence but still emitted `INDEPENDENT_WITNESS_MISSING` as a finding.

This disagreement is the result, not an error to be voted away.

## WALDO/Mirror boundary

WALDO is added **after** the ring as an external receipt witness. It validates the observed ring, bindings, counts, dissent pattern, truth closure and receipt digest. WALDO does not become a fourth voter and does not retroactively rewrite the historical guarded holds.

The independent Go witness produced:

`1113f359f149214e48c20902a73aaffca612b0a54d35c79e8499806b91845739`

The Go verifier also rejects a receipt whose top-level authority is changed to `CANON`.

## Receipts

Observed v0.13 receipt:

`sha256:9daf12733393e19c6149215e6a8f4f2d774f870ff21ba22c04efd785d34c8303`

The sealed receipt used by the Go witness is in `internal/axmmirror/testdata/peer-audit-ring-v0.13.json`. A detached replay regenerated the receipt and peer-audit pack byte-for-byte from the frozen v0.12 donor inputs; `replay-summary.json` records the replay result.

## Truth boundary

Observed generated auditor code executed in disposable experiment roots. Target code was not mutated. No live AI provider was called by the harness and the harness requested no network access. No candidate was selected, installed, registered, staged, promoted, merged, admitted to persistent learning or made CANON. No quorum or majority result has authority.
