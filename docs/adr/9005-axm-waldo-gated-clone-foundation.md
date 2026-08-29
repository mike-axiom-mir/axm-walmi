# ADR 9005: Put deterministic gates around the WALDO witness clone

Status: experimental fork decision

## Context

Wave 1 binds downstream behavior evidence to WALDO corpus, run, model, and
release lineage. It does not yet define what provenance a learned specialist
may see, how its source claims are checked, or how an evaluation is frozen
before the model produces an answer.

Two new evidence sources sharpen that boundary:

- OpenWALDO now names training profiles by their data behavior: shuffled,
  balanced, and weighted. The former v1/v2/v3 names remain compatibility
  aliases. The exact upstream delta is recorded in
  `research/openwaldo-upstream-delta-2026-08-15.json`.
- The passive 115-organ Mirror archive supplies contract knowledge about
  source-labelled evidence, held-out evaluation, precommitment, typed HOLDs,
  and refusal of retrospective credit. Archived JavaScript remains inert and
  is not imported, executed, bulk-copied, connected, or promoted.

The clone should therefore inherit knowledge, not anatomy. Its implementation
must remain ordinary Go over documented WALDO artifacts and must keep learned
weights between deterministic input and output gates.

## Decision

Add four deterministic organs plus one integrated seal path under
`internal/axmmirror` and expose them through the separate
`waldo-axm-mirror` CLI.

### Training Profile Contract Lens

Schema: `axm.waldo-witness.training-profile-contract/v0.1`.

The lens consumes the exact documented `RUN-BOM.json` bytes and a valid
Training Run Witness. It exposes the declared and canonical profile identities,
historical alias state, profile schema, seed, data order, shuffle bounds,
packing, corpus-weight-set digest and bounded projection, and held-out selection
policy.

It recognizes the exact upstream contracts:

| Profile | Schema | Data order | Evaluation selection | Weights |
| --- | ---: | --- | --- | --- |
| `causal-pretrain-shuffled` | 1 | `bounded-shuffle-v1` | `lowest-sha256-v1` | refused |
| `causal-pretrain-balanced` | 1 | `corpus-balanced-shuffle-v1` | `stratified-lowest-sha256-v1` | refused |
| `causal-pretrain-weighted` | 1 | `corpus-weighted-shuffle-v1` | `stratified-lowest-sha256-v1` | required and complete |

Historical `causal-pretrain-v1/v2/v3` receipts retain their declared names and
historical schemas 1/2/3 while projecting the equivalent current behavior.
Unknown profiles, wrong schemas, behavior mismatches, and a different valid run
witness produce typed HOLD receipts. The lens never claims that selection or
weighting caused a later output.

### Provenance Context Surface

Schemas:

- request: `axm.waldo-witness.provenance-context-request/v0.1`;
- packet: `axm.waldo-witness.provenance-context/v0.1`.

The request embeds verified receipts, an explicit field allow-list, and a fact
byte limit. The output is a sorted list of facts. Every fact keeps four values
together:

1. field path;
2. JSON value;
3. evidence class and exact source-receipt digest;
4. claim ceiling.

Evidence classes are `STRUCTURALLY_WITNESSED`, `RECORDED_ASSERTION`,
`DECLARED`, `UNKNOWN`, and `NOT_PROVIDED`. Optional missing facts are emitted as
explicit null `NOT_PROVIDED` facts. A run-backed anchor requires a READY run
witness and a witnessed profile contract. Identity drift or mismatched receipt
joins produce `HOLD_CONTEXT_JOIN_MISMATCH`.

The packet is all-or-nothing. If its facts exceed `max_fact_bytes`, it emits
`HOLD_CONTEXT_LIMIT` with zero facts; it never silently truncates a provenance
surface. Artifact bodies, raw corpus text, prompts, hidden reasoning, secrets,
absolute paths, and authority are excluded.

### Source-Claim Gate

Schemas:

- submission: `axm.waldo-witness.source-claim-submission/v0.1`;
- assessment: `axm.waldo-witness.source-claim-assessment/v0.1`.

A submission binds typed claims to the exact digest of the model output and the
exact context packet. The gate checks answering identity, selected corpus-path
membership, recorded license assertions, canonical training profile identity,
and bounded exact-inventory evaluation independence.

It deliberately cannot confirm:

- source-to-output causality;
- legal usability or a grant of rights;
- exact reproducibility without a later Reproducibility Twin;
- unknown claim kinds.

Those become `UNPROVEN_CAUSAL_CLAIM`,
`LEGAL_CONCLUSION_NOT_PROVIDED`, `HOLD_INCOMPLETE_EVIDENCE`, or
`OUT_OF_SCOPE`, rather than optimistic prose. Findings remain independent; the
aggregate state is only a routing state and never a composite quality score.
The gate does not rewrite the bound output and does not claim that the submitted
typed list exhaustively represents every sentence in it.

### Evaluation Protocol Seal

Schemas:

- draft: `axm.waldo-witness.evaluation-protocol-draft/v0.1`;
- seal: `axm.waldo-witness.evaluation-protocol-seal/v0.1`.

Strict draft decoding excludes output and result fields. Before any output is
accepted, the seal freezes:

- exact pack and document digests;
- case count and case-order digest;
- authorship declaration and evidence digest;
- answer-key digest and visibility boundary;
- exact answering identity, anchor, and context packet;
- exact contamination comparison;
- request-set digest;
- allowed metrics and independent comparison dimensions;
- case, output-byte, and time limits;
- permission and requested authority.

Only a matched target, allowed permission, CLEAR exact declared overlap check,
outside-authored declaration, withheld or inapplicable answer key, and closed
requested authority produce `SEALED_READY_FOR_EXACT_EVALUATION`. Contamination,
unknown independence, answer-key visibility, target mismatch, denied/unknown
permission, and authority growth produce separate HOLD or REFUSED states.

The seal's digest detects mutation but does not by itself prove wall-clock
chronology. A later behavior receipt must bind the exact seal digest.

### Gated Behavior Seal

`SealGated` is the integration boundary. In addition to the existing run and
anchor joins, it requires:

- a witnessed profile contract matching the run witness;
- a READY context packet that retains the run and profile receipts;
- a source-claim assessment bound to the same context and output;
- a READY evaluation protocol bound to the same target, context, request set,
  permission, and experiment ID.

The final behavior record gains verified links to the profile, context, claim
assessment, protocol seal, and Training Run Witness. Contradicted or unresolved
source claims may still be sealed as durable failure/HOLD evidence, but they
cannot be represented as a passing behavior outcome.

## CLI

The strengthened flow is:

```text
lens-corpus
  -> witness-run
  -> profile-contract
  -> anchor / lock / contamination
  -> context
  -> seal-evaluation
  -> model output + typed claims
  -> gate-claims
  -> seal-gated
  -> verify
```

`seal-evaluation` must happen before model execution; `seal-gated` later proves
that the resulting evidence names that precommitted seal.

## Durability and authority

All new receipts carry closed authority. No organ may execute tools, launch
training, grant permission, promote a model, alter CANON, or act in the world.
Self-digests are mutation evidence, not signatures or authorship proofs.

The schemas above are experimental v0.1 durable contracts. Changes require a
new schema or an explicit compatibility decision, fixtures, and tests.

## Consequences

The future learned specialist can receive useful WALDO provenance without
becoming its author, and fluent output can be checked without granting the
model power to rewrite evidence. Evaluation comparison, dissent continuity,
and reproducibility remain later work; they must consume this precommitted
foundation rather than reconstructing it retrospectively.

No learned clone is trained by this decision, no archived Mirror organ is
activated, and no upstream WALDO or fork `main` behavior is changed.
