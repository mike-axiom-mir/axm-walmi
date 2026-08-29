# 9020: Keep selection authority separate from execution authority

## Status

Accepted for `EXPERIMENTAL` v0.18 evidence only.

## Context

v0.17 established that multiple feasible capability compositions do not authorize WALDO to silently choose a winner. Feasibility and selection authority are separate facts.

The next boundary appears after a selection is supplied from outside WALDO: a system could incorrectly treat "this proposal was selected" as permission to build, install, promote, or change CANON.

The real local Capability Fabric and real AXM governance implementation are not present in this repository, and no real governance decision is supplied to this experiment.

## Decision

Add a contract-only selection-receipt probe. It binds the verified v0.17 receipt, head, candidate-set digest, and candidate digests.

A fixture representing external selection may be structurally accepted only when:

- it references the frozen candidate set;
- the selected candidate ID and digest match that set;
- its scope is exactly `SELECT_PROPOSAL`;
- it does not grant execution, installation, promotion, or CANON authority;
- it does not identify WALDO as the issuer;
- it remains explicitly labelled `FIXTURE_ONLY_NOT_REAL_APPROVAL`.

A structurally accepted selection moves only to `SELECTED_PROPOSAL_ONLY`. With no execution authority, the overall state remains `HOLD / EXECUTION_AUTHORITY_ABSENT`.

Conflicting individually valid selection receipts remain first-class evidence and yield `HOLD / CONFLICTING_SELECTION_RECEIPTS`; later ordering does not overwrite earlier evidence.

## Consequence

The experiment separates:

1. feasibility — can this proposal satisfy the declared contract?
2. selection — which frozen proposal does an external selection receipt point to?
3. execution authority — may anything actually act on that selection?
4. promotion/CANON — still a separate governance layer.

WALDO may verify these boundaries but does not gain any of the missing authority merely by observing them.

## Non-goals

v0.18 does not authenticate a real governance signature, claim that a human approved a candidate, execute the Capability Fabric, build a selected proposal, install or promote anything, change CANON, or send AXM work upstream to OpenWALDO.
