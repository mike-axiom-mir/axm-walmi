# AXM / WALDO v0.11 — Organ Fabric runtime bridge

Status: `EXPERIMENTAL / PUBLIC AXM RESEARCH / downstream WALDO fork only`

v0.11 repeats the exact frozen v0.10 `ONE_CONCEPT_MANY_BODIES` challenge against a newer AXM donor packet containing three deliberately separate source sets:

- Code Capability Fabric v0.6;
- Deterministic Organ Fabric v1;
- Organ Archive snapshot data.

The source packet SHA-256 is:

`8f4688db3abcba988011f2a2f36f172a09c3d848e57a2da54db90abd2669e634`

The frozen concept SHA-256 remains:

`sha256:9779b14236b2717ce1e8694adbacfd95c07d3afbacea10d3e4255cb52950856a`

## Two-stage evidence history

The packet was first inspected statically under its original no-execute/no-publish boundary. A later explicit human authorization allowed public publication and a disposable runtime experiment. The earlier bytes are not rewritten.

The runtime experiment first ran the donor's focused Organ Fabric tests. The observed total was 151 passing checks. It then sent four already-bounded proposal envelopes through the real proposal adapter and normal deterministic Organ Fabric gates:

1. `evidence-trail-mini-game` — `EXTERNAL` proposal source;
2. `continuity-diagnostic-tool` — `EXTERNAL` proposal source;
3. `lineage-how-to-guide` — `EXTERNAL` proposal source;
4. `tool-episode-quarantine-body` — explicit `AI` proposal source.

The harness did **not** call a live AI provider. `AI` identifies the source class of the fourth supplied proposal, not a hidden model dependency.

## Observed result

Every proposal produced all three declared strategies:

`lean / balanced / guarded`

Observed totals:

- proposals: **4**
- generated candidates: **12**
- generation failures: **0**
- package-verification failures: **0**
- winner selections: **0**

All candidate packages were regenerated for deterministic replay and verified. Scores remained advisory. Ties were preserved rather than silently broken.

All twelve verified packages were then written to a new disposable Organ Archive. Archive verification reported 12 objects / 12 events / 0 failures. Reinserting the first package returned `DEDUPLICATED` and did not add another event. The archive was exported and imported into a second fresh archive; the same 12-object index identity verified after the round trip.

Compact runtime receipt:

`sha256:0e110c9e0c56c72b52410db87a4ec019de140a31face79d7e7023f286d2283ec`

WALDO/Mirror independent witness digest:

`2235bcfbdc1beefd4b3ec3835e1ad96eaf50433f58a393e3c1d102ad0c644db9`

## Truth boundary

Observed here: the supplied packet checksum verified; donor focused tests executed; Organ runtime executed in a disposable test root; 12 candidates were generated and package-verified; archive dedup and export/import round trip verified.

Not observed or not performed: live AI-provider invocation, automatic winner selection, installation, registration, staging, promotion, persistent-learning admission, platform-main integration, or CANON change.

The source packet names local AXM review commits/branches that were not reachable on GitHub at publication time. v0.11 therefore preserves the exact packet bytes and source identities rather than pretending those local commits were already public history.
