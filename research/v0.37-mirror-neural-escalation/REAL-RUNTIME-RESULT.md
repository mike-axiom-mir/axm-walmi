# v0.37 real reverse-stack result

Run date: 2026-08-24

Status: `COMPLETED_WITH_NEGATIVE_QUALITY_RESULT_RETAINED`

Challenge: `CAPABILITY_GROWTH_MUST_NOT_BE_HIDDEN`

## What is connected

The experimental stack now has both real runtime directions:

1. WALDO neural-primary chat can pass through deterministic Mirror response grounding (v0.35/v0.36).
2. Mirror deterministic-primary reasoning can explicitly recruit the existing local WALDO model as a non-authoritative candidate (v0.37).
3. A visible chat-learning record can be marked review-required or approved, ingested with user-data provenance, and consumed by WALDO's existing real training lifecycle.

## Observed proof

| Stage | Observation |
|---|---|
| Stable Mirror request | `DETERMINISTIC_RESOLVED`; neural model not opened |
| Unresolved without opt-in | `HOLD_NEURAL_OPT_IN_REQUIRED`; neural model not called |
| Unresolved with opt-in | `NEURAL_CANDIDATE`; real `pytorch-smoke` session called |
| Identity context | Visible Axiom/Mir roots bound by SHA-256 before model template rendering |
| Neural learning capture | Written as `REVIEW_REQUIRED`; ledger mutation reported |
| Approved learning capture | Known deterministic exchange marked `APPROVED_FOR_TRAINING` |
| Ingestion | 1 user-data record, 33 reference tokens, 1 verified Parquet shard |
| Real training | PyTorch `2.8.0+cpu`, 11 steps, 176 consumed tokens, simulated `false` |
| Weight identity | Changed from `6c5274a9...` to `22679bd2...` |

## Negative result retained

The first bounded neural call (temperature `0`, eight-token limit) produced only trim-removable output. The command returned `HOLD_NEURAL_ESCALATION_FAILED`; it did not create a learning record.

A second sampled call produced a non-empty but unusable byte-model candidate. Its hash is retained, its raw text remains private, and its learning state is `REVIEW_REQUIRED`. It was deliberately not approved or trained. This proves connectivity, not reasoning quality.

## Learning truth

The disposable continuation run genuinely changed model weights, so a real learning path is observed. It does not prove that the model became better. One approved deterministic boundary exchange and a tiny smoke model cannot support that claim.

Visible states remain separate:

- learning-ledger mutation: yes when a record is appended;
- model-memory mutation during inference: no;
- training mutation: yes only in the later explicit continuation run;
- identity mutation: no;
- quality improvement: not measured.

## Key identities

- Runtime binary SHA-256: `9af0d086faa71ebc91b098830d2832096eba8ca3fad35cf0ab366a5487dc9baa`
- Identity capsule SHA-256: `a4a48bd2fe11943cf95bca18fcf246778d248bf07dd085c2175ea6b26766a68c`
- Neural candidate SHA-256: `ab4eb7bfc9f1cd7bee70939af0a801031c7a014f5d0c1593956aeba4f2fe63ad`
- Approved learning record SHA-256: `d3ba7be6d66aba9674ad346d769531183cb94bc7ccfc87ea78a7fea4de51f9cd`
- Ingest plan identity: `dc41b8466e73d568ce8c086e92d113bd14dd963e8f1cdff7f7e30236ea6216e2`
- Training run: `7d1bda8f70a03178`
- Training run BOM SHA-256: `cb64925f23f60579c12c9a69eba5a2ff6d1f0d5650a9e6d9bb7f989c84432b66`
- Initial weights SHA-256: `6c5274a9ab8e90a4d3204326dc6d043894be3462076211693c5c1dad5c98df41`
- Result weights SHA-256: `22679bd2d5050181e14be2d951821505a4aa3a204fc03732fd2eb518bcd01eba`

Raw chat prompts, candidates, and learning records are private and are not included in the public receipt.
