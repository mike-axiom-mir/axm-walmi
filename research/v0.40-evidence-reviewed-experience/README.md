# WALDO / AXM Mirror v0.40 — evidence-reviewed experience

## Challenge

`OBSERVED_OUTCOME_LABEL_IS_NOT_TRAINING_AUTHORITY`

v0.38 made experience visible and durable. v0.39 separated synthetic starting ground from observed chat and execution data. The remaining seam was subtler: a sealed experience could still describe its own outcome as `HELPFUL` or `CORRECTED` and immediately carry training-ready text.

That is useful as a learning candidate, but the label itself is not enough evidence for a weight-training promotion.

## Result

**PASS for the additive promotion gate.**

v0.40 adds a deterministic evidence-review boundary between a v0.38 experience learning candidate and the v0.39 observed-ground training path.

The flow is now representable as:

```text
SENSED
  -> REACTED
  -> OUTCOME_OBSERVED
  -> REFLECTED
  -> learning candidate
  -> independent evidence review
       | APPROVE_TRAINING -> reviewed promotion -> observed ground -> normal WALDO training intake
       | REJECT_TRAINING  -> no positive training target
```

Memory and retrieved experience remain available regardless of training approval. Rejecting a weight-training promotion does not erase the original episode, feedback, reflection, or Hermes memory projection.

## New contracts

### `axm.waldo.mirror-experience-training-review/v0.40`

A review binds:

- the exact v0.38 `learningRecordSha256`;
- a separate evidence receipt;
- an evidence class;
- `APPROVE_TRAINING` or `REJECT_TRAINING`;
- a visible rationale and review timestamp; and
- closed authority.

Accepted evidence classes are intentionally explicit:

- `USER_CONFIRMED`
- `TOOL_VERIFIED`
- `EXECUTION_VERIFIED`
- `INDEPENDENT_REVIEW`

The evidence receipt may not simply reuse the learning candidate digest. Promotion also rejects recycling the episode reflection digest as verification evidence.

### `axm.waldo.mirror-experience-reviewed-training/v0.40`

An approved review produces a sealed promotion that transitively binds:

- original episode ID and outcome signal;
- prompt and target response;
- v0.38 learning-record digest;
- reflection-event digest;
- review digest;
- independent evidence receipt and evidence class; and
- capture/review timestamps.

It carries **no execution, model-training, promotion, CANON, or world-action authority**. It is evidence, not permission.

## Bridge into v0.39 ground

`MirrorExperienceReviewedTrainingRecord.GroundRecord(...)` converts the reviewed promotion into the existing strict `OBSERVED_CHAT` or `OBSERVED_EXECUTION_TRACE` ground shape.

The resulting ground record uses the reviewed promotion digest as its source receipt. That means the v0.39 verifier receives one digest that transitively binds the candidate, reflection, review, and independent evidence receipt instead of accepting the outcome label alone as the provenance story.

The existing ground verifier and structured training projection remain unchanged.

## Deterministic checks added

The v0.40 tests require that:

1. a learning candidate cannot serve as its own evidence receipt;
2. a reflection event cannot be recycled as independent evidence;
3. a rejected review produces no positive training promotion;
4. an approved externally receipted review produces a hash-bound reviewed promotion;
5. that promotion can enter the v0.39 observed-ground verifier without losing its transitive receipt chain;
6. unknown review fields fail closed; and
7. post-seal target tampering is rejected.

## Claim boundary

This branch proves the **review primitive and the reviewed bridge**. It does **not** claim that every historical or legacy WALDO training entrypoint has already been rewired to require v0.40 review records. The v0.38 experience producer remains compatible and its learning record should be treated as a candidate until passed through this gate.

A later integration step can make this review gate mandatory at CLI/training-ingest boundaries after compatibility impact is measured. That should be an explicit migration, not a silent semantic rewrite.

## Why this matters

Experience should count, including successful behavior, corrections, failures, and uncertainty. But “the system logged that this was helpful” and “there is independent evidence supporting weight training on this target” are different claims.

v0.40 keeps both: fast visible learning through retained experience, and a stronger evidence threshold before that experience is allowed to teach the model weights.
