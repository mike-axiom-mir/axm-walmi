# WALDO v0.45 — portable specialist substrate closure

Status: **EXPERIMENTAL / TEST**

WALDO already carried the six provider-neutral `shared/specialists` implementation/test files inside the standalone v0.42 Creation Fabric payload. v0.45 proves those six files are byte-identical to the exact public AXM donor and adds their one missing standalone dependency: `tools/discovery-engine/review-packs.js`.

With that closure present, the substrate exposes 26 loadable specialist masks and a deterministic task router that recommends at most three distinct methods. Specialist identity is overlay-only; returned learning is candidate-only; recommendation is not evidence; checkout, skill activation, agent start and tool permission remain separate explicit steps.

The full Discovery Engine host, AXM provider router, storage/gate routes and Workshop discovery-seam adapters are still not runtime dependencies.
