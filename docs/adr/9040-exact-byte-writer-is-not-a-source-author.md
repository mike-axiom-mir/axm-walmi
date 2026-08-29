# ADR 9040: The exact-byte writer is not a source author

## Status

Experimental.

## Context

The Creation Fabric can plan a website but deliberately has no workspace-mutation authority. WALDO neural output can supply source text, while applying those bytes safely is a separate machine concern.

## Decision

Add a bounded candidate-writer command that accepts hash-bound create, update, and delete operations. The writer preflights the full transaction, refuses unsafe paths and stale targets, applies only the supplied bytes, and emits a non-authoring mutation receipt.

Absence of a WALDO source candidate is a hold. The writer must not fill that gap with templates, generated code, or another model.

## Consequence

WALDO can later use the writer as a hand without giving the hand authorship. The v0.51 website probe remains honest when the neural source runtime or artifact is unavailable.

## Boundary

`EXACT_BYTE_WRITER_IS_NOT_SOURCE_AUTHOR`

No execution beyond the bounded file transaction, network, install, deployment, merge, promotion, CANON, training, or model-weight authority is created.
