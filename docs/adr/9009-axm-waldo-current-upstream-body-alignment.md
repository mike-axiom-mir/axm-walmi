# ADR 9009: Align the witness clone with the current OpenWALDO body

Status: experimental fork decision

## Context

The v0.5 witness stack was based on the original fork point while OpenWALDO
continued to evolve. The previously reviewed upstream head was
`4944072bc3f2fcd5d98d40915d25c7e1535d4877`; the current head selected for this
experiment is `451e029abd1f74fd77625984526a1980c48fb477`.

The new upstream body is not merely documentation. It adds durable row-level
content assessments, main-content classification, compatibility behavior for
mixed schema-1/schema-2 filtering, a conversational compose, more precise
artifact verification, and mandatory privacy redaction before canonical
identity. Those changes alter fields that participate in a corpus BOM's
canonical JSON identity.

The original AXM corpus lens deliberately decoded a typed subset of the WALDO
BOM. Although it accepted unknown additive fields at the JSON boundary, its
canonical `bom_sha256` did not include fields it did not understand. Keeping
that implementation after the upstream change would therefore risk emitting a
digest different from the current WALDO corpus BOM while appearing READY.

## Decision

### Preserve both lineages with a stacked merge

Create v0.6 on the v0.5 experiment head and merge the exact current OpenWALDO
head. The merge keeps the upstream commit ancestry visible instead of copying
or re-authoring upstream changes. The only textual conflict is the ADR index,
which is resolved as the union of upstream ADRs 0052-0055 and downstream AXM
ADRs 9001-9009.

Neither OpenWALDO `main` nor the fork's `main` is changed. v0.6 remains stacked
on v0.5, and merge, promotion, or CANON decisions remain human gates.

The exact observation, source hashes, commit list, and merge parents are in
`research/openwaldo-upstream-alignment-2026-08-16.json`.

### Upgrade the corpus evidence lens to v0.2

Current lens output uses
`axm.waldo-witness.corpus-evidence-lens/v0.2`. It includes three explicit
surfaces that are part of current OpenWALDO corpus identity:

- a digest-bound record-filter policy summary, including whether a global
  policy exists and the number of per-corpus policies;
- deterministic assessment detector identities and aggregate email-address,
  repetitive-content, and boilerplate-content record counts; and
- the exact privacy-redaction policy, names-retained flag, compatible versus
  redacted shard counts, and aggregate transformation counts.

The lens validates current record schema 2 writer v9 and the compatible v8 and
v7 forms, as well as existing schema-1 v5/v4 shards. It verifies matching
assessment and redaction evidence across manifests, shards, and embedded shard
BOMs. Mixed older shards remain visible as legacy or unredacted-compatible;
they are never silently relabelled as assessed or redacted.

For current BOMs, `bom_sha256` is calculated over the complete typed current
WALDO representation, including record filtering, assessment, and redaction.
An integration test constructs and validates a real current `corpus.BOM` with
the upstream package, marshals it with the upstream type, and requires the AXM
lens to emit the identical digest.

### Retain legacy receipt validation

Existing v0.1 corpus-lens receipts remain valid only in their original shape;
they may not acquire v0.2 fields while retaining a v0.1 schema identifier. New
lens operations always emit v0.2. Legacy schema-1 corpus input remains accepted
and receives explicit `NOT_DECLARED`, `NOT_APPLICABLE_LEGACY`, and
`NOT_RECORDED` projections where the current evidence does not exist.

Because the compiled capability catalog changes from corpus lens v0.1 to v0.2,
all checked digest-bound capability handoff examples are deliberately
re-sealed. That digest cascade is a compatibility fact, not a provider run or
content approval.

## Privacy and assessment limits

`waldo/privacy-redaction-v1` replaces recorded email addresses, IP addresses,
phone numbers, and high-confidence credentials and removes recognized mail
routing headers before canonical identity. It explicitly retains names. The
lens reports only the recorded policy and counts. It does not claim anonymity,
GDPR compliance, removal of indirect identifiers, or detection without misses.

Assessment booleans are pinned classifier observations. They do not establish
meaning, quality, harmlessness, legal usability, or causal influence on a
model output. A record-filter digest proves the declared selection contract,
not that a later trainer consumed the selected rows.

## Consequences

The experimental clone now has the current OpenWALDO body and can witness the
new durable corpus facts without pretending the Workshop is local or that the
new body grants autonomy. The change improves provenance continuity and input
hygiene; it does not create a trained clone, a specialist-growth loop, live
provider access, automatic repair, promotion authority, or CANON authority.

Future upstream updates must repeat the same process: pin a fresh head, inspect
the durable-contract delta, preserve both ancestries, test digest compatibility
against upstream types, and add a new receipt rather than rewriting this one.
