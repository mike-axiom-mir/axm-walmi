# ADR 9004: Complete Wave 1 with corpus and training-run witnesses

Status: experimental fork decision

## Context

The first fork organs could anchor a model or release, detect answering-identity
drift, check declared evaluation overlap, and seal behavior evidence. They could
not yet show which exact corpus BOM and immutable run plan sat behind a
run-backed answering identity. A caller could provide `corpus_bom_sha256` and
`run_bom_sha256`, but no deterministic organ checked those links against
WALDO's durable records.

WALDO already separates immutable `RUN-BOM.json` from mutable lifecycle state in
`RUN.json`. Its corpus BOM also separates recorded source/license identities,
selected shard pins, and exact aggregates from claims that hashes cannot prove.
The fork should preserve those distinctions rather than invent a parallel
provenance format.

## Decision

Add two fork-only, read-only organs and a stronger behavior-seal path.

### Corpus Evidence Lens

`LensCorpusBOM` accepts the documented schema-1 `openwaldo-bom` subject
`corpus`. It validates the current durable fields, including:

- lowercase Git identity when a commit is recorded;
- sorted, unique selection paths, including WALDO's empty-string index-root
  selection;
- manifest, source, source-file, submanifest, shard, records-root, and embedded
  attestation digests;
- resolved conversion fields and manifest/shard references;
- license policy syntax and selected-shard policy compliance;
- exact shard, document, token, byte, modality, manifest, and license totals;
- multi-license usage partitions; and
- the structural forms of embedded, implicit-v4, deep-validated, and absent
  shard attestations.

The compact receipt preserves selected paths, index state, manifest and source
pins, source-file set digests, the ordered shard-set digest, exact totals,
license assertions, and an attestation summary. License entries are always
labelled `RECORDED_ASSERTION`; they are not converted into legal advice.

The lens reports Git identity honestly:

- `CLEAN_PINNED` for a clean recorded commit;
- `DIRTY_MANIFEST_PINNED` when the checkout was dirty; and
- `MANIFEST_PINNED_NO_COMMIT` when no Git commit was available.

Missing or dirty Git identity does not erase the exact manifest and shard
digests already carried by the BOM.

The original v0.1 schema-1 projection accepts unknown additive JSON fields, as
WALDO's corpus compatibility contract requires. Duplicate object keys and
trailing JSON values remain ambiguous and are rejected. `bom_sha256` hashes the
known typed representation used for WALDO BOM identity; `document_sha256`
separately binds the exact bytes observed by the organ. A future durable field
that changes BOM identity requires an explicit schema or compatibility update
in this fork rather than a silent claim of support.

ADR 9009 applies that rule after OpenWALDO added record-filter, assessment, and
privacy-redaction fields. New projections use v0.2 and bind those current fields
into the canonical BOM digest. They expose exact policy/detector identities and
counts without turning assessment into meaning or redaction into anonymity.
Legacy v0.1 receipts remain validation-compatible in their original shape.

The lens also carries `receipt_sha256`, calculated over the receipt with that
field omitted. This detects accidental mutation of the projected receipt; it
is not a signature or proof of who produced it.

### Training Run Witness

`WitnessTrainingRun` consumes one documented `RUN-BOM.json` and its `RUN.json`.
It:

- validates the immutable run, model, stage, architecture, backend, framework,
  runtime, host, topology, parameter, evaluation-set, and initialization facts;
- projects and validates the embedded corpus BOM;
- requires the embedded corpus digest to equal `corpus_bom_sha256`;
- requires `RUN.json` to pin the canonical run BOM digest;
- validates planned, running, complete, failed, and interrupted lifecycle
  invariants, attempts, progress, observations, checkpoints, evaluations,
  consumption accounting, and recorded artifact identities; and
- emits separate canonical and exact-document digests for the run BOM and run
  record, plus observation or progress digests when present.

A structurally valid non-terminal, failed, interrupted, or simulated run still
produces a receipt, but its state is `HOLD`. Only a complete, non-simulated run
with a complete observation and recorded output artifact can be `READY`.

The run witness uses the same self-digest convention. Its receipt digest binds
the embedded corpus lens, run lifecycle state, observation/progress digest,
environment, parameter summary, and boundary notices. It remains tamper
evidence, not authentication.

The organ does not follow artifact paths or reread artifact bytes. WALDO's own
model inspection and verification paths remain responsible for proving that a
persisted file currently matches its recorded digest. The witness reports the
identity WALDO recorded; it does not upgrade that record into a new byte-level
verification claim.

### Witnessed behavior sealing

`SealWitnessed` is restricted to run-backed anchors. It requires:

- an `ANCHORED` model or release identity;
- a `READY` training-run witness;
- matching model and run IDs between anchor and witness;
- the behavior draft's run BOM digest to match the witness; and
- the behavior draft's corpus BOM digest to match the witness's embedded corpus
  lens.

It then applies the existing anchor match and behavior seal. Origin-backed
models continue to use `seal-anchored`, because inventing a training-run link
for an imported origin would be false.

The witnessed seal adds (or validates) a `waldo-training-run-witness` passing
verification whose evidence digest is the exact run-witness receipt digest.
The final behavior record therefore retains the specific completed-run receipt,
not merely the immutable plan digest.

## Durability and authority

The receipt schemas are:

- `axm.waldo-witness.corpus-evidence-lens/v0.2` for new projections;
- `axm.waldo-witness.corpus-evidence-lens/v0.1` for legacy validation; and
- `axm.waldo-witness.training-run-witness/v0.1`.

Both carry closed authority. They cannot execute tools, start training, promote
a model, alter CANON, or act in the world. CLI writes retain the existing
atomic no-replace behavior. A HOLD run receipt is written before the CLI exits
nonzero so incomplete evidence remains inspectable.

## Consequences

Wave 1 now has a deterministic chain from corpus selection and run plan through
answering identity, evaluation comparison, and sealed behavior evidence. The
chain remains an identity and evidence mechanism: it does not prove source
truth, legal usability, safety, actual causal influence on an output, or
semantic independence of an evaluation set.

The next useful work is comparative rather than adding general-assistant
organs: a bounded provenance context surface, source-claim gate, behavior delta
comparator, dissent continuity, and reproducibility comparison.
