# v0.22 — Platform Hermes run-capsule evidence contract

## Question

Can WALDO verify a complete run-evidence capsule without treating evidence, authorization, or a Return Packet as promotion authority?

## Donor

The donor is AXM Platform PR #48 at exact head:

`7290dacf4fe2bb9a502f6a339fc53ecdbe0d420b`

The dedicated `Hermes AXM Runtime Gate` passed at workflow run `32672295737`.

The donor already defines an AXM-owned vocabulary:

`axm.hermes-run-manifest/v1` → `axm.hermes-tool-receipt/v1` + `axm.hermes-provider-receipt/v1` + `axm.hermes-session-event/v1` → `axm.hermes-return-packet/v1`.

v0.22 does not copy or execute that runtime. It freezes the contract semantics into a pure WALDO-side verifier fixture.

## Challenge

`RUN_CAPSULE_EVIDENCE_IS_NOT_PROMOTION`

Seven deterministic cases are retained:

1. complete evidence → `CANDIDATE / EVIDENCE_COMPLETE_NOT_PROMOTED`
2. authorized call without matching completion receipt → `HOLD / COMPLETION_RECEIPT_GAP`
3. remote provider observed under local-only policy → `HOLD / PROVIDER_POLICY_MISMATCH`
4. policy hash changes during run → `HOLD / POLICY_CHANGED_DURING_RUN`
5. source not verified after run → `HOLD / SOURCE_NOT_VERIFIED_AFTER_RUN`
6. consent revoked after authorization while completion evidence arrives → `OBSERVED / COMPLETION_EVIDENCE_RETAINED_AFTER_REVOCATION`
7. raw prompt storage requested → `REFUSED / RAW_CONTENT_STORAGE_FORBIDDEN`

The revocation case is deliberately asymmetric: evidence for a previously authorized in-flight action may finish recording, but revocation does not grant another action.

## Truth boundary

- Platform Hermes runtime copied: **false**
- Platform Hermes runtime executed by WALDO: **false**
- live Hermes run observed: **false**
- live AI provider called: **false**
- raw prompt/response/tool bodies retained: **false**
- Return Packet promotes automatically: **false**
- WALDO owns Hermes runtime: **false**
- CANON changed: **false**

## Detached verification

- `gofmt`: PASS
- `go vet ./...`: PASS
- `go test ./... -count=1`: PASS
- exact donor identities: PASS
- seven semantic cases: PASS
- resealed semantic tampering: rejected
- stale receipt: rejected
- unknown field: rejected

Canonical contract receipt:

`sha256:2bd0aeb46770bae2a95a0f0c12bf63af02ad80a8abb746f6355692e22a51a879`

Deterministic witness:

`8605099f2176b7bc0c9587ae1ad6cf53dd504523bcc494aee8fb1f0d660755e3`
