# ADR 9008: Add a portable capability spine without importing the Workshop

Status: experimental fork decision

## Context

The WALDO witness clone now has provenance, evaluation, situated-evidence, and
small deterministic asset-creation boundaries. It still does not contain the
Workshop. That is intentional, but it leaves one practical gap: the specialist
cannot describe its own bounded deterministic surfaces, compare them with an
external capability inventory, or prepare and verify a narrow external handoff.

Copying the Workshop would erase the specialist boundary. Allowing the learned
clone to discover or invoke providers directly would also make fluent model
output an authority source for capability, permission, schema compatibility,
and execution.

The public AXM platform `main` snapshot at commit
`a4f99fbfc05268173458bf3fb8f3fe616919e376` contains four useful contracts:

- Technical Glasses exposes source-fingerprinted machine-readable technical
  ground truth and refuses to treat structural scans or stale snapshots as
  current runtime proof;
- Workshop Needs Observatory exposes an exact capability inventory and typed
  gaps while refusing fuzzy capability claims and automatic building;
- Handoff Wiring Observatory preserves exact producer/consumer schemas,
  unmatched handoffs, source attribution, and freshness while refusing
  semantic or wildcard compatibility inference and automatic rewiring;
- Adapter Translation Garden preserves interface mappings, translation loss,
  confinement, and fidelity evidence while refusing hidden loss, permission
  inference, automatic orchestration, installation, promotion, and CANON.

Their exact paths, Git blob IDs, SHA-256 digests, and claim ceilings are pinned
in `research/portable-capability-spine-knowledge-sources-2026-08-15.json`.
Those files are source-only knowledge. No platform runtime or Workshop hand was
invoked or observed when this decision was made.

## Decision

Add five ordinary-Go, fork-only organs that form a portable capability spine:

1. self capability census;
2. external capability snapshot intake;
3. exact capability-gap and handoff planning;
4. translation-loss sealing;
5. digest-bound return verification.

The organs exchange strict versioned JSON. They do not contain a provider
client, Workshop router, adapter runtime, installer, repair path, permission
surface, or transport.

### Self capability census

Schemas:

- `axm.waldo-witness.self-capability-census-request/v0.1`;
- `axm.waldo-witness.self-capability-snapshot/v0.1`.

The census emits a canonical compile-time catalog of the deterministic
`waldo-axm-mirror` contract surfaces. Each item names its stable capability ID,
version, CLI command, exact accepted schemas, exact emitted schemas, and
`COMPILED_DECLARATION` evidence state.

The request supplies an exact observed build digest and a separate observation
receipt digest. Those are external bindings. The census validates and carries
them but does not read its own executable, inspect its Git checkout, rerun
tests, or author its own build provenance. A snapshot therefore says what the
compiled catalog declares, not that every command is healthy on the current
host or that a learned clone can use it successfully.

The catalog and complete snapshot each carry their own SHA-256 digest.

### External capability intake

Schemas:

- `axm.waldo-witness.external-capability-snapshot/v0.1`;
- `axm.waldo-witness.external-capability-receipt/v0.1`.

An external snapshot carries exact source documents, observation and assessment
times, TTL, inventory completeness, and one of four observation states:

- `LIVE` requires a runtime-evidence digest and may carry
  `OBSERVED_AVAILABLE`, `OBSERVED_UNAVAILABLE`, or `UNKNOWN` capabilities;
- `SOURCE_ONLY` carries declarations but cannot claim observed availability;
- `ABSENT` requires a complete empty inventory plus an external evidence
  digest;
- `UNKNOWN` requires an explicitly incomplete empty inventory.

The intake derives age and freshness only from supplied timestamps. Future,
stale, and incomplete observations remain distinct holds. It never consults
wall-clock time during replay. A fresh `SOURCE_ONLY` receipt remains
source-only; freshness cannot promote it into live readiness.

Each capability preserves provider identity, version, execution class,
declaration state, exact input/output schemas, accepted data classes, required
permissions, and refusals. Metadata is not permission or runtime proof.

### Exact gap and handoff planning

Schemas:

- `axm.waldo-witness.capability-gap-request/v0.1`;
- `axm.waldo-witness.capability-handoff-plan/v0.1`.

A gap request binds one target answering identity, capability ID, input artifact
schema/digest/size, required output schema, data class, output-byte ceiling,
creation time, and expiry.

Planning first checks the exact self catalog. An exact local match produces
`NO_EXTERNAL_HANDOFF_REQUIRED`. Otherwise, candidates are derived only from
exact capability IDs. The planner does not normalize schema names, use
wildcards, infer semantic compatibility, or choose by prose similarity.
Before considering an external candidate, it re-evaluates the snapshot age at
the gap request's explicit `created_at`; a receipt that was fresh when assessed
cannot be replayed as fresh after its TTL.

The main outcomes are deliberately separate:

- `HANDOFF_CONTRACT_READY`: one current exact provider candidate with no
  declared permission requirements;
- `HOLD_EXTERNAL_CAPABILITY_SOURCE_ONLY` or
  `HOLD_EXTERNAL_CAPABILITY_SNAPSHOT`: external evidence is not current live
  readiness;
- `HOLD_CAPABILITY_UNAVAILABLE`: no observed available provider;
- `HOLD_CAPABILITY_DATA_CLASS`: no available provider accepts the declared
  data class;
- `HOLD_SCHEMA_TRANSLATION_REQUIRED`: capability and data class match but
  schemas do not;
- `HOLD_PROVIDER_AMBIGUITY`: multiple exact providers require explicit human
  selection;
- `HOLD_PERMISSION_REVIEW`: the unique exact provider declares permissions
  that must be decided outside the plan.

Even a READY plan records only a `unique_candidate_provider_id`. It always has
`provider_selected: false`, `external_execution_allowed: false`, closed
authority, and an explicit human-review requirement. A unique candidate is not
an invocation or permission grant.

### Translation-loss sealing

Schemas:

- `axm.waldo-witness.capability-translation-declaration/v0.1`;
- `axm.waldo-witness.translation-loss-receipt/v0.1`.

A declaration binds one exact plan and one named candidate. It records input
and output schema legs plus canonical field mappings. Supported mapping methods
are `IDENTITY`, `RENAME`, `UNIT_CONVERSION`, `APPROXIMATION`, `DROP`, and
`SYNTHESIZE`; every mapping separately declares `NONE`, `DECLARED`, or
`BLOCKING` loss.

Loss-bearing methods cannot declare `NONE`. Different schemas require an
explicit mapping in the affected direction. The resulting states are:

- `TRANSLATION_NOT_REQUIRED`;
- `TRANSLATION_NO_DECLARED_LOSS`;
- `REVIEW_DECLARED_TRANSLATION_LOSS`;
- `HOLD_BLOCKING_TRANSLATION_LOSS`.

`NO_DECLARED_LOSS` is intentionally not called lossless. It means only that
the declaration names no loss. The receipt neither inspects schema definitions
nor proves equivalence, conformance, fidelity, or successful conversion. It
does not generate or execute an adapter and cannot upgrade a held plan.

### Digest-bound return verification

Schemas:

- `axm.waldo-witness.capability-return-draft/v0.1`;
- `axm.waldo-witness.capability-return-receipt/v0.1`.

A return draft contains no paths, URLs, or raw payloads. It binds:

- exact plan and translation-receipt digests;
- provider ID and original input artifact identity;
- output artifact IDs, schemas, digests, and sizes;
- a provider receipt digest;
- start, completion, and assessment times;
- externally reported runtime and permission state;
- cleanup state and retained-input byte count;
- closed authority.

Verification preserves independent checks for joins, plan readiness,
translation state, expiry and timestamp order, runtime completion, external
permission report, cleanup/retention, and output schema/byte limits. The first
failed category produces a typed HOLD. `RETURN_BINDINGS_VERIFIED` requires all
checks to pass.

The verifier does not fetch or rehash output bytes. It therefore proves only
the integrity of supplied declarations and receipt bindings. It cannot prove
provider identity, output content, semantic or visual quality, usefulness,
safety, legality, permission enforcement, or successful cleanup in the world.

## CLI

The separate experimental CLI adds:

```text
waldo-axm-mirror census-capabilities <census-request.json> <self-snapshot.json>
waldo-axm-mirror intake-capabilities <external-snapshot.json> <external-receipt.json>
waldo-axm-mirror plan-handoff <self-snapshot.json> <external-receipt.json> <gap-request.json> <plan.json>
waldo-axm-mirror seal-translation <plan.json> <translation-declaration.json> <translation-receipt.json>
waldo-axm-mirror verify-handoff-return <plan.json> <translation-receipt.json> <return-draft.json> <return-receipt.json>
```

HOLD receipts are written before the command exits nonzero. Source-only intake
is itself a valid intake result, but any plan based on it remains held. Writes
remain atomic and no-replace.

## Durability and compatibility

Strict decoding rejects duplicate keys, unknown fields, and trailing content.
Sets, capability inventories, provider candidates, field mappings, and output
artifacts are canonicalized before digesting. All time evaluation is explicit
and replayable. Source documents, external capabilities, translation mappings,
artifact sizes, output counts, TTL, and plan lifetime have hard bounds.

The public-safe example chain intentionally uses the real platform contracts as
`SOURCE_ONLY`. It pins self snapshot, external receipt, held handoff plan,
translation receipt, and held return receipt digests. Separate synthetic tests
exercise a LIVE unique-provider path through `RETURN_BINDINGS_VERIFIED`, plus
stale, ambiguous, permission, schema, translation-loss, expiry, retention,
tamper, and authority-injection paths.

Changing durable meaning requires a new schema or explicit compatibility
decision. A self-digest remains an integrity check, not a signature or author
identity.

## Authority boundary

The portable capability spine does not:

- copy or emulate the Workshop runtime;
- discover a live provider by itself;
- invoke, select, install, enable, repair, or monitor a provider or adapter;
- infer schema semantics or hide translation loss;
- inherit, request, or grant permissions;
- move raw task payloads or retrieve returned artifact bytes;
- approve output content, visual quality, usefulness, safety, or legality;
- train a clone, publish an asset, promote a result, alter CANON, or act in the
  world.

Mike remains the merge and CANON gate.

## Consequences

The WALDO specialist can now know its own narrow declared surfaces and prepare
portable, reviewable capability contracts when it lacks a local capability.
It can also preserve why a route is held, what translation loss was declared,
and whether a returned digest set matches the approved contract shape.

This improves the clone without making it the same as Mirror or placing the
Workshop inside it. Live transport, resource leases, provider selection,
adapter execution, artifact-byte verification, visual review, continuity, and
replay remain later independent decisions.
