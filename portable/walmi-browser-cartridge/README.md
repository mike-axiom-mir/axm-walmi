# WALMI browser cartridge build v0.1

This build target packages the current WALDO / AXM Mirror research checkpoint into an offline mobile/browser cartridge without requiring Go on the target device.

## Included runtime

- `runtime/browser-entry.js` — the cartridge entry loaded by the single-file WALMI console.
- `runtime/go/walmi-axm-mirror.wasm` — Go `js/wasm` build that imports the real `internal/axmmirror` package.
- `runtime/go/wasm_exec.js` — copied from the exact Go toolchain used to build the WASM binary.
- full repository source archive for offline inspection/rebuild.
- license, notice, manifest and SHA-256 checksums.

The browser Go bridge exposes real deterministic AXM Mirror operations including capability census and the identity tool-experience / tool-memory lifecycle.

## Explicit neural hold

The current concrete WALDO PyTorch inference backend launches an external Python process. That backend is retained in source but is not treated as a valid mobile/browser neural runtime.

This cartridge therefore reports `HELD_BROWSER_NEURAL_ENGINE_NOT_BUNDLED` until a browser-native neural engine **and the intended WALMI model weights** are physically included in the cartridge. It does not call an internet model and does not generate a substitute neural reply.

## Deployment rule

Go, GitHub and build dependencies are build-machine concerns only. The target device receives precompiled runtime artifacts.

Target device requirements for this rung:

- modern browser with WebAssembly;
- enough memory/storage for the cartridge;
- the standalone WALMI single-file HTML console.

No runtime network dependency is introduced. Capability remains separate from authority.
