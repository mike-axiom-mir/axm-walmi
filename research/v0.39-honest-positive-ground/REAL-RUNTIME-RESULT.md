# v0.39 real positive-ground result

## Verdict

**PASS for an honestly labeled synthetic starting curriculum, normal WALDO
ingestion, and real local weight training. Model-quality improvement is not
established. No platform-log or Hermes-runtime consumption is claimed.**

## 1. Ground verification

The strict verifier accepted the generated dataset with the positive-seed
coverage floor enabled:

- records: `32`;
- training targets: `32`;
- synthetic records: `32`;
- observed records: `0`;
- roots: `8`, with `4` examples per root;
- challenge kinds: `32`;
- canonical dataset SHA-256:
  `ff41d52966997c5db46a3d34600830c083f52de4634b7641f8f34ac410e2ca0d`.

The explicit structured training projection contained 32 records, was written
mode `0600`, and had SHA-256
`b1dde0a3231fb6af8d31c3a42254fde12a22b166c776d063f9a37e7c4fb24d47`.
Writing the projection did not mutate model weights or identity.

## 2. Synthetic provenance and ingestion

The normal WALDO recipe declared the source category `synthetic`, generator
`OpenAI Codex`, version `work-session-2026-08-24`, machine-generated content
`yes`, personal data `no`, and Apache-2.0 project authorship.

The audited ingestion produced:

- plan identity:
  `d973ce99446007c248c15fc0743a6f089f8a673860fc35b1deabc74904bb9327`;
- retained structured conversations: `32`;
- reference tokens: `2,132`;
- canonical conversation shard:
  `1501d2a31e297e3bbe5d3bfb9a28e6358ceb451facddac7a522f6629c73b74dc`;
- embedded shard BOM:
  `89e1347fd5eb7ca545e47375e1ae8382dc2a655c91261f9dcca0068621930862`;
- training corpus BOM:
  `3eea3eacf59f0f61a15fca031d87d92f800f3132865f5c550155ef535e78007e`.

Object verification and a complete deep audit passed. The ingest privacy scan
reported zero email addresses, IP addresses, phone numbers, routing headers,
or credentials.

## 3. Real PyTorch training

WALDO created `axiom-mir-positive-ground-v039`, a deliberately tiny 115,200
parameter byte model with the `user-assistant-v1` interaction contract. The
CPU runtime was Python 3.12.13 with PyTorch 2.8.0+cpu; both runs reported
`simulated: false`.

The causal starting stage completed:

- run: `de322df6a5b2ffc2`;
- steps: `32`;
- consumed tokens: `4,096`;
- final training loss: `3.9793143272399902`;
- logical run BOM:
  `560ef9cac0733b86b97ca4d0b1528b7e8816fd0b79b4b1018c615ea920ce6ce8`;
- result weights:
  `f4e56f7f78b0669d9d7b5777c32f551290e6cda897fd2439b9493cd35942e193`.

The assistant-response SFT stage then completed with loss masked to assistant
content:

- run: `27de5caf8dea2e75`;
- steps: `32`;
- consumed assistant-target tokens: `3,072`;
- final training loss: `3.3933801651000977`;
- logical run BOM:
  `fea3a4244e8b88bacdacdcfb223b23869344adf350dab40ad9a5db5501aa41ea`;
- initialization weights:
  `f4e56f7f78b0669d9d7b5777c32f551290e6cda897fd2439b9493cd35942e193`;
- result weights:
  `e59deac2be35a6547742e99b04f2d5967caa9aa4e5d1d8474cec1c7286e6dfc8`.

The differing SFT initialization and result hashes prove a real gradient-driven
weight mutation. They do not prove semantic improvement.

## 4. Negative quality result retained

A deterministic chat probe returned no usable response after interaction
trimming. A sampled probe produced nonsensical text. This is expected from a
115K-parameter byte-level smoke model with a 64-token context and a tiny seed.

The result proves the curriculum, provenance, ingestion, objective, runtime,
and weight path. It does **not** make the smoke model suitable for real use. The
same corpus and contracts are intended to initialize or tune a capable local
base model later, followed by a separate held-out behavioral evaluation.

## 5. Platform and Codex trace boundary

The v0.39 schema can validate future `OBSERVED_CHAT` and
`OBSERVED_EXECUTION_TRACE` projections when they carry timestamps and source
receipts. No platform or Codex logs were available to this run, so none were
ingested or described as connected runtime experience.

Raw authorized logs should remain in the sensory/experience layer. Positive
targets are projections: helpful and corrected outcomes may train; harmful,
inconclusive, and unreviewed outcomes remain memory/evidence. This preserves the
full history without rewarding every action that happened to appear in a log.
