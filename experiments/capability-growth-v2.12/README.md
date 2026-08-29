# WALDO v0.43 — portable capability-growth graft

Status: **EXPERIMENTAL / TEST**

v0.43 grows the standalone WALDO body instead of linking it back to AXM. The model core remains replaceable; capability lives in local, inspectable modules, contracts and deterministic helpers that travel with this repository.

“Transformer-style” is an architecture metaphor here: composable capability blocks around a replaceable model core. This experiment does not modify transformer weights or claim a new neural architecture.

Added now: Agent Tool Forge, provider-independent Identity Shell Fabric, Hand Verification Lab, the exact platform Module Contract Verifier, and the Knowledge Canvas core needed by the already-vendored Evidence Desk.

The eight Keel example files inside Identity Shell Fabric are retained only because the adversarial verifier uses them as fixtures. They are not selected as WALDO identity, memory, authority, or runtime configuration.

The Discovery Engine was inspected but not grafted yet: its product contract still names AXM-specific provider/storage/gate routes and helper scripts reach into Mirror/Shared Controls. It is a candidate for a later neutral-host adapter rather than a hidden runtime dependency.

No new runtime dependency on the AXM repository is introduced. AXM commit `386736aaae3993089dfaf970cf2360894959e3c0` is provenance only; selected source files are copied into WALDO.

The uploaded Capability Growth showcase contained 1,395 files. 1,156 were already byte-identical to v0.42. Of 239 unique files, 93 closed portable capability/fixture files are grafted here; environment-facing seams, public-index generators, steward history, and Discovery Engine are left out intentionally.

Capability is not authority. No automatic execution, network, install, merge, release, CANON, identity acceptance, provider control, deployment, or model-weight mutation is granted. v0.41 evidence review remains the promotion boundary toward training material.
