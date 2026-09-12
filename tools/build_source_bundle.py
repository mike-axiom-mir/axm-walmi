#!/usr/bin/env python3
"""Build one deterministic ZIP from the verified WALMI source manifest."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import zipfile

from seal_source_tree import inventory, manifest_value


ROOT = Path(__file__).resolve().parents[1]
MANIFEST_PATH = ROOT / "WALMI-SOURCE-MANIFEST.json"
OUTPUT = ROOT / "release" / "WALMI_STANDALONE_SOURCE.zip"
FIXED_TIME = (2020, 1, 1, 0, 0, 0)
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def source_path(root: Path, relative: object) -> Path:
    if not isinstance(relative, str) or not relative or "\\" in relative:
        raise RuntimeError(f"unsafe source path in manifest: {relative!r}")
    portable = PurePosixPath(relative)
    if portable.is_absolute() or portable.as_posix() != relative or any(
        part in {"", ".", ".."} for part in portable.parts
    ):
        raise RuntimeError(f"unsafe source path in manifest: {relative!r}")
    path = root.joinpath(*portable.parts)
    current = root
    for part in portable.parts:
        current = current / part
        if current.is_symlink():
            raise RuntimeError(f"linked source path is forbidden: {relative}")
    if not path.is_file():
        raise RuntimeError(f"manifest source is not a regular file: {relative}")
    try:
        path.resolve(strict=True).relative_to(root.resolve(strict=True))
    except (FileNotFoundError, ValueError) as error:
        raise RuntimeError(f"source path escapes the sealed root: {relative}") from error
    return path


def load_verified_manifest(root: Path, manifest_path: Path) -> tuple[dict[str, object], bytes]:
    source_path(root, manifest_path.relative_to(root).as_posix())
    manifest_bytes = manifest_path.read_bytes()
    try:
        manifest = json.loads(manifest_bytes)
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise RuntimeError(f"invalid source manifest: {manifest_path.name}") from error
    if not isinstance(manifest, dict) or not isinstance(manifest.get("files"), list):
        raise RuntimeError("invalid source manifest structure")
    seen: set[str] = set()
    for item in manifest["files"]:
        if not isinstance(item, dict):
            raise RuntimeError("invalid source manifest file entry")
        relative = item.get("path")
        source_path(root, relative)
        if relative in seen:
            raise RuntimeError(f"duplicate source path in manifest: {relative}")
        seen.add(relative)
        if type(item.get("bytes")) is not int or item["bytes"] < 0:
            raise RuntimeError(f"invalid byte count in manifest: {relative}")
        if not isinstance(item.get("sha256"), str) or not SHA256_PATTERN.fullmatch(item["sha256"]):
            raise RuntimeError(f"invalid SHA-256 in manifest: {relative}")
    declared = manifest_value(manifest["files"])
    if manifest != declared:
        raise RuntimeError("source manifest metadata does not match its file entries")
    actual = manifest_value(inventory(root, manifest_path))
    if manifest != actual:
        raise RuntimeError("source tree does not match the sealed manifest")
    return manifest, manifest_bytes


def build_bundle(
    root: Path = ROOT,
    manifest_path: Path = MANIFEST_PATH,
    output: Path = OUTPUT,
) -> dict[str, object]:
    root, manifest_path, output = Path(root), Path(manifest_path), Path(output)
    manifest, manifest_bytes = load_verified_manifest(root, manifest_path)
    members = [*manifest["files"], {
        "path": manifest_path.name,
        "bytes": len(manifest_bytes),
        "sha256": sha256_bytes(manifest_bytes),
    }]
    output.parent.mkdir(parents=True, exist_ok=True)
    temporary = output.with_suffix(".zip.tmp")
    try:
        with zipfile.ZipFile(
            temporary, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9
        ) as archive:
            for item in sorted(members, key=lambda value: value["path"]):
                path = source_path(root, item["path"])
                payload = path.read_bytes()
                if len(payload) != item["bytes"] or sha256_bytes(payload) != item["sha256"]:
                    raise RuntimeError(f"source changed after manifest seal: {item['path']}")
                info = zipfile.ZipInfo(item["path"], FIXED_TIME)
                info.compress_type = zipfile.ZIP_DEFLATED
                info.external_attr = 0o100644 << 16
                archive.writestr(info, payload, compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)
        temporary.replace(output)
    finally:
        temporary.unlink(missing_ok=True)
    with zipfile.ZipFile(output) as archive:
        bad = archive.testzip()
        if bad:
            raise RuntimeError(f"ZIP verification failed at {bad}")
    return {
        "status": "PASS",
        "path": str(output),
        "members": len(members),
        "bytes": output.stat().st_size,
        "sha256": hashlib.sha256(output.read_bytes()).hexdigest(),
        "sourceTreeSha256": manifest["treeSha256"],
    }


def main() -> int:
    report = build_bundle()
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
