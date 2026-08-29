# 9024: Treat run-capsule evidence as candidate evidence, not promotion authority

## Status

Accepted for `EXPERIMENTAL` v0.22 evidence only.

## Context

AXM Platform PR #48 upgraded the Hermes adapter into a bounded runtime lane with per-run manifests, metadata-only tool/provider/session receipts, integrity flags, and candidate-only Return Packets. It also preserves completion evidence for an already-authorized in-flight tool when action consent is later revoked.

This is useful to WALDO as an evidence contract, not as execution machinery.

## Decision

Add a pure verifier for a frozen run-capsule fixture. The verifier binds exact PR #48 donor identities and requires:

- source verification before and after the run;
- completion-receipt count not below authorized tool-call count;
- explicit provider-policy mismatch evidence;
- explicit policy/profile drift evidence;
- raw prompt/response/tool bodies excluded;
- `canon=false` and `promotion=candidate-only`;
- completion evidence may survive revocation for a previously authorized in-flight action;
- revocation cannot authorize any additional action.

Evidence completeness can produce `CANDIDATE`, never promotion.

## Boundary

WALDO does not import `axm_gate.py`, sanitize the environment, launch Hermes, execute tools, call a model provider, approve a Return Packet, or change CANON.

Platform PR #48 remains the runtime donor. WALDO remains witness/verifier only.
