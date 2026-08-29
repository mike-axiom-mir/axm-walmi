# v0.13 replay protocol

The observed v0.13 run was replayed against the same detached donor inputs used by v0.12. The replay regenerated both the observed receipt and the generated peer-audit field pack byte-for-byte.

## Required frozen inputs

1. v0.12 detached Organ donor using candidate factory `1.0.1-v012-parity`.
2. v0.11 proposal bridge containing the same three deterministic proposals plus the AI-classified fourth proposal.
3. No live AI provider. The fourth proposal remains a pre-existing proposal whose source class is `AI`.
4. No network dependency in the harness.

## Procedure

1. Generate three `peer-evidence-audit` candidates: lean, balanced, guarded.
2. Materialize each package in a disposable root and run its package selftest.
3. Check required/held-out parity and an unknown-input refusal against the trusted deterministic runtime.
4. Regenerate the 12 v0.12 target candidates from the same four proposals × three strategies.
5. Feed each target's typed evidence summary to each auditor: 36 executions.
6. Run the four incomplete-evidence dissent cases without changing auditor policy.
7. Feed each generated auditor's typed evidence summary to the other two auditors; exclude self-audit.
8. Seal the receipt with authority `NONE` and no quorum authority.
9. Compare regenerated receipt bytes and field-pack bytes with the original observed artifacts.

## Expected identities

- receipt: `sha256:9daf12733393e19c6149215e6a8f4f2d774f870ff21ba22c04efd785d34c8303`
- peer-audit pack: `sha256:5ebd8a83f405243f501dcd4930e5279b86c4e39dbc082f66ffc6ccab26aaaaf7`
- auditor generation run: `sha256:fa7fd4557973eb0afa5cf39c0ccdeb438586046053a9834d0281a4a60f63010f`
- WALDO witness: `1113f359f149214e48c20902a73aaffca612b0a54d35c79e8499806b91845739`

## Expected topology

- 36/36 known-good target audits: `EVIDENCE_PASS`.
- dissent cases: 4, preserving lean/balanced/guarded disagreement.
- peer audits: 6 total; 4 pass; 2 hold.
- both peer holds are guarded-strategy holds for missing independent witness.
- no self-audit and no majority/quorum authority.

`replay-summary.json` records the completed replay result. The Go witness independently verifies the sealed observed receipt rather than trusting this protocol document.
