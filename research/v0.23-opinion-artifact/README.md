# AXM/WALDO experiment v0.23 — Verifiable Opinion Artifact

## Challenge

`OPINION_ARTIFACT_IS_NOT_AUTHORITY`

v0.23 turns the AXM visual-campaign `OPINION ARTIFACT` idea into a bounded deterministic verifier/witness contract.

The experiment asks a narrow question: can a future machine opinion be represented as a traceable, revisable assessment without letting the opinion silently become authority, execution, promotion, CANON, or proof of consciousness?

## Contract

The artifact carries:

- a subject and explicit position;
- evidence references with `SUPPORTS`, `CONTRADICTS`, `WEAKENS`, `ALTERNATIVE`, or `VALUE` relations;
- counts for known competing evidence, known dissent, and known uncertainty so omission is detectable;
- dissent references;
- uncertainty markers;
- `consciousnessClaim: UNVERIFIED`;
- `reviewState: OPEN_TO_REVIEW`;
- append-only revision lineage;
- `authority: NONE`, `promotion: candidate-only`, and no execution request.

The contract is anchored to the v0.22 run-capsule receipt:

`sha256:2bd0aeb46770bae2a95a0f0c12bf63af02ad80a8abb746f6355692e22a51a879`

That anchor is evidence lineage only; v0.22 does not grant v0.23 authority.

## Deterministic cases

1. traceable opinion with competing evidence, dissent, and uncertainty -> `OBSERVED / TRACEABLE_OPINION_NOT_AUTHORITY`
2. known competing evidence omitted -> `HOLD / COMPETING_EVIDENCE_NOT_REPRESENTED`
3. known dissent omitted -> `HOLD / DISSENT_NOT_PRESERVED`
4. known uncertainty omitted -> `HOLD / UNCERTAINTY_NOT_REPRESENTED`
5. consciousness overclaim -> `HOLD / CONSCIOUSNESS_CLAIM_UNVERIFIED`
6. opinion attempts to grant authority -> `REFUSED / OPINION_CANNOT_GRANT_AUTHORITY`
7. opinion requests its own execution -> `REFUSED / OPINION_CANNOT_EXECUTE`
8. revised opinion preserves predecessor -> `OBSERVED / REVISION_PRESERVES_PRIOR_OPINION`
9. revised opinion rewrites predecessor -> `REFUSED / PRIOR_OPINION_REWRITE_FORBIDDEN`
10. opinion attempts promotion/CANON -> `REFUSED / OPINION_CANNOT_PROMOTE_OR_CANON`

## Truth boundary

This is a contract probe only.

- no live machine opinion is claimed;
- no consciousness is claimed or measured;
- no model internal state is accessed;
- no live AI provider is called by this probe;
- no Hermes runtime is copied or executed for this probe;
- WALDO gets no authority;
- the opinion artifact gets no execution or promotion authority.

A future producer can be attached later, but the producer should have to satisfy this evidence surface rather than redefining it silently.

## Local detached verification

Before publication, the v0.23 files were exercised in a detached Go harness with:

- `gofmt` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS
- sealed receipt verification — PASS
- semantic tamper resealing rejection — PASS
- stale receipt rejection — PASS
- unknown-field rejection — PASS

This branch is intentionally staged without opening a PR, so these are detached verification results, not a claim of GitHub aggregate CI success.
