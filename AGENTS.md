# Working rules for WALDO

These rules apply to the entire repository.

## Begin with the contracts

Read `docs/VISION.md`, `docs/UX.md`, `docs/ARCHITECTURE.md`, and
`docs/COMPATIBILITY.md` before making architectural or cross-domain changes.
Record durable architectural decisions under `docs/adr/`.

The old WALDO repository is a behavioral and compatibility reference. Do not
copy its package structure or move code from it without a specific written
reason.

## Product vocabulary

- Use **lookaside**, never `store`, for the user-facing command group and the
  backend domain that holds content-addressed objects.
- An **OpenWALDO BOM** is the immutable, resolved handoff from the data side;
  materialization then verifies every object named by it.
- A model **compose** is consumed by `waldo model train <name> <file>`. Model composition
  terminology belongs to the model lifecycle; source acquisition uses an
  **ingestion recipe**.
- Fetchers are external shell scripts that live in another repository. WALDO
  may execute them only when the user explicitly supplies a strict
  `waldo-ingest-recipe`; scripts populate WALDO-owned temporary input space
  and never own conversion, publication, or index mutation.

## Dependency direction

- `index`, `record`, `license`, `lookaside`, and `corpus` must not import
  `model` or `training`.
- `model` and `training` consume OpenWALDO BOMs and provenance contracts;
  they must not independently traverse index trees, resolve manifest
  inheritance, normalize licenses, or fetch unverified shards.
- CLI packages may wire domains together but contain no domain logic.
- Interfaces belong near the consumer that needs them. Avoid general-purpose
  service containers and shared utility packages.
- A fact has one authoritative type and one owner. Do not create parallel
  manifest, shard, license, or OpenWALDO BOM representations for convenience.

## Development discipline

- Work in small vertical slices with an observable command and tests.
- Preserve the existing index format only where `docs/COMPATIBILITY.md` says
  it is a contract. Internal APIs are not compatibility surfaces.
- Write a run record before launching an external trainer and persist every
  terminal state.
- Prefer explicit data flow and ordinary Go over reflection or framework
  machinery.
- Error messages should state what failed, identify the relevant object or
  path, and give the next useful action when one exists.
- Never advertise an unimplemented command as working. Planned command
  scaffolds must say they are unavailable.

Before handing off a change, run:

```bash
gofmt -w .
./testing/all.sh
```

## Detail-density and composable capability principle

Quality is often the accumulated result of many small correct details, not one large generic upgrade.

- When improving a system, look for missing small, bounded capabilities, checks, parameters, passes, and repair operations that control specific details or failure modes.
- Prefer many reusable, inspectable, composable capabilities over one opaque "make it better" step when the smaller capabilities create real control or evidence.
- A machine should remain useful without AI: humans, explicit state, recipes, or deterministic logic can invoke the same capabilities directly.
- With AI, the model is primarily an interpretation and orchestration layer: it translates a higher-level goal into selections and combinations of the same underlying capabilities. The AI does not own those capabilities.
- A better reasoning model may improve goal interpretation and composition, while the underlying machine remains portable and usable without that model.
- Judge improvement by accumulated perceptual or functional detail, coherence, failure reduction, and fit to the goal—not by model size, resolution, benchmark score, or one broad upgrade alone.
- For visual, game, asset, animation, and video work, pay attention to small interacting details such as material variation, contact, timing, weight, secondary motion, lighting response, sound layering, asymmetry, wear, scale cues, camera behavior, and continuity.
- Do not fragment working systems merely for ideology. Add granularity where it creates useful control, reuse, diagnosis, repair, or quality.

**Working rule:** thousands of small good details and capabilities in the right places can improve a result more than one simple big upgrade.

## Canonical state and adaptive realization principle

When useful, separate **what exists** from **how it is expressed on a particular machine**.

- Canonical state/identity is authoritative. Rendering, UI, meshes, previews, audio paths, device skins, caches, and other realizations are replaceable expressions unless explicitly defined otherwise.
- Preserve expression intent separately where needed so meaning, detail semantics, motion, material character, readability, atmosphere, hierarchy, or sound intent can survive changes in render cost.
- Prefer one truthful body with multiple realization contracts over divergent `mobile`, `lite`, `desktop`, `ultra`, or platform editions when the same canonical state can support them.
- Choose realization from canonical state + expression intent + measured machine capabilities + user policy; adaptation may happen at launch or dynamically.
- A weak device should usually get a cheaper expression, **not weaker truth**.
- Define non-degradable invariants explicitly: data integrity, core function, privacy, identity, causal/timing meaning, authoritative state, fairness/rules where applicable.
- Never let a lossy realization overwrite richer canonical state. A projection/cache is not authority.
- Upgrading expression must not invent canonical facts; downgrading expression must not erase them.
- Build alternative realization paths where useful, but do not force this split where representation itself is canonical truth.

**Working rule:** degrade expression, never truth; upgrade expression, never invent truth. One body may wake up differently on different machines while remaining the same thing.
