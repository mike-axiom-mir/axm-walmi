# v0.39 — honest positive starting ground

## Result

**PASS — v0.39 defines and verifies a constructive synthetic seed for the
connected AXM Mirror/WALDO stack, ingests it with synthetic provenance, and
uses it in real causal plus assistant-response PyTorch training.** It also
defines the compatible source classes for later authorized chat, platform,
tool, build, and Codex execution traces.

The central rule is:

```text
synthetic seed != observed experience != retrieval memory != weight training
```

All four can influence future behavior, but they do so through different
mechanisms and must remain visibly distinct.

## What counts as AI training data

Modern AI training commonly consumes several shapes of data:

| Training phase | Typical record | What it teaches |
| --- | --- | --- |
| Pretraining or continued pretraining | token sequences | statistical next-token patterns |
| Supervised fine-tuning | instruction and desired response demonstrations | how to respond to a task |
| Preference tuning | two or more candidate responses plus rankings | which behavior to prefer |
| Critique and revision tuning | initial response, critique, and improved response | how to identify and repair a bad answer |
| Agent/experience learning | state, action/tool result, outcome, and reviewed lesson | which trajectories worked, failed, or require correction |

The GPT-3 paper describes autoregressive pretraining and text-only in-context
demonstrations. InstructGPT uses human-written demonstrations followed by
ranked model outputs. Constitutional AI uses generated critiques, revisions,
and AI preferences. Self-Instruct proves that generated instruction examples
can be useful after filtering. ReAct demonstrates interleaved reasoning/action
trajectories with environment observations.

Synthetic data is therefore real training data, but synthetic does not mean
observed. Indiscriminate recursive training on model-generated content can lose
rare parts of the real distribution. The v0.39 contract preserves synthetic
labels and makes future genuine outcomes the reality anchor.

Primary references:

- [Language Models are Few-Shot Learners](https://arxiv.org/abs/2005.14165)
- [Training language models to follow instructions with human feedback](https://arxiv.org/abs/2203.02155)
- [Self-Instruct](https://arxiv.org/abs/2212.10560)
- [Constitutional AI](https://arxiv.org/abs/2212.08073)
- [ReAct](https://arxiv.org/abs/2210.03629)
- [Retrieval-Augmented Generation](https://arxiv.org/abs/2005.11401)
- [AI models collapse when trained on recursively generated data](https://www.nature.com/articles/s41586-024-07566-y)

## Seed curriculum

`positive-ground.jsonl` contains 32 sealed `SYNTHETIC_SEED` demonstrations:

- 8 visible roots;
- 4 examples per root;
- 32 explicit challenge kinds;
- 32 positive assistant targets; and
- 0 records falsely labeled as observed.

The roots are truth before story, constructive user agency, visible non-hidden
operation, honest uncertainty and repair, reality-anchored learning, consent
and privacy, evidence before authority, and respectful disagreement.

The hard prompts matter. The seed directly trains against fake certainty,
sycophancy, hidden learning, hidden control, privacy leakage, erased failures,
stale memory, capability-as-permission, and empty praise. Positive means
truthful help plus repair—not automatic agreement.

Regenerate and verify it:

```bash
go run ./research/v0.39-honest-positive-ground/generate \
  --output research/v0.39-honest-positive-ground/positive-ground.jsonl

waldo mirror ground verify \
  research/v0.39-honest-positive-ground/positive-ground.jsonl \
  --positive-seed
```

Create an explicit structured SFT projection:

```bash
waldo mirror ground verify \
  research/v0.39-honest-positive-ground/positive-ground.jsonl \
  --positive-seed \
  --training-to /private/path/positive-ground-training.jsonl
```

The projection is mode `0600`, contains only `POSITIVE_TARGET` records, and
retains data class, root, challenge, evidence signal, and source-record digest
as structured metadata for WALDO ingestion.

## Later platform and Codex logs

Authorized raw logs remain full sensory evidence. They are not designed away
or rewritten into cheerful demonstrations. A later projector can bind a
reviewed episode to its source receipt as `OBSERVED_CHAT` or
`OBSERVED_EXECUTION_TRACE`:

```text
prompt/state -> action/tool call -> tool/build output -> observed outcome
                                              -> correction/lesson
```

Helpful and corrected episodes can produce positive targets. Harmful and
inconclusive episodes remain negative or unresolved memory. Secret scanning and
access control occur before training intake without deleting the underlying
authorized provenance.

## Claim boundary

The public seed is a strong starting curriculum, not a claim that a tiny smoke
model has become a strong assistant. Dataset verification, ingestion, actual
gradient training, changed weight hashes, and measured quality remain separate
receipts.

## Recorded runtime proof

The 32 records became one verified structured-conversation shard with 2,132
reference tokens. A real CPU PyTorch run consumed 4,096 causal targets, followed
by a real assistant-response SFT run consuming 3,072 masked assistant targets.
The SFT stage started from weights
`f4e56f7f78b0669d9d7b5777c32f551290e6cda897fd2439b9493cd35942e193`
and produced
`e59deac2be35a6547742e99b04f2d5967caa9aa4e5d1d8474cec1c7286e6dfc8`.

That proves a real model mutation. A bounded chat probe from the 115K-parameter
smoke model was still unusable, so there is no quality claim. See
[REAL-RUNTIME-RESULT.md](REAL-RUNTIME-RESULT.md) and
[PUBLIC-RECEIPT.json](PUBLIC-RECEIPT.json).
