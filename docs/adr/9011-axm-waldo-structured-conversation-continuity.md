# 9011: Bind structured conversation lineage to a digest-only continuity spine

## Status

Experimental downstream AXM/Mirror decision. This is not an OpenWALDO decision
and does not change OpenWALDO upstream behavior.

## Context

OpenWALDO now has two useful properties that did not exist at the beginning of
this fork experiment:

1. a model interaction template is an explicit, versioned part of model
   identity; and
2. dialogue ingestion preserves ordered structured messages and optional tool
   definitions until the model-side training transformation is selected.

Those changes matter for Mirror because earlier continuity experiments had to
choose between two bad representations:

- retain flattened conversational text, which leaks more private payload than a
  continuity receipt needs and loses role/tool structure; or
- retain only one whole-blob digest, which protects content but cannot tell us
  whether turn order, role boundaries, tool context, or the active interaction
  contract changed.

The new WALDO representation gives the downstream experiment a third option:
keep structure as evidence while replacing payloads with digests.

This also gives us a concrete route to the portable continuity capsule already
listed in the AXM experiment roadmap. The capsule can bind answering identity,
conversation lineage, verifier generation, task state, memory-shard catalog,
evidence set, and open dissent without copying the underlying private objects.

## Decision

Add two evidence-only AXM organs in `internal/axmmirror`.

### 1. Structured conversation witness

`WitnessConversation` accepts an already-normalized `record.Conversation` and a
small AXM request that names:

- the answering-identity digest;
- the canonical WALDO record digest;
- model interaction template;
- conversation training template;
- training objective;
- supervised roles; and
- requested authority.

The witness validates the WALDO conversation through the upstream record
contract and verifies that `source_record_sha256` equals the canonical encoded
conversation identity.

The retained receipt contains:

- whole canonical conversation digest;
- ordered turn number and role;
- per-turn content digest;
- optional per-turn context digest;
- a turn digest binding position, role, and those payload digests;
- role counts;
- optional canonical tool-definition digest;
- model/training interaction declarations;
- request and receipt self-digests; and
- closed authority.

It intentionally does **not** retain raw prompt text, raw assistant text, raw
tool results, tool definitions, hidden reasoning, private memory, or model
weights.

A mismatch between the declared model interaction template and training
template produces a typed HOLD rather than silently normalizing one side to the
other. For `assistant-response-modeling`, AXM also refuses to call the receipt a
Mirror answering-identity witness when `assistant` is absent from the declared
supervised roles. That is a downstream AXM boundary, not an additional claim
about what OpenWALDO itself permits.

Any request for execution, training, promotion, CANON, or world-action
authority produces `REFUSED_AUTHORITY_GROWTH`; the emitted receipt still carries
closed authority.

### 2. Portable continuity capsule

`SealContinuityCapsule` binds one verified conversation witness to a set of
already-sealed state references:

- answering identity;
- task-state digest;
- active verifier-registry digest;
- optional skill-continuity receipt;
- zero or more memory-shard digests;
- zero or more open-dissent digests; and
- zero or more evidence digests.

The capsule copies only the conversation lineage needed for resumption checks:
interaction template, whole conversation digest, last-turn digest, turn count,
and the witness digest/state.

Memory, dissent, evidence, task state, skills, verifier definitions, and raw
conversation remain separate governed artifacts. The capsule cannot fetch,
restore, install, execute, train on, promote, or make those objects CANON.

Digest sets are sorted and duplicate-free before sealing so the same logical
catalog receives the same identity independent of input order.

`captured_at` and `expires_at` are explicit. `FreshAt` answers only whether a
supplied time lies inside that declared interval. It does not claim that any
referenced object is still available, compatible, safe, or semantically fresh.

## Relationship to current OpenWALDO

This slice depends on the current upstream body rather than copying or replacing
it:

- ADR 0056 supplies the versioned model interaction contract;
- ADR 0060 supplies structured conversation identity and model-side rendering;
- `internal/record.Conversation` supplies the canonical ordered message/tool
  representation; and
- the tool-use compose demonstrates a current upstream pattern in which tool
  definitions and tool results remain context while assistant outputs are the
  supervised targets.

AXM does not fork those semantics inside `internal/axmmirror`. If OpenWALDO adds
a new template identifier, role, conversation schema, or training objective,
this witness should HOLD/reject until the downstream contract is deliberately
reviewed and extended.

## Why this is useful

A continuity consumer can now distinguish several failure classes without
opening the private payloads:

- same conversation bytes but wrong answering identity;
- same identity but different interaction template;
- reordered or replaced turns;
- changed tool definitions;
- changed active verifier generation;
- changed task-state snapshot;
- changed memory catalog;
- changed open-dissent set; or
- an expired continuity claim.

That is materially stronger than a single transcript hash and materially safer
than copying the transcript into every checkpoint.

It also creates a future bridge to headless or cross-session AXM use: a host can
first compare the digest-only capsule, then retrieve only the separately
permitted artifacts that are actually needed. The capsule itself is not a
recovery mechanism.

## Non-goals and truth boundary

This decision does not:

- train a Mirror model;
- decide what data should be training data;
- turn tool-memory evidence into SFT examples;
- infer permissions from tool definitions;
- validate tool schemas semantically;
- prove a model followed the declared interaction template;
- prove a training run used the declared transformation merely because a
  witness says so;
- restore a conversation from hashes;
- synchronize private memory;
- select an AI provider;
- execute tools or shell commands;
- approve verifier changes;
- close dissent;
- grant runtime authority; or
- change OpenWALDO upstream.

A real model/run/release chain still needs its own WALDO provenance and AXM
behavior evidence. This continuity spine only makes the relationship between
structured dialogue state and portable resumption state inspectable.

## Tests

The v0.9 tests cover:

- a tool-bearing structured conversation with assistant-only supervision;
- removal of raw dialogue/tool payloads from the retained receipt;
- interaction-template mismatch HOLD;
- missing assistant target HOLD;
- authority-growth refusal;
- digest-only continuity sealing;
- canonical memory/evidence catalog ordering;
- freshness interval checks;
- continuity/witness binding mismatch HOLD;
- propagation of a held conversation witness; and
- duplicate digest rejection.

## Follow-up experiments

1. Bind the capsule to an actual WALDO model/run/release chain rather than only
   synthetic fixtures.
2. Add an append-only dissent ledger receipt so the capsule can reference a
   specific open-dissent frontier instead of a digest set alone.
3. Add a Reproducibility Twin that attempts to resolve every referenced digest
   in an isolated read-only store and reports PRESENT / MISSING / MISMATCH
   without restoring anything.
4. Add reviewed compaction manifests for memory shards so a continuity capsule
   can reference both the full evidence lineage and a smaller model-facing view.
5. Only after those boundaries hold, experiment with a reviewed transformation
   from selected public-safe evidence into candidate structured training
   conversations. Evidence must never become training material merely because
   it exists in a continuity capsule.
