#!/usr/bin/env python3
"""Create or verify the deterministic manifest for the standalone source tree."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "WALMI-SOURCE-MANIFEST.json"
EXCLUDED_ROOTS = {
    ".git",
    ".cache",
    ".mypy_cache",
    ".pytest_cache",
    ".ruff_cache",
    ".walmi-intake",
    "WALMI_PC_FULL_STACK_v0_2",
    "checkpoints",
    "dist",
    "experience",
    "models",
    "node_modules",
    "private-state",
    "release",
    "rollback",
    "runtime-state",
    "simulator-saves",
    "state",
}
EXCLUDED_DIRECTORY_NAMES = {
    "__pycache__",
    ".mypy_cache",
    ".pytest_cache",
    ".ruff_cache",
    "node_modules",
}
EXCLUDED_SUFFIXES = {".pyc", ".pyo"}


def sha256(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def inventory(
    root: Path = ROOT, manifest: Path | None = None
) -> list[dict[str, object]]:
    root = Path(root)
    manifest = Path(manifest) if manifest is not None else root / MANIFEST.name
    files = []
    paths = sorted(
        root.rglob("*"),
        key=lambda path: path.relative_to(root).as_posix().encode("utf-8"),
    )
    for path in paths:
        relative = path.relative_to(root)
        if relative.parts[0] in EXCLUDED_ROOTS:
            continue
        if any(part in EXCLUDED_DIRECTORY_NAMES for part in relative.parts[:-1]):
            continue
        if path.suffix.lower() in EXCLUDED_SUFFIXES:
            continue
        if path == manifest:
            continue
        if path.is_symlink():
            raise RuntimeError(f"linked path is forbidden: {relative.as_posix()}")
        if not path.is_file():
            continue
        files.append({
            "path": relative.as_posix(),
            "bytes": path.stat().st_size,
            "sha256": sha256(path),
        })
    return files


def manifest_value(files: list[dict[str, object]]) -> dict[str, object]:
    aggregate = hashlib.sha256()
    for item in files:
        aggregate.update(
            f"{item['sha256']}  {item['bytes']}  {item['path']}\n".encode("utf-8")
        )
    return {
        "schema": "axm.walmi.standalone-source-manifest/v1",
        "copyMode": "FULL_FILES_NO_LINKS",
        "fileCount": len(files),
        "totalBytes": sum(int(item["bytes"]) for item in files),
        "treeSha256": aggregate.hexdigest(),
        "files": files,
    }


def difference_summary(
    expected: dict[str, object], actual: dict[str, object]
) -> dict[str, object]:
    expected_files = {
        str(item["path"]): item for item in expected.get("files", [])
    }
    actual_files = {
        str(item["path"]): item for item in actual.get("files", [])
    }
    missing = sorted(set(expected_files) - set(actual_files))
    unexpected = sorted(set(actual_files) - set(expected_files))
    changed = sorted(
        path
        for path in set(expected_files) & set(actual_files)
        if expected_files[path] != actual_files[path]
    )
    return {
        "expectedTreeSha256": expected.get("treeSha256"),
        "actualTreeSha256": actual["treeSha256"],
        "expectedFileCount": expected.get("fileCount"),
        "actualFileCount": actual["fileCount"],
        "missingPaths": missing[:50],
        "unexpectedPaths": unexpected[:50],
        "changedPaths": changed[:50],
        "differenceCount": len(missing) + len(unexpected) + len(changed),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--verify", action="store_true")
    args = parser.parse_args()
    actual = manifest_value(inventory())
    if args.verify:
        expected = json.loads(MANIFEST.read_text(encoding="utf-8"))
        if actual != expected:
            print(json.dumps({
                "status": "FAIL",
                **difference_summary(expected, actual),
            }, indent=2))
            return 1
    else:
        MANIFEST.write_text(
            json.dumps(actual, indent=2, ensure_ascii=False) + "\n",
            encoding="utf-8",
            newline="\n",
        )
    print(json.dumps({
        "status": "PASS",
        "mode": "VERIFY" if args.verify else "SEAL",
        "fileCount": actual["fileCount"],
        "totalBytes": actual["totalBytes"],
        "treeSha256": actual["treeSha256"],
    }, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
