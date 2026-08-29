# WALMI v0.2 — Standalone Workspace + Asset Body

Status: experimental packaging branch.

## Invariant

WALMI must boot and retain its local body without AXM Workshop and without a network connection. Workshop may donate newer capabilities later, but it is not a runtime dependency.

## Workspace Hand

`walmi_workspace_hand.py` attaches one explicit local directory and exposes bounded `status`, `tree`, `read`, `stat`, `hash`, and `apply` operations.

`apply` accepts only hash-bound create/update/delete candidates. It preflights the whole transaction, refuses absolute/unclean paths, `.git`, symlink traversal and stale expected hashes, writes atomically, and rolls back previously applied operations when a later operation fails. The hand carries supplied bytes; it does not claim authorship, install software, publish, promote, mutate CANON, or use the network.

No workspace is required at boot. A missing attachment is a normal capability absence, not a WALMI startup failure.

## Asset body

The Windows cartridge carries WALMI's existing inner-asset machinery plus a pinned copy of the AXM asset capability line from `mike-axiom-mir/axm-collaboration-platform` commit `fd6ec98a6a98a6666a980c359730ccec57a8cbe9`:

- Asset Hands v2.5.0
- Asset Fabric v0.12.0
- Visual Kernel
- Visual FX
- shared output support and jsPDF runtime
- Spatial Studio core/geometry dependencies
- Film & Motion Studio core dependency

A bundled Node runtime drives `walmi_asset_hand_runner.js`, so the shared hands are executable locally rather than merely archived source. The package smoke test loads the registry, verifies the pinned versions and dependency closure, and requires at least 30 registered hands before the cartridge is accepted.

Native DCC/engine/tool bridges remain optional. If Blender, Godot, Unity, Unreal, FreeCAD, FFmpeg or another external substrate is absent, the relevant hand must report the missing capability rather than making the whole WALMI runtime dependent on that program.

## Truth boundaries

- Bundled capability is not execution authority.
- Asset results remain candidates.
- Workspace mutation is limited to the explicitly attached root.
- Git metadata mutation is refused by the Workspace Hand.
- No internet fallback is introduced.
- The model-weight scan reports whether starting neural weights are actually present; packaging does not fabricate them.
- A generic PyTorch package smoke verifies that the bundled runtime can change and save weights.
- A separate isolated autolearn smoke starts with one observed incomplete trajectory, ingests its failed attempt as unsupervised tool evidence, supervises only the retained lesson, runs two optimizer steps, persists a fresh conversational checkpoint, reopens it for inference, and refuses to retrain when no new projection exists.
- Those receipts prove the experience-to-weights mechanism and its non-completion path. They do not prove useful language behavior, generalization, reasoning growth, or quality improvement; the one-record smoke has no held-out evaluation set.
