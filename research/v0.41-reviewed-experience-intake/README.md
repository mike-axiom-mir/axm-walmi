# v0.41 — reviewed experience intake

## Goal

Strengthen the v0.38 visible experience spine and v0.40 evidence-review gate by
adding a concrete intake executable that can turn a reviewed experience into a
v0.39 observed-ground record without letting the experience grade itself.

The pipeline is deliberately staged:

```text
SENSE / REACT / OBSERVE / REFLECT
             |
             +--> visible future context / Hermes projection
             |
             +--> experience learning candidate
                         |
                         v
                independent evidence review
                    |              |
                 REJECT          APPROVE
                    |              |
               memory only     reviewed promotion
                                      |
                                      v
                              observed ground record
                                      |
                                      v
                          normal WALDO verify / ingest / train
```

A rejection is not erasure. The underlying episode remains available to the
existing experience-memory path; only positive weight-training promotion is
withheld.

## New entry point

`cmd/waldo-mirror-review` consumes exactly one sealed v0.38 experience learning
candidate and one sealed v0.40 training review.

Example:

```bash
go run ./cmd/waldo-mirror-review \
  --reviewed-to private-reviewed.jsonl \
  --ground-to private-ground.jsonl \
  --data-class OBSERVED_CHAT \
  --root truth-before-story \
  --challenge OBSERVED_HELPFUL_RESPONSE \
  candidate.json review.json
```

For an approved review it writes a mode `0600` reviewed promotion. When
`--ground-to` is also supplied it writes a mode `0600` v0.39 ground record whose
source receipt is the reviewed-promotion digest. That digest transitively binds:

- the original experience learning record;
- the reflection event;
- the independent evidence receipt;
- the explicit review decision; and
- the final prompt/target training text.

The resulting ground record can then use the existing verifier:

```bash
waldo mirror ground verify private-ground.jsonl \
  --training-to private-structured-training.jsonl
```

The review executable never opens a model, mutates weights, changes identity,
executes tools, edits CANON, or deletes experience memory.

## Evidence classes

v0.40 already recognizes these independent evidence classes and v0.41 carries
them through the intake receipt:

- `USER_CONFIRMED`
- `TOOL_VERIFIED`
- `EXECUTION_VERIFIED`
- `INDEPENDENT_REVIEW`

The evidence receipt must not be the learning candidate hash, reflection hash,
or review hash. This blocks the simple circular case where generated learning
artifacts certify themselves.

## Runtime boundary

This is an additive reviewed-intake lane. The older v0.38
`waldo mirror experience observe --learn-to` output still exists for
compatibility and should be treated as a **learning candidate**, not as v0.41
evidence-reviewed approval. v0.41 does not claim that every generic WALDO
corpus-ingest path can recognize and reject legacy candidate JSONL by schema.
That broader in-place CLI cutover should be made only with CI-backed edits to
the existing command surface.

So the claim is narrow and testable:

> v0.41 provides a deterministic, source-bound reviewed intake path from
> experience candidate to observed ground, while preserving rejected episodes
> as memory/context and performing no model-weight mutation itself.

## Verification

`go test ./...` will automatically include the new `cmd/waldo-mirror-review`
tests under the repository's normal CI. The tests cover approved promotion,
reviewed-to-ground provenance, mode `0600`, rejected-memory-only behavior,
aliased output refusal, and pre-mutation validation.

The experiment branch itself does not receive the normal `ci.yml` workflow
unless it is used in a pull request; therefore a branch-only commit is not a CI
pass claim.
