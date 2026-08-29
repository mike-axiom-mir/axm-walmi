# ADR 9045: Persistent paced simulator play is not a disposable probe

Status: accepted locally

## Context

The simulator bridge proved that a named player could choose a bounded visible action and that all three engines could preserve deterministic evidence. It did not provide developmental play. Each episode created its world underneath an ephemeral run directory; packaging then deleted that expanded state. Turns also ran as quickly as the machine allowed, and a multi-turn episode produced only one training projection.

That design is useful for contract probes but it cannot honestly represent a player returning to the same park, expedition, or life. A passing probe is not evidence of accumulated experience.

## Decision

Each simulator player owns explicit named save slots below the private WALMI data home. A slot contains authoritative simulator state plus bounded metadata; episode evidence remains append-only outside the slot. Reopening a slot resumes its existing seed and state. Creating a new slot never overwrites an old one.

A play session records wall-clock start, decision, action, and end times. Successive turns use an explicit minimum cadence. A zero cadence remains available only as a labelled test/probe setting.

Every observed turn becomes its own outcome-conditioned learning projection while the complete session remains one direct-experience episode. Failed model output stays evidence. A deterministic replacement action is optional and visibly labelled; normal WALMI play stops instead of silently attributing another player's action to WALMI.

Slot metadata keeps counters and only the latest session summary. It does not grow an unbounded duplicate history. The append-only experience ledgers remain the historical authority.

## Consequences

- WALMI can return to the same three worlds across processes and days.
- Simulator ZIP receipts no longer contain or delete the authoritative save.
- Long play can accumulate turn-level experience without making completion the learning gate.
- Wall-clock duration becomes auditable and cannot be confused with simulated clock minutes.
- Save round-trip and live paced-session evidence are required before claiming persistent play works.
- Theme Park and Factual Space retain their native save semantics. Living City gains named browser slots in addition to its existing autosave and JSON export/import.
