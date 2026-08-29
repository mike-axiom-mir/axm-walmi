# ADR 9010: Grow verifiers and tool wisdom without self-authority

Status: experimental fork decision

## Context

The v0.6 WALDO witness clone can preserve provenance, evaluate bounded claims,
compare current and backup skill manifests, forge small deterministic assets,
and prepare external capability handoffs. It still has two connected growth
gaps.

First, a verifier can become stale or incomplete. Freezing every verifier
forever would preserve old errors. Letting a verifier rewrite itself after it
declares its own change valid would be worse: the thing being changed would
also control the evidence, acceptance rule, and rollback decision.

Second, the clone needs to learn from tool use without receiving an ever-growing
memory blob. A global blob would mix identities, tool versions, task classes,
stale observations, failures, raw payloads, and unverified repetition. It would
make retrieval accidental and could turn mere frequency into an unsupported
competence claim.

The desired growth loop must therefore allow revision and experience while
keeping provenance, independence, freshness, dissent, rollback, promotion, and
CANON boundaries explicit.

## Decision

Add seven fork-only organs in two linked surfaces:

1. compiled verifier invariant kernel;
2. digest-bound verifier registry;
3. mutual verifier change witness;
4. retained-generation rollback guard;
5. bounded Repair Buddy;
6. identity-and-tool-scoped experience memory;
7. selective tool-wisdom recall.

All durable inputs use strict versioned JSON. Unknown fields, duplicate keys,
trailing values, malformed digests, open authority, and self-digest mutation
are refused. CLI outputs remain atomic and no-replace.

## Evolvable verifier definitions, fixed invariant kernel

Schemas:

- `axm.waldo-witness.verifier-registry/v0.1`;
- `axm.waldo-witness.verifier-change-request/v0.1`;
- `axm.waldo-witness.verifier-change-receipt/v0.1`.

A verifier definition is data. It names an ID, version, verifier class,
independence group, rule-definition digest, test-pack digest, implementation
digest, and protected status. The v0.1 path does not load or execute those
implementation bytes.

The invariant kernel is not a member of the changing verifier set. Its compiled
v0.1 contract requires:

- no target verifier may review its own change;
- at least two currently registered passing peers;
- at least two independent reviewer groups;
- exact reviewer version and definition binding to the active registry;
- one sealed intent and one frozen fixture/protocol identity;
- exact consensus on baseline and candidate result-set digests;
- zero regressions and zero unknown results;
- at most three retained rollback-generation digests;
- explicit Merge Gate review for protected verifiers;
- closed execution, training, promotion, CANON, and world-action authority.

The change protocol cannot edit those rules. Changing the invariant kernel
requires a new reviewed protocol and ADR outside the self-change path.

The evolvable verifier classes identify where mutual verification is needed:

| Class | Required boundary |
| --- | --- |
| `SCHEMA_INVARIANT` | strict decoding, canonical form, self-digests, limits |
| `PROVENANCE_BINDING` | exact source, identity, request, receipt, and generation joins |
| `BEHAVIOR_EVALUATION` | frozen replay and regression decisions |
| `PRIVACY_BOUNDARY` | redaction declarations and memory-admission content limits |
| `CONTINUITY_FRESHNESS` | TTL, active generation, previous generation, rollback retention |
| `AUTHORITY_PERMISSION` | tool, installation, activation, promotion, CANON, and human gates |
| `ARTIFACT_RUNTIME` | implementation/test-pack identity and separately observed runtime evidence |

No single verifier is sufficient at a boundary where its own definition or
evidence supply is changing. A valid change uses peers from other registered
independence groups and preserves their individual findings rather than
collapsing them into one confidence score.

## Fail-safe generation and rollback semantics

`assess-verifier-change` recomputes every binding and emits a typed receipt.
`VERIFIER_CHANGE_READY` embeds a sealed next-generation registry with:

- generation incremented by one;
- exact previous-registry digest;
- current registry first in the retained rollback list;
- only the target definition replaced;
- unchanged invariant kernel and closed authority.

READY is permission to materialize a candidate file only. It is not runtime
activation. `materialize-verifier-change` recomputes the entire assessment and
writes the candidate to a new no-replace path.

Any peer rejection, binding drift, self-approval, or shadow-result mismatch
emits a `ROLLBACK_*` receipt. Missing evidence or quorum emits a `HOLD_*`
receipt. Neither path emits a candidate registry. The current registry remains
active and is copied into `rollback_target_sha256`. This is fail-safe rollback:
the active pointer never moves, so the rejected candidate has nothing to undo
or overwrite.

Protected verifier changes may receive passing peer analysis, but remain
`HOLD_PROTECTED_MERGE_GATE`. The clone cannot authenticate or replace the human
Merge Gate.

## Repair Buddy

Schemas:

- `axm.waldo-witness.repair-buddy-request/v0.1`;
- `axm.waldo-witness.repair-buddy-plan/v0.1`.

Repair Buddy accepts an exact failed or held change receipt and fresh incident
evidence. It reruns the original assessment against the active registry before
diagnosing anything. It then preserves the complete bounded lifecycle:

`OBSERVE -> DETECT -> CONTAIN -> DIAGNOSE -> PROPOSE -> SIMULATE -> AUTHORIZE -> REPAIR -> VERIFY -> LEARN`

Only the first four stages may complete automatically. `PROPOSE` is a
candidate. Simulation and authorization are required. Repair, verification,
and learning remain blocked until a new change intent independently passes the
original invariant kernel.

The plan can recommend collecting missing reviews, rebuilding evidence from
the frozen baseline, revising or withdrawing a regressing candidate,
quarantining a self-approval path, or escalating a protected change. It has no
patch, command, installer, target path, active-pointer mutation, promotion, or
CANON mechanism. A READY change produces `HOLD_REPAIR_NOT_NEEDED` rather than a
gratuitous repair candidate.

## Identity-bound tool memory and selective wisdom

Schemas:

- `axm.waldo-witness.identity-tool-experience/v0.1`;
- `axm.waldo-witness.identity-tool-memory-shard/v0.1`;
- `axm.waldo-witness.identity-wisdom-query/v0.1`;
- `axm.waldo-witness.identity-wisdom-view/v0.1`.

Memory is sharded by exact answering-identity digest, tool ID, and tool
version. An experience binds input and output artifact digests, tool receipt,
independent verification receipt, task class, use tags, outcome, observed time,
and TTL. It contains only a bounded public-safe `REUSE` or `AVOID` distillation.
Raw prompts, tool payloads, artifact bytes, hidden reasoning, and secrets are
not accepted.

Only independently verified success or independently verified failure can be
appended. Repetition without verification is not memory growth. Cross-identity,
cross-tool, cross-version, duplicate, backdated, mutated, and over-limit
entries are refused. Each generation self-digests and links the exact previous
shard.

`recall-tool-wisdom` selects by exact identity, tool version, task class, and
intersecting use tags. Stale and future entries are omitted. The model-facing
view contains at most eight newest relevant distillations plus their evidence
digests. It never receives the entire shard or an implicit cross-tool memory
blob.

This is evidence-based context growth, not hidden weight training. The memory
does not claim general competence, authorize the tool, install a specialist,
or promote a lesson to CANON.

## Compatibility and verification

Checked fixtures pin the initial registry, READY and rollback change paths,
Repair Buddy plan, sealed tool experience, initial memory shard, and selective
wisdom view. Tests cover self-approval, reviewer drift, peer regression,
insufficient quorum, protected changes, fresh reassessment, tamper detection,
cross-tool admission, raw or unverified content, bounded retrieval, and stale
wisdom.

The compiled self-capability catalog gains the new command surfaces. Its
snapshot and downstream handoff-example digests are re-sealed as a deliberate
compatibility cascade. No external provider run is implied.

## Alternatives rejected

- A verifier approving its own rewrite gives the changed rule circular
  authority.
- Majority voting without exact current-definition and replay binding permits
  stale or incomparable peers.
- Automatic rollback by overwriting the active file destroys evidence and
  introduces partial-write risk.
- Repair Buddy directly patching a verifier collapses diagnosis, authorization,
  and repair into one authority source.
- One global memory blob mixes identities, tools, versions, freshness, and data
  boundaries.
- Storing raw tool interactions or hidden reasoning adds unnecessary privacy
  and prompt-injection risk.
- Treating successful repetition as wisdom confuses frequency with independent
  verification.

## Consequences

The clone can now prepare bounded verifier evolution, retain a trusted rollback
generation, diagnose failed changes, and accumulate tool-specific verified
wisdom without a live platform. It still cannot execute peer verifiers from
their digests, authenticate human approval, activate registry generations,
repair code, install tools, train itself, or make an AXM result CANON.
