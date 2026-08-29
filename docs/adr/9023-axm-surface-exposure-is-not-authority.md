# 9023: Keep capability exposure separate from authority and context budget

## Status

Accepted for `EXPERIMENTAL` v0.21 evidence only.

## Context

v0.20 closed the execution-permit/revocation boundary while preserving WALDO as a verifier rather than an executor. A separate Hermes donor review exposed another useful distinction: a capability can exist in a runtime without being appropriate to expose on every interaction surface.

The reviewed `NousResearch/hermes-agent` donor uses composable toolsets, keeps desktop-only affordances out of the generic core path, constrains default webhook exposure because webhook content can be untrusted, and attributes prompt/schema cost to toolsets and other context categories. The exact three source blobs used for this observation are unchanged between the AXM local Hermes donor pin `739bc555...` and reviewed Hermes head `2ebb1cb4...`.

AXM already has a separate local Hermes runtime-adapter lane. This WALDO experiment must not rebuild or claim that adapter.

## Decision

Add a GitHub-only **surface capability exposure contract**.

The contract separates:

1. capability declared;
2. capability installed/present;
3. capability exposed on a particular surface;
4. authority to act through that capability;
5. actual execution.

None of those states implies the next.

The probe also introduces an explicit context-budget boundary. Tool-schema costs in v0.21 are fixture estimates used to test the contract shape; they are not measurements of Hermes runtime token usage. If declared exposure exceeds the profile's context budget, the contract returns `HOLD / CONTEXT_BUDGET_EXCEEDED`. It must not silently delete rules, shrink evidence, widen authority, or invent a priority order to make the profile fit.

## Surface rules tested

- untrusted webhook exposure is constrained to the Hermes-observed safe set: `clarify`, `vision_analyze`, `web_extract`, `web_search`;
- `terminal` on the untrusted webhook surface is refused;
- desktop-only affordances such as `open_preview` remain unavailable on CLI even when the capability is declared and installed;
- an exposed desktop capability without authority is held;
- exposure plus authority reaches only `READY / AUTHORIZED_NOT_EXECUTED`;
- an over-budget profile holds before action semantics are considered.

## Donor boundary

The Hermes source files remain external donor evidence. v0.21 does not import the Hermes agent loop, terminal runtime, shell-hook executor, context engine, skills runtime, memory runtime, cron runtime, or local AXM Hermes adapter.

The shell-hook source is retained in the donor receipt because its typed event vocabulary may justify a later witness-receipt experiment, but v0.21 does not execute hooks.

## Authority boundary

WALDO does not decide which surface should receive a capability and cannot make exposure grant authority. It verifies a frozen contract fixture and emits a digest-only witness.

No selection, execution, install, promotion, CANON change, or upstream OpenWALDO mutation is authorized by this ADR.
