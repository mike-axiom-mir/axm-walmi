# ADR 0021: Bounded project Workspace Hand

Status: experimental

## Context

WALMI could read and write its own model, checkpoint, experience, and export state, while the Creation Fabric deliberately refused arbitrary source-workspace access. The local neural runtime exposed generation, but no tool loop connected a selected project root to list, read, hash, bounded write, verification, and repair observations. Service Modes named file capabilities without providing them.

The v0.51 website experiment already contained a transactional candidate writer. Leaving that tested boundary only in history made the current PC package look more capable than its active runtime.

## Decision

Add an opt-in Workspace Hand beneath `waldo mirror workspace`.

- The human selects one existing, non-symlink root.
- Listing, reading, stat, and hashing remain inside that root and refuse `.git`, symlinks, traversal, absolute paths, drive-qualified paths, and NULs.
- Neural output is a strict one-object tool protocol, not hidden native tool authority.
- Writes are disabled unless the human supplies `--allow-write`.
- Writes reuse the v0.51 limits: 64 operations, 512 KiB per file, 2 MiB per transaction, expected hashes for update/delete, content hashes, atomic replacement, and rollback on failure.
- Verification commands are exact argument arrays in the human-authored request. They execute directly without a shell; the neural model selects only an allowed index.
- The loop is bounded to 32 turns and retains visible tool/error receipts.
- Optional experience output is review-required evidence. It does not train, mutate weights, promote, integrate, install, or change CANON.

## Consequences

WALMI can inspect and modify a selected real project without gaining general disk authority. A local model can now observe failures and attempt repairs within one bounded loop. Fun, quality, autonomy, and learning effectiveness are not implied. Model weights are separate runtime material and are never fabricated by the package.
