from __future__ import annotations

import hashlib
import json
import sys
import zipfile
from pathlib import Path

DIST = Path("dist")
HOST = DIST / "host"
CARTRIDGE = DIST / "cartridge"
NATIVE = CARTRIDGE / "native/windows/WALMI_WINDOWS_FULL_HOST_v0_2.zip"


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def zip_tree(root: Path, out: Path, include=None) -> None:
    out.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED, compresslevel=6, allowZip64=True) as zf:
        for path in sorted(p for p in root.rglob("*") if p.is_file()):
            if include is not None and not include(path.relative_to(root)):
                continue
            zf.write(path, path.relative_to(root).as_posix())
    with zipfile.ZipFile(out) as zf:
        bad = zf.testzip()
        if bad:
            raise SystemExit(f"bad zip member: {bad}")


if sys.argv[1:] == ["native"]:
    mutable_roots = {"models", "checkpoints", "experience", "rollback", "state", "workspace", "assets"}

    def clean_host_member(relative: Path) -> bool:
        if relative.parts and relative.parts[0] in mutable_roots:
            return len(relative.parts) == 2 and relative.name == "README.txt"
        return True

    zip_tree(HOST, NATIVE, clean_host_member)
    with zipfile.ZipFile(NATIVE) as zf:
        names = set(zf.namelist())
    required = {
        "bin/waldo.exe",
        "bin/waldo-axm-mirror.exe",
        "bin/waldo-mirror-review.exe",
        "tools/walmi_workspace_hand.py",
        "tools/walmi_portable_bootstrap.py",
        "tools/walmi_autolearn.py",
        "runtime/python/python.exe",
        "runtime/node/node.exe",
        "tools/walmi_asset_hand_runner.js",
        "body/shared/asset-hands/asset-hands.js",
        "body/shared/visual-kernel/visual-kernel.js",
        "body/shared/visual-fx/fx-blocks.js",
        "body/shared/output/axm-output-core.js",
        "body/shared/vendor/jspdf/jspdf.umd.min.js",
        "body/tools/spatial-studio/spatial-core.js",
        "body/tools/film-motion-studio/film-motion-core.js",
        "body/tools/asset-fabric/fabric-core.js",
        "START_WALMI_PC.cmd",
        "WALMI_LOCAL_SHELL.cmd",
        "WALMI_WORKSPACE_HAND.cmd",
        "WALMI_ASSET_HAND.cmd",
        "WALMI_AUTOLEARN.cmd",
        "WALMI_EXPERIENCE_OBSERVE.cmd",
        "models/README.txt",
        "checkpoints/README.txt",
        "experience/README.txt",
        "rollback/README.txt",
        "workspace/README.txt",
        "assets/README.txt",
        "MODEL_WEIGHT_SCAN.json",
        "ASSET_CAPABILITY_RECEIPT.json",
        "WORKSPACE_HAND_SMOKE.json",
        "AUTOLEARN_EXPERIENCE_SMOKE.json",
    }
    missing = sorted(required - names)
    if missing:
        raise SystemExit(f"native host missing: {missing}")
    print("WINDOWS_FULL_HOST_V02_PASS", NATIVE.stat().st_size, "bytes", len(names), "files")
    raise SystemExit(0)

if sys.argv[1:] != ["final"]:
    raise SystemExit("usage: finalize_cartridge.py native|final")

inventory = CARTRIDGE / "inventory"
source = json.loads((inventory / "CURRENT_SOURCE.json").read_text(encoding="utf-8"))
weights = json.loads((inventory / "MODEL_WEIGHT_SCAN.json").read_text(encoding="utf-8"))
assets = json.loads((inventory / "ASSET_CAPABILITY_RECEIPT.json").read_text(encoding="utf-8"))
workspace = json.loads((inventory / "WORKSPACE_HAND_SMOKE.json").read_text(encoding="utf-8"))
autolearn = json.loads((inventory / "AUTOLEARN_EXPERIENCE_SMOKE.json").read_text(encoding="utf-8"))

if source.get("status") != "PASS" or not source.get("commit"):
    raise SystemExit("current source receipt is not PASS")
if assets.get("status") != "PASS" or assets.get("platformCommit") != "fd6ec98a6a98a6666a980c359730ccec57a8cbe9":
    raise SystemExit("asset capability receipt is not PASS or not pinned")
if not assets.get("workshopRequired") is False or int(assets.get("handCount", 0)) < 30:
    raise SystemExit("asset capability runtime is incomplete")
if workspace.get("status") != "PASS" or workspace.get("networkUsed") is not False:
    raise SystemExit("workspace hand smoke is not PASS")
if autolearn.get("status") != "PASS" or autolearn.get("weightsChanged") is not True or autolearn.get("checkpointReopened") is not True or autolearn.get("behaviorQuality") != "UNKNOWN_NO_HELD_OUT_EVALUATION":
    raise SystemExit("autolearn experience smoke is not PASS or overclaims behavior quality")

manifest = {
    "schema": "walmi.cartridge/v1",
    "name": "WALMI-PC-FULL-STACK",
    "version": "0.2.0",
    "browserEntry": "runtime/browser-entry.js",
    "source": {
        "repository": "mike-axiom-mir/waldo-axm-mirror-research",
        "packageCommit": source["commit"],
        "currentStackArchive": "source/WALMI_CURRENT_STACK_v0_2.zip",
        "allExperimentHistory": "history/WALDO_ALL_EXPERIMENTS.bundle",
        "assetPlatformRepository": "mike-axiom-mir/axm-collaboration-platform",
        "assetPlatformCommit": assets["platformCommit"],
    },
    "stack": [
        "MIRROR", "WALDO", "HERMES", "EPHEMERAL_SPECIALISTS", "NEURAL_SOCKETS",
        "DETERMINIZATION_STEWARD", "SERVICE_MODES", "GRAMMAR_GLASS", "CREATION_FABRIC",
        "MODEL_LIFECYCLE", "TRAINING", "INFERENCE", "INNER_ASSET",
        "WALMI_WORKSPACE_HAND", "ASSET_FABRIC_V0_12", "ASSET_HANDS_V2_5",
    ],
    "nativeHosts": [{
        "platform": "windows", "arch": "x64", "label": "WALMI Windows Full Host v0.2",
        "path": "native/windows/WALMI_WINDOWS_FULL_HOST_v0_2.zip",
        "fileName": "WALMI_WINDOWS_FULL_HOST_v0_2.zip", "sha256": sha256(NATIVE),
    }],
    "runtime": {
        "go": {"target": "js/wasm", "wasm": "runtime/go/walmi-axm-mirror.wasm", "exec": "runtime/go/wasm_exec.js", "targetNeedsGoInstalled": False},
        "windows": {"portablePython": True, "portablePyTorchCPU": True, "portableNode": True, "targetNeedsPythonInstalled": False, "targetNeedsNodeInstalled": False, "trainingAndInferenceCodeBundled": True},
        "workspace": {"optional": True, "workshopRequired": False, "read": True, "tree": True, "hash": True, "transactionalWrites": True, "gitMutation": False},
        "assets": {"innerAsset": True, "assetFabric": assets.get("assetFabricVersion"), "assetHandsVersion": assets.get("assetHandsVersion"), "handCount": assets.get("handCount"), "workshopRequired": False, "internetFallback": False},
        "neural": {"startingWeightsBundled": weights["startingWeightsBundled"], "weightFiles": weights["recognizedWeightFiles"], "internetFallback": False, "automaticExperienceThreshold": 8, "completionRequiredForLearning": False, "failedAttemptSupervised": False, "experienceGrowthSmoke": "inventory/AUTOLEARN_EXPERIENCE_SMOKE.json", "behaviorQuality": autolearn["behaviorQuality"]},
    },
    "experienceExport": True,
    "rollbackStorage": True,
    "offlineRequired": True,
    "authority": "NONE",
}
(CARTRIDGE / "walmi.manifest.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
(CARTRIDGE / "README_FIRST.txt").write_text(
    "WALMI PC FULL STACK v0.2\n\n"
    "This cartridge contains a standalone Windows host. Workshop is optional. "
    "Workspace access is attached per folder, and the latest pinned Asset Fabric/Asset Hands code is copied into WALMI itself. "
    "Network removal removes external information sources, not WALMI's local body.\n",
    encoding="utf-8",
)

required = [
    "walmi.manifest.json", "runtime/browser-entry.js", "runtime/go/walmi-axm-mirror.wasm", "runtime/go/wasm_exec.js",
    "native/windows/WALMI_WINDOWS_FULL_HOST_v0_2.zip", "source/WALMI_CURRENT_STACK_v0_2.zip",
    "history/WALDO_ALL_EXPERIMENTS.bundle", "inventory/CURRENT_SOURCE.json", "inventory/MODEL_WEIGHT_SCAN.json",
    "inventory/ASSET_CAPABILITY_RECEIPT.json", "inventory/WORKSPACE_HAND_SMOKE.json", "inventory/AUTOLEARN_EXPERIENCE_SMOKE.json", "inventory/GO_TESTS.txt",
]
missing = [name for name in required if not (CARTRIDGE / name).is_file()]
if missing:
    raise SystemExit(f"FINAL_COMPLETENESS_HOLD {missing}")

rows = []
file_manifest_path = inventory / "CARTRIDGE_FILE_MANIFEST.json"
for path in sorted(p for p in CARTRIDGE.rglob("*") if p.is_file() and p != file_manifest_path):
    rows.append({"path": path.relative_to(CARTRIDGE).as_posix(), "bytes": path.stat().st_size, "sha256": sha256(path)})
file_manifest_path.write_text(
    json.dumps({"schema": "walmi.cartridge-files/v1", "count": len(rows), "files": rows}, indent=2), encoding="utf-8"
)

out = DIST / "WALMI_PC_FULL_STACK_v0_2.zip"
zip_tree(CARTRIDGE, out)
digest = sha256(out)
(DIST / "WALMI_PC_FULL_STACK_v0_2.sha256.txt").write_text(f"{digest}  {out.name}\n", encoding="utf-8")
print("FINAL_WALMI_PC_V02_ZIP_PASS", out.stat().st_size, digest, "payload_files", len(rows), "asset_hands", assets.get("handCount"))
