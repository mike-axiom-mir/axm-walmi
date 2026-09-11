# 0066: Reject stale Workshop checkpoint writers

## Status

Accepted for cooperating local Workshop processes.

## Context

ADR 0064 makes each Workshop checkpoint self-verifying and recoverable. The
mutation transaction then persists a detached candidate before exposing it as
live in-process state. Those guarantees do not coordinate two Workshop
processes sharing the same data directory.

Two processes can load checkpoint A, derive different successors B and C, and
both pass validation. Atomic rename prevents either write from being torn, but
without admission it allows the later writer to replace B with stale successor
C. The resulting file is internally valid while silently losing B's change.

## Decision

The Workshop retains a process-local token for the exact `state.json` bytes it
loaded or last committed. Every state mutation:

1. takes a non-blocking OS advisory lock for the Workshop data directory;
2. rereads the current checkpoint while holding that lock;
3. compares the exact current-byte SHA-256 (or observed absence) with the
   process-local token;
4. refuses a busy or mismatched writer without changing current or backup
   state; and
5. on a match, performs ADR 0064's verified backup and atomic checkpoint write,
   then advances the process-local token to the committed bytes.

The fixed `state.json.lock` path contains no canonical state. Lock ownership is
held by the operating system and releases when the file descriptor closes or
the process exits. A stale process is not automatically merged or refreshed;
the operator must retry after a busy result or restart after a token mismatch.

## Consequences

- Two cooperating processes cannot both admit different successors to the
  same observed checkpoint.
- First-launch seeding, legacy migration, and backup recovery use the same
  compare-before-commit boundary.
- Identical bytes remain identified by content rather than timestamps or
  process identity.
- The token is derived and reconstructable; canonical state remains the
  verified checkpoint envelope.
- This is one-machine, cooperating-process coordination. It is not a merge
  algorithm, authentication boundary, distributed lock, or guarantee for
  network/FAT filesystems with incompatible lock or rename semantics.
