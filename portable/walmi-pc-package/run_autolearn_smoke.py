from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import subprocess
from pathlib import Path


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def run(arguments: list[str], cwd: Path, environment: dict[str, str]) -> subprocess.CompletedProcess[str]:
    completed = subprocess.run(
        arguments,
        cwd=cwd,
        env=environment,
        text=True,
        capture_output=True,
        check=False,
    )
    if completed.returncode != 0:
        raise SystemExit(
            f"autolearn smoke command failed ({completed.returncode}): {subprocess.list2cmdline(arguments)}\n"
            f"stdout:\n{completed.stdout}\nstderr:\n{completed.stderr}"
        )
    return completed


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", type=Path, default=Path("dist/host"))
    parser.add_argument(
        "--smoke-root", type=Path, default=Path("dist/autolearn-package-smoke")
    )
    parser.add_argument(
        "--receipt",
        type=Path,
        default=Path("dist/inventory/AUTOLEARN_EXPERIENCE_SMOKE.json"),
    )
    arguments = parser.parse_args()

    repository = Path.cwd().resolve()
    host = arguments.host.resolve()
    smoke = arguments.smoke_root.resolve()
    receipt_path = arguments.receipt.resolve()
    if smoke.exists():
        raise SystemExit(f"autolearn smoke root already exists: {smoke}")
    for directory in [smoke / "bin", smoke / "tools", smoke / "experience"]:
        directory.mkdir(parents=True, exist_ok=True)

    shutil.copy2(host / "bin/waldo.exe", smoke / "bin/waldo.exe")
    shutil.copy2(
        host / "tools/walmi_portable_bootstrap.py",
        smoke / "tools/walmi_portable_bootstrap.py",
    )
    shutil.copy2(host / "tools/walmi_autolearn.py", smoke / "tools/walmi_autolearn.py")
    source_projection = (
        repository
        / "experiments/website-creation-v0.51/failure-training-projection.jsonl"
    )
    copied_projection = smoke / "experience/training-projections.jsonl"
    shutil.copy2(source_projection, copied_projection)
    source_projection_sha256 = sha256(source_projection)

    runtime = host / "runtime"
    python = runtime / "python/python.exe"
    state = smoke / "state"
    environment = dict(os.environ)
    environment.update(
        {
            "WALMI_HOME": str(smoke),
            "WALDO_CONFIG": str(state / "waldo-config.json"),
            "PYTHONHOME": str(runtime / "python"),
            "TEMP": str(state / "tmp"),
            "TMP": str(state / "tmp"),
            "PIP_CACHE_DIR": str(state / "pip-cache"),
            "TORCH_HOME": str(state / "torch-cache"),
            "HF_HOME": str(state / "model-cache"),
            "XDG_CACHE_HOME": str(state / "model-cache"),
            "PYTHONPYCACHEPREFIX": str(state / "pycache"),
            "PATH": os.pathsep.join(
                [
                    str(runtime / "python"),
                    str(runtime / "python/Scripts"),
                    str(runtime / "node"),
                    environment.get("PATH", ""),
                ]
            ),
        }
    )

    run([str(python), str(smoke / "tools/walmi_portable_bootstrap.py")], repository, environment)
    model_name = "walmi-autolearn-smoke"
    autolearn = [
        str(python),
        str(smoke / "tools/walmi_autolearn.py"),
        "--model",
        model_name,
        "--threshold",
        "1",
        "--epochs",
        "1",
        "--learning-rate",
        "0.001",
    ]
    run(autolearn, repository, environment)
    run(autolearn, repository, environment)

    receipt_rows = [
        json.loads(line)
        for line in (smoke / "experience/autolearn-receipts.jsonl")
        .read_text(encoding="utf-8")
        .splitlines()
        if line.strip()
    ]
    if len(receipt_rows) != 2:
        raise SystemExit(f"expected two autolearn receipts, got {len(receipt_rows)}")
    trained, repeated = receipt_rows
    if (
        trained.get("state") != "EXPERIENCE_GROWTH_CHECKPOINT_PERSISTED"
        or trained.get("trainingInvoked") is not True
        or trained.get("weightsChanged") is not True
        or trained.get("directExperienceRewritten") is not False
        or trained.get("qualityImproved") is not None
    ):
        raise SystemExit(f"invalid persisted growth receipt: {trained}")
    if (
        repeated.get("state") != "WAITING_FOR_EXPERIENCE_THRESHOLD"
        or repeated.get("newRecords") != 0
        or repeated.get("trainingInvoked") is not False
        or repeated.get("weightsChanged") is not False
    ):
        raise SystemExit(f"autolearn repeated without new experience: {repeated}")

    model_bom_path = smoke / f"models/{model_name}/MODEL-BOM.json"
    model_bom = json.loads(model_bom_path.read_text(encoding="utf-8"))
    current_run = model_bom["current_run_id"]
    run_record = next(item for item in model_bom["runs"] if item["id"] == current_run)
    run_bom_path = smoke / "models" / model_name / run_record["run_bom"]
    run_bom = json.loads(run_bom_path.read_text(encoding="utf-8"))
    if run_bom["parameters"]["steps"] != 2:
        raise SystemExit(f"unexpected optimizer steps: {run_bom['parameters']['steps']}")

    chat = run(
        [
            str(smoke / "bin/waldo.exe"),
            "--json",
            "model",
            "chat",
            model_name,
            "What should you do after an incomplete attempt?",
            "--max-tokens",
            "24",
            "--temperature",
            "0",
            "--seed",
            "1",
        ],
        repository,
        environment,
    )
    chat_receipt = json.loads(chat.stdout)
    if chat_receipt.get("run_id") != current_run:
        raise SystemExit("inference did not reopen the current growth checkpoint")

    compose = json.loads(
        (smoke / "experience/training/experience-compose.json").read_text(
            encoding="utf-8"
        )
    )
    stage = compose["stages"][0]
    if (
        stage["objective"] != "assistant-response-modeling"
        or stage["conversation"]["supervised_roles"] != ["assistant"]
    ):
        raise SystemExit("autolearn compose does not isolate the assistant lesson target")
    if sha256(copied_projection) != source_projection_sha256:
        raise SystemExit("source experience projection changed during autolearn")

    output = {
        "schema": "walmi.autolearn-experience-smoke/v1",
        "status": "PASS",
        "sourceOutcome": "INCONCLUSIVE",
        "sourceProjectionSHA256": source_projection_sha256,
        "sourceProjectionUnchanged": True,
        "trainingObjective": stage["objective"],
        "supervisedRoles": stage["conversation"]["supervised_roles"],
        "failedAttemptRole": "tool",
        "optimizerSteps": run_bom["parameters"]["steps"],
        "currentRunID": current_run,
        "currentWeightSHA256": trained["currentWeightSHA256"],
        "checkpointReopened": True,
        "repeatedWithoutNewExperience": True,
        "secondRunTrainingInvoked": False,
        "directExperienceRewritten": False,
        "weightsChanged": True,
        "behaviorQuality": "UNKNOWN_NO_HELD_OUT_EVALUATION",
        "networkUsed": False,
        "smokeRoot": str(smoke),
    }
    receipt_path.parent.mkdir(parents=True, exist_ok=True)
    receipt_path.write_text(json.dumps(output, indent=2) + "\n", encoding="utf-8")
    shutil.copy2(receipt_path, host / "AUTOLEARN_EXPERIENCE_SMOKE.json")
    print(
        "WALMI_AUTOLEARN_EXPERIENCE_PASS",
        current_run,
        trained["currentWeightSHA256"],
        "quality",
        output["behaviorQuality"],
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
