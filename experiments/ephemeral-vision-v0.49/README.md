# WALDO v0.49 — ephemeral visual buffer

Status: **EXPERIMENTAL / TEST**

v0.49 vendors the provider-neutral Ephemeral Vision Hand as a bounded sensory buffer. It does not create camera, screen-capture, browser, operating-system or network authority. A host must explicitly supply a `captureFrame` function; without one, construction fails.

Accepted frames are JPEG data URLs kept only inside bounded memory. Frame and byte ceilings evict oldest raw frames, metadata excludes image payloads, sealing produces a digest receipt rather than a raw-video archive, and cleanup wipes buffered data. Browser-only contact-sheet rendering remains optional and fails closed when no DOM exists.

This gives WALDO a portable way to reason over short visual sequences whenever a future host supplies consented frames, while keeping the sensory substrate separate from the model body.
