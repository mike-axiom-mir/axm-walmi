# AXM / WALDO v0.18 — selection receipt does not grant execution

v0.18 extends the verified v0.17 choice checkpoint without importing or executing the real local AXM Capability Fabric.

Frozen source evidence:

- v0.17 contract receipt: `sha256:62d12d9ef02d3527878ba5e1c9baee6e79dc973d3617caf03796170fc5787dbe`
- v0.17 verified GitHub head: `6b5071cf7a632291e5d44c6d80909d0ae73ee937`
- v0.17 candidate-set digest: `sha256:d396b84ddaa15f230273e96cb7b6c35dfece8d4d81ffbb46a127cccb2e0b8aa0`

## Question

Can a selection-only governance receipt identify one already-frozen feasible proposal without granting WALDO execution, installation, promotion, or CANON authority?

The experiment uses a **contract fixture**, not a real governance approval. The fixture explicitly records:

`authenticity: FIXTURE_ONLY_NOT_REAL_APPROVAL`

and:

`realGovernanceDecisionObserved: false`

This prevents a contract probe from being narrated later as an actual human decision.

## Primary case

The fixture references the exact v0.17 candidate set and selects:

`direct-read-chain`

The selected chain is additionally bound by its canonical candidate digest:

`sha256:8ed1d23b0c7020fe1f3ee2427c2984c6132c46e49f86b0183828c4c1dc775241`

The receipt scope is only:

`SELECT_PROPOSAL`

The verifier may therefore record:

`CONTRACT_SELECTION_ACCEPTED`

and:

`SELECTED_PROPOSAL_ONLY`

but the overall state remains:

`HOLD / EXECUTION_AUTHORITY_ABSENT`

Selection is a fact about which proposal a governance contract fixture points to. It is not permission to act.

## Refusal cases

The verifier independently refuses:

- stale candidate-set identity → `REFUSED / CANDIDATE_SET_MISMATCH`
- unknown candidate ID → `REFUSED / CANDIDATE_NOT_IN_FROZEN_SET`
- candidate digest mismatch → `REFUSED / CANDIDATE_DIGEST_MISMATCH`
- selection receipt that widens into build authority → `REFUSED / SELECTION_SCOPE_EXCEEDED`
- WALDO issuing its own selection authority → `REFUSED / WALDO_CANNOT_SELF_ISSUE_SELECTION`

A duplicate of the same receipt does not create a conflict or extra authority.

## Conflicting receipts

A second individually valid fixture selects the other frozen candidate:

`projected-read-chain`

Both receipts remain addressable. The result is:

`HOLD / CONFLICTING_SELECTION_RECEIPTS`

`lastWriterWins` is explicitly false. No majority rule, time ordering, array ordering, or WALDO preference resolves the conflict.

## Derived candidate digests

The two candidate digests were independently recomputed from the exact v0.17 candidate structs using Go `json.Marshal` plus SHA-256:

- `direct-read-chain`: `sha256:8ed1d23b0c7020fe1f3ee2427c2984c6132c46e49f86b0183828c4c1dc775241`
- `projected-read-chain`: `sha256:721ba695c6d8243ea8e24c5c5a52414c7ef6bf456ca5ea67975069a80b74fdf9`

## Detached verification

Before publication, the detached v0.18 package passed:

- gofmt
- Go vet
- Go tests
- exact fixture verification
- deterministic witness generation
- duplicate-same-selection non-conflict
- resealed semantic-tamper rejection
- stale receipt rejection
- unknown JSON field rejection

Repository CI remains independently authoritative after publication.

Canonical v0.18 receipt:

`sha256:768d32a9bc59008065f09b31e8b09be6b1fd20ce77dd9d578c1b76af2474b06f`

Deterministic witness digest:

`1bbb03715c787a397987d50fb0629c928bb4108f773e48cced8838ec1adcfe57`

## Truth boundary

- real governance approval observed: **false**
- local Capability Fabric executed: **false**
- live AI provider called: **false**
- WALDO issued selection authority: **false**
- selected proposal executed: **false**
- build started: **false**
- installed: **false**
- promoted: **false**
- CANON changed: **false**

WALDO remains a verifier/witness over the contract evidence. It does not become the chooser or executor.
