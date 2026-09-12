# 0064: Bind Workshop state to verified local checkpoints

## Status

Accepted for the Workshop state format introduced by this change.

## Context

The Local Workshop owns identities, sessions, messages, explicit memories,
consent records, media references, and background-runtime state. It previously
wrote that canonical state as an unsealed `state.json`. A truncated or
otherwise damaged file stopped startup, while a valid but unintended edit was
indistinguishable from state the Workshop had committed. The temporary-file
rename reduced partial-write exposure but left no verified restart point.

Treating damaged state as a new installation would be worse: absence and
corruption are different facts, and silently seeding a fresh identity could
overwrite evidence that recovery needs.

## Decision

`state.json` is now a schema-1 `axm.workshop.state` envelope. It contains the
version-2 Workshop state and a SHA-256 digest of that exact normalized state.
The envelope is the canonical checkpoint; `/api/state` remains the existing
state projection so the browser contract does not change.

Every successful save:

1. enters ADR 0066's local single-writer and observed-checkpoint admission
   boundary;
2. validates identity, session, memory, consent, and media references;
3. preserves the previous verified checkpoint as `state.json.backup`;
4. writes and syncs a same-directory temporary file;
5. atomically renames it to `state.json`; and
6. syncs the containing directory where the operating system supports it.

On startup, the Workshop verifies the current envelope before using it. If the
current checkpoint is missing or rejected and the backup verifies, the backup
is restored. Rejected current bytes are first preserved exactly under
`recovery/`, named by their SHA-256 identity. If neither checkpoint verifies,
startup fails closed and leaves both inputs untouched.

Legacy direct version-0, version-1, and version-2 state documents remain a
one-way read compatibility surface. They are normalized, validated, retained
as the first backup, and replaced by the envelope on successful startup.

## Consequences

- Restart behavior distinguishes first launch, legacy migration, verified
  current state, and backup recovery.
- A single previous checkpoint is available without a service, account, or
  network dependency.
- Corrupt bytes are not silently discarded when recovery is possible.
- The digest detects accidental modification; it is not a signature and does
  not identify an author.
- Restoring the previous checkpoint can lose only changes newer than that
  checkpoint. The quarantined rejected bytes remain available for inspection.
