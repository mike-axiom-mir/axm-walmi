# ADR 9003: Add bounded Wave 1 provenance organs before any learned clone

Status: experimental fork decision

## Context

ADR 9002 defines the WALDO Witness Mirror as a provenance witness rather than
a second general-purpose Mirror. The first behavior-evidence seal accepted a
lineage digest supplied by its caller, but it did not yet prove that the digest
came from a structurally valid WALDO model or release BOM. It also lacked an
identity-drift gate and a bounded held-out overlap check.

Those checks must exist outside learned weights. A model can reason about their
receipts, but it cannot author its own origin, clear its own evaluation, or gain
authority from them.

## Decision

Add three fork-only deterministic organs under `internal/axmmirror` and expose
them through the separate `waldo-axm-mirror` CLI.

### Origin Anchor Organ

`AnchorWALDOBOM` strictly decodes the documented schema-1 `openwaldo-bom`
subjects `model` and `model-release`. It rejects unknown and duplicate JSON
fields, ambiguous current selections, simulated selected runs, invalid or
escaping paths, malformed digests, duplicate artifacts, and selected managed
models without the weights, configuration, and tokenizer roles WALDO requires
for inference. Release packages must retain a weights artifact.

The receipt records two different digests:

- `bom_sha256` is the SHA-256 of the typed canonical JSON representation used
  by WALDO's BOM hashing contract;
- `document_sha256` is the SHA-256 of the exact input bytes observed by this
  organ.

`ANCHORED` means the BOM structurally selects a complete non-simulated run or a
recorded origin, or is a structurally valid release BOM. A model BOM without a
selected answering source produces `HOLD`. The organ validates recorded
artifact identities but deliberately does not follow paths or reread artifact
bytes, so the receipt is not an independent artifact-verification claim.

### Release Identity Lock Organ

`CompareIdentity` compares an expected anchor with a newly observed BOM anchor.
It emits:

- `LOCKED` when the selected answering identity is unchanged;
- `DRIFT` when model/release identity, selected source, architecture, format,
  source BOM, run, or recorded artifacts change;
- `HOLD` when either model anchor lacks a selected answering source.

Append-only model history may change the model BOM while leaving the lock
`LOCKED` if the selected real run and artifacts remain identical. A release BOM
is immutable identity: any release BOM digest change produces `DRIFT`, even if
its artifact list happens to remain the same.

### Evaluation Independence / Contamination Guard Organ

`CheckContamination` intersects explicitly declared training and evaluation
inventories across record hashes, text hashes, source groups, and corpus paths.
The report binds the canonical comparison by SHA-256 and lists the dimensions
that were actually comparable on both sides.
It emits:

- `CONTAMINATED` for any exact overlap;
- `HOLD` when no overlap is found but either declared inventory is incomplete;
- `HOLD` when populated inventories lack a shared identity dimension;
- `CLEAR` only when both declared inventories are complete and disjoint within
  the dimensions they enumerate.

`CLEAR` is bounded to exact declared identities. It does not establish semantic
independence, absence of paraphrase leakage, absence of memorization, or
chronological separation. Outside authorship remains a recorded declaration,
with an optional evidence-document digest; the organ never upgrades that
declaration to independently verified fact.

### Anchored behavior evidence

`SealAnchored` requires the behavior draft's model or release BOM digest to
match an `ANCHORED` receipt before applying the existing behavior-evidence
seal. The original `seal` command remains for compatibility and isolated
contract testing, but `seal-anchored` is the intended experimental path.

## Durability and authority

Every new receipt carries closed authority: no tool execution, training,
promotion, CANON, or world action. CLI outputs are written from a synced
temporary file with an atomic no-replace commit. Existing receipts are never
silently overwritten.

The v0.1 schemas are:

- `axm.waldo-witness.origin-anchor/v0.1`;
- `axm.waldo-witness.release-identity-lock/v0.1`;
- `axm.waldo-witness.evaluation-comparison/v0.1`;
- `axm.waldo-witness.contamination-report/v0.1`.

Schema changes require a new version or an explicit compatibility decision.

## Consequences

The fork can bind behavior evidence to a real WALDO model/release BOM contract,
detect silent answering-identity changes, and refuse obvious or unresolved
evaluation overlap. ADR 9004 extends this with corpus and run witnesses. The
fork still has no learned clone, inference integration, semantic leakage
detector, legal conclusion, or source-to-output causal attribution.
