# 9012: Bridge the public AXM Code Fabric to the WALDO Mirror experiment by receipts

## Status

Accepted for `EXPERIMENTAL` v0.10 only.

## Context

The public AXM collaboration platform now contains an early Code Capability Fabric with deterministic routing, grounded-consent scope, a bounded slow-creation pilot, declarative blueprint composition, and blueprint-to-schema compilation.

The private Organ Factory is newer and is not present in the public donor snapshot used by this experiment. It must not be inferred, reconstructed, or credited to this baseline.

Copying the platform JavaScript directly into the WALDO fork would blur ownership, create duplicate source bodies, and make later provenance harder to inspect. A cross-repository Git blob graft was also rejected by GitHub because the source blob object does not exist in the WALDO repository object store.

## Decision

Keep the Code Fabric executable body in its native platform repository and bridge it into the WALDO/Mirror experiment through typed, digest-bound receipts.

The pinned public donor is platform commit:

`fd6ec98a6a98a6666a980c359730ccec57a8cbe9`

with the exact donor blob identities recorded in `research/platform-code-fabric-donor-v0.1.json`.

The detached platform experiment is PR 45 on branch:

`experiment/waldo-mirror-code-fabric-bridge-v0.1`

Its dedicated GitHub Actions run `32622264684` completed successfully. Before the bridge challenge, the runner passed the existing donor selftests for:

- deterministic JSON core;
- Code Capability Fabric v2;
- declarative blueprint composer v1; and
- blueprint schema compiler v1.

## Challenge

One sealed ancestor concept was sent through the existing deterministic Fabric into three different inert bodies:

1. `evidence-trail-mini-game`;
2. `continuity-diagnostic-tool`; and
3. `lineage-how-to-guide`.

All three retain one concept digest:

`sha256:9779b14236b2717ce1e8694adbacfd95c07d3afbacea10d3e4255cb52950856a`

but produce distinct blueprint identities.

The deterministic receipt is:

`sha256:1b54ba8a56945110becbcd7198d4c5988da5eea8724ff048e0c2f362844682c7`

A second run explicitly opted into an externally authored AI proposal for a fourth body, `tool-episode-quarantine-body`. The AI proposal did not replace the Fabric. It supplied a candidate body declaration which passed through the same deterministic composer and schema compiler.

The AI-opt-in receipt is:

`sha256:f88c2ca3da0774340cff484574d13980ab14d7004be9bfd386a00faab28eeb1c`

The fourth blueprint is:

`sha256:07a338cd9bc8f498c38d68b2fa28cf6dc90cc5ea874b020d33ba930208d5f4cd`

## Interpretation

The result supports a useful split:

```text
proposal source
  deterministic declaration OR explicit AI proposal
        |
        v
same deterministic Code Fabric
        |
        +--> inert blueprint
        +--> strict input schema
        +--> strict output schema
        +--> UNRUN acceptance matrix
        +--> receipt
        |
        v
WALDO / Mirror evidence and continuity layer
```

AI therefore remains optional at the proposal boundary. Deterministic composition, schema compilation, lineage, and truth ceilings do not depend on a live model call.

This also gives the later private Organ Factory a clean future comparison point. When an explicitly supplied donor snapshot becomes available, rerun the exact same challenge and compare what additional body-generation capability appears.

## Truth boundary

The successful runner proves that these exact public donor modules and experiment code executed successfully in GitHub Actions and produced the recorded receipts. It does **not** prove that the three deterministic bodies are good implementations, that the AI-proposed fourth body is correct, or that any blueprint is executable software.

The platform harness reported no live AI invocation, no requested network use by the harness, no generated executable code, no candidate execution, no installation, no promotion, no human quality approval, and no CANON change.

The private Organ Factory was explicitly absent:

`NOT_PUBLIC_IN_DONOR`.

No capability from it is included in or implied by this result.

## Consequences

- The platform remains the authoritative executable home of the public Code Fabric donor.
- WALDO/Mirror consumes or witnesses receipts rather than silently forking Fabric implementation bytes.
- Deterministic and AI-opt-in proposal modes share the same downstream composition gate.
- The early public Fabric now has a frozen measured baseline that can be compared with later Creation Fabric and Organ Factory generations.
- v0.9 remains unchanged as rollback; v0.10 is additive and experimental.
