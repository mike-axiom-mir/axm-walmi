#!/usr/bin/env python3
"""Build one deterministic ZIP from the verified WALMI source manifest."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
import zipfile


ROOT = Path(__file__).resolve().parents[1]
MANIFEST_PATH = ROOT / "WALMI-SOURCE-MANIFEST.json"
OUTPUT = ROOT / "release" / "WALMI_STANDALONE_SOURCE.zip"
FIXED_TIME = (2020, 1, 1, 0, 0, 0)


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def main() -> int:
    manifest_bytes = MANIFEST_PATH.read_bytes()
    manifest = json.loads(manifest_bytes)
    members = [*manifest["files"], {
        "path": MANIFEST_PATH.name,
        "bytes": len(manifest_bytes),
        "sha256": sha256_bytes(manifest_bytes),
    }]
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    temporary = OUTPUT.with_suffix(".zip.tmp")
    with zipfile.ZipFile(
        temporary, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9
    ) as archive:
        for item in sorted(members, key=lambda value: value["path"]):
            path = ROOT / item["path"]
            payload = path.read_bytes()
            if len(payload) != item["bytes"] or sha256_bytes(payload) != item["sha256"]:
                raise RuntimeError(f"source changed after manifest seal: {item['path']}")
            info = zipfile.ZipInfo(item["path"], FIXED_TIME)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o100644 << 16
            archive.writestr(info, payload, compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)
    temporary.replace(OUTPUT)
    with zipfile.ZipFile(OUTPUT) as archive:
        bad = archive.testzip()
        if bad:
            raise RuntimeError(f"ZIP verification failed at {bad}")
    report = {
        "status": "PASS",
        "path": str(OUTPUT),
        "members": len(members),
        "bytes": OUTPUT.stat().st_size,
        "sha256": hashlib.sha256(OUTPUT.read_bytes()).hexdigest(),
        "sourceTreeSha256": manifest["treeSha256"],
    }
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
