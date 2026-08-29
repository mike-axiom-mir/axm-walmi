# ADR 9044: Simulator play is observed experience, not hidden training

## Status

Experimental local Windows integration.

## Context

Theme Park, Factual Space, and Living City are three standalone AXM simulator repositories on the PC. WALMI needs to play them and candidate organs need the same test lane, while the simulator repositories remain independently reviewable and WALMI's experience store remains bounded enough for ordinary gaming hardware.

## Decision

1. Keep the three simulator repositories external and unchanged. A WALMI-owned adapter loads their public headless interfaces from configurable roots, defaulting to `D:\AXM_ACTIVE`.
2. Give every player the same request: one visible observation and one finite menu of action IDs. The player returns exactly one current action ID. Hidden world state, arbitrary native action payloads, and direct state-file writes are outside the player contract.
3. Support three player kinds: a deterministic contract player, a candidate organ process, and a named local WALDO model. Model play remains unavailable until local weights exist and produce a valid response.
4. Treat accepted, refused, invalid, and incomplete play as observed simulated experience. Simulator acceptance defaults to `INCONCLUSIVE`; it is not automatically labeled helpful or strategically correct. A refused or invalid selection becomes `HARMFUL` experience whose failed attempt is not supervised.
5. Append one bounded episode receipt to `simulator-direct-experience.jsonl` and one outcome-conditioned reflection to `training-projections.jsonl`. Projection does not invoke training or claim changed weights.
6. Cap each invocation at 64 turns. After verification, compact all per-run world files into one verified ZIP. The direct ledger and projection remain append-only, so file count grows by one archive per episode rather than by every simulator file and turn.
7. Factual Space must pass its native ledger replay. Theme Park and Living City validate native state on every load and step. Evidence from one simulator never grants authority in another.
8. Keep the old v0.25 research robot probe separate. Its Windows memory measurement is a compatibility repair, not one of the three playable worlds.
9. When a neural model or candidate organ violates the strict player response contract, append a typed `axm.walmi.simulator-capability-gap/v1` record. Preserve the failed response as unsupervised tool evidence, then permit one deterministic `deterministic-visible-action-selector/v1` repair chosen only from the already visible action menu. If the simulator accepts the repair, the supervised reflection is the exact strict repair JSON rather than the failed neural response or an abstract completion lesson. This proves response-contract compliance only, not strategic quality. The episode remains `HARMFUL` learning evidence and the repair grants no authority outside that run.
10. A named local model uses the bounded `axm.walmi.simulator-menu-index-codec/v1` adapter. The adapter exposes a fixed eight-slot alphabet selected deterministically from the visible menu. When a world exposes fewer than eight actions, later slots are deterministic aliases of those same visible actions; no hidden or fabricated action is introduced. UTF-8 labels are byte-bounded and the full prompt remains at most 480 bytes. The adapter accepts exactly one digit with deterministic decoding controls. The digit is translated to the existing strict JSON player response; the adapter does not choose the digit. Failed digits retain their output hash and byte count without copying arbitrary model text into the gap ledger. Accepted or repaired digits are the supervised contract lesson. Candidate organs and deterministic players retain the strict JSON contract directly.

## Consequences

WALMI and candidate organs can now exercise the same local world boundary without merging the worlds into WALMI. Episodes are portable and replay evidence remains available inside their archives, but raw file count stays bounded.

Capability gaps have a separate append-only ledger. This lets the deterministic side gain a bounded repair while the neural side receives an outcome-conditioned lesson about the original contract failure; neither side can relabel the failure as successful neural play.

The bridge can create training-ready reflections from experience, but it does not train automatically during play. Later autolearn policy still owns review, thresholds, checkpoints, rollback, and weight mutation.

## Boundary

`world accepted action != good strategy`

`simulated experience != human experience`

`experience projection != training invoked != weights changed`

`player action choice != filesystem authority != promotion != CANON`
