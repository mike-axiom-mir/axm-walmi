# v0.38 real runtime result

## Verdict

**PASS for the visible experience connection and real learning path. No model-quality claim. No live Hermes-runtime claim.**

## 1. Sense and reaction

The stable fixture produced `DETERMINISTIC_RESOLVED` without opening a model and appended two events to the private experience ledger:

- episode: `v038-helpful-experience`
- request SHA-256: `efa1baaa43508ab6b34da9144d5a2d9c5eb8538167bb76752dd143702cb61312`
- experience events: `SENSED`, `REACTED`
- experience-ledger mutation: `true`
- neural called: `false`

## 2. Outcome and reflection

The explicit `HELPFUL` outcome closed the episode:

- outcome event SHA-256: `b42090d24ea369d31aa6f0f75c01790b4437f0c521c1fb7fa1436fe651fefe91`
- reflection event SHA-256: `1b2cfcb89bd5e924ff06704b224d1a894cc2cfae922fa59e2849bddbe35cfa71`
- WALDO learning-record SHA-256: `bdbdebb4b2580dc48c8eb6edb775290af894db814e576da6bc9d95d417ec398e`
- Hermes memory-record SHA-256: `9834975ed45d313a0928a01cba6960cc0b95a7cf75f6359ac920aff9b498ab93`
- actual Hermes runtime mutation: `false`

The experience, WALDO projection, and Hermes projection files were separate append-only private files with mode `0600`.

## 3. Experience-informed real neural reaction

The unresolved follow-up used the completed episode as visible local prompt context:

- local model: `pytorch-smoke`
- backend: `pytorch`
- experience context applied: `true`
- retrieved episode: `v038-helpful-experience`
- experience context SHA-256: `e87066aae8dbc029799ad11db61ddf3d77408b26846cbd0dc5de3eb3329fada4`
- neural prompt SHA-256: `1cff663a68530479a6de4bbfad0e4c5d69091125d9d2c0f5d7544ffc57ec3076`
- neural candidate SHA-256: `69ecbc862d040eec494cbaf002a80ba329fd30d92154f7f0602891ffe6bc1d13`
- generated tokens: `32`
- finish reason: `max_tokens`

The raw tiny-model output was nonsensical and remains private. It was recorded only as a reaction in the open `v038-neural-reaction` episode. It was not projected into training.

## 4. Real WALDO ingestion and PyTorch learning

The training-ready helpful episode was ingested through the normal user-data provenance path:

- records retained: `1`
- reference tokens: `37`
- published shard SHA-256: `69daa9a8ccea7833dfb1ad4a3c9323171a73dc56cec4d9c9f75f4ba7e3f8be1b`
- license: `LicenseRef-AXM-User-Private`

The resulting corpus drove an actual CPU PyTorch continuation run:

- run ID: `9c08402a82c1b361`
- state: `complete`
- simulated: `false`
- backend: `pytorch@builtin-pytorch-worker-schema-1-r7`
- runtime: Python `3.12.13`, PyTorch `2.8.0+cpu`, CPU
- steps: `12`
- consumed tokens: `192`
- final loss: `5.034520149230957`
- logical run BOM SHA-256: `61d2d602d4b86d4f22440da31e1698c5de3e31428aa0733f8cf985fcc8d6dc22`
- initial weights SHA-256: `1881b144beaba8f31ae2b6543e10caaea0560f9435c7c86b9f1e93a425b66d7b`
- result weights SHA-256: `a5a4a3eacd3b205ba4703fc6c789c6465c2ce3ee16f2079efbc176da4d2a0cbf`

The live training stream emitted all 12 steps, checkpoint persistence, and terminal `complete`. The run record and artifact inventory confirmed the hashes afterward. Ears verdict: `PASS`; the rolling raw buffer was released.

## Interpretation

This proves that a chat-derived experience can affect the next neural context and can later change real local model weights after a helpful/corrected outcome. It also proves a portable Hermes memory handoff was created.

It does not prove that the next answer improved, that the retained lesson is universally correct, or that Hermes has already consumed the capsule.
