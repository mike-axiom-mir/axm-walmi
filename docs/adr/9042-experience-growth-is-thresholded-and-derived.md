# ADR 9042: Experience growth is thresholded and derived

## Status

Experimental.

## Context

The direct experience ledger is historical evidence. Training needs a revisable
view over that history, but treating the view as the history would let later
importance, categorization, or training policy rewrite what actually happened.
Training on every individual write also creates noisy, expensive checkpoint
churn.

## Decision

The portable WALMI host keeps three distinct layers:

1. The append-only direct experience ledger records sensed, reacted, observed,
   and reflected events.
2. A derived projection ledger contains either observed positive response
   targets or outcome-conditioned reflections. Failed attempts remain tool
   evidence and only reflection lessons are supervised.
3. A threshold controller waits for a configurable number of new unique
   projections. At the threshold it deterministically normalizes the complete
   accumulated projection set into a structured conversation corpus, replaces
   that derived corpus snapshot, and trains the next checkpoint from the
   accumulated corpus.
4. Each threshold generation publishes that accumulated snapshot under a new
   logical corpus identity (`experience/walmi-generation-NNNNNN`). WALDO model
   stages use corpus identity as completed-work evidence, so reusing one path
   after its manifest changes would correctly skip it rather than prove new
   training. A failed cycle retains its generation number and can retry the
   same generation identity with `--update`.
5. The replaceable training view deterministically categorizes harmful
   `WALMI_MENU_INDEX_V1` contract-gap reflections and replays each four times.
   This bounded curriculum weighting changes neither the projection ledger nor
   direct experience. Generated replay rows have unique derived IDs, disclose
   their category and replay ordinal, and are replaced with the next snapshot.
   Replays after the first also carry a short, bounded `DERIVED_REPLAY=N/4`
   marker in unsupervised tool evidence. That makes the conversation contents
   distinct to WALDO's canonical deduplicator without changing the supervised
   one-digit assistant target.

The first growth cycle creates a clean 10M-preset conversational WALDO model.
Later cycles require the same immutable architecture and interaction contract.
The controller records processed projection digests only after a checkpoint is
successfully persisted, so a failed run remains retryable. Concurrent cycles
are refused by an exclusive local lock.

The default threshold is eight new projections. It is an explicit experimental
batching policy, not a claim that eight experiences are sufficient for useful
behavior.

## Consequence

The controller still retains every unique projection. Its first importance
policy is deliberately narrow: exact menu-index contract failures receive four
derived training rows so the small byte model can learn the transport boundary.
Broader importance, strategic scoring, and representative replay remain later
experiments and cannot relabel source experience.

Every growth receipt distinguishes `trainingInvoked`, `weightsChanged`, and
`qualityImproved`. A new artifact hash can establish weight mutation. Quality
remains unknown without held-out behavioral evidence. The controller reads the
persisted run BOM after training and refuses to advance its experience state if
WALDO retained fewer records than the disclosed derived training-row count.

## Boundary

`DIRECT_EXPERIENCE_IS_NOT_THE_TRAINING_VIEW`

Automatic batching does not grant network, install, promotion, merge, CANON,
workspace-write, or world-action authority.
