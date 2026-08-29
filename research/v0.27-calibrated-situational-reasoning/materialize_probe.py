from pathlib import Path
import gzip
import hashlib

ROOT = Path(__file__).parent
CAPSULE = ROOT / "probe.py.gz"
OUTPUT = ROOT / "probe.py"
EXPECTED_SOURCE_SHA256 = "11a011906b056bcb3a75f43ecd3354e763f9f4202eba2fee08a3025adba3c2bd"

data = gzip.decompress(CAPSULE.read_bytes())
actual = hashlib.sha256(data).hexdigest()
if actual != EXPECTED_SOURCE_SHA256:
    raise SystemExit(f"source SHA-256 mismatch: {actual} != {EXPECTED_SOURCE_SHA256}")
OUTPUT.write_bytes(data)
print(f"materialized {OUTPUT.name} sha256:{actual}")
