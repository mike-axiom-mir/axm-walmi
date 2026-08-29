from __future__ import annotations

import hashlib
import json
import zipfile
from pathlib import Path

DIST = Path("dist-v02")
HOST = DIST / "host"
CARTRIDGE = DIST / "cartridge"
NATIVE = CARTRIDGE / "native/windows/WALMI_WINDOWS_FULL_HOST_v0_2.zip"
OUT = DIST / "WALMI_PC_FULL_STACK_v0_2.zip"


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def zip_tree(root: Path, output: Path) -> None:
    output.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(output, "w", zipfile.ZIP_DEFLATED, compresslevel=6, allowZip64=True) as archive:
        for path in sorted(item for item in root.rglob("*") if item.is_file()):
            archive.write(path, path.relative_to(root).as_posix())
    with zipfile.ZipFile(output) as archive:
        bad = archive.testzip()
        if bad:
            raise SystemExit(f"bad ZIP member: {bad}")


zip_tree(HOST, NATIVE)
native_required = {
    "bin/waldo.exe",
    "bin/waldo-axm-mirror.exe",
    "bin/waldo-mirror-review.exe",
    "runtime/python/python.exe",
    "START_WALMI_PC.cmd",
    "WALMI_LOCAL_SHELL.cmd",
    "RUN_WORKSPACE_HAND.cmd",
    "WALMI_WORKSPACE_REQUEST_EXAMPLE.json",
}
with zipfile.ZipFile(NATIVE) as archive:
    native_names = set(archive.namelist())
missing = sorted(native_required - native_names)
if missing:
    raise SystemExit(f"native host missing: {missing}")

inventory = CARTRIDGE / "inventory"
receipt = json.loads((inventory / "WALMI_V02_BUILD_RECEIPT.json").read_text(encoding="utf-8"))
if receipt.get("status") != "PASS" or not receipt.get("workspaceHand") or not receipt.get("neuralToolLoop"):
    raise SystemExit("workspace Hand build receipt is not PASS")

manifest = {
    "schema": "walmi.cartridge/v1",
    "name": "WALMI-PC-FULL-STACK",
    "version": "0.2.0",
    "source": {
        "repository": "mike-axiom-mir/waldo-axm-mirror-research",
        "commit": receipt["commit"],
        "archive": "source/WALMI_CURRENT_STACK_v0_2.zip",
        "completeHistory": "history/WALDO_ALL_EXPERIMENTS_v0_2.bundle",
    },
    "stack": [
        "MIRROR", "WALDO", "HERMES_HANDOFF", "EPHEMERAL_SPECIALISTS",
        "NEURAL_SOCKETS", "DETERMINIZATION_STEWARD", "WORKSHOP_CAPABILITY_INTAKE",
        "SERVICE_MODES", "GRAMMAR_GLASS", "CREATION_FABRIC", "PROJECT_WORKSPACE_HAND",
        "TRANSACTIONAL_CANDIDATE_WRITER", "NEURAL_TOOL_LOOP", "MODEL_LIFECYCLE",
        "TRAINING", "INFERENCE",
    ],
    "nativeHosts": [{
        "platform": "windows", "arch": "x64",
        "path": "native/windows/WALMI_WINDOWS_FULL_HOST_v0_2.zip",
        "sha256": sha256(NATIVE),
    }],
    "runtime": {
        "targetNeedsGoInstalled": False,
        "targetNeedsPythonInstalled": False,
        "portablePython": True,
        "portablePyTorchCPU": True,
        "browserWASM": "runtime/go/walmi-axm-mirror.wasm",
        "startingModelWeightsBundled": False,
        "modelWeightTruth": "NO_MODEL_WEIGHTS_FOUND_OR_FABRICATED; select an existing local WALDO model",
    },
    "workspace": {
        "selectedRootOnly": True,
        "writeDefault": False,
        "transactionalWriter": True,
        "verificationWithoutShell": True,
        "experienceCandidateOptIn": True,
    },
    "authority": {
        "install": False, "integration": False, "promotion": False,
        "learning": False, "canon": False,
    },
}
(CARTRIDGE / "walmi.manifest.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
(CARTRIDGE / "README_FIRST.txt").write_text(
    "WALMI PC FULL STACK v0.2\n\n"
    "This cartridge contains the complete current source archive, complete Git experiment history, "
    "native Windows host with portable Python/PyTorch, browser WASM fallback, and the bounded project Workspace Hand.\n\n"
    "Extract native/windows/WALMI_WINDOWS_FULL_HOST_v0_2.zip and open README_FIRST.txt. "
    "No model weights are fabricated or silently downloaded; select an existing local WALDO model.\n",
    encoding="utf-8",
)

required = [
    "walmi.manifest.json",
    "native/windows/WALMI_WINDOWS_FULL_HOST_v0_2.zip",
    "runtime/go/walmi-axm-mirror.wasm",
    "runtime/go/wasm_exec.js",
    "runtime/browser-entry.js",
    "source/WALMI_CURRENT_STACK_v0_2.zip",
    "history/WALDO_ALL_EXPERIMENTS_v0_2.bundle",
    "inventory/WALMI_V02_BUILD_RECEIPT.json",
    "inventory/GO_TEST_ALL.txt",
    "inventory/WINDOWS_RUNTIME_VERIFY.txt",
    "inventory/PYTORCH_RUNTIME_VERIFY.txt",
]
missing = [name for name in required if not (CARTRIDGE / name).is_file()]
if missing:
    raise SystemExit(f"FINAL_COMPLETENESS_HOLD {missing}")

rows = []
for path in sorted(item for item in CARTRIDGE.rglob("*") if item.is_file()):
    rows.append({"path": path.relative_to(CARTRIDGE).as_posix(), "bytes": path.stat().st_size, "sha256": sha256(path)})
(inventory / "CARTRIDGE_FILE_MANIFEST.json").write_text(json.dumps({"schema": "walmi.cartridge-files/v1", "count": len(rows), "files": rows}, indent=2), encoding="utf-8")
zip_tree(CARTRIDGE, OUT)
digest = sha256(OUT)
(DIST / "WALMI_PC_FULL_STACK_v0_2.sha256.txt").write_text(f"{digest}  {OUT.name}\n", encoding="utf-8")
print("FINAL_WALMI_PC_V02_ZIP_PASS", OUT.stat().st_size, digest, len(rows))
