from pathlib import Path
import gzip
import hashlib

ROOT = Path(__file__).parent
CAPSULE = ROOT / "probe.py.gz"
OUTPUT = ROOT / "probe.py"
EXPECTED_SOURCE_SHA256 = "92d938959ed022ff5c6f66d01caa1c45a203b0bad2e2626ec01c3f8067974fd5"

data = gzip.decompress(CAPSULE.read_bytes())
actual = hashlib.sha256(data).hexdigest()
if actual != EXPECTED_SOURCE_SHA256:
    raise SystemExit(f"source SHA-256 mismatch: {actual} != {EXPECTED_SOURCE_SHA256}")
OUTPUT.write_bytes(data)
print(f"materialized {OUTPUT.name} sha256:{actual}")
