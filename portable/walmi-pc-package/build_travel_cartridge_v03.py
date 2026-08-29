from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import subprocess
import zipfile
from pathlib import Path


MUTABLE_ROOTS = {
    "assets",
    "checkpoints",
    "experience",
    "models",
    "rollback",
    "state",
    "workspace",
}
REQUIRED_RUNTIME = {
    "bin/waldo.exe",
    "runtime/node/node.exe",
    "runtime/python/python.exe",
    "tools/walmi_autolearn.py",
    "tools/walmi_device_pack_builder.py",
    "tools/walmi_node_world_adapter.mjs",
    "tools/walmi_portable_bootstrap.py",
    "tools/walmi_simulator_bridge.py",
}


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def source_commit(repository: Path) -> str:
    completed = subprocess.run(
        ["git", "-C", str(repository), "rev-parse", "HEAD"],
        check=True,
        capture_output=True,
        text=True,
    )
    return completed.stdout.strip()


def runtime_members(host: Path) -> list[Path]:
    return sorted(
        path
        for path in host.rglob("*")
        if path.is_file() and path.relative_to(host).parts[0] not in MUTABLE_ROOTS
    )


def build_runtime(repository: Path, host: Path, output: Path) -> int:
    overrides = {
        "tools/walmi_autolearn.py": repository
        / "portable/walmi-pc-package/walmi_autolearn.py",
        "tools/walmi_device_pack_builder.py": repository
        / "portable/walmi-pc-package/walmi_device_pack_builder.py",
        "tools/walmi_node_world_adapter.mjs": repository
        / "portable/walmi-pc-package/walmi_node_world_adapter.mjs",
        "tools/walmi_portable_bootstrap.py": repository
        / "portable/walmi-pc-package/walmi_portable_bootstrap.py",
        "tools/walmi_simulator_bridge.py": repository
        / "portable/walmi-pc-package/walmi_simulator_bridge.py",
    }
    members = runtime_members(host)
    names = {path.relative_to(host).as_posix() for path in members}
    names.update(overrides)
    missing = sorted(REQUIRED_RUNTIME - names)
    if missing:
        raise SystemExit(f"runtime source is missing required files: {missing}")

    output.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(
        output,
        "w",
        compression=zipfile.ZIP_DEFLATED,
        compresslevel=6,
        allowZip64=True,
    ) as archive:
        for path in members:
            relative = path.relative_to(host).as_posix()
            if relative in overrides:
                continue
            archive.write(path, relative)
        for relative, path in sorted(overrides.items()):
            archive.write(path, relative)
    with zipfile.ZipFile(output) as archive:
        bad = archive.testzip()
        if bad:
            raise SystemExit(f"runtime ZIP failed verification at {bad}")
        archived = set(archive.namelist())
        missing = sorted(REQUIRED_RUNTIME - archived)
        if missing:
            raise SystemExit(f"runtime ZIP is missing required files: {missing}")
        return len(archive.infolist())


def reuse_runtime(source: Path, output: Path) -> int:
    if not source.is_file():
        raise SystemExit(f"reusable runtime archive does not exist: {source}")
    output.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, output)
    with zipfile.ZipFile(output) as archive:
        bad = archive.testzip()
        if bad:
            raise SystemExit(f"reused runtime ZIP failed verification at {bad}")
        archived = set(archive.namelist())
        missing = sorted(REQUIRED_RUNTIME - archived)
        if missing:
            raise SystemExit(f"reused runtime ZIP is missing required files: {missing}")
        return len(archive.infolist())


def write_outer_zip(root: Path, output: Path) -> None:
    with zipfile.ZipFile(output, "w", allowZip64=True) as archive:
        for path in sorted(item for item in root.rglob("*") if item.is_file()):
            relative = path.relative_to(root.parent).as_posix()
            compression = (
                zipfile.ZIP_STORED
                if path.name.startswith("WALMI_RUNTIME_WINDOWS_X64_")
                else zipfile.ZIP_DEFLATED
            )
            archive.write(path, relative, compress_type=compression)
    with zipfile.ZipFile(output) as archive:
        bad = archive.testzip()
        if bad:
            raise SystemExit(f"travel ZIP failed verification at {bad}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Build the few-file WALMI Windows travel cartridge."
    )
    parser.add_argument(
        "--host", type=Path, default=Path("WALMI_PC_FULL_STACK_v0_2/host")
    )
    parser.add_argument(
        "--output", type=Path, default=Path("dist/WALMI_TRAVEL_CARTRIDGE_v0_3_6")
    )
    parser.add_argument(
        "--reuse-runtime",
        type=Path,
        help="reuse and fully verify an existing runtime ZIP when only outer launchers changed",
    )
    return parser.parse_args()


def main() -> int:
    arguments = parse_args()
    repository = Path.cwd().resolve()
    host = arguments.host.resolve()
    output = arguments.output.resolve()
    if not host.is_dir():
        raise SystemExit(f"source host does not exist: {host}")
    if output.exists():
        raise SystemExit(f"output already exists: {output}")

    output.mkdir(parents=True)
    runtime = output / "WALMI_RUNTIME_WINDOWS_X64_v0_3_6.zip"
    runtime_count = (
        reuse_runtime(arguments.reuse_runtime.resolve(), runtime)
        if arguments.reuse_runtime
        else build_runtime(repository, host, runtime)
    )
    runtime_hash = sha256(runtime)

    travel_source = repository / "portable/walmi-pc-package/travel"
    for name in [
        "BUILD_DEVICE_PACKAGE.cmd",
        "START_WALMI.cmd",
        "WALMI_AUTOLEARN.cmd",
        "WALMI_CARTRIDGE.ps1",
        "WALMI_EXPERIENCE_OBSERVE.cmd",
        "WALMI_PLAY_SIMULATORS.cmd",
    ]:
        shutil.copy2(travel_source / name, output / name)

    manifest = {
        "schema": "walmi.travel-cartridge/v1",
        "name": "WALMI-TRAVEL-CARTRIDGE",
        "version": "0.3.6",
        "sourceCommit": source_commit(repository),
        "interfaces": {
            "desktop": {
                "kind": "desktop-local-page",
                "state": "WAITING_FOR_USER_WALMI_HTML",
                "entry": None,
            },
            "mobile": {
                "kind": "mobile-local-page",
                "state": "WAITING_FOR_USER_WALMI_HTML",
                "entry": None,
            },
        },
        "frontendOrganIntake": {
            "state": "WAITING_FOR_USER_CONTRACT",
            "humanAndWalmiUseSameJsonIntake": True,
            "walmiPosition": "TOP_LEVEL_OVER_LOCAL_AXM_PAGES",
            "pageAppliesItsOwnValidatedUpdate": True,
            "hiddenMutation": False,
        },
        "platformPacks": {
            "windows-x64": "BUNDLED",
            "android": "COMPANION_NOT_BUILT_YET",
            "linux-x64": "NOT_BUILT_YET",
            "ios": "COMPANION_NOT_BUILT_YET",
        },
        "devicePackBuilder": {
            "host": "windows-x64",
            "targets": ["windows-x64", "linux-x64", "android", "ios-web", "ios-native"],
            "nativeIOSFinalization": "REQUIRES_MACOS_XCODE_SIGNING",
        },
        "runtime": {
            "platform": "windows",
            "architecture": "x64",
            "archive": runtime.name,
            "sha256": runtime_hash,
            "expandedFiles": runtime_count,
            "cacheRoot": "D:\\WALMI_RUNTIME_CACHE",
        },
        "personalData": {
            "defaultPath": "WALMI_DATA",
            "travelsWithCartridge": True,
            "sharedWithRuntimeCache": False,
            "platformNeutralBoundary": True,
            "mobilePolicy": "COMPANION_STATE_ONLY_NO_MODELS_OR_CHECKPOINTS",
            "mobileCapabilitySurface": "FULL_WALMI_VIA_PC_HOME",
            "startingModelWeights": False,
        },
        "claims": {
            "offlineAfterTransfer": True,
            "runtimeExtractedOncePerMachine": True,
            "behaviorQuality": "UNKNOWN_UNTIL_HELD_OUT_EVALUATION",
        },
        "simulatorBridge": {
            "worlds": ["theme-park", "factual-space", "living-city"],
            "defaultRoots": "D:\\AXM_ACTIVE",
            "playerKinds": ["deterministic", "organ", "model"],
            "modelState": "WAITING_FOR_LOCAL_WEIGHTS",
            "autolearn": "AUTOMATIC_THRESHOLD_8_AFTER_SUCCESSFUL_PLAY",
            "runStorage": "PERSISTENT_NAMED_SAVE_PER_WORLD_PLUS_ONE_VERIFIED_RECEIPT_ZIP_PER_SESSION",
            "capabilityGapLedger": "APPEND_ONLY_TYPED_GAPS",
            "invalidPlayerRepair": "STOP_BY_DEFAULT; OPTIONAL_VISIBLE_DETERMINISTIC_FALLBACK_IS_EXPLICIT",
            "laterGrowthCorpora": "UNIQUE_LOGICAL_IDENTITY_PER_GENERATION",
        },
    }
    (output / "walmi-cartridge.json").write_text(
        json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )
    (output / "README_FIRST.txt").write_text(
        "WALMI TRAVEL CARTRIDGE v0.3.6\n\n"
        "Run START_WALMI.cmd. WALMI prepares its heavy runtime once on D:.\n"
        "Copy this folder, or the matching ZIP, when moving WALMI. Personal experience, models and state live in WALMI_DATA beside the launcher.\n"
        "The reusable runtime cache is not personal and does not need to travel.\n"
        "This build contains the full Windows home engine. The desktop and mobile HTML interfaces will be added from the user-provided WALMI pages. Android and iPhone are lightweight companions to the PC home; Linux may receive a full engine later.\n"
        "WALMI_PLAY_SIMULATORS.cmd lets the local walmi model return to its own named Theme Park, Factual Space, and Living City saves under D:\\AXM_ACTIVE. Decisions are wall-clock paced by default, every accepted simulator mutation is checkpointed, and every observed turn becomes a separate outcome-conditioned reflection while the full session receives one verified ZIP.\n",
        encoding="utf-8",
    )

    transport_files = len([path for path in output.iterdir() if path.is_file()])
    receipt = {
        "schema": "walmi.travel-cartridge-build/v1",
        "status": "PASS",
        "transportFiles": transport_files,
        "expandedRuntimeFiles": runtime_count,
        "runtimeSha256": runtime_hash,
        "startingModelWeights": False,
        "runtimeCacheDrive": "D:",
        "personalDataSeparated": True,
        "quality": "UNKNOWN_UNTIL_HELD_OUT_EVALUATION",
    }
    (output / "BUILD_RECEIPT.json").write_text(
        json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )

    outer = output.with_suffix(".zip")
    if outer.exists():
        raise SystemExit(f"outer ZIP already exists: {outer}")
    write_outer_zip(output, outer)
    outer_hash = sha256(outer)
    (output.parent / f"{output.name}.sha256.txt").write_text(
        f"{outer_hash}  {outer.name}\n", encoding="utf-8"
    )
    print(
        "WALMI_TRAVEL_CARTRIDGE_V036_PASS",
        f"transport_files={transport_files + 1}",
        f"runtime_files={runtime_count}",
        f"runtime_sha256={runtime_hash}",
        f"outer_sha256={outer_hash}",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
