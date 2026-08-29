# ADR 9038: Experience is a visible shared spine

## Status

Experimental runtime integration only.

## Context

v0.37 connected deterministic Mirror reasoning, optional local WALDO neural candidates, visible identity grounding, and chat-derived training input. A chat record could therefore be retained and later trained, but the system still did not represent experience itself: what was sensed, how the stack reacted, what happened afterward, and what lesson should affect a later situation.

AXM's separate Hermes lane also provides longer-lived memory and learning capabilities. Rebuilding a second opaque memory system inside WALDO would duplicate ownership and lose the connection between a remembered lesson and the exact Mirror/WALDO episode that produced it.

## Decision

1. Add a private append-only experience ledger with four typed events: `SENSED`, `REACTED`, `OUTCOME_OBSERVED`, and `REFLECTED`.
2. Bind each episode with a per-episode SHA-256 chain. Raw chat, reaction, feedback, correction, and lesson remain visible to the local owner; operational receipts may remain hash-only.
3. Let `waldo mirror reason --experience-ledger` retrieve completed lessons for unresolved neural reasoning and append the new sensed/reaction pair. Stable deterministic reasoning still does not consult or open a neural model.
4. Let `waldo mirror experience observe` close one open episode with a declared `HELPFUL`, `CORRECTED`, `HARMFUL`, or `INCONCLUSIVE` outcome.
5. Retain all four outcomes as future context. Negative and inconclusive experience must not disappear merely because it is unsuitable as a positive training target.
6. Treat a helpful response or explicit correction as training-ready. Project it to ordinary WALDO JSONL intake only when the invocation supplies `--learn-to`. Recording or projecting the episode does not claim that model weights already changed.
7. Permit every completed episode to become a portable Hermes memory capsule through `--hermes-to`. The capsule carries raw visible context and provenance hashes. Writing it does not claim that the separate Hermes runtime is installed, running, or mutated.
8. Keep Mirror experience context, WALDO training projection, Hermes memory projection, actual Hermes runtime memory, model-weight mutation, and identity mutation as separate receipt fields.
9. Force experience, training, and Hermes projection files to mode `0600`.
10. Fail closed on unknown fields, broken event chains, duplicate or missing sequences, invalid outcome transitions, digest mismatch, or attempts to treat harmful/inconclusive reactions as positive training targets.

## Consequences

The experimental stack now shares one experience spine:

```text
chat / environment signal
        ↓
Mirror SENSED → WALDO/Mirror REACTED
        ↓
observed outcome → REFLECTED lesson
        ├── later Mirror context
        ├── WALDO training projection (helpful/corrected)
        └── Hermes episodic-memory projection (all outcomes)
```

This makes an interaction matter beyond a static chat transcript. It still does not prove that a lesson is correct, that retrieval improves the next response, that a training run improves model quality, or that Hermes accepted the memory. Those claims require their own runtime receipts and measured outcomes.

## Boundary

`experience retained != lesson correct`

`future context != hidden control`

`training projection != changed weights != improvement`

`Hermes memory capsule != Hermes runtime mutation`

`memory or learning != permission != execution != promotion != CANON`
