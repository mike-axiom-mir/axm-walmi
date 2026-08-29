#!/usr/bin/env python3
"""Materialize the exact v0.26 probe source from the committed gzip capsule."""
from pathlib import Path
import gzip, hashlib

ROOT = Path(__file__).parent
source = gzip.decompress((ROOT / "probe.py.gz").read_bytes())
want = "4c0c28385046bbbea19079fcc3333a6a4de6f1e01431c2fb24d09d7441fcdc8a"
got = hashlib.sha256(source).hexdigest()
if got != want:
    raise SystemExit(f"probe source digest mismatch: {got} != {want}")
(ROOT / "probe.py").write_bytes(source)
print(f"materialized probe.py sha256:{got}")
