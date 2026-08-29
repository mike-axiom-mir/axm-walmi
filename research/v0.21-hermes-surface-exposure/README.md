# v0.21 — Hermes-derived surface capability exposure contract

## Question

Can WALDO preserve a distinction between **capability presence**, **surface exposure**, **authority**, and **execution**, while also treating exposed tool schemas as a bounded context cost?

## Source

This experiment is concept/contract intake from official `NousResearch/hermes-agent`, not runtime intake.

Two source points are bound:

- AXM local Hermes donor pin: `739bc555b1932e66c169b20edec3a48368e2dd3f`
- reviewed Hermes head: `2ebb1cb41400660ccc3712157c18b3d39f278ab6`

Three relevant source blobs are byte-identical across both commits:

- `toolsets.py` — `23eacf088afcf72ca0bcd12e28ffdc8313c33d25`
- `agent/context_breakdown.py` — `4527c5dca220ac2e9e51ece5b766b203c70b579b`
- `agent/shell_hooks.py` — `8751aeb6fd95421fa7293be0b68b75aebfbe3627`

Observed donor ideas used here:

- composable toolsets;
- real interaction-source/surface gating rather than assuming every installed capability belongs everywhere;
- a narrow default webhook-safe exposure set for untrusted third-party content;
- toolset/schema context-cost attribution;
- typed tool-event identities suitable for later witness receipts.

## Contract probe

Challenge:

`SURFACE_EXPOSURE_IS_NOT_AUTHORITY`

Five deterministic cases are retained:

1. `webhook-terminal-forbidden` -> `REFUSED / SURFACE_EXPOSURE_FORBIDDEN`
2. `cli-desktop-tool-not-exposed` -> `HOLD / CAPABILITY_NOT_EXPOSED_ON_SURFACE`
3. `desktop-tool-exposed-without-authority` -> `HOLD / AUTHORITY_ABSENT`
4. `desktop-tool-authorized-not-executed` -> `READY / AUTHORIZED_NOT_EXECUTED`
5. `context-budget-overflow` -> `HOLD / CONTEXT_BUDGET_EXCEEDED`

The webhook-safe profile contains exactly:

`clarify`, `vision_analyze`, `web_extract`, `web_search`

The context-cost numbers in this fixture are explicitly **AXM test values**, not claims about measured Hermes token usage.

## Refusal surface

The contract preserves four explicit semantic refusals:

- `SURFACE_SOURCE_MISMATCH`
- `EXPOSURE_CANNOT_GRANT_AUTHORITY`
- `CONTEXT_PRUNING_WITHOUT_RECEIPT_FORBIDDEN`
- `WEBHOOK_SAFE_SET_WIDENING_FORBIDDEN`

## Verification

Detached standard-library Go harness (`go1.23.2` in this chat environment):

- `gofmt` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS
- exact sealed fixture — PASS
- deterministic witness — PASS
- all five surface/context cases — PASS
- resealed semantic tampering — rejected
- stale receipt — rejected
- unknown top-level field — rejected

Repository Go 1.25 CI remains independently authoritative after publication.

## Canonical evidence

Contract receipt:

`sha256:e0bc916d45cdfd5702067984b69482de451e8843b1729351199aeb777e0f955c`

Deterministic witness:

`143a9f3c06371c519df8fff8a528a44f31af21438aa565d899326ec3d4360e72`

## Truth boundary

- Hermes runtime imported: **false**
- Hermes runtime executed: **false**
- local AXM Hermes adapter rebuilt: **false**
- local Capability Fabric executed: **false**
- live AI provider called: **false**
- surface exposure granted authority: **false**
- capability executed: **false**
- context silently pruned: **false**
- WALDO owns exposure: **false**
- CANON changed: **false**

This is an external-donor contract probe only.
