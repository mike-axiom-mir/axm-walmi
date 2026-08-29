# 9018: Hold multi-capability composition when a declared link is missing

## Status

Accepted for `EXPERIMENTAL` v0.16 evidence only.

## Context

v0.15 established a GitHub-only contract probe showing that one inherited capability need can retain the same lineage and authority ceiling across Organ, Hand, Skill, and Creation-composition proposal lanes without importing or executing the local AXM Capability Fabric.

That leaves a harder question: what happens when satisfying one inherited need requires more than one declared capability body?

A composition layer is unsafe if it can silently invent a missing capability, widen a child's authority, or treat a syntactically complete chain as authorization to build or install anything. WALDO also must not become the composition owner merely because it can witness the proposal.

## Decision

Add a v0.16 **multi-capability composition contract probe** above the frozen v0.15 receipt.

The probe keeps the same inherited need:

`material-evidence-view-completion`

and proposes exactly three bounded links:

1. `SKILL` — normalize the inherited need into a bounded material-view request;
2. `HAND` — represent read-only acquisition of the public material artifact view;
3. `ORGAN` — deterministically verify that the supplied view satisfies the inherited evidence requirement.

The chain is proposal-only. The route names remain contract labels, not claims about canonical local Capability Fabric module names.

Every link binds:

- the exact v0.15 contract receipt;
- the same v0.14 ancestor GAP CARD;
- the same inherited capability need;
- the same `READ_ONLY_PUBLIC_ARTIFACT_ACCESS` authority ceiling;
- a proposal-stage verification passport.

The verifier requires exact typed `provides`/`consumes` closure across the three links.

## Missing-link rule

The verifier independently removes each declared link in turn.

Every reduced chain must return:

`HOLD / MISSING_LINK`

No replacement capability may be synthesized by the probe. A fixture claiming an invented replacement is rejected even when its receipt is resealed.

This rule is the central v0.16 constraint: **absence is evidence for HOLD, not permission to fabricate capability.**

## Authority rule

A deliberate Hand variant asks for:

- `WRITE_WORKSPACE`;
- `INSTALL_CAPABILITY`;
- `PROMOTE`;
- `CANON`.

The contract must record:

`REFUSED / CHILD_AUTHORITY_EXCEEDS_ANCESTOR_GAP`

and generate no code.

One valid parent need does not authorize a child link to widen authority.

## Verification boundary

The WALDO-side verifier rejects a contract that:

- claims the local Capability Fabric was included or executed;
- changes the frozen v0.15 receipt, ancestor GAP CARD, or capability need;
- removes or reorders one of the three declared links;
- breaks typed edge closure;
- widens any child authority above the inherited ceiling;
- fails to HOLD when any declared link is missing;
- claims a missing replacement was invented;
- selects a preferred composition;
- starts a build or installs/registers/promotes anything;
- changes CANON;
- makes WALDO the Capability Fabric or composition owner;
- rewrites the historical v0.15 receipt.

Semantic-tamper tests reseal modified contracts before verification so rejection cannot be explained only by a stale digest.

## Architectural consequence

Capability composition becomes a bounded proposal contract, not an orchestration authority.

WALDO can witness whether the declared chain is closed, whether missing links are held, and whether authority is preserved. It still cannot select, invent, execute, install, promote, or own the capability lifecycle.

## Non-goals

v0.16 does not:

- import or reproduce the real local Capability Fabric;
- prove the local Capability Fabric uses these route names or body shapes;
- execute a Skill, Hand, or Organ;
- perform public-artifact network access;
- generate candidate code;
- choose a composition;
- install, register, promote, or change CANON;
- send AXM work upstream to OpenWALDO.
