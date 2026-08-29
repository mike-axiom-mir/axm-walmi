from pathlib import Path
import gzip
import hashlib

ROOT = Path(__file__).parent
CAPSULE = ROOT / "probe.py.gz"
OUTPUT = ROOT / "probe.py"
EXPECTED_SOURCE_SHA256 = "953eef23ea669badfa71c148678ee8eb814ad08a79f7171b037b825833d3daad"

data = gzip.decompress(CAPSULE.read_bytes())
actual = hashlib.sha256(data).hexdigest()
if actual != EXPECTED_SOURCE_SHA256:
    raise SystemExit(f"source SHA-256 mismatch: {actual} != {EXPECTED_SOURCE_SHA256}")
OUTPUT.write_bytes(data)
print(f"materialized {OUTPUT.name} sha256:{actual}")
