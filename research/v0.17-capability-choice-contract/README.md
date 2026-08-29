# AXM / WALDO v0.17 — feasibility is not selection authority

v0.17 extends the verified v0.16 composition checkpoint without importing or executing the real local AXM Capability Fabric.

Frozen source:

- v0.16 receipt: `sha256:7e8188eb87772ae6ae47ab67bd1dbf158008513ae43c63da680cfb388ca5f81d`
- verified v0.16 GitHub head: `16900d91e60cde5c74bcfe0d51a209d508683742`
- inherited need: `material-evidence-view-completion`
- authority ceiling: `READ_ONLY_PUBLIC_ARTIFACT_ACCESS`

## Experiment

Two proposal-only compositions independently satisfy the same need:

1. `direct-read-chain` — `SKILL -> HAND -> ORGAN` (3 links)
2. `projected-read-chain` — `SKILL -> ORGAN -> HAND -> ORGAN` (4 links)

Both close their typed inputs/outputs and remain read-only. That makes both **feasible**, not authorized for selection.

With `selectionAuthority: NONE`, the required result is:

`HOLD / MULTIPLE_VALID_COMPOSITIONS`

No shortest-path rule, array-order rule, score, winner, or preferred chain is introduced. Advisory priority entries are explicitly null.

## Order and survivor checks

The witness sorts stable candidate IDs before hashing the candidate set. Reversing serialized candidate order therefore preserves the same candidate-set identity and the same HOLD decision.

If only one feasible chain remains while selection authority is still `NONE`, the result is still:

`HOLD / SELECTION_AUTHORITY_ABSENT`

Feasibility and absence of competitors do not manufacture permission.

## Cycle and authority refusal

A retained `cycle-a -> cycle-b -> cycle-a` case must return `HOLD / CYCLE_DETECTED` rather than being silently repaired.

A deliberately invalid child request containing `WRITE_WORKSPACE`, `INSTALL_CAPABILITY`, `PROMOTE`, and `CANON` is retained as `REFUSED / CHILD_AUTHORITY_EXCEEDS_ANCESTOR_GAP`.

## Verification

Detached checks before publication:

- gofmt: PASS
- go vet: PASS
- Go tests: PASS
- sealed fixture: PASS
- candidate order reversal: same set digest / same HOLD
- single survivor: HOLD
- cycle: HOLD
- resealed semantic tampering: rejected
- stale receipt: rejected
- unknown JSON fields: rejected

Repository CI remains independently authoritative after publication.

Canonical v0.17 receipt:

`sha256:62d12d9ef02d3527878ba5e1c9baee6e79dc973d3617caf03796170fc5787dbe`

Order-independent candidate-set digest:

`sha256:d396b84ddaa15f230273e96cb7b6c35dfece8d4d81ffbb46a127cccb2e0b8aa0`

Deterministic witness digest:

`06d3528871b5e3c243af8eac7462a16b919c8d3258bf9a5c171f22dabcd4daf7`

## Truth boundary

- local Capability Fabric executed: **false**
- live AI called: **false**
- candidate selected: **false**
- build/install/promote/CANON: **false**
- serialization order used as priority: **false**
- single survivor auto-selected: **false**
- cycle silently repaired: **false**
- WALDO owns selection: **false**

WALDO remains a downstream scout/witness. This experiment is draft evidence and does not target OpenWALDO upstream.
