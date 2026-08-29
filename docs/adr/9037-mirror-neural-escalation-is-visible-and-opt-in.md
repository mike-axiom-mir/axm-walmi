# ADR 9037: Mirror neural escalation is visible and opt-in

## Status

Experimental runtime integration only.

## Context

v0.35 connected WALDO's real neural `model chat` path to deterministic AXM Mirror grounding at the response boundary. v0.36 then exercised that direction through a real local PyTorch model.

The reverse direction remained detached. Mirror could describe a deterministic-primary architecture with neural escalation for ambiguity, novelty, or conflict, but no observable shell command connected that decision to WALDO's existing local inference session.

AXM also intends to learn from chats and carry an Axiom/Mir identity locally. Removing those capabilities would make the connection incomplete. Hiding them would make the resulting growth uninspectable.

## Decision

1. Add an experimental `waldo mirror reason <request.json>` command.
2. Keep deterministic Mirror reasoning primary. A `STABLE` request returns its deterministic response without opening a model, even when `--neural` is present.
3. An `UNCERTAIN` or `CONFLICT` request remains on HOLD unless the invocation explicitly includes `--neural` and a local `--model`.
4. Open the existing raw `inference.Session` for reverse escalation. Do not route it through the v0.35 response wrapper, because Mirror already owns candidate status at this boundary and wrapping again would create a control loop.
5. Build the Mirror escalation capsule before `model.Interaction.Prompt` renders model-specific tokens. The local model therefore sees the typed grounding state and visible identity roots without rewriting an already-rendered prompt.
6. Treat the neural answer as a non-authoritative candidate. It cannot grant permission, execute, mutate identity, train weights, promote, or change CANON.
7. Permit an optional identity capsule whose digest must bind its complete visible instruction list. The capsule is context, not an identity mutation.
8. Permit visible chat learning through `--learn-to`. The private append-only JSONL record contains the raw prompt and selected response, source mode, consent, provenance hashes, and identity digest.
9. Support two declared learning states: `candidate` requires later review; `approved` marks the record ready for the normal auditable training intake. Neither mode silently trains during inference.
10. Keep the optional operational trace hash-only and append-only. Trace and learning files are forced to mode `0600`.
11. Report learning-ledger mutation separately from model-memory, weight-training, and identity mutation. Persisting a chat record must not be described as "no mutation."
12. Fail closed on unknown request fields, identity-digest mismatch, unresolved deterministic certainty, missing neural opt-in, missing model, invalid generation bounds, timeout, backend failure, or empty neural output.

## Consequences

The WALDO/Mirror experiment now has both runtime directions:

```text
WALDO neural primary -> deterministic Mirror response grounding
Mirror deterministic primary -> explicit local WALDO neural candidate
```

Chat-derived learning and visible Axiom/Mir identity context are retained rather than designed away. Their state is explicit: capture is not training, training-ready input is not a completed model update, and identity context is not identity mutation.

This connection does not prove that the neural candidate improves Mirror reasoning, that chat-derived training improves the model, or that any identity revision is good. Those require separate measured runs with before/after model identities and retained negative outcomes.

## Boundary

`visible identity context != identity mutation`

`chat capture != accepted training input != completed training != improvement`

`neural candidate != evidence != permission != execution != promotion != CANON`
