# ADR 9007: Give the WALDO specialist a bounded inner asset foundry

Status: experimental fork decision

## Context

The WALDO witness clone now has deterministic provenance, evaluation, sensory
receipt, skill-continuity, and dual discovery boundaries. It still has no
Workshop and no native way to turn a small visual intent into a real inspectable
asset candidate. The next experiment needs that capability without copying the
Workshop, importing its JavaScript, or hiding a general agent behind an asset
command.

The public AXM platform `main` snapshot at commit
`a4f99fbfc05268173458bf3fb8f3fe616919e376` contains the Asset Factory,
Asset Hands, target-canvas, sprite-atlas, visual-grammar, and visual-proof
contracts. Their exact paths and byte digests are pinned in
`research/asset-foundry-knowledge-sources-2026-08-15.json`.

Those sources support several distinctions worth preserving:

- an asset brief, target canvas, creation recipe, artifact set, and validation
  receipt are separate objects;
- a primary delivery artifact and a preview are different outputs;
- deterministic pixel composition is a real bounded hand, while a missing
  capability must stay visible;
- semantic shape and presentation style are different planes;
- technical structure checks do not establish appearance, meaning, usefulness,
  accessibility, originality, or human approval;
- a generated preview is a candidate, not automatic admission to a shared asset
  vocabulary, release, or CANON.

The full Asset Factory is intentionally broader than this first WALDO slice. It
can coordinate multiple hands and external generation capabilities. WALDO does
not receive those routes here.

## Decision

Add an ordinary-Go deterministic pixel compiler called the inner asset foundry.
It consumes one strict recipe and produces one portable ZIP-compatible
`.axmasset` candidate bundle. It performs no network call, model call, source
retrieval, external process execution, filesystem lookup, plugin routing, or
Workshop action.

The schemas are:

- `axm.waldo-witness.inner-asset-recipe/v0.1`;
- `axm.waldo-witness.inner-asset-grid/v0.1`;
- `axm.waldo-witness.inner-asset-candidate/v0.1`;
- `axm.waldo-witness.inner-asset-validation/v0.1`;
- the public-compatible metadata shape `axm.sprite-atlas/v1`.

### Recipe and compiler

A human or later bounded WALDO specialist may author a recipe. Authoring intent
does not author the artifact bytes or technical verdict: the compiler owns both
deterministically.

The first recipe language supports only:

- `icon`, `sprite`, `tile`, `effect`, and `ui` kinds;
- `pixel-8bit`, `pixel-16bit`, and bounded `pixel-custom` profiles;
- explicit canvas, palette, seed, frames, layers, and closed authority;
- `pixel`, `rect`, `line`, `ellipse`, and seeded `scatter` operations.

There are no scripts, prompts, paths, URLs, shaders, arbitrary code, external
hands, or opaque generator settings. Dimensions, frame counts, palettes,
operations, pixel count, file size, texture memory, and bundle expansion all
have hard bounds. Seeded scatter uses a compiler-owned stable PRNG so it does
not depend on Go's library RNG behavior.

### Semantic and presentation planes

Each layer declares `semantic` or `presentation`, plus `protected` and
`editable` flags. A protected layer cannot also be editable. A READY candidate
requires at least one protected semantic layer in every frame. This records the
shape that later style variants must not silently rewrite; it does not claim
that the shape is correct or meaningful.

### Portable candidate

The deterministic `.axmasset` contains exactly:

```text
candidate.json
validation.json
normalized-recipe.json
resolved-grid.json
asset.png
preview.png
atlas.json
```

The primary and preview PNGs use nearest-neighbor pixel scaling and remain
distinct artifacts. The normalized recipe and resolved colour-ID grid remain
editable evidence. The sprite-atlas metadata describes exact frame geometry.

ZIP entries are stored uncompressed, sorted by name, given a fixed timestamp,
limited to safe flat filenames, and capped at 16 MiB. Verification rejects
duplicate, unsafe, compressed, missing, or extra entries; rehashes every bound
artifact; and recompiles the normalized recipe to compare the exact candidate,
validation, and artifact bytes. A self-digest is still not a signature or an
author identity.

### Technical checks and visual review

Technical assessment preserves separate checks for:

- non-empty output;
- declared alpha behavior;
- primary and preview file budgets;
- texture-memory budget;
- protected semantic-core coverage.

A failing constraint produces a portable HOLD candidate rather than silently
discarding the evidence. A technical PASS proves only bounded deterministic
compilation and those declared checks. Every candidate remains:

- `visual_status: UNREVIEWED`;
- `candidate_only: true`;
- not human-approved;
- not canonical;
- not automatically promoted;
- closed for tool execution, training, promotion, CANON, and world action.

The synthetic witness-orb preview received one static visual inspection for
coherence and clipping during development. That observation is not encoded as
approval, animation verification, game fitness, or shared-vocabulary admission.

## CLI

The separate experimental CLI adds:

```text
waldo-axm-mirror forge-asset <inner-asset-recipe.json> <candidate.axmasset>
waldo-axm-mirror verify-asset <candidate.axmasset>
```

Writes are atomic and no-replace. A HOLD bundle is written for inspection before
the forge command returns a nonzero status.

## Compatibility

Strict decoding rejects duplicate keys, unknown fields, and trailing values.
The example recipe pins its normalized recipe, candidate, validation, and full
portable-bundle digests. Any format change therefore requires an explicit
schema/version decision rather than accidental drift.

## Authority boundary

This foundry does not:

- copy or emulate the full Workshop or Asset Factory;
- invoke an image model, external generator, browser, terminal, renderer, or
  Asset Hand;
- search for, ingest, install, publish, export, or promote assets;
- prove animation, appearance, meaning, originality, accessibility, usefulness,
  product fit, legal status, safety, or human preference;
- alter protected semantics automatically;
- train a clone, grant permissions, change CANON, or act in the world.

Mike remains the merge and CANON gate.

## Consequences

The experimental WALDO branch can now make a small real pixel asset from an
inspectable recipe, carry the editable sources and exact bytes together, and
reverify the whole candidate without the Workshop. A future specialist clone
can learn to propose recipes against this narrow substrate, while the ordinary
compiler and human visual gate remain outside its learned claims.

This is a foundation, not a complete asset workshop. Additional hands, format
conversion, asset-vocabulary admission, and richer visual proof require later
separate decisions.
