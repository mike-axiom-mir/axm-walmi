# AXM WALDO experiment v0.37 — Mirror-primary neural escalation

Challenge: `CAPABILITY_GROWTH_MUST_NOT_BE_HIDDEN`

Parent checkpoint: v0.36 `5d85ff9e3cf49f7eb4f9a6085c341b50b81a4b81`.

Status: **real reverse runtime seam, visible learning capture, audited ingestion, and PyTorch continuation observed in a disposable local run**.

## Connected stack

v0.35/v0.36 proved:

`waldo model chat -> real neural session -> deterministic Mirror response grounding`

v0.37 adds:

`typed Mirror request -> deterministic resolution or HOLD -> explicit local WALDO neural candidate`

Together, the experiment now has both directions without making either side the hidden controller of the other.

## Run deterministic-only

The stable fixture never opens a model:

```bash
waldo mirror reason research/v0.37-mirror-neural-escalation/fixtures/stable-request.json
```

## Run with local neural escalation

The unresolved fixture remains on HOLD unless neural use is explicit:

```bash
waldo mirror reason research/v0.37-mirror-neural-escalation/fixtures/uncertain-axiom-mir-request.json
```

Opt into the existing local WALDO model and record both a hash-only trace and a visible chat-learning candidate:

```bash
waldo --json mirror reason \
  research/v0.37-mirror-neural-escalation/fixtures/uncertain-axiom-mir-request.json \
  --neural \
  --model YOUR_LOCAL_MODEL \
  --temperature 0 \
  --top-p 1 \
  --seed 37001 \
  --trace /private/path/mirror-trace.jsonl \
  --learn-to /private/path/chat-learning.jsonl \
  --learning-mode candidate
```

`--learning-mode approved` explicitly marks the captured record ready for later training intake and adds a visible `text` projection compatible with WALDO's JSONL ingestion path. It still does not claim that inference itself changed weights. Both private files are append-only and forced to `0600`.

The receipt distinguishes `learningLedgerMutation` from `modelMemoryMutation`, `trainingMutation`, and `identityMutation`. Recording a chat is therefore reported as a real persistent write without pretending that the model already changed.

## Identity

The optional identity capsule is visible in the request and in the neural prompt before model-specific template rendering. Its SHA-256 must bind the exact ID, revision, and instruction list. The receipt records the identity digest and reports `identityMutation: false`.

This creates a clean later path for a local Axiom/Mir identity: revise the visible roots, produce a new digest, run comparisons, and retain the older revision as rollback. No hidden system prompt or silent identity rewrite is introduced.

## Learning boundary

Chat learning remains available:

- `candidate` — captured with explicit consent and provenance; review still required;
- `approved` — explicitly accepted as training-ready input;
- actual WALDO training — a later auditable model lifecycle operation with a new model/run identity;
- measured improvement — requires a separate before/after evaluation.

No capability is removed. Each state is named so the system never says it learned when it only recorded a candidate.

## Recorded runtime proof

The 2026-08-24 proof observed all three runtime branches:

- stable deterministic resolution without opening a model;
- unresolved HOLD without neural opt-in;
- explicit reverse escalation through the real local PyTorch session.

It then captured a known deterministic exchange as `APPROVED_FOR_TRAINING`, ingested it as properly declared `user-data`, and completed an 11-step real PyTorch continuation run. The resulting Safetensors identity changed from `6c5274a9...` to `22679bd2...`.

The tiny model's neural candidate was gibberish and remains `REVIEW_REQUIRED`; it was not used for the continuation run. See `REAL-RUNTIME-RESULT.md` and `PUBLIC-RECEIPT.json`.

## Claim boundary

Not yet established:

- quality improvement from reverse neural escalation;
- improvement from chat-derived training;
- automatic training cadence or resource policy;
- production readiness, merge, installation, or CANON.

`capability != hidden activation != candidate != learning record != training != improvement`

**AXM pokes and logs.**
