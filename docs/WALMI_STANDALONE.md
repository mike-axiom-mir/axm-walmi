# WALMI standalone boundary

This repository separates the WALMI product experiment from the historical
OpenWALDO / AXM Mirror integration worktree without linking back to it. Required
implementation files are copied in full and sealed by hash.

## Three state classes

1. **Source state** is the code, schemas, tests, docs, and packaging logic in
   Git. It is portable and reviewable.
2. **Private instance state** is one WALMI's identity, direct experience,
   weights, conversations, simulator saves, and local settings. It stays on the
   owner's device and must never be committed by default.
3. **Shared state** is an optional public projection: signed discoveries,
   verified capability gaps, shared-world progress, and intentionally published
   artifacts. A private record is not automatically a shared record.

The blank fixture in `starter-state/shared/SHARED-STATE.json` defines only the
portable boundary. A future network synchronizer must preserve append-only event
identity, deterministic reduction, conflict evidence, and explicit publication
authority. Browser IndexedDB alone is device-local; global progress requires an
actual shared transport or exchanged signed state packs.

## Source and runtime

The repository carries complete source rather than submodules, junctions, or
runtime links. Installed Python, Node, Go caches, generated PC hosts, model
weights, and evolving WALMI instance state are deliberately absent. Packaging
tools recreate runtime products from the source boundary.

`WALMI-SOURCE-PROVENANCE.json` records the integration worktree snapshot that
was copied. `WALMI-SOURCE-MANIFEST.json` records the final standalone tree.
Neither document claims that a learned model or private state is present.

## Porting

Git keeps individual files because they remain inspectable and diffable. For
transport to a website, another PC, or a device build machine, run:

```text
python tools/seal_source_tree.py --verify
python tools/build_source_bundle.py
```

The resulting ZIP is one file with deterministic member ordering, fixed ZIP
timestamps, and manifest verification before packaging. A target platform can
unpack and build it without access to the historical experiment repository.

## Windows verification

`tools/test_walmi_windows.ps1` is the native Windows lane. It tests the WALMI,
Mirror, CLI, model, training, composition, and workshop integration packages;
builds `cmd/walmi-browser-wasm` under `GOOS=js GOARCH=wasm`; runs the portable
Python suites; and verifies the source manifest.

Do not interpret a raw Windows `go test ./...` as the portable acceptance test.
That command mixes a browser-only `syscall/js` command with upstream tests that
deliberately exercise POSIX mode bits, executable shell fixtures, symlinks, and
optional tools such as `cosign` and `llama-quantize`. Those remain useful in
their intended lanes but are not native Windows prerequisites for WALMI.

## Evidence language

Use narrow claims. A valid action proves codec and simulator acceptance. A
changed weight hash proves a different checkpoint. Resume digests prove save
continuity. Strategy quality, general reasoning, safety, and beneficial
self-modification require separate evidence.
