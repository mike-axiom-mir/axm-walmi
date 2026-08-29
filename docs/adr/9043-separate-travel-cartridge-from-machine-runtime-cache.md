# ADR 9043: Separate the travel cartridge from the machine runtime cache

- Status: experimental
- Date: 2026-08-29

## Context

The standalone WALMI Windows host contains more than twenty-three thousand
files because it bundles Python, PyTorch, Node, native binaries, and their
supporting data. That is useful offline, but copying so many small files is
slow and fragile across removable media and machines. WALMI's personal
experience must remain portable without treating the replaceable runtime as
personal memory.

## Decision

Distribute the Windows experiment as a few-file travel cartridge. The
immutable host is one SHA-256-bound runtime ZIP. On first launch, the cartridge
validates and extracts that runtime into a content-addressed cache on `D:`.
Later launches and other cartridges with the same runtime reuse the cache.

Personal models, experience, checkpoints, rollback data, configuration, and
caches live under `WALMI_DATA` beside the cartridge by default. The launcher
passes this location as `WALMI_DATA_HOME`; the runtime remains selected by
`WALMI_HOME`. Existing extracted hosts retain their old behavior because the
data home defaults to the runtime host when no override is supplied.

The cartridge reserves local-browser interface boundaries rather than making
the user experience part of the Windows engine. The intended AXM HTML pages
will enter through separate desktop and mobile boundaries when their actual
source is supplied. A Windows x64 home engine is bundled at this rung. Linux
may receive a full host pack later. Android and iOS are companion packs for
chat, intake, observations, and approvals against the user's PC home; they do
not carry or modify the full neural system. This is not a reduced user-facing
capability claim: the mobile page may expose the full WALMI collaboration
surface through the PC home while deep modification remains on that PC, like a
game client using a separate editor and build host.

The PC host includes a bounded device-pack assembler. It can seal a selected
desktop or mobile page, a selected platform engine, and an unchanged snapshot
of `WALMI_DATA` for a full desktop host into one hash-bound transport ZIP.
Mobile targets refuse the full `WALMI_DATA` tree and reserve only a bounded
companion-state/local-bridge contract. Assembly does not claim that the target
engine or bridge exists or runs. Native iOS finalization remains a macOS,
Xcode, and signing step; the PC can prepare its inputs but cannot truthfully
produce a signed native iPhone application by itself.

Future local-page integration reuses the AXM frontend-organ JSON intake that
the user will supply with the actual pages. Humans and WALMI submit through the
same page-owned intake. WALMI sits above the installed local pages and can
propose a JSON update, but the receiving page validates and applies its own
contract. This decision deliberately does not invent that schema from memory
or copy a different page implementation as a substitute.

The runtime still expands to ordinary files because compiled Python and neural
libraries cannot be claimed to execute safely from a ZIP without separate
compatibility evidence. The reduction applies to transport and repeated
machine preparation, not to the number of files required by the active native
runtime.

## Consequences

- Moving a clean WALMI build requires one outer ZIP or a handful of cartridge
  files instead of copying the expanded runtime tree.
- Personal growth remains separate from the disposable machine cache.
- Runtime extraction occurs once per runtime hash on each Windows machine.
- The current full engine pack is Windows x64 only. Linux requires its own
  verified full engine. Android and iOS require a verified local companion
  bridge and mobile page, not a copy of the PC brain.
- Packing changes portability and does not establish model-quality improvement.
