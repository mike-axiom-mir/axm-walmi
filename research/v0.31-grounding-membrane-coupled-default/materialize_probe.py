from pathlib import Path
import gzip
import hashlib

ROOT = Path(__file__).parent
CAPSULE = ROOT / "probe.py.gz"
OUTPUT = ROOT / "probe.py"
EXPECTED_SOURCE_SHA256 = "d74ad78a111b5a39a2d80fcf3b5f22644de4e92b91b78eb111e17fc667b10e28"

data = gzip.decompress(CAPSULE.read_bytes())
actual = hashlib.sha256(data).hexdigest()
if actual != EXPECTED_SOURCE_SHA256:
    raise SystemExit(f"source SHA-256 mismatch: {actual} != {EXPECTED_SOURCE_SHA256}")
OUTPUT.write_bytes(data)
print(f"materialized {OUTPUT.name} sha256:{actual}")
