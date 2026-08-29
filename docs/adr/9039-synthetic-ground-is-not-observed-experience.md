# 9039: Synthetic starting ground is not observed experience

## Status

Accepted for the experimental AXM Mirror lane.

## Context

Training data is broader than a document corpus. A model may learn through
causal token prediction, supervised demonstrations, ranked preferences,
critiques and revisions, or reviewed action/outcome trajectories. Synthetic
examples can provide a useful starting distribution, while chat, tool, build,
and platform logs can later provide direct observations of behavior and
outcomes.

Calling all of these records "memory" hides an important distinction. Retrieval
memory changes what context is available at inference time. Gradient training
changes model weights. Synthetic curriculum is generated evidence; it is not a
real event merely because it is stored in a durable ledger.

## Decision

The experimental Mirror lane accepts strict
`axm.waldo.mirror-ground-record/v0.39` JSONL records with one of these source
classes:

- `SYNTHETIC_SEED` for deliberately generated demonstrations;
- `OBSERVED_CHAT` for authorized conversation projections with a source
  receipt; or
- `OBSERVED_EXECUTION_TRACE` for authorized action/tool/outcome projections
  with a source receipt.

Every synthetic seed names its generator and sets `synthetic: true`. It cannot
carry an observation timestamp or source receipt. Every observed record does
the reverse: it carries a timestamp and valid receipt, and cannot name a
synthetic generator.

The evidence signal and training disposition are also independent. Curated
synthetic examples and observed `HELPFUL` or `CORRECTED` examples may be
`POSITIVE_TARGET`. `HARMFUL` and `INCONCLUSIVE` observations may be retained as
`MEMORY_ONLY` but cannot carry positive training text. Unreviewed records are
`REVIEW_REQUIRED`.

`waldo mirror ground verify` checks these invariants, record hashes, IDs, and
source labels. `--positive-seed` additionally enforces a minimum curriculum
coverage floor. `--training-to` is an explicit mutation that writes only
positive targets as structured user/assistant JSONL for the normal WALDO
ingestion and SFT path.

Raw logs are not flattened or deleted by this decision. They stay in their
authorized sensory or experience store. Ground records are reviewable training
projections that retain a binding receipt back to the source trace.

## Consequences

- Synthetic data can provide a constructive start without being described as
  lived or observed experience.
- Platform and Codex logs have a later compatible route into the same training
  intake while their raw detail remains retained.
- Positive does not mean agreeable. The v0.39 seed includes false-premise,
  hidden-control, privacy, authority, failure, uncertainty, and dissent cases.
- A valid dataset, a written training projection, a real weight update, and a
  measured quality improvement remain separate claims.
