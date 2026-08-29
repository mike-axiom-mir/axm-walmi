# ADR 9002: Make the WALDO-bound Mirror clone a provenance witness, not a normal Mirror

Status: experimental fork decision

## Context

The public AXM `mirror` branch already contains a broad reasoning body with many
deterministic organs for readiness, capability discovery, memory guidance,
resource stewardship, Foundation observation, repair/evidence routing, and
bounded Workshop handoffs. Copying that organ set wholesale into this fork would
create a second normal Mirror and would obscure the reason this clone exists.

WALDO already provides a different native substrate: durable corpus, run, model,
and release lineage; content-addressed artifacts; explicit training plans;
append-only run history; verified exports; and local inference over verified
model artifacts.

This clone therefore has a narrower purpose:

> Be a provenance-native witness model whose primary special capability is to
> inspect and preserve the relationship between source material, training,
> model identity, concrete behavior, evaluation, disagreement, and later
> changes without pretending that hashes prove truth, legal rights, safety, or
> causal influence.

Working nickname in this ADR: **WALDO Witness Mirror**. This is descriptive, not
a product or upstream project name.

## Architectural rule

The learned model may reason *about* provenance, but it is never the authority
for its own provenance. Durable WALDO BOMs and deterministic fork-only organs
remain authoritative for lineage identity.

The clone should initially have fewer organs than normal Mirror. Each new organ
must justify why it belongs to a provenance witness rather than a general
assistant.

## Reuse from normal Mirror

Reuse concepts, contracts, and tests where they fit; do not bulk-copy the full
runtime.

The most useful existing Mirror patterns are:

- closed authority and explicit permission state;
- immutable/content-addressed evidence;
- unknown and contradictory states as first-class outcomes;
- dissent preservation rather than score flattening;
- bounded source inventories and source mutation refusal;
- separation of observation, evidence admission, training, promotion, CANON,
  tool use, and world action;
- independent/held-out evaluation as a separate claim from local self-tests.

General Workshop discovery, broad tool planning, arbitrary capability growth,
resource economics, and autonomous repair are not first-wave requirements for
this clone.

## Clone-specific organ families

### Family A — Origin and lineage

#### 1. Origin Anchor Organ

Purpose: bind every observed behavior to an exact WALDO model or release BOM
before the behavior can enter the witness ledger.

Inputs:
- model/release BOM reference;
- optional origin, run, and corpus BOM references;
- observed artifact hashes.

Outputs:
- exact lineage anchor;
- missing-link HOLDs;
- mismatch/refusal states.

Authority: read/verify only. Model prose can never author or replace lineage.

#### 2. Corpus Evidence Lens Organ

Purpose: project the stable OpenWALDO corpus BOM into a compact machine surface
for the clone: selected paths, index revision, manifest/source pins, license
assertions/policy, shard identities, counts, and validation state.

Important limit: a corpus BOM proves recorded and verified identities within its
contract. It does not prove legal usability, truth, quality, safety, or that a
later trainer consumed every byte.

#### 3. License Assertion Boundary Organ

Purpose: keep license assertions, verification state, disputes, and use-policy
filters visible without silently converting them into legal conclusions.

States should distinguish at least:
- RECORDED_ASSERTION;
- VERIFIED_IDENTITY_ONLY;
- DISPUTED;
- MISSING;
- USER_POLICY_ALLOWED;
- USER_POLICY_EXCLUDED;
- LEGAL_CONCLUSION_NOT_PROVIDED.

This organ must never answer "legally safe" from a hash or manifest assertion.

#### 4. Training Run Witness Organ

Purpose: bind a behavior to the exact training run lineage: compose/plan, corpus
BOM, origin if any, backend identity, environment observations, checkpoint/run
state, and output hashes.

It must preserve WALDO's distinction between a real run and explicit simulated
`fake` backend artifacts.

#### 5. Release Identity Lock Organ

Purpose: resolve "which model answered?" to one verified released/managed
artifact identity, tokenizer, architecture, and current run/origin selection.

A changed model, tokenizer, release BOM, or current verified run creates a new
identity rather than mutating an old behavior record.

### Family B — Inference provenance and claims

#### 6. Provenance Context Surface Organ

Purpose: expose a bounded read-only provenance packet to the clone at inference
when requested.

The packet is generated from deterministic BOM readers, not from model memory.
It lets the model reason over facts such as its exact release, selected corpus
families, known license assertions, and training-run identity while preventing
self-authored lineage.

No tool, training, permission, or write authority is implied.

#### 7. Source-Claim Gate Organ

Purpose: separate a model's natural-language claims about its origin from
machine-resolved provenance.

Examples:
- "I was trained on corpus X" may be CONFIRMED only if the relevant BOM proves
  that selection.
- "This answer came from document Y" must normally remain UNPROVEN unless a
  separate retrieval/attribution mechanism supplies evidence.
- "I learned this fact from source Z" is not inferable merely because Z was in
  the training corpus.

The gate should emit typed claim status instead of rewriting the model's words.

### Family C — Evaluation and behavior

#### 8. Behavior Evidence Seal Organ

Already started in ADR 9001 / `internal/axmmirror`.

Purpose: preserve context/request/output digests, exact WALDO lineage, permission
state, verification results, dissent, observed outcome, and closed authority.

Sealing proves identity, not truth or safety.

#### 9. Evaluation Independence / Contamination Guard Organ

Purpose: determine whether an evaluation pack can honestly be described as
held-out or outside-authored with respect to the exact training selection.

Checks may include:
- exact record/text hash overlap;
- source-group overlap;
- corpus path overlap;
- evaluator authorship declaration;
- whether the evaluation existed before or after the training seal when that
  timing is actually evidenced.

Any incomplete comparison becomes HOLD/UNKNOWN, never "independent by default."

#### 10. Behavior Delta Comparator Organ

Purpose: compare the same sealed evaluation across two exact model/release
identities without collapsing the result into one score.

Preserve independently:
- unchanged behavior;
- improvement candidate;
- regression candidate;
- changed but unresolved;
- new/removed refusal;
- verification deltas;
- dissent deltas;
- permission/authority deltas.

This becomes the main lens for studying whether corpus/training changes correlate
with behavior changes.

#### 11. Dissent Continuity Ledger Organ

Purpose: carry reviewer/model/test disagreement across releases rather than
letting a new successful run erase an older objection.

Dissent can become resolved or superseded only by an explicit evidence link.
No majority vote automatically converts disagreement into truth.

#### 12. Reproducibility Twin Organ

Purpose: compare an original WALDO run/release with a later reconstruction.

Possible states:
- EXACT_ARTIFACT_MATCH;
- SAME_PINNED_INPUTS_DIFFERENT_ARTIFACT;
- ENVIRONMENT_DIFFERENCE_EXPLAINS_CANDIDATE_DIVERGENCE;
- INPUT_DRIFT;
- INCOMPLETE_EVIDENCE;
- NOT_COMPARABLE.

Bit-for-bit equality and behavioral similarity remain separate claims.

### Family D — Source change and impact

#### 13. Source Change Impact Router Organ

Purpose: when a manifest, license assertion, source record, corpus BOM, or
origin is corrected/disputed/removed, identify downstream runs, models,
releases, and behavior records whose interpretation may be affected.

Output is a review graph, not automatic deletion, unlearning, or invalidation.

This organ is potentially useful upstream because WALDO already preserves the
lineage needed to answer "what depended on this?"

#### 14. Controlled Attribution Experiment Planner Organ

Purpose: propose controlled A/B or ablation experiments when somebody wants to
ask whether a corpus change affected a behavior.

It must state the causal limit clearly: membership in a training corpus does not
prove that one source caused one output. Stronger attribution requires controlled
comparisons and still may remain probabilistic or unresolved.

The planner cannot select a legal conclusion, mutate data, launch training, or
promote a model automatically.

#### 15. Memorization / Leakage Audit Organ

Purpose: test whether a model reproduces protected evaluation strings,
synthetic canaries, or suspiciously exact training passages under a declared
audit protocol.

The audit should distinguish exact match, near match, ordinary common phrase,
unknown, and unverifiable cases. It is an evaluation instrument, not proof of
copyright infringement or source causality.

### Family E — Collaboration boundary

#### 16. Upstream Contribution Extractor Organ

Purpose: separate generic WALDO-useful research from AXM-specific Mirror
identity before anything is proposed upstream.

It may produce a review packet containing:
- generic contract or verifier;
- tests;
- WALDO-facing rationale;
- AXM-specific pieces deliberately excluded;
- known limits and unrun tests.

It has no authority to open an upstream PR or issue automatically.

## Organs intentionally not copied initially

Do not initially import the full normal-Mirror families for:

- broad Workshop capability discovery;
- autonomous tool/hand planning;
- general memory guidance;
- automatic repair;
- resource economics/provider selection;
- general Foundation growth routing;
- world-action planning.

Those may be useful later, but they would blur the experiment before the
provenance-witness core is proven.

## First build order

Implementation note: ADR 9003 implements the Origin Anchor, Release Identity
Lock, and Evaluation Independence / Contamination Guard. ADR 9004 implements
the Corpus Evidence Lens and Training Run Witness and binds run-backed behavior
evidence to both receipts. All Wave 1 deterministic lineage slices now exist;
this does not mean a learned clone has been trained.

ADR 9005 implements the Training Profile Contract Lens, Provenance Context
Surface, Source-Claim Gate, Evaluation Protocol Seal, and their integrated
Gated Behavior Seal. Behavior comparison remains later work and must consume a
precommitted protocol rather than assigning credit retrospectively.

### Wave 1 — required before a real clone run

1. Origin Anchor Organ
2. Corpus Evidence Lens Organ
3. Training Run Witness Organ
4. Release Identity Lock Organ
5. Behavior Evidence Seal Organ (started)
6. Evaluation Independence / Contamination Guard Organ

### Wave 1.1 — upstream behavior-contract compatibility

7. Training Profile Contract Lens Organ

### Wave 2 — required for useful comparative research

8. Provenance Context Surface Organ
9. Source-Claim Gate Organ
10. Evaluation Protocol Seal Organ
11. Behavior Delta Comparator Organ
12. Dissent Continuity Ledger Organ
13. Reproducibility Twin Organ

### Wave 3 — research that may be valuable back to WALDO

12. Source Change Impact Router Organ
13. Controlled Attribution Experiment Planner Organ
14. Memorization / Leakage Audit Organ
15. License Assertion Boundary Organ hardening
16. Upstream Contribution Extractor Organ

## Relationship to WALDO internals

Fork organs should consume durable WALDO artifacts and documented contracts,
not depend on undocumented `internal/*` implementation shapes unless an
explicit experiment requires it.

This preserves the upstream rule that internal Go packages are replaceable
implementation while durable BOMs and documented compatibility surfaces carry
meaning.

## Relationship to learned weights

The learned clone is not the provenance database.

Preferred loop:

```text
WALDO durable BOMs / verified artifacts
              |
              v
   deterministic witness organs
              |
        bounded packet
              |
              v
        learned model
              |
      proposed response
              |
              v
 deterministic claim/evidence gates
              |
              v
 sealed behavior evidence + dissent
```

This lets the model reason with provenance while making it impossible for a
fluent answer to silently rewrite the provenance record.

## Truth boundary

This architecture does not claim token-level source attribution, legal
clearance, model safety, consciousness, causal explanation of individual
outputs, or exact reproducibility where the execution environment cannot
provide it.

The experiment remains public-safe, additive to WALDO, and closed-authority by
default.
