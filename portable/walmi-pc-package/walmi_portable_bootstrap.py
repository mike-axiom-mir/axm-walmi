from __future__ import annotations

import json
import os
import subprocess
from pathlib import Path


def inside(root: Path, candidate: Path) -> bool:
    try:
        candidate.relative_to(root)
    except ValueError:
        return False
    return True


def write_json_atomic(path: Path, value: dict[str, object]) -> None:
    temporary = path.with_name(f".{path.name}.tmp")
    with temporary.open("w", encoding="utf-8", newline="\n") as stream:
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def main() -> int:
    script_host = Path(__file__).resolve().parent.parent
    host = Path(os.environ.get("WALMI_HOME", script_host)).resolve()
    if host != script_host:
        raise SystemExit(
            f"WALMI_HOME {host} does not match this portable host {script_host}"
        )

    # The immutable runtime may be cached on the machine while the personal
    # WALMI data travels beside a cartridge. Existing standalone hosts keep
    # their original behavior because WALMI_DATA_HOME defaults to the host.
    data_home = Path(os.environ.get("WALMI_DATA_HOME", host)).resolve()
    data_home.mkdir(parents=True, exist_ok=True)
    state = data_home / "state"
    paths = {
        "index": state / "index",
        "lookaside": state / "lookaside",
        "lookaside_cache": state / "lookaside-cache",
        "lookaside_scratch": state / "lookaside-scratch",
        "ingest_staging": state / "ingest-staging",
        "models": data_home / "models",
        "experience": data_home / "experience",
        "checkpoints": data_home / "checkpoints",
        "rollback": data_home / "rollback",
        "tmp": state / "tmp",
        "pip_cache": state / "pip-cache",
        "torch_cache": state / "torch-cache",
        "model_cache": state / "model-cache",
        "pycache": state / "pycache",
    }
    for path in paths.values():
        path.mkdir(parents=True, exist_ok=True)

    config_path = Path(
        os.environ.get("WALDO_CONFIG", state / "waldo-config.json")
    ).resolve()
    if not inside(data_home, config_path):
        raise SystemExit(
            f"portable WALDO_CONFIG must remain inside {data_home}, got {config_path}"
        )
    config_path.parent.mkdir(parents=True, exist_ok=True)

    configuration: dict[str, object] = {}
    if config_path.is_file():
        try:
            loaded = json.loads(config_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as error:
            raise SystemExit(f"cannot read portable WALDO config {config_path}: {error}")
        if not isinstance(loaded, dict):
            raise SystemExit(f"portable WALDO config {config_path} is not an object")
        configuration.update(loaded)

    # These locations deliberately follow the extracted host whenever the ZIP
    # is moved. Unrelated user preferences are preserved.
    configuration["schema"] = 1
    configuration["index"] = str(paths["index"])
    lookaside = configuration.get("lookaside")
    if not isinstance(lookaside, dict):
        lookaside = {}
    lookaside.update(
        {
            "cache": str(paths["lookaside_cache"]),
            "scratch": str(paths["lookaside_scratch"]),
            "publish": {"url": paths["lookaside"].as_uri(), "workers": 4},
        }
    )
    configuration["lookaside"] = lookaside
    configuration["ingest"] = {"staging": str(paths["ingest_staging"])}
    model = configuration.get("model")
    if not isinstance(model, dict):
        model = {}
    model.update({"root": str(paths["models"]), "backend": "pytorch"})
    configuration["model"] = model
    ai = configuration.get("ai")
    if not isinstance(ai, dict) or not ai.get("provider"):
        configuration["ai"] = {"provider": "deterministic"}
    write_json_atomic(config_path, configuration)

    index_metadata = paths["index"] / "index.yaml"
    if not index_metadata.is_file():
        waldo = host / "bin" / "waldo.exe"
        completed = subprocess.run(
            [str(waldo), "index", "init", str(paths["index"])],
            cwd=host,
            text=True,
            capture_output=True,
            check=False,
        )
        if completed.returncode != 0:
            detail = completed.stderr.strip() or completed.stdout.strip()
            raise SystemExit(f"cannot initialize portable WALDO index: {detail}")

    print(f"WALMI_PORTABLE_BOOTSTRAP_OK runtime={host} data={data_home}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
