#!/usr/bin/env python3
"""Regression tests for the standalone source seal and bundle boundary."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
import tempfile
import unittest
import zipfile

from build_source_bundle import build_bundle
from seal_source_tree import inventory, manifest_value


class SourcePackagingTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="walmi-source-package-")
        self.base = Path(self.temporary.name)
        self.root = self.base / "source"
        self.root.mkdir()
        self.manifest = self.root / "WALMI-SOURCE-MANIFEST.json"
        self.output = self.root / "release" / "source.zip"

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def write(self, relative: str, content: bytes) -> Path:
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(content)
        return path

    def seal(self) -> dict[str, object]:
        value = manifest_value(inventory(self.root, self.manifest))
        self.manifest.write_text(
            json.dumps(value, indent=2) + "\n", encoding="utf-8", newline="\n"
        )
        return value

    def test_verified_bundle_is_deterministic(self) -> None:
        self.write("README.md", b"sealed source\n")
        self.write("nested/code.go", b"package sealed\n")
        sealed = self.seal()
        first = build_bundle(self.root, self.manifest, self.output)
        first_bytes = self.output.read_bytes()
        second = build_bundle(self.root, self.manifest, self.output)
        self.assertEqual(first["sha256"], second["sha256"])
        self.assertEqual(first_bytes, self.output.read_bytes())
        self.assertEqual(first["sourceTreeSha256"], sealed["treeSha256"])
        with zipfile.ZipFile(self.output) as archive:
            self.assertEqual(
                archive.namelist(),
                ["README.md", "WALMI-SOURCE-MANIFEST.json", "nested/code.go"],
            )

    def test_bundle_rejects_manifest_path_escape(self) -> None:
        outside = self.base / "private.txt"
        outside.write_bytes(b"must stay outside\n")
        payload = outside.read_bytes()
        files = [{
            "path": "../private.txt",
            "bytes": len(payload),
            "sha256": hashlib.sha256(payload).hexdigest(),
        }]
        self.manifest.write_text(
            json.dumps(manifest_value(files), indent=2) + "\n", encoding="utf-8"
        )
        with self.assertRaisesRegex(RuntimeError, "unsafe source path"):
            build_bundle(self.root, self.manifest, self.output)
        self.assertFalse(self.output.exists())

    def test_bundle_rejects_unsealed_file(self) -> None:
        self.write("sealed.txt", b"sealed\n")
        self.seal()
        self.write("later.txt", b"not in manifest\n")
        with self.assertRaisesRegex(RuntimeError, "does not match the sealed manifest"):
            build_bundle(self.root, self.manifest, self.output)
        self.assertFalse(self.output.exists())

    def test_inventory_rejects_directory_and_dangling_links(self) -> None:
        target = self.base / "outside"
        target.mkdir()
        (target / "private.txt").write_text("outside\n", encoding="utf-8")
        try:
            (self.root / "linked-directory").symlink_to(target, target_is_directory=True)
            (self.root / "dangling-file").symlink_to(self.base / "missing.txt")
        except (NotImplementedError, OSError) as error:
            self.skipTest(f"symlink creation unavailable: {error}")
        with self.assertRaisesRegex(RuntimeError, "linked path is forbidden"):
            inventory(self.root, self.manifest)


if __name__ == "__main__":
    unittest.main()
