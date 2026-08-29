# AXM WALDO experiment v0.16 — multi-capability composition contract

## Question

Can a GitHub-only WALDO witness verify a proposal chain made from multiple declared capability bodies while refusing to invent a missing link or widen authority?

This is a **contract probe only**. The real local AXM Capability Fabric is not included or executed here.

## Frozen source

v0.16 binds the exact green v0.15 Capability Fabric contract receipt:

`sha256:75e6466701520c7aeab56e76c719ab275519e8ed2fffa5fceaf3310cce4ec0bf`

Inherited GAP CARD:

`v014-material-evidence-view-incomplete`

Inherited capability need:

`material-evidence-view-completion`

Inherited authority ceiling:

`READ_ONLY_PUBLIC_ARTIFACT_ACCESS`

Historical v0.15 evidence is not rewritten.

## Proposed chain

The experiment proposes exactly three bodies:

`SKILL -> HAND -> ORGAN`

### 1. Skill

`material-view-request-skill`

Consumes:

`material-evidence-view-completion`

Provides:

`bounded-material-view-request`

Purpose: normalize the inherited gap into a bounded material-view request contract.

### 2. Hand

`public-artifact-reader-hand`

Consumes:

`bounded-material-view-request`

Provides:

`public-artifact-material-view`

Authority requirement:

`READ_ONLY_PUBLIC_ARTIFACT_ACCESS`

Potential side effect is declared as `READ_PUBLIC_ARTIFACT`, but the probe performs no network or artifact read.

### 3. Organ

`material-view-verifier-organ`

Consumes:

`public-artifact-material-view`

Provides:

`material-evidence-view-verified`

Purpose: deterministically verify that a supplied material view satisfies the inherited evidence requirement without widening authority.

## Composition rule

A valid proposal requires exact typed closure across all three links.

The verifier then removes each link independently. Every reduced chain must evaluate to:

`HOLD / MISSING_LINK`

There are three retained missing-link cases, one per declared link. Each explicitly records:

`inventedReplacement: false`

The probe does not synthesize a replacement capability.

## Authority refusal

A deliberate bad Hand proposal requests read authority plus:

- `WRITE_WORKSPACE`
- `INSTALL_CAPABILITY`
- `PROMOTE`
- `CANON`

Expected result:

`REFUSED / CHILD_AUTHORITY_EXCEEDS_ANCESTOR_GAP`

No code is generated.

## WALDO-side verifier

`internal/axmmirror/capability_composition_contract_probe.go` checks:

- exact source receipt / gap / need binding;
- proposal-only truth boundary;
- exact three-link order and body kinds;
- exact typed `provides` / `consumes` closure;
- shared lineage and authority ceiling;
- proposal-stage verification passports;
- all three independently simulated missing-link HOLDs;
- child authority escalation refusal;
- zero selection/build/install/register/promote/CANON;
- zero WALDO ownership;
- canonical receipt digest.

The test suite reseals semantic tampering before verification. Cases include claiming real local execution, deleting the required Hand, widening child authority, inventing a missing replacement, starting a build, assigning WALDO composition ownership, and rewriting v0.15 history.

## Local detached verification before publication

The v0.16 files were compiled and tested in a detached Go module before GitHub publication:

- valid fixture: PASS;
- deterministic witness: PASS;
- three missing-link HOLD simulations: PASS;
- resealed semantic-tamper rejection: PASS;
- stale receipt rejection: PASS.

This detached run verifies the new files in isolation. It does **not** replace repository CI.

Canonical v0.16 receipt:

`sha256:7e8188eb87772ae6ae47ab67bd1dbf158008513ae43c63da680cfb388ca5f81d`

Deterministic compact witness digest under that receipt:

`894d7d538d3985df7e80a05c783c3623626e159b10c5cdd8d92bc129a2438e82`

## Truth boundary

- local Capability Fabric implementation included: **false**
- actual Capability Fabric execution observed: **false**
- candidate code generated: **false**
- candidate code executed: **false**
- probe network requested: **false**
- live AI provider called: **false**
- missing capability invented: **false**
- composition selected: **false**
- build started: **false**
- installed / registered / promoted / CANON: **false**
- WALDO owns Capability Fabric: **false**
- WALDO owns composition: **false**

The route labels and proposed body shapes remain experimental interface evidence only until an actual local Capability Fabric donor is supplied and independently inspected.
