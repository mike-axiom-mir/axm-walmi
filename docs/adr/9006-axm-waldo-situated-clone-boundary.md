# ADR 9006: Give the WALDO witness clone a situated evidence boundary

Status: experimental fork decision

## Context

The Wave 2 gated-clone foundation is provenance-aware but not situated. It can
receive bounded WALDO lineage, assess typed source claims, and bind a frozen
evaluation protocol. It does not yet know whether a current observation was
actually supplied, whether the skills needed to interpret that observation are
still present and backed up, or which unresolved seams and human meanings
surround the result.

The relevant Mirror knowledge is spread across two exact public snapshots:

- the AXM platform `main` Sensorium at commit `a4f99fbfc05268173458bf3fb8f3fe616919e376`;
- the separate `mirror` branch at commit
  `4a3727505576c4c95198a8287dd8ce1e82619f32`.

The source receipt is
`research/mirror-situated-knowledge-sources-2026-08-15.json`.

Those snapshots establish three important contracts.

1. The Sensorium contains thirteen TEST senses. Raw frames, stream buffers,
   recordings, and probes are ephemeral; a bounded typed receipt survives.
   Perception never transfers permission.
2. Mirror's discovery paths are deliberately different. The AI-native Seam
   Cell is primary and inspects evidence, contradiction, boundary, outcome,
   calibration, recovery, lineage, and access. Human Discovery/Stance is an
   explicit secondary advisory skill for usefulness, clarity, beginner
   friction, craft, accessibility, and product soul.
3. Skill continuity is not one automatic backup switch. The platform has a
   portable Sensorium skill pack, a passive 115-organ archive, manual backup
   guidance, and a separate permissioned Recovery Center. Archive or backup
   membership grants no install, restore, compatibility, permission, or
   runtime authority.

The WALDO specialist should learn from these contracts without copying the
Workshop runtime, importing Mirror JavaScript, or turning itself into a general
agent.

## Decision

Add four additive, ordinary-Go domains under `internal/axmmirror` and expose
them only through the separate `waldo-axm-mirror` CLI. Existing Wave 1 and Wave
2 schemas remain unchanged.

### Sensory Evidence Intake

Schemas:

- `axm.waldo-witness.sensory-evidence-draft/v0.1`;
- `axm.waldo-witness.sensory-evidence-receipt/v0.1`.

The intake recognizes the exact thirteen Sensorium sense/capability pairs. It
does not implement their executors. A caller supplies a sealed, attributed
typed observation with:

- exact sense and capability identity;
- subject, claim, seat, and optional backend identity;
- observation, seal, and assessment timestamps;
- TTL and derived age/freshness;
- observation state and independent PASS/FAIL/UNKNOWN verdict;
- typed-observation and sense-specific receipt digests;
- authority-inheritance declaration;
- post-seal raw-retention counters and cleanup state.

`PASS` and `FAIL` can both be `SENSORY_EVIDENCE_READY`: READY describes intake
integrity and freshness, not success of the observed claim. Stale, untimed,
future, incomplete, unavailable, simulated, retained, or authority-inheriting
inputs become distinct HOLD/REFUSED receipts.

The draft and receipt contain no frame, image, transcript, recording, stream,
log, directory tree, probe output, or raw sensory buffer. WALDO never claims it
performed capture merely because it can validate a supplied receipt.

### Skill and Backup Continuity

Schemas:

- `axm.waldo-witness.skill-continuity-request/v0.1`;
- `axm.waldo-witness.skill-continuity-receipt/v0.1`.

The request separates three use classes:

- `KNOWLEDGE`: inert material such as the 115-organ archive;
- `INSTRUCTION`: portable skill guidance such as the Sensorium skill pack;
- `ADAPTER`: a host-mediated or executable capability with an exact declared
  compatibility state.

It compares a complete current manifest with a complete backup manifest for an
explicit set of requirements. Each item binds version, source digest, optional
instruction digest, TEST/WORKING status, execution status, proof status,
compatibility status, and optional aggregate member count.

Findings remain separate: `MATCHED`, `MISSING_CURRENT`, `MISSING_BACKUP`,
`DRIFTED`, or `INCOMPATIBLE`. Missing or drifted items may produce a
`REVIEW_RESTORE_CANDIDATE`, but that object is only a digest-bound review plan.
It has no path, command, installer, permission, or auto-apply authority.

This intentionally does not reproduce the Workshop Recovery Center. The
current Workshop backup note says whole-workspace backup is manual in v0.1,
while the separate Recovery Center requires preview, permission, exact
confirmation, safety copy, and lineage. WALDO therefore inventories and routes
holds only; it never installs, restores, enables, or executes a skill.

### Dual-native Discovery Stance

Schemas:

- `axm.waldo-witness.discovery-stance-request/v0.1`;
- `axm.waldo-witness.discovery-stance-packet/v0.1`.

The AI-native surface is primary. Each seam preserves:

- one of the eight native stances;
- severity and OPEN/BLOCKED/CLOSED state;
- statement and exact evidence references;
- blocked interpretations and optional repair hint;
- the cheapest disconfirming check;
- append-only closure history.

A CLOSED seam requires the latest closure attempt to be `PASS` and carry
evidence references. Fluent wording cannot close it.

The human-native surface is secondary and explicit-only. It preserves the
Workshop stages `knownSpace`, `rejectedDirections`, `blindSpots`, `seams`,
`patterns`, `realityChecks`, `soulChecks`, and `minimalChecks`. Every entry is
labelled `AI_ADVISORY_REQUIRES_HUMAN_REVIEW`. The surface cannot close native
seams, promote learning, grant permissions, or implement its own suggestions.

The two surfaces have independent digests. They are not averaged into a score,
ranked against each other, or forced to imitate one another. Machine-native
structure and human meaning remain separately reviewable.

### Situated Context Envelope

Schemas:

- `axm.waldo-witness.situated-context-request/v0.1`;
- `axm.waldo-witness.situated-context-envelope/v0.1`.

An envelope binds one exact answering identity and one exact subject to:

- a READY provenance context packet;
- an explicit required-sense list and exactly one receipt per required sense;
- a skill/backup continuity receipt for the same answering identity;
- a dual discovery packet for the same subject.

All thirteen sensory contracts are supported, but no task silently invokes all
thirteen. The task must name its required senses. A missing or held required
sense, skill-continuity hold, discovery hold, or identity/subject mismatch
remains a distinct envelope HOLD. The envelope is bounded to 512 KiB and still
contains typed receipts only.

`SealSituatedGated` adds four verification bindings to the existing gated
behavior seal:

- sensory evidence set;
- skill and backup continuity;
- dual discovery stance;
- complete situated envelope.

A held situated envelope may be sealed as durable `hold` or `fail` evidence,
but never as a passing behavior outcome.

## CLI

The additive situated flow is:

```text
existing provenance context
  + intake-sensory (one per explicitly required sense)
  + assess-skills
  + discover
  -> situated-context
  -> seal-situated
  -> verify
```

The earlier `seal-gated` path remains available and unchanged for experiments
that do not claim a situated boundary.

## Durability and compatibility

Every new schema is experimental v0.1. Strict JSON decoding rejects duplicate
keys, unknown fields, and trailing values. Inputs are canonicalized before
digesting; output receipts carry self-digests for mutation detection. The
checked synthetic fixtures and digest compatibility tests pin the initial
format.

A self-digest is not a signature, author identity, fresh byte re-verification,
or truth proof. Source metadata and earlier runtime tests retain their original
claim ceilings.

## Authority boundary

All outputs carry closed authority. These domains do not:

- open a camera, microphone, browser, terminal, stream, or file probe;
- retain raw sensory buffers;
- install, restore, enable, connect, or execute skills or archived organs;
- infer that archive membership proves behavior or compatibility;
- run a disconfirming check or implement a repair hint;
- let human advice close AI-native seams;
- grant permission, train, promote, alter CANON, or act in the world.

Mike remains the merge and CANON gate.

## Consequences

The future WALDO witness clone can now be surrounded by a machine-readable
account of what it was allowed to observe, whether required knowledge and
capability manifests remained continuous, what machine-native seams stayed
open, and what human-facing meaning required review.

This makes the specialist more situated without making it a copy of Mirror or
giving learned weights hidden body, backup, repair, or promotion authority. No
learned clone is trained by this decision, no archived source is imported or
executed, and upstream WALDO and the fork's `main` remain unchanged.
