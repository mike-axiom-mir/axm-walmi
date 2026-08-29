# AXM/WALDO v0.14 — independent witness dissent ledger

v0.14 keeps the v0.13 generated peer-audit ring unchanged and asks a temporal question:

> Can later independent evidence resolve uncertainty without rewriting earlier dissent or turning a newer witness into a final judge?

## Frozen source

The experiment binds the v0.13 peer-audit receipt:

`sha256:9daf12733393e19c6149215e6a8f4f2d774f870ff21ba22c04efd785d34c8303`

At the original peer vote the ring had 4 `EVIDENCE_PASS` results and 2 guarded `HOLD` results because no independent witness was then present. Those two historical HOLD edges are copied into the v0.14 source binding and must remain unchanged.

## Two independent implementations

### Witness A — Go structural verifier

`witness_a.go` independently checks the canonical v0.13 receipt digest, peer-ring topology, closed authority and no-CANON/no-quorum truth boundary.

Observed T1 result: **EVIDENCE_PASS**.

### Witness B — Python public-material verifier

`witness_b.py` consumes an explicit public evidence view and verifies two inherited v0.13 artifacts: the sealed peer-ring receipt and the readable pack projection that binds the full generated-pack digest.

At **T1**, its view intentionally contains the peer-ring receipt but omits the public peer-audit pack projection. It therefore returns **HOLD / MATERIAL_EVIDENCE_VIEW_INCOMPLETE**.

At **T2**, the exact same implementation receives the missing public field-pack projection and returns **EVIDENCE_PASS**.

The T2 observation references the T1 HOLD as superseded evidence, but the old observation remains in the append-only ledger.

## Result

At T1 the independent witnesses genuinely disagree because they use different implementations and different evidence requirements:

- witness A: PASS
- witness B: HOLD

At T2 witness B resolves its own uncertainty after receiving more evidence. This does **not** rewrite:

- witness B's T1 HOLD;
- either v0.13 guarded peer HOLD;
- the fact that no independent witness existed during the original peer vote.

No majority/quorum field can grant authority and the ledger has no final judge.

## Observed identities

- v0.14 receipt: `sha256:d8f5407b6fde3335385967c5d837d563d8474841e2a00038267e6010b06f70ec`
- compact WALDO witness: `1493389d5e0bb6d0654a7df078a3b147a379d720ddb06632c78050db6fb423ce`
- ledger head: `sha256:4807f6264c74bf013e7ccc1fdc68f87ec92bffc5be990b9c062b758d7cfd19e3`

## Replay artifacts

- `witness-b-view-t1.json` — incomplete public-material view by design.
- `witness-b-view-t2.json` — same view with the pack projection supplied.
- `../../internal/axmmirror/testdata/peer-audit-ring-v0.13.json` — inherited sealed v0.13 peer-ring receipt.
- `../v0.13-peer-audit/peer-audit-pack-public-projection.json` — inherited public projection binding the full generated-pack digest.
- `witness-dissent-ledger.full.json` — public duplicate of the observed append-only ledger.
- `build_ledger.py` — deterministic ledger-sealing helper.

## Truth boundary

The experiment is detached and evidence-only. It does not select, install, register, stage, promote, merge, train, mutate target code or change CANON. The harness made no live AI provider call and requested no network access.
