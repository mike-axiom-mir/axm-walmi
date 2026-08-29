# ADR 9005: Witness training profile contracts without rewriting run history

Status: experimental fork decision

## Context

ADR 9004 completed the first deterministic chain from a WALDO corpus BOM and
training run through behavior evidence. The next upstream comparison found that
WALDO now names three different data-selection behaviors explicitly:

- `causal-pretrain-shuffled`;
- `causal-pretrain-balanced`; and
- `causal-pretrain-weighted`.

The older `causal-pretrain-v1`, `causal-pretrain-v2`, and
`causal-pretrain-v3` values remain historical aliases. Current upstream also
persists token budgets and structured conversation transforms in schema-1 run
BOMs. Those are immutable training facts, but a learned model must not infer or
rewrite them from memory.

The exact read-only comparison is recorded in
`research/openwaldo-upstream-delta-2026-08-22.json`. No upstream commit was
merged into this fork.

## Decision

Add a fork-only Training Profile Contract Lens with schema:

`axm.waldo-witness.training-profile-contract/v0.1`

The lens consumes:

1. one exact documented `RUN-BOM.json`; and
2. its already validated Training Run Witness.

It emits a separate self-digested receipt rather than changing an existing
training-run witness receipt silently. The receipt binds both observed and
witnessed run-BOM, document, parameter, and witness-receipt digests.

### Profile identity

Current behavior names require `profile_schema: 1`. Historical names retain
both their declared identity and canonical behavior:

| Declared name | Historical schema | Canonical behavior |
| --- | ---: | --- |
| `causal-pretrain-v1` | 1 | `causal-pretrain-shuffled` |
| `causal-pretrain-v2` | 2 | `causal-pretrain-balanced` |
| `causal-pretrain-v3` | 3 | `causal-pretrain-weighted` |

The receipt never rewrites historical run bytes. A valid historical pair emits
`LEGACY_ALIAS_WITNESSED`; the declared and canonical fields both remain visible.

### Projected behavior

The receipt projects:

- data order, shuffle bounds, and packing contract;
- a sorted corpus-weight projection and its digest;
- evaluation selection and limits;
- seed, epochs, requested tokens, steps, and planned capacity; and
- explicit evidence references and claim ceilings.

Corpus-weight projection is bounded to 1,024 entries and 4,096 bytes per path.
Inputs outside that resource contract are rejected instead of truncated.

The lens recognizes these data contracts:

- shuffled: `bounded-shuffle-v1` with `lowest-sha256-v1` evaluation;
- balanced: `corpus-balanced-shuffle-v1` with stratified evaluation; and
- weighted: `corpus-weighted-shuffle-v1`, a positive weight map, and stratified
  evaluation.

Balanced and weighted observations may legitimately identify WALDO's
whole-index selection with the empty string. Consumption validation therefore
checks membership in the pinned corpus paths rather than treating an empty path
as missing evidence.

### Typed outcomes

The receipt states are:

- `PROFILE_CONTRACT_WITNESSED`;
- `LEGACY_ALIAS_WITNESSED`;
- `HOLD_UNKNOWN_PROFILE_CONTRACT`;
- `HOLD_PROFILE_SCHEMA_MISMATCH`; and
- `HOLD_PARAMETER_WITNESS_MISMATCH`.

Valid but unsupported or mismatched inputs produce an inspectable HOLD receipt.
Malformed inputs still fail without manufacturing evidence.

## Current upstream compatibility boundary

The local run-BOM wire projection retains the optional schema-1
`requested_tokens` and `conversation` fields so current inputs do not lose those
bytes during typed canonicalization. Conversation content remains bound through
the run BOM and witness; it is not reclassified as a training profile.

This decision does not yet claim full current-upstream compatibility. Newer
corpus `record_kind`, assessment, redaction, and source-language fields need a
separate Corpus Evidence Lens review. Current model interaction identity also
needs a separate Origin Anchor update. Both remain explicit HOLDs in the
upstream-delta receipt.

## Authority and truth boundary

Every profile receipt carries closed authority. It cannot execute tools, launch
training, promote a model, alter CANON, or act in the world.

A witnessed profile establishes only the declared data-selection contract in
the supplied immutable run evidence. It does not prove model quality, safety,
legal usability, semantic independence, or that a selected or weighted source
caused a particular output.
