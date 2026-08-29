from __future__ import annotations

import json
import unittest
import zipfile
from pathlib import Path
from tempfile import TemporaryDirectory

import walmi_device_pack_builder


class DevicePackBuilderTests(unittest.TestCase):
    def test_builds_one_verified_transport_without_rewriting_data(self) -> None:
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            page = root / "mobile.html"
            page.write_text("<!doctype html><title>WALMI</title>", encoding="utf-8")
            engine = root / "engine.bin"
            engine.write_bytes(b"engine")
            data = root / "data"
            data.mkdir()
            experience = data / "direct-experience.jsonl"
            experience.write_text('{"observed":true}\n', encoding="utf-8")
            before = experience.read_bytes()
            output = root / "linux-pack.zip"

            result = walmi_device_pack_builder.build_device_pack(
                "linux-x64", page, output, engine, data
            )

            self.assertEqual(result["personalData"]["directExperienceRewritten"], False)
            self.assertEqual(experience.read_bytes(), before)
            with zipfile.ZipFile(output) as archive:
                self.assertIsNone(archive.testzip())
                names = set(archive.namelist())
                self.assertIn("interface/index.html", names)
                self.assertIn("engine/engine.bin", names)
                self.assertIn("WALMI_DATA.zip", names)
                manifest = json.loads(archive.read("walmi-device-pack.json"))
            self.assertEqual(manifest["target"], "linux-x64")
            self.assertFalse(manifest["applicationReady"])

    def test_mobile_companion_refuses_full_pc_data_tree(self) -> None:
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            page = root / "mobile.html"
            page.write_text("<!doctype html><title>WALMI</title>", encoding="utf-8")
            data = root / "data"
            data.mkdir()
            (data / "models").mkdir()
            with self.assertRaisesRegex(ValueError, "do not embed the full"):
                walmi_device_pack_builder.build_device_pack(
                    "android", page, root / "android-pack.zip", None, data
                )

    def test_ios_native_requires_mac_finalization(self) -> None:
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            page = root / "mobile.html"
            page.write_text("<!doctype html><title>WALMI</title>", encoding="utf-8")
            output = root / "ios-pack.zip"
            result = walmi_device_pack_builder.build_device_pack(
                "ios-native", page, output, None, None
            )
            self.assertEqual(result["finalization"], "REQUIRES_MACOS_XCODE_SIGNING")
            self.assertEqual(
                result["engine"]["state"],
                "BROWSER_COMPANION_ENGINE_NOT_SUPPLIED",
            )
            self.assertEqual(result["role"], "COMPANION_TO_PC_HOME")
            self.assertFalse(result["pcBridge"]["mobileWeightMutation"])


if __name__ == "__main__":
    unittest.main()
