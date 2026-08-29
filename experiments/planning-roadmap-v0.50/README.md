# WALDO v0.50 — visible adaptive planning roadmap

Status: **EXPERIMENTAL / TEST**

v0.50 adds a provider-neutral planning membrane for WALDO's neural generation path without turning private chain-of-thought into an artifact or making every request pay for deep planning.

The deterministic assessment selects one of four public planning depths:

- `DIRECT` — no separate planning call; normal conversation/simple work continues directly.
- `BRIEF` — one bounded public plan call, at most 4 milestones.
- `ROADMAP` — one bounded public plan call, at most 10 milestones.
- `DEEP` — two bounded public calls: proposal, then an explicit visible revision; both proposal digests remain in lineage so the revision is not silent.

Depth is derived from declared task kind, consequence, uncertainty, estimated steps, artifact count, external effects, irreversibility, and capability state. A caller may explicitly request a mode. `DEEP` therefore means a richer **visible planning artifact**, not hidden reasoning.

The neural output must use `axm.waldo.visible-plan-proposal/v0.50`. Fields named `analysis`, `reasoning`, `chain_of_thought`, `thoughts`, `scratchpad`, or equivalent hidden-reasoning surfaces are refused. Plans remain candidates and grant no execution, permission, promotion, merge, CANON, or world-action authority.

The deterministic compiler turns the accepted public plan into `axm.waldo.visible-roadmap/v0.50`:

- milestone dependencies are validated and cycles are refused;
- required capabilities are checked through the already-vendored Capability Gap Hand;
- every acceptance claim receives an `UNTESTED` evidence route through the already-vendored Evidence Router Hand;
- missing required capabilities hold the roadmap rather than being silently installed or quality-reduced;
- progress is receipt-driven, dependency-ordered, and evidence-backed before a milestone can become `DONE`;
- creation roadmaps emit a candidate handoff targeting the existing `bounded-creation-program-planner-v1`.

Hermes is **optional continuity/queue infrastructure**, not a second planner. The membrane can emit Hermes-compatible `/queue` payloads for each roadmap milestone using Hermes' existing proposal-only contract. It performs no HTTP call and has no runtime dependency on Hermes; a host can choose to submit those payloads under Hermes' own consent gate.

This makes the intended default architecture explicit: Mirror/WALDO can remain the neural reasoning pair, while deterministic fabrics decide planning depth, validate the public plan, expose missing capabilities/evidence, and turn the plan into a durable creation roadmap.
