# 9017: Put a Capability Fabric contract above WALDO gap evidence without importing the local implementation

## Status

Accepted for `EXPERIMENTAL` v0.15 evidence only.

## Context

v0.14 established an append-only witness-dissent ledger and a WALDO return packet containing a GAP CARD. At that point the return packet named `Creation Fabric evidence intake` as its candidate destination.

Later architecture context clarified a higher layer: a Capability Fabric sits above Organ Fabric and Creation Fabric and can also produce modular Hands and Skills. WALDO is not installed locally and the actual local Capability Fabric implementation is not present in this repository.

Rewriting the v0.14 return packet would destroy truthful history. Importing or pretending to execute an unavailable local implementation would overclaim the experiment.

## Decision

Add a **contract probe**, not a Capability Fabric implementation.

The v0.15 probe:

- binds the exact v0.14 return-packet and dissent-ledger digests;
- preserves `v014-material-evidence-view-incomplete` as the ancestor GAP CARD;
- records a new routing interpretation from the historical Creation-Fabric destination to `Capability Fabric intake` without mutating v0.14;
- represents one capability need as four proposal-only embodiment kinds:
  - `ORGAN` routed by contract label to `ORGAN_FABRIC`;
  - `HAND` routed by contract label to `MODULAR_HAND_BUILDER`;
  - `SKILL` routed by contract label to `MODULAR_SKILL_BUILDER`;
  - `CREATION_COMPOSITION` routed by contract label to `CREATION_FABRIC`;
- preserves one authority ceiling: `READ_ONLY_PUBLIC_ARTIFACT_ACCESS`;
- requires common lineage and proposal-stage verification passports;
- performs no selection or build;
- explicitly refuses a Hand proposal requesting write/install/promote/CANON authority.

The route labels are interface labels for this experiment. They are not assertions about canonical names inside the unavailable local Capability Fabric.

## Verification boundary

The WALDO-side verifier must reject a probe that:

- claims the local Capability Fabric was included or executed;
- changes the frozen source return-packet, ledger, or GAP CARD identity;
- rewrites the historical v0.14 routing packet;
- gives any accepted embodiment authority above the GAP CARD ceiling;
- loses common lineage or verification-passport boundaries;
- fails to refuse the deliberate authority-escalation proposal;
- selects, builds, installs, registers, promotes, or changes CANON;
- says WALDO owns the Capability Fabric.

The verifier independently recomputes the canonical external receipt digest.

## Architectural consequence

WALDO remains a scout/witness and evidence producer. A later Capability Fabric may consume its GAP CARD, but WALDO does not decide which embodiment is canonical and does not gain creation or promotion authority.

Capability Fabric becomes the **future higher-level routing seam**, while Organ Fabric and Creation Fabric remain distinct downstream embodiment lanes rather than being replaced.

The experiment also preserves an important distinction between:

- **capability need** — one evidence-backed ancestor requirement;
- **embodiment proposal** — one possible Organ, Hand, Skill, or Creation-Fabric composition;
- **authority** — bounded separately from capability;
- **promotion** — still outside this probe.

## Non-goals

v0.15 does not:

- reproduce the real local Capability Fabric;
- build a Hand or Skill framework inside WALDO;
- execute any proposed embodiment;
- choose a preferred embodiment;
- install or register a capability;
- merge anything upstream to OpenWALDO;
- change AXM CANON.

A future experiment may replace this contract probe with an actual donor receipt only after the real Capability Fabric is explicitly supplied to the GitHub research lane.
