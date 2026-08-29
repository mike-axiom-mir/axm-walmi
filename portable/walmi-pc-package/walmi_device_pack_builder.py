from __future__ import annotations

import argparse
import hashlib
import json
import os
import tempfile
import zipfile
from pathlib import Path


TARGETS = ("windows-x64", "linux-x64", "android", "ios-web", "ios-native")
MOBILE_TARGETS = {"android", "ios-web", "ios-native"}


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def files_under(path: Path) -> list[tuple[Path, Path]]:
    if path.is_file():
        return [(path, Path(path.name))]
    if path.is_dir():
        return [
            (item, item.relative_to(path))
            for item in sorted(path.rglob("*"))
            if item.is_file()
        ]
    raise ValueError(f"input does not exist: {path}")


def tree_sha256(path: Path) -> str:
    digest = hashlib.sha256()
    for item, relative in files_under(path):
        name = relative.as_posix().encode("utf-8")
        digest.update(len(name).to_bytes(8, "big"))
        digest.update(name)
        digest.update(item.stat().st_size.to_bytes(8, "big"))
        with item.open("rb") as stream:
            for block in iter(lambda: stream.read(1024 * 1024), b""):
                digest.update(block)
    return digest.hexdigest()


def zip_tree(path: Path, output: Path) -> int:
    members = files_under(path)
    with zipfile.ZipFile(
        output,
        "w",
        compression=zipfile.ZIP_DEFLATED,
        compresslevel=6,
        allowZip64=True,
    ) as archive:
        for item, relative in members:
            archive.write(item, relative.as_posix())
    with zipfile.ZipFile(output) as archive:
        bad = archive.testzip()
        if bad:
            raise ValueError(f"nested package failed verification at {bad}")
    return len(members)


def build_device_pack(
    target: str,
    interface: Path,
    output: Path,
    engine: Path | None,
    data: Path | None,
) -> dict[str, object]:
    if target not in TARGETS:
        raise ValueError(f"unsupported target {target!r}")
    interface = interface.resolve()
    output = output.resolve()
    engine = engine.resolve() if engine else None
    data = data.resolve() if data else None
    if target in MOBILE_TARGETS and data is not None:
        raise ValueError(
            "mobile companion packs do not embed the full WALMI_DATA tree"
        )
    if output.exists():
        raise ValueError(f"output already exists: {output}")

    interface_members = files_under(interface)
    if not interface_members:
        raise ValueError("interface input contains no files")
    if interface.is_dir() and not (interface / "index.html").is_file():
        raise ValueError("interface directory must contain index.html")
    if interface.is_file() and interface.suffix.lower() != ".html":
        raise ValueError("single-file interface must be HTML")

    output.parent.mkdir(parents=True, exist_ok=True)
    temp_root = Path(tempfile.mkdtemp(prefix="walmi-device-pack-"))
    data_archive: Path | None = None
    try:
        data_count = 0
        data_hash = ""
        if data:
            data_archive = temp_root / "WALMI_DATA.zip"
            data_count = zip_tree(data, data_archive)
            data_hash = sha256(data_archive)

        finalization = (
            "REQUIRES_MACOS_XCODE_SIGNING"
            if target == "ios-native"
            else "DEVICE_RUNTIME_OR_BROWSER_INSTALL_STEP_REQUIRED"
        )
        manifest: dict[str, object] = {
            "schema": "walmi.device-pack/v1",
            "target": target,
            "role": "COMPANION_TO_PC_HOME"
            if target in MOBILE_TARGETS
            else "FULL_HOST_OR_COMPANION",
            "capabilitySurface": "FULL_WALMI_VIA_PC_HOME"
            if target in MOBILE_TARGETS
            else "LOCAL_FULL_HOST_WHEN_ENGINE_IS_VERIFIED",
            "interface": {
                "kind": "desktop-local-page"
                if target in {"windows-x64", "linux-x64"}
                else "mobile-local-page",
                "sha256": tree_sha256(interface),
                "files": len(interface_members),
            },
            "engine": {
                "state": "SEALED"
                if engine
                else (
                    "BROWSER_COMPANION_ENGINE_NOT_SUPPLIED"
                    if target in MOBILE_TARGETS
                    else "HELD_ENGINE_NOT_SUPPLIED"
                ),
                "sha256": tree_sha256(engine) if engine else "",
                "files": len(files_under(engine)) if engine else 0,
            },
            "personalData": {
                "state": "SEALED" if data else "CLEAN_START",
                "archive": "WALMI_DATA.zip" if data else None,
                "sha256": data_hash,
                "files": data_count,
                "directExperienceRewritten": False,
                "fullWalmiDataBundled": data is not None,
            },
            "pcBridge": {
                "required": target in MOBILE_TARGETS,
                "internetRequired": False,
                "mobileRole": "FULL_INTERFACE_INTAKE_AND_COLLABORATION"
                if target in MOBILE_TARGETS
                else "NOT_APPLICABLE",
                "deepModificationHost": "PC_WALMI_HOME"
                if target in MOBILE_TARGETS
                else "LOCAL_HOST",
                "state": "WAITING_FOR_LOCAL_WALMI_INTERFACE_BRIDGE"
                if target in MOBILE_TARGETS
                else "NOT_REQUIRED",
                "mobileWeightMutation": False,
            },
            "assemblyState": "INPUTS_SEALED",
            "applicationReady": False,
            "finalization": finalization,
            "quality": "UNKNOWN_UNTIL_DEVICE_RUNTIME_AND_BEHAVIOR_VERIFICATION",
        }

        with zipfile.ZipFile(
            output,
            "w",
            compression=zipfile.ZIP_DEFLATED,
            compresslevel=6,
            allowZip64=True,
        ) as archive:
            archive.writestr(
                "walmi-device-pack.json",
                json.dumps(manifest, indent=2, sort_keys=True) + "\n",
            )
            for item, relative in interface_members:
                destination = (
                    Path("interface/index.html")
                    if interface.is_file()
                    else Path("interface") / relative
                )
                archive.write(item, destination.as_posix())
            if engine:
                for item, relative in files_under(engine):
                    archive.write(item, (Path("engine") / relative).as_posix())
            if data_archive:
                archive.write(data_archive, "WALMI_DATA.zip")

        with zipfile.ZipFile(output) as archive:
            bad = archive.testzip()
            if bad:
                raise ValueError(f"device pack failed verification at {bad}")
        manifest["packageSha256"] = sha256(output)
        manifest["packageBytes"] = output.stat().st_size
        return manifest
    finally:
        if data_archive and data_archive.exists():
            data_archive.unlink()
        try:
            temp_root.rmdir()
        except OSError:
            pass


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Assemble a WALMI device transport pack on the PC."
    )
    parser.add_argument("--target", required=True, choices=TARGETS)
    parser.add_argument("--interface", required=True, type=Path)
    parser.add_argument("--engine", type=Path)
    parser.add_argument("--data", type=Path)
    parser.add_argument("--output", required=True, type=Path)
    return parser.parse_args()


def main() -> int:
    arguments = parse_args()
    data = arguments.data
    if (
        data is None
        and arguments.target not in MOBILE_TARGETS
        and os.environ.get("WALMI_DATA_HOME")
    ):
        candidate = Path(os.environ["WALMI_DATA_HOME"])
        data = candidate if candidate.is_dir() else None
    result = build_device_pack(
        arguments.target,
        arguments.interface,
        arguments.output,
        arguments.engine,
        data,
    )
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
