#!/usr/bin/env python3
from pathlib import Path
import hashlib

EXPECTED = "8f4688db3abcba988011f2a2f36f172a09c3d848e57a2da54db90abd2669e634"
root = Path(__file__).resolve().parent
parts = sorted((root / "source-packet-parts").glob("part-*.bin"))
if len(parts) not in (4, 16):
    raise SystemExit(f"expected 4 or 16 parts, found {len(parts)}")
out = root / "axm-fabrics-current-waldo-2026-08-23.zip"
h = hashlib.sha256()
with out.open("wb") as target:
    for part in parts:
        data = part.read_bytes()
        target.write(data)
        h.update(data)
actual = h.hexdigest()
if actual != EXPECTED:
    out.unlink(missing_ok=True)
    raise SystemExit(f"SHA-256 mismatch: {actual}")
print(f"OK {out.name} sha256:{actual}")
