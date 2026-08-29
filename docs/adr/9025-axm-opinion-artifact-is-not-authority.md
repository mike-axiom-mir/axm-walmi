# ADR 9025: Treat a machine opinion as traceable reported assessment, not authority

## Status

Accepted for the downstream AXM/WALDO research experiment only.

## Context

A visual AXM campaign seed proposed a future `OPINION ARTIFACT`: a machine identity may state an opinion, but should expose how it reached that opinion through source packets, competing evidence, uncertainty markers, and dissent. The same seed explicitly refused an unsupported consciousness claim and described the artifact as traceable, reversible, and open to review.

The v0.22 experiment already separates run-capsule evidence from promotion. The next boundary is whether an opinion built from evidence can accidentally become authority, execution permission, CANON, or a claim that an internal conscious state has been proven.

## Decision

v0.23 introduces a deterministic opinion-artifact contract probe.

An artifact may carry a subject, a stated position, traceable evidence relations, known competing evidence, dissent references, uncertainty markers, and revision lineage. A valid artifact must remain `OPEN_TO_REVIEW` and must preserve a prior opinion when it is revised.

The artifact itself has `NONE` authority. It cannot execute, grant authority, promote itself, or become CANON. Its consciousness field remains `UNVERIFIED`; the contract treats an opinion as a reported assessment, not proof of consciousness or privileged access to model internals.

Known contrary evidence, known dissent, and known uncertainty cannot be silently omitted. A revision that replaces rather than preserves the prior artifact is refused.

## Consequences

- A machine can eventually express a durable, inspectable perspective without that expression becoming action authority.
- Opinion changes can be studied longitudinally without rewriting the earlier state.
- Evidence, dissent, and uncertainty remain first-class parts of the artifact rather than explanatory text added later.
- A complete opinion artifact is still only evidence for review; it does not imply correctness, consciousness, promotion, or execution permission.
- WALDO remains a verifier/witness in this experiment. No live machine opinion, live AI provider call, Hermes execution, or model-internal-state access is claimed.

## Revisit when

A real AXM identity/opinion producer exists locally with a separately reviewed authority model and can emit signed artifacts under the same or a stricter evidence boundary.
