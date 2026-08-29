# 9016: Preserve independent witness disagreement as append-only evidence

## Status

Accepted for `EXPERIMENTAL` v0.14 evidence only.

## Context

v0.13 generated three bounded audit organs and let them audit one another. The ring preserved two guarded HOLDs instead of converting a 4-to-2 majority into authority. WALDO then witnessed the ring externally.

That still leaves a harder continuity problem: later evidence can legitimately resolve an observer's uncertainty. A naive system may overwrite the earlier HOLD with a later PASS, or treat the newest/strongest witness as a final judge. Either destroys historical truth.

## Decision

Record independent witness observations in an append-only digest chain.

A later observation may reference an earlier one as superseded evidence, but it must not remove, mutate or retroactively relabel the earlier observation. Peer-ring history is independently frozen and cannot be rewritten merely because external witnesses appear later.

No ledger aggregation rule grants authority. There is no final-judge field with an active verdict.

## v0.14 experiment

Two independent implementations observe the frozen v0.13 ring:

1. a Go structural verifier receives the full sealed receipt and returns `EVIDENCE_PASS`;
2. a Python material-evidence verifier at T1 receives an incomplete artifact view and returns `HOLD`;
3. the same Python verifier at T2 receives the missing field-pack artifact and returns `EVIDENCE_PASS`.

The resulting ledger therefore preserves both temporal facts:

- at T1 the witnesses disagreed;
- at T2 witness B's uncertainty was resolved by additional evidence.

The T1 HOLD remains present and digest-addressable. The original v0.13 peer-ring HOLDs also remain present because they describe what was known at the peer-vote boundary, not what became known later.

## Verification boundary

The WALDO-side verifier requires:

- exact v0.13 receipt/witness/peer-pack identities;
- the original 4 PASS / 2 HOLD peer topology;
- both original guarded HOLD edges;
- independent Go/Python witness identities;
- T1 PASS-vs-HOLD disagreement;
- T2 resolution by additional evidence;
- a three-entry chained ledger;
- the T2 entry to reference the retained T1 HOLD;
- no peer or witness history rewrite;
- no majority authority, final judge, installation, promotion or CANON change.

Tampering the fixture to erase the historical HOLD or assign a final judge must be rejected.

## Consequences

Resolution becomes additive rather than destructive. This permits stronger later evidence without pretending uncertainty never existed.

It also separates **current confidence** from **historical evidence**: a newer PASS can be useful without becoming a license to rewrite earlier witnesses or peer dissent.

## Non-goals

v0.14 does not define governance, voting, consensus, promotion, training admission or automatic repair. It does not make WALDO, either independent witness, or the generated peer ring authoritative over AXM.
