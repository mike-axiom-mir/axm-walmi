# v0.20 — revocation stops future action without rewriting history

This GitHub-only experiment extends the independently green v0.19 checkpoint without importing or executing the real local AXM Capability Fabric.

## Frozen source

- v0.19 contract receipt: `sha256:88885cdb000a567746030914f170c161444db87ec6116ba8ce4b6a58da6257dc`
- verified v0.19 head: `ecf1be23166e8445677aaede78cfb20956ad7a7e`
- v0.19 deterministic witness: `960c7eab77597a87256da843663e88fe22d1290fcc1e42dbff1b1da14bfee7d3`
- inherited permit: `external-disposable-execution-permit-direct-read`
- selected proposal: `direct-read-chain`

## Challenge

`REVOCATION_STOPS_FUTURE_ACTION_WITHOUT_REWRITING_HISTORY`

v0.19 separated permission from action. v0.20 asks whether a later revocation can close future execution authority without erasing the earlier accepted permit.

The answer in this contract probe is yes, through an append-only authority ledger.

### Sequence 1

`v019-permit-accepted`

- event: `PERMIT_ACCEPTED`
- outcome: `ACCEPTED`
- rewritten later: **false**

### Sequence 2

`v020-permit-revoked`

- event: `PERMIT_REVOKED`
- outcome: `REVOKED`
- references sequence-1 permit event
- rewrites earlier event: **false**

Current state:

`REVOKED_BEFORE_EXECUTION`

The revocation itself is explicitly `FIXTURE_ONLY_NOT_REAL_REVOCATION`.

## Temporal evaluator

The pure evaluator exercises both sides of the revocation boundary without executing anything:

| Request position | Body | Result |
| --- | --- | --- |
| before revocation | present (hypothetical) | `READY / EXECUTION_AUTHORIZED_NOT_EXECUTED` |
| before revocation | absent | `HOLD / EXECUTABLE_BODY_ABSENT` |
| after revocation | present (hypothetical) | `REFUSED / PERMIT_REVOKED` |
| after revocation | absent | `REFUSED / PERMIT_REVOKED` |

A revocation therefore changes current authority, not historical truth.

## Refusal / HOLD surface

The experiment preserves seven explicit cases:

1. `REVOCATION_SOURCE_MISMATCH`
2. `REVOCATION_TARGET_MISMATCH`
3. `REVOCATION_SCOPE_EXCEEDED`
4. `WALDO_CANNOT_SELF_ISSUE_REVOCATION`
5. `HISTORY_REWRITE_FORBIDDEN`
6. `REVOCATION_CANNOT_GRANT_AUTHORITY`
7. `REVOCATION_ORDER_UNRESOLVED`

An exact duplicate revocation is `DEDUPLICATED` with no extra authority and no history rewrite.

## Detached verification

The new v0.20 files were exercised in an isolated standard-library Go module using the runtime available to this chat (`go1.23.2`). This is deliberately not represented as a substitute for the repository's Go 1.25 CI matrix.

Passed locally:

- `gofmt`
- `go vet ./...`
- `go test ./... -count=1`
- exact sealed fixture verification
- deterministic witness generation
- pre/post-revocation temporal outcomes
- seven refusal/HOLD evaluator cases
- append-only ledger validation
- resealed semantic-tamper rejection
- stale-receipt rejection
- unknown-field rejection

Repository CI remains independently authoritative for whole-repo compatibility.

## Canonical evidence

- v0.20 contract receipt: `sha256:edb16e8f804b6cace4e59eaf7f66344ef31648e52b028129eae373224e3645c2`
- deterministic witness: `8038f833ad35b2ea7714161c70ad473a2dd1e7feb441d6805d43415b97d81f73`

## Truth boundary

- real revocation observed: **false**
- local Capability Fabric included/executed: **false**
- executable candidate body included: **false**
- live AI provider called: **false**
- candidate executed: **false**
- earlier permit history deleted/rewritten: **false**
- install/promote/CANON: **false**
- WALDO owns revocation: **false**

This experiment is authority-history evidence only. Keep the PR draft and unmerged by default; do not send AXM changes to OpenWALDO upstream.
