# AXM/WALDO v0.15 — Capability Fabric contract probe

v0.15 does **not** import or execute the local AXM Capability Fabric. That implementation is not present in this repository.

This checkpoint tests only the boundary needed for a future encounter:

> Can one evidence-backed capability gap be represented as several bounded embodiment proposals while preserving one ancestor need, one authority ceiling, provenance, verification-passport semantics, and a refusal path for authority growth?

## Frozen source

The source is the v0.14 WALDO return packet:

`sha256:b82cf3a8d5d02e8a979c044b4bcd98c1516390950647949172e34651616b90d2`

Ancestor GAP CARD:

`v014-material-evidence-view-incomplete`

The prior v0.14 packet routed that gap to `Creation Fabric evidence intake`. After the later architecture clarification, v0.15 records a **new routing interpretation** to `Capability Fabric intake` without rewriting the old packet.

## One capability / many bodies

Capability:

`material-evidence-view-completion`

Authority ceiling:

`READ_ONLY_PUBLIC_ARTIFACT_ACCESS`

Four proposal-only embodiments share that exact ancestor gap and ceiling:

1. **ORGAN** → contract label `ORGAN_FABRIC`
   - deterministic evidence-view requirement checker;
   - emits a missing-artifact report;
   - uses no authority in the probe.

2. **HAND** → contract label `MODULAR_HAND_BUILDER`
   - bounded read of one specifically referenced public artifact;
   - may request only `READ_ONLY_PUBLIC_ARTIFACT_ACCESS`;
   - must verify returned bytes against the expected digest before evidence admission.

3. **SKILL** → contract label `MODULAR_SKILL_BUILDER`
   - bounded procedure for resolving an incomplete material-evidence view;
   - preserves the earlier HOLD;
   - stops if required read authority is unavailable.

4. **CREATION_COMPOSITION** → contract label `CREATION_FABRIC`
   - proposal-only multi-piece chain when one reusable body is insufficient;
   - no composition is executed in v0.15.

The route labels above are **contract labels only**. They are not claims about canonical local module names.

## Deliberate authority-escalation refusal

A fifth proposal asks for a Hand with:

- `WRITE_WORKSPACE`
- `INSTALL_CAPABILITY`
- `PROMOTE`
- `CANON`

The probe records:

`REFUSED / AUTHORITY_EXCEEDS_ANCESTOR_GAP`

No code is generated for the refused proposal.

## Verification passports

Every accepted proposal carries a lightweight proposal-stage passport containing:

- source lineage binding;
- declared tests;
- authority used;
- side effects observed;
- deterministic proposal replay state;
- gate state.

Because this is a contract probe, `testsExecutedByProbe`, `authorityUsed`, and `sideEffectsObserved` remain empty. The gate state is:

`HOLD_FOR_REAL_CAPABILITY_FABRIC`

## WALDO-side witness

`internal/axmmirror/capability_fabric_contract_probe.go` independently verifies:

- exact v0.14 return-packet and ledger identities;
- the routing adjustment without rewriting v0.14 history;
- one common capability ID and authority ceiling;
- exactly one Organ, Hand, Skill, and Creation-Fabric composition proposal;
- proposal-only state and shared lineage;
- verification-passport boundaries;
- refusal of the authority-escalating Hand;
- zero selection/build/install/register/promote/CANON state;
- the canonical external receipt digest.

The test suite also reseals semantic tampering so failures cannot be explained only by a stale hash. It rejects attempts to:

- claim the real local Capability Fabric was executed;
- add write authority to the Hand;
- select an embodiment;
- make WALDO own the Capability Fabric;
- rewrite the previous routing history.

## Observed identities

Contract-probe receipt:

`sha256:75e6466701520c7aeab56e76c719ab275519e8ed2fffa5fceaf3310cce4ec0bf`

Expected compact witness digest from the deterministic witness representation:

`c2ec52373ef24773ea59631ccd3a3ce7be1a7b1d958f98e05fdf09dce0e4d6f6`

The witness digest is only meaningful if the Go verifier passes against the exact fixture. Full repository CI is not claimed until GitHub reports it.

## EXISTING / EXTEND / ADAPT / NEW / HOLD

**EXISTING**
- v0.14 append-only dissent ledger;
- v0.14 GAP CARD / return packet;
- v0.13 peer-audit evidence plane;
- existing Organ/Creation Fabric concepts and WALDO witness boundary.

**EXTEND**
- return routing so the higher-level Capability Fabric can become the future intake point.

**ADAPT**
- the existing GAP CARD becomes a capability requirement;
- verification-passport semantics are applied to multiple embodiment proposals.

**NEW**
- only the v0.15 GitHub contract-probe schema and WALDO verifier/witness.

**HOLD**
- importing the real local Capability Fabric;
- building any Organ/Hand/Skill/composition from this probe;
- automatic embodiment selection;
- automatic installation or promotion;
- CANON change.

## Truth boundary

WALDO is not installed locally by this experiment and does not become the Capability Fabric. The local Capability Fabric implementation was not included or executed. No candidate code was generated or run. The probe requested no network access and called no live AI provider. Nothing was selected, built, installed, registered, promoted, merged, or made CANON.
