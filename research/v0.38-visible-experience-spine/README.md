# v0.38 — visible experience spine

## Result

**PASS — the experimental Mirror/WALDO stack can now retain a complete interaction as experience, expose a completed lesson to later local neural reasoning, project useful outcomes into real WALDO training, and hand the same episode to Hermes as portable memory.**

This is learning by experience in an explicit machine form:

```text
SENSED → REACTED → OUTCOME_OBSERVED → REFLECTED
                                      ├── future Mirror context
                                      ├── WALDO training projection
                                      └── Hermes memory projection
```

The event chain records raw local context in a private `0600` JSONL ledger. Each event binds the previous event hash. A completed lesson is therefore connected to the prompt, reaction, feedback, and correction that produced it rather than becoming an unexplained memory fragment.

## Commands

Start an episode while resolving through Mirror:

```bash
waldo mirror reason request.json \
  --experience-ledger private-experience.jsonl
```

Unresolved reasoning can also use completed experience with the existing explicit neural path:

```bash
waldo mirror reason request.json \
  --neural --model local-model \
  --experience-ledger private-experience.jsonl
```

Close the episode after observing what happened:

```bash
waldo mirror experience observe outcome.json \
  --ledger private-experience.jsonl \
  --learn-to private-training.jsonl \
  --hermes-to private-hermes-memory.jsonl
```

Outcome signals have different consequences:

| Signal | Future Mirror context | WALDO positive training target | Hermes episodic memory |
| --- | --- | --- | --- |
| `HELPFUL` | yes | the observed reaction | yes |
| `CORRECTED` | yes | the explicit correction | yes |
| `HARMFUL` | yes, as a negative lesson | no | yes |
| `INCONCLUSIVE` | yes, preserving uncertainty | no | yes |

## Hermes boundary

Hermes remains a separate AXM runtime lane. v0.38 writes a strict `axm.waldo.hermes-memory-capsule/v0.38` record containing the episode, outcome, lesson, and reflection provenance. This is the compatible handoff for Hermes's longer-lived memory layer.

WALDO did **not** import or launch the Hermes runtime in this experiment. A memory projection is not silently described as an actual Hermes runtime mutation.

## Real proof

The retained helpful episode was exposed to a later unresolved request through a visible experience capsule. The real local `pytorch-smoke` model received that capsule through the existing inference session and returned a 32-token candidate.

The tiny byte model's text was nonsensical. It remains an open, unapproved reaction and was not used as a positive training target. The runtime connection is proven; answer quality is not.

The helpful deterministic episode was then ingested as one private user-data record and used in real CPU PyTorch continuation training:

- retained records: `1`
- reference tokens: `37`
- training steps: `12`
- consumed training tokens: `192`
- simulated: `false`
- initial weights: `1881b144beaba8f31ae2b6543e10caaea0560f9435c7c86b9f1e93a425b66d7b`
- result weights: `a5a4a3eacd3b205ba4703fc6c789c6465c2ce3ee16f2079efbc176da4d2a0cbf`

Different weight hashes prove a real model update. They do not prove improvement.

See [REAL-RUNTIME-RESULT.md](REAL-RUNTIME-RESULT.md) and [PUBLIC-RECEIPT.json](PUBLIC-RECEIPT.json) for the hash-only public evidence.

## Boundaries retained

- Stable deterministic Mirror reasoning still never opens a neural model.
- Retrieved experience is visible prompt context, not hidden authority.
- Experience retention, WALDO training projection, actual weight training, Hermes memory projection, Hermes runtime mutation, and identity mutation are distinct.
- Raw chat, feedback, reactions, and memories remain private local files.
- No merge, promotion, execution authority, identity mutation, or CANON change occurs.
