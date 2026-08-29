#!/usr/bin/env python3
"""Bounded WALMI play bridge for the three local AXM simulator worlds."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time
import uuid
import zipfile


PLAYER_REQUEST_SCHEMA = "axm.walmi.simulator-player-request/v1"
PLAYER_RESPONSE_SCHEMA = "axm.walmi.simulator-player-response/v1"
EPISODE_SCHEMA = "axm.walmi.simulator-experience/v1"
SAVE_SLOT_SCHEMA = "axm.walmi.simulator-save-slot/v1"
CAPABILITY_GAP_SCHEMA = "axm.walmi.simulator-capability-gap/v1"
DETERMINISTIC_REPAIR_SCHEMA = "axm.walmi.simulator-player-repair/v1"
MODEL_PLAYER_CODEC_SCHEMA = "axm.walmi.simulator-menu-index-codec/v1"
PROJECTION_SCHEMA = "axm.waldo.experience-trajectory-training/v0.51"
TRAJECTORY_SCHEMA = "axm.waldo.experience-trajectory/v0.51"
MAX_TURNS = 512
MAX_PROCESS_OUTPUT = 2 * 1024 * 1024
MAX_MODEL_MENU_ACTIONS = 8
MAX_MODEL_PROMPT_BYTES = 480
DEFAULT_ROOTS = {
    "theme-park": Path(r"D:\AXM_ACTIVE\axm-theme-park-simulator"),
    "factual-space": Path(r"D:\AXM_ACTIVE\axm-factual-space-simulator"),
    "living-city": Path(r"D:\AXM_ACTIVE\axm-living-city-simulator"),
}
SPACE_DETERMINISTIC_AI_COMMAND = """
import json
import sys
from argparse import Namespace
from pathlib import Path
from axm_star_sim.cli import _execute_decision, _load_verified
from axm_star_sim.command import plan_command

output = Path(sys.argv[1])
action_id = sys.argv[2]
system, state, _events = _load_verified(output)
proposal = {
    "actor": "ai",
    "action": action_id,
    "rationale": "Selected from WALMI's visible bounded action menu.",
    "created_at": f"deterministic:{system['system_id']}:turn:{int(state['turn']) + 1}",
}
plan = plan_command(system=system, state=state, mode="ai_command", ai_proposal=proposal)
args = Namespace(output=output, entropy_mode="deterministic", beacon_file=None, reveal=None)
print(json.dumps(_execute_decision(args, system, state, plan["decision"]), ensure_ascii=False))
"""


def canonical_bytes(value: object) -> bytes:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode("utf-8")


def digest(value: object) -> str:
    return hashlib.sha256(canonical_bytes(value)).hexdigest()


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def safe_identity(value: str, label: str) -> str:
    normalized = str(value).strip()
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,79}", normalized):
        raise ValueError(
            f"{label} must start with a letter or digit and contain at most 80 letters, digits, dot, dash, or underscore characters"
        )
    if normalized in {".", ".."}:
        raise ValueError(f"{label} cannot be a path marker")
    return normalized


def atomic_write_json(path: Path, value: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f"{path.name}.{os.getpid()}.{uuid.uuid4().hex}.tmp")
    temporary.write_bytes(canonical_bytes(value) + b"\n")
    os.replace(temporary, path)


def sealed_slot_metadata(value: dict) -> dict:
    sealed = {**value, "metadataSha256": ""}
    sealed["metadataSha256"] = digest(sealed)
    return sealed


def validate_slot_metadata(value: dict, owner: str, world: str, slot: str) -> dict:
    if value.get("schema") != SAVE_SLOT_SCHEMA:
        raise ValueError("save slot metadata has an unsupported schema")
    for name, expected in (("owner", owner), ("world", world), ("slot", slot)):
        if value.get(name) != expected:
            raise ValueError(f"save slot {name} does not match the requested identity")
    supplied = value.get("metadataSha256")
    if not isinstance(supplied, str) or len(supplied) != 64:
        raise ValueError("save slot metadata has no valid digest")
    if digest({**value, "metadataSha256": ""}) != supplied:
        raise ValueError("save slot metadata digest does not match")
    return value


def open_save_slot(
    data_home: Path,
    owner_value: str,
    world: str,
    slot_value: str,
    requested_seed: str | None,
    new_save: bool,
) -> dict:
    owner = safe_identity(owner_value, "save owner")
    slot = safe_identity(slot_value, "save slot")
    saves_root = (data_home / "simulator-saves").resolve()
    slot_dir = (saves_root / owner / world / slot).resolve()
    try:
        slot_dir.relative_to(saves_root)
    except ValueError as error:
        raise ValueError("save slot path escapes the configured save root") from error
    metadata_path = slot_dir / "SAVE-SLOT.json"
    if new_save and slot_dir.exists():
        raise FileExistsError(f"refusing to overwrite existing save slot: {owner}/{world}/{slot}")
    if metadata_path.exists():
        metadata = validate_slot_metadata(read_json(metadata_path), owner, world, slot)
        if requested_seed is not None and requested_seed != metadata.get("seed"):
            raise ValueError(
                f"save slot seed is {metadata.get('seed')!r}; refusing requested seed {requested_seed!r}"
            )
        seed = str(metadata["seed"])
        resumed = True
    else:
        if slot_dir.exists() and any(slot_dir.iterdir()):
            raise ValueError(f"save slot directory is incomplete and has no metadata: {slot_dir}")
        seed = requested_seed or "WALMI-SIM-001"
        created_at = utc_now()
        metadata = sealed_slot_metadata({
            "schema": SAVE_SLOT_SCHEMA,
            "owner": owner,
            "world": world,
            "slot": slot,
            "seed": seed,
            "createdAt": created_at,
            "updatedAt": created_at,
            "stateDigest": None,
            "totalSessions": 0,
            "totalTurns": 0,
            "totalAcceptedTurns": 0,
            "totalInterruptedSessions": 0,
            "totalWallSeconds": 0.0,
            "activeSession": None,
            "lastSession": None,
        })
        slot_dir.mkdir(parents=True, exist_ok=True)
        atomic_write_json(metadata_path, metadata)
        resumed = False
    return {
        "owner": owner,
        "world": world,
        "slot": slot,
        "slotDir": slot_dir,
        "metadataPath": metadata_path,
        "metadata": metadata,
        "seed": seed,
        "resumed": resumed,
    }


def write_slot_metadata(slot_info: dict, updates: dict) -> dict:
    metadata = sealed_slot_metadata({**slot_info["metadata"], **updates})
    atomic_write_json(slot_info["metadataPath"], metadata)
    slot_info["metadata"] = metadata
    return metadata


def read_json(path: Path) -> dict:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain one JSON object")
    return value


def parse_json_object(text: str, label: str) -> dict:
    if len(text.encode("utf-8")) > MAX_PROCESS_OUTPUT:
        raise ValueError(f"{label} output exceeds {MAX_PROCESS_OUTPUT} bytes")
    value = json.loads(text)
    if not isinstance(value, dict):
        raise ValueError(f"{label} must return one JSON object")
    return value


def run_process(command: list[str], *, cwd: Path, input_value: dict | None = None, env: dict | None = None) -> dict:
    result = subprocess.run(
        command,
        cwd=cwd,
        env=env,
        input=None if input_value is None else canonical_bytes(input_value),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=180,
        check=False,
    )
    stdout = result.stdout.decode("utf-8", errors="replace")
    stderr = result.stderr.decode("utf-8", errors="replace")
    if result.returncode != 0:
        raise RuntimeError(
            f"command refused with exit {result.returncode}: {' '.join(command[:3])}\n{stderr[-4000:]}"
        )
    return parse_json_object(stdout, command[0])


def append_jsonl(path: Path, value: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    lock = path.with_name(path.name + ".lock")
    deadline = time.monotonic() + 10
    descriptor = None
    while descriptor is None:
        try:
            descriptor = os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
        except FileExistsError:
            if time.monotonic() >= deadline:
                raise TimeoutError(f"timed out waiting for ledger lock {lock}")
            time.sleep(0.05)
    try:
        line = canonical_bytes(value) + b"\n"
        with path.open("ab", buffering=0) as stream:
            stream.write(line)
            os.fsync(stream.fileno())
    finally:
        os.close(descriptor)
        lock.unlink(missing_ok=True)


def default_data_home() -> Path:
    configured = os.environ.get("WALMI_DATA_HOME", "").strip()
    if configured:
        return Path(configured)
    configured_home = os.environ.get("WALMI_HOME", "").strip()
    if configured_home:
        return Path(configured_home)
    bundled = bundled_data_home(Path(__file__))
    if bundled is not None:
        return bundled
    raise RuntimeError(
        "WALMI data home is ambiguous; use the packaged launcher or pass --data-home explicitly"
    )


def bundled_data_home(script_path: Path) -> Path | None:
    resolved = script_path.resolve()
    if resolved.parent.name.lower() != "tools" or len(resolved.parents) < 2:
        return None
    host = resolved.parents[1]
    required = ("bin", "models", "state", "tools")
    return host if all((host / name).is_dir() for name in required) else None


def infer_waldo_bin(configured: Path | None, data_home: Path) -> Path | None:
    if configured is not None:
        return configured.resolve()
    bundled = (data_home.resolve() / "bin" / "waldo.exe").resolve()
    return bundled if bundled.is_file() else None


def waldo_environment(
    data_home: Path, waldo_bin: Path | None = None
) -> dict[str, str]:
    environment = os.environ.copy()
    runtime_host = (
        waldo_bin.resolve().parent.parent if waldo_bin is not None else data_home.resolve()
    )
    python_root = runtime_host / "runtime" / "python"
    state = data_home.resolve() / "state"
    portable_path = [python_root, python_root / "Scripts", runtime_host / "runtime" / "node"]
    environment["PATH"] = os.pathsep.join(
        [*(str(path) for path in portable_path), environment.get("PATH", "")]
    )
    environment["PYTHONHOME"] = str(python_root)
    environment["WALMI_HOME"] = str(runtime_host)
    environment["WALDO_CONFIG"] = str(
        (state / "waldo-config.json").resolve()
    )
    environment["TEMP"] = str(state / "tmp")
    environment["TMP"] = str(state / "tmp")
    environment["PIP_CACHE_DIR"] = str(state / "pip-cache")
    environment["TORCH_HOME"] = str(state / "torch-cache")
    environment["HF_HOME"] = str(state / "model-cache")
    environment["XDG_CACHE_HOME"] = str(state / "model-cache")
    environment["PYTHONPYCACHEPREFIX"] = str(state / "pycache")
    return environment


class NodeWorld:
    def __init__(self, world: str, root: Path, state_path: Path, seed: str, adapter: Path):
        self.world = world
        self.root = root
        self.state_path = state_path
        self.seed = seed
        self.adapter = adapter

    def _call(self, command: str, action_id: str | None = None) -> dict:
        request = {
            "schema": "axm.walmi.simulator-adapter-request/v1",
            "command": command,
            "world": self.world,
            "worldRoot": str(self.root),
            "statePath": str(self.state_path),
            "seed": self.seed,
        }
        if action_id is not None:
            request["actionId"] = action_id
        return run_process(["node", str(self.adapter)], cwd=self.adapter.parent, input_value=request)

    def create(self) -> dict:
        return self._call("create")["after"]

    def observe(self) -> dict:
        return self._call("observe")["after"]

    def open_or_create(self) -> tuple[dict, bool]:
        if self.state_path.is_file():
            return self.observe(), True
        if self.state_path.exists():
            raise ValueError(f"simulator state path is not a file: {self.state_path}")
        return self.create(), False

    def act(self, action_id: str) -> tuple[dict, dict]:
        response = self._call("act", action_id)
        return response["outcome"], response["after"]


class SpaceWorld:
    def __init__(self, root: Path, output: Path, seed: str):
        self.root = root
        self.output = output
        self.seed = seed
        self.environment = os.environ.copy()
        self.environment["PYTHONPATH"] = str(root / "src")
        self.environment["PYTHONDONTWRITEBYTECODE"] = "1"

    def _cli(self, arguments: list[str]) -> dict:
        command = [
            sys.executable,
            "-c",
            "from axm_star_sim.cli import main; raise SystemExit(main())",
            *arguments,
        ]
        return run_process(command, cwd=self.root, env=self.environment)

    def create(self) -> dict:
        self._cli(["generate", "--seed", self.seed, "--output", str(self.output), "--command-mode", "ai_command"])
        return self.observe()

    def open_or_create(self) -> tuple[dict, bool]:
        state_path = self.output / "runtime_state.json"
        if state_path.is_file():
            verification = self.verify()
            if verification.get("valid") is False or verification.get("status") in {"FAIL", "FAILED"}:
                raise ValueError("factual-space save ledger did not verify before resume")
            return self.observe(), True
        if self.output.exists() and any(self.output.iterdir()):
            raise ValueError(f"factual-space save directory is incomplete: {self.output}")
        return self.create(), False

    def observe(self) -> dict:
        state = read_json(self.output / "runtime_state.json")
        menu = state.get("action_menu") or {}
        actions = []
        for item in list(menu.get("actions") or [])[:32]:
            actions.append({
                "id": str(item.get("action_id")),
                "label": str(item.get("label")),
                "description": str(item.get("intent") or ""),
                "category": str(item.get("category") or "unknown"),
            })
        state_bytes = (self.output / "runtime_state.json").read_bytes()
        return {
            "schema": "axm.walmi.simulator-observation/v1",
            "world": "factual-space",
            "version": "local-source",
            "stateDigest": hashlib.sha256(state_bytes).hexdigest(),
            "visible": {
                "turn": state.get("turn"),
                "resources": state.get("resources"),
                "activeThreads": menu.get("active_thread_ids") or [],
                "recentObservations": list(state.get("observations") or [])[-3:],
            },
            "actions": actions,
        }

    def act(self, action_id: str) -> tuple[dict, dict]:
        current = self.observe()
        if action_id not in {item["id"] for item in current["actions"]}:
            return {"ok": False, "reason": "actionId is not present in the current visible menu"}, current
        resolved = run_process(
            [sys.executable, "-c", SPACE_DETERMINISTIC_AI_COMMAND, str(self.output), action_id],
            cwd=self.root,
            env=self.environment,
        )
        return {
            "ok": resolved.get("status") == "event_resolved",
            "reason": resolved.get("outcome") or resolved.get("status"),
            "eventHash": resolved.get("event_hash"),
            "action": resolved.get("action"),
        }, self.observe()

    def verify(self) -> dict:
        return self._cli(["verify-ledger", "--output", str(self.output)])


def validate_player_response(value: dict, observation: dict) -> dict:
    if set(value) != {"schema", "actionId"}:
        raise ValueError("player response must contain exactly schema and actionId")
    if value.get("schema") != PLAYER_RESPONSE_SCHEMA:
        raise ValueError(f"player response schema must be {PLAYER_RESPONSE_SCHEMA}")
    action_id = value.get("actionId")
    if not isinstance(action_id, str) or action_id not in {item["id"] for item in observation["actions"]}:
        raise ValueError("player actionId is not present in the visible action menu")
    return value


def deterministic_player(request: dict) -> dict:
    actions = request["observation"]["actions"]
    if not actions:
        raise ValueError("world exposed no actions")
    selector = int(hashlib.sha256(f"{request['world']}|{request['seed']}|{request['turn']}".encode()).hexdigest()[:16], 16)
    return {"schema": PLAYER_RESPONSE_SCHEMA, "actionId": actions[selector % len(actions)]["id"]}


def bounded_utf8_label(value: object, byte_budget: int = 36) -> str:
    normalized = " ".join(str(value).split())
    bounded = normalized.encode("utf-8")[:byte_budget].decode("utf-8", errors="ignore")
    return bounded.strip() or "visible action"


def compact_visible_state(observation: dict, byte_budget: int = 120) -> str:
    visible = observation.get("visible") or {}
    candidates = []
    for key in (
        "day", "minute", "turn", "cash", "visitorsPresent", "lifetimeVisitors",
        "ticketPrice", "parkOpen", "time", "resources", "activeThreads", "player",
    ):
        if key not in visible:
            continue
        rendered = json.dumps(visible[key], ensure_ascii=False, sort_keys=True, separators=(",", ":"))
        candidates.append(f"{key}={rendered}")
    text = " ".join(candidates) or f"state={observation.get('stateDigest', '')[:16]}"
    return text.encode("utf-8")[:byte_budget].decode("utf-8", errors="ignore").strip()


def model_player_codec(request: dict) -> dict:
    actions = request["observation"]["actions"]
    if not actions:
        raise ValueError("world exposed no actions for the neural menu codec")
    selector_material = (
        f"{request['world']}|{request['seed']}|{request['turn']}|"
        f"{request['observation']['stateDigest']}"
    )
    start = int(hashlib.sha256(selector_material.encode()).hexdigest()[:16], 16) % len(actions)
    selected = [
        actions[(start + offset) % len(actions)]
        for offset in range(MAX_MODEL_MENU_ACTIONS)
    ]
    prompt = None
    for label_budget, state_budget in ((36, 120), (28, 80), (20, 48), (12, 0)):
        lines = [
            "WALMI_MENU_INDEX_V1",
            f"world={request['world']}",
        ]
        if state_budget:
            lines.append(compact_visible_state(request["observation"], state_budget))
        lines.append("Choose one visible action. Return one digit only.")
        for index, action in enumerate(selected):
            label = bounded_utf8_label(action.get("label") or action["id"], label_budget)
            lines.append(f"{index}={label}")
        candidate = "\n".join(lines)
        if len(candidate.encode("utf-8")) <= MAX_MODEL_PROMPT_BYTES:
            prompt = candidate
            break
    if prompt is None:
        raise ValueError("neural menu prompt cannot fit its fixed visible-action codec")
    return {
        "schema": MODEL_PLAYER_CODEC_SCHEMA,
        "prompt": prompt,
        "menuActionIds": [action["id"] for action in selected],
        "maxTokens": 4,
        "temperature": 0.0,
        "topP": 1.0,
    }


def decode_model_choice(text: str, codec: dict) -> dict:
    normalized = text.strip()
    if not re.fullmatch(r"[0-7]", normalized):
        raise ValueError(
            "menu-index/v1 output was not exactly one available digit "
            f"(bytes={len(text.encode('utf-8'))}, sha256={hashlib.sha256(text.encode()).hexdigest()})"
        )
    index = int(normalized)
    actions = codec["menuActionIds"]
    if index >= len(actions):
        raise ValueError(f"menu-index/v1 choice {index} is outside the visible codec menu")
    return {"schema": PLAYER_RESPONSE_SCHEMA, "actionId": actions[index]}


def repair_player_gap(request: dict, failure: str) -> dict:
    """Apply one menu-bounded repair without erasing the failed player attempt."""
    repair_request = request
    codec = request.get("playerCodec")
    if isinstance(codec, dict) and codec.get("schema") == MODEL_PLAYER_CODEC_SCHEMA:
        allowed = set(codec["menuActionIds"])
        repair_request = {
            **request,
            "observation": {
                **request["observation"],
                "actions": [
                    action for action in request["observation"]["actions"] if action["id"] in allowed
                ],
            },
        }
    response = validate_player_response(deterministic_player(repair_request), request["observation"])
    return {
        "schema": DETERMINISTIC_REPAIR_SCHEMA,
        "state": "BOUNDED_VISIBLE_ACTION_SELECTED",
        "hand": "deterministic-visible-action-selector/v1",
        "failure": failure[:1000],
        "response": response,
        "authority": {
            "hiddenWorldAccess": False,
            "directFileMutation": False,
            "training": False,
            "promotion": False,
            "canon": False,
        },
    }


def capability_gap_record(episode: dict, step: dict) -> dict:
    repair = step.get("deterministicRepair")
    failure = repair["failure"] if repair else str(step.get("playerFailure") or "player action failed")
    record = {
        "schema": CAPABILITY_GAP_SCHEMA,
        "id": "",
        "capabilityId": "simulator.player.choose-visible-action",
        "gapType": "CONTRACT",
        "world": episode["world"],
        "player": episode["player"],
        "episodeReceiptSha256": episode["receiptSha256"],
        "observedAt": episode["observedAt"],
        "evidence": {
            "turn": step["turn"],
            "requestSha256": step["requestSha256"],
            "failure": failure,
        },
        "repair": {
            "state": (
                "APPLIED" if repair and step["outcome"].get("ok")
                else "REFUSED" if repair
                else "NOT_APPLIED"
            ),
            "hand": repair["hand"] if repair else None,
            "response": repair["response"] if repair else None,
            "worldActionApplied": bool(repair and step["outcome"].get("ok")),
        },
        "authority": {
            "training": False,
            "promotion": False,
            "canon": False,
            "persistentWorldAction": False,
        },
    }
    record["id"] = "gap-" + digest(record)[:24]
    return record


def describe_attempt(step: dict) -> str:
    repair = step.get("deterministicRepair")
    if repair:
        return (
            f"turn {step['turn']}: player failure: {repair['failure']}; "
            f"deterministic repair selected {step.get('actionId') or 'no valid action'}"
        )
    if step.get("playerFailure"):
        return f"turn {step['turn']}: player failure: {step['playerFailure']}; no replacement action was applied"
    return f"turn {step['turn']}: {step.get('actionId') or 'no valid action'}"


def experience_prompt(episode: dict) -> str:
    for step in episode["steps"]:
        codec = step.get("playerCodec")
        if isinstance(codec, dict) and codec.get("schema") == MODEL_PLAYER_CODEC_SCHEMA:
            return codec["prompt"]
    return f"Choose actions from the visible {episode['world']} simulator menu."


def experience_lesson(all_ok: bool, gap_records: list[dict], steps: list[dict]) -> str:
    for step in steps:
        codec = step.get("playerCodec")
        if (
            isinstance(codec, dict)
            and codec.get("schema") == MODEL_PLAYER_CODEC_SCHEMA
            and step["outcome"].get("ok")
            and step.get("actionId") in codec["menuActionIds"]
        ):
            # The digit proves only that the named model can satisfy the codec.
            # Simulator acceptance still does not prove strategic quality.
            return str(codec["menuActionIds"].index(step["actionId"]))
    accepted_repairs = [
        gap["repair"]["response"]
        for gap in gap_records
        if gap["repair"].get("worldActionApplied")
    ]
    if accepted_repairs:
        # This target proves only response-contract compliance. The HARMFUL
        # outcome signal and tool message retain the original neural failure.
        return canonical_bytes(accepted_repairs[0]).decode("utf-8")
    if all_ok:
        return "The simulator accepted the selected actions and preserved deterministic evidence. Acceptance alone does not prove the strategy was good; compare future outcomes before preferring it."
    return "Treat a refused or invalid simulator action as direct experience: select only a currently visible action ID, preserve the refusal reason, and do not imitate the failed attempt."


def organ_player(request: dict, organ_path: Path) -> dict:
    suffix = organ_path.suffix.casefold()
    if suffix == ".py":
        command = [sys.executable, str(organ_path)]
    elif suffix in {".js", ".mjs", ".cjs"}:
        command = ["node", str(organ_path)]
    else:
        command = [str(organ_path)]
    return run_process(command, cwd=organ_path.parent, input_value=request)


def extract_model_text(value: object) -> str | None:
    if isinstance(value, str):
        return value
    if isinstance(value, dict):
        for key in ("text", "response", "content"):
            found = value.get(key)
            if isinstance(found, str):
                return found
        for key in ("result", "message", "output"):
            found = extract_model_text(value.get(key))
            if found:
                return found
    return None


def model_player(
    request: dict, waldo_bin: Path, model: str, environment: dict[str, str]
) -> dict:
    codec = request.get("playerCodec")
    if not isinstance(codec, dict) or codec.get("schema") != MODEL_PLAYER_CODEC_SCHEMA:
        raise ValueError("model player request has no supported neural menu codec")
    seed = int(hashlib.sha256(
        f"{request['episodeId']}|{request['turn']}".encode()
    ).hexdigest()[:8], 16)
    result = run_process([
        str(waldo_bin), "--json", "model", "chat", model, codec["prompt"],
        "--max-tokens", str(codec["maxTokens"]),
        "--temperature", str(codec["temperature"]),
        "--top-p", str(codec["topP"]),
        "--seed", str(seed),
    ], cwd=waldo_bin.parent, env=environment)
    text = extract_model_text(result)
    if not text:
        raise ValueError("WALDO model response contained no text")
    return decode_model_choice(text, codec)


def select_action(args: argparse.Namespace, request: dict) -> dict:
    if args.player == "deterministic":
        result = deterministic_player(request)
    elif args.player == "organ":
        if args.organ is None:
            raise ValueError("--organ is required when --player organ")
        result = organ_player(request, args.organ.resolve())
    else:
        if args.waldo_bin is None or not args.model:
            raise ValueError(
                "model player requires --model and either --waldo-bin or bundled bin/waldo.exe"
            )
        result = model_player(
            request, args.waldo_bin.resolve(), args.model, args.waldo_environment
        )
    return validate_player_response(result, request["observation"])


def summarize_delta(before: dict, after: dict) -> dict:
    changes = {}
    for key in sorted(set(before.get("visible", {})) | set(after.get("visible", {}))):
        old = before.get("visible", {}).get(key)
        new = after.get("visible", {}).get(key)
        if old != new:
            changes[key] = {"before": old, "after": new}
    return {"stateChanged": before.get("stateDigest") != after.get("stateDigest"), "visibleChanges": changes}


def compact_run(run_dir: Path) -> tuple[Path, int, int]:
    archive = run_dir.parent / f"{run_dir.name}.zip"
    files = [item for item in run_dir.rglob("*") if item.is_file()]
    with zipfile.ZipFile(archive, "x", compression=zipfile.ZIP_DEFLATED, compresslevel=6) as output:
        for item in files:
            output.write(item, item.relative_to(run_dir.parent).as_posix())
    with zipfile.ZipFile(archive) as check:
        bad = check.testzip()
        if bad:
            archive.unlink(missing_ok=True)
            raise RuntimeError(f"run archive verification failed at {bad}")
    expanded_bytes = sum(item.stat().st_size for item in files)
    shutil.rmtree(run_dir)
    return archive, len(files), expanded_bytes


def direct_projection(episode: dict, signal: str, lesson: str, projections: Path) -> dict:
    attempts = []
    outcomes = []
    for step in episode["steps"]:
        attempts.append(describe_attempt(step))
        outcomes.append(f"turn {step['turn']}: {step['outcome'].get('reason', 'no reason')}")
    projection = {
        "schema": PROJECTION_SCHEMA,
        "id": episode["episodeId"],
        "dataClass": "OBSERVED_EXECUTION_TRACE",
        "sourceReceiptSha256": episode["receiptSha256"],
        "observedAt": episode["observedAt"],
        "outcomeSignal": signal,
        "targetKind": "OUTCOME_CONDITIONED_REFLECTION",
        "trainingObjective": "assistant-response-modeling",
        "supervisedRoles": ["assistant"],
        "messages": [
            {"role": "user", "content": experience_prompt(episode)},
            {"role": "tool", "content": "Observed attempt:\n" + "\n".join(attempts) + "\n\nObserved outcome:\n" + "\n".join(outcomes)},
            {"role": "assistant", "content": lesson},
        ],
    }
    append_jsonl(projections, projection)
    return {"mode": "DIRECT_COMPATIBLE_V051", "projectionSha256": digest(projection), "weightsChanged": False}


def project_with_waldo(episode: dict, signal: str, lesson: str, args: argparse.Namespace, records: Path, projections: Path, run_dir: Path) -> dict:
    if not getattr(args, "waldo_projection_supported", False):
        return direct_projection(episode, signal, lesson, projections)
    attempts = "; ".join(describe_attempt(step) for step in episode["steps"])
    outcomes = "; ".join(f"turn {step['turn']}: {step['outcome'].get('reason', 'unknown')}" for step in episode["steps"])
    draft = {
        "schema": TRAJECTORY_SCHEMA,
        "id": episode["episodeId"],
        "sourceReceiptSha256": episode["receiptSha256"],
        "observedAt": episode["observedAt"],
        "outcomeSignal": signal,
        "prompt": experience_prompt(episode),
        "attemptTrace": attempts,
        "observedOutcome": outcomes,
        "lesson": lesson,
        "targetKind": "OUTCOME_CONDITIONED_REFLECTION",
        "trainingObjective": "assistant-response-modeling",
        "completionRequired": False,
        "failedAttemptSupervised": False,
        "reflectionTargetSupervised": True,
        "authority": {"tool_execution": False, "training": False, "promotion": False, "canon": False, "world_action": False},
        "recordSha256": "",
    }
    turn = int(episode["steps"][0].get("turn", 0)) if len(episode["steps"]) == 1 else 0
    draft_path = run_dir / f"trajectory-draft-turn-{turn:04d}.json"
    draft_path.write_bytes(canonical_bytes(draft) + b"\n")
    try:
        return run_process([
            str(args.waldo_bin), "--json", "mirror", "experience", "project-trajectory", str(draft_path),
            "--record-to", str(records), "--learn-to", str(projections),
        ], cwd=args.waldo_bin.parent, env=args.waldo_environment)
    except Exception as error:
        fallback = direct_projection(episode, signal, lesson, projections)
        fallback["waldoProjectionUnavailable"] = str(error)[:1000]
        return fallback


def project_episode_turns(
    episode: dict,
    gap_records: list[dict],
    args: argparse.Namespace,
    records: Path,
    projections: Path,
    run_dir: Path,
) -> dict:
    gap_turns = {int(gap["evidence"]["turn"]): gap for gap in gap_records}
    receipts = []
    for step in episode["steps"]:
        turn = int(step["turn"])
        turn_gap = gap_turns.get(turn)
        turn_ok = bool(step["outcome"].get("ok"))
        signal = "INCONCLUSIVE" if turn_ok and turn_gap is None else "HARMFUL"
        turn_episode = {
            **episode,
            "episodeId": f"{episode['episodeId']}-turn-{turn:04d}",
            "steps": [step],
        }
        lesson = experience_lesson(turn_ok, [turn_gap] if turn_gap else [], [step])
        receipt = project_with_waldo(
            turn_episode, signal, lesson, args, records, projections, run_dir
        )
        receipts.append({"turn": turn, "outcomeSignal": signal, "receipt": receipt})
    return {
        "mode": "PER_TURN_OUTCOME_CONDITIONED_V051",
        "turnsProjected": len(receipts),
        "weightsChanged": False,
        "receipts": receipts,
    }


def _play_world(world_name: str, args: argparse.Namespace, data_home: Path, roots: dict[str, Path]) -> dict:
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%S.%fZ")
    episode_id = f"sim-{world_name}-{stamp}-{uuid.uuid4().hex[:8]}"
    run_dir = data_home / "experience" / "simulator-runs" / episode_id
    run_dir.mkdir(parents=True, exist_ok=False)
    slot_info = open_save_slot(
        data_home,
        args.save_owner,
        world_name,
        args.save_slot,
        args.seed,
        args.new_save,
    )
    adapter = Path(__file__).with_name("walmi_node_world_adapter.mjs").resolve()
    if world_name in {"theme-park", "living-city"}:
        world = NodeWorld(
            world_name,
            roots[world_name],
            slot_info["slotDir"] / "state.json",
            slot_info["seed"],
            adapter,
        )
    else:
        world = SpaceWorld(
            roots[world_name], slot_info["slotDir"] / "expedition", slot_info["seed"]
        )
    observation, world_resumed = world.open_or_create()
    if slot_info["resumed"] != world_resumed:
        raise ValueError("save metadata and authoritative simulator state disagree about resume state")

    prior_metadata_digest_matched = (
        not world_resumed
        or slot_info["metadata"].get("stateDigest") in {None, observation["stateDigest"]}
        or slot_info["metadata"].get("activeSession") is not None
    )
    if not prior_metadata_digest_matched:
        raise ValueError("save slot state changed without an active or completed session receipt")
    starting_state_digest = observation["stateDigest"]

    session_started_at = utc_now()
    session_started_mono = time.monotonic()
    deadline = (
        session_started_mono + args.session_minutes * 60.0
        if args.session_minutes is not None
        else None
    )
    base_turns = int(slot_info["metadata"].get("totalTurns", 0))
    base_accepted = int(slot_info["metadata"].get("totalAcceptedTurns", 0))
    base_interrupted = int(slot_info["metadata"].get("totalInterruptedSessions", 0))
    recovered_session = slot_info["metadata"].get("activeSession")
    write_slot_metadata(slot_info, {
        "updatedAt": session_started_at,
        "stateDigest": observation["stateDigest"],
        "activeSession": {
            "episodeId": episode_id,
            "startedAt": session_started_at,
            "paceSeconds": args.pace_seconds,
            "turnsObserved": 0,
            "recoveredFromInterruptedSession": recovered_session,
        },
    })

    steps = []
    last_decision_started = None
    for turn in range(1, args.turns + 1):
        if last_decision_started is not None and args.pace_seconds > 0:
            next_decision_due = last_decision_started + args.pace_seconds
            if deadline is not None and next_decision_due >= deadline:
                break
            remaining = next_decision_due - time.monotonic()
            if remaining > 0:
                time.sleep(remaining)
        if deadline is not None and time.monotonic() >= deadline:
            break
        decision_started_mono = time.monotonic()
        decision_started_at = utc_now()
        last_decision_started = decision_started_mono
        request = {
            "schema": PLAYER_REQUEST_SCHEMA,
            "episodeId": episode_id,
            "world": world_name,
            "seed": slot_info["seed"],
            "turn": turn,
            "observation": observation,
            "save": {
                "owner": slot_info["owner"],
                "slot": slot_info["slot"],
                "persistentTurn": base_turns + turn,
            },
            "authority": {"chooseVisibleActionOnly": True, "hiddenWorldAccess": False, "directFileMutation": False},
        }
        if args.player == "model":
            request["playerCodec"] = model_player_codec(request)
        deterministic_repair = None
        player_failure = None
        try:
            response = select_action(args, request)
        except Exception as error:
            response = None
            failure = f"player response refused: {error}"
            player_failure = failure
            if args.repair_policy == "visible-fallback":
                try:
                    deterministic_repair = repair_player_gap(request, failure)
                    action_id = deterministic_repair["response"]["actionId"]
                    outcome, after = world.act(action_id)
                except Exception as repair_error:
                    deterministic_repair = None
                    action_id = None
                    outcome = {
                        "ok": False,
                        "reason": f"{failure}; deterministic repair refused: {repair_error}",
                    }
                    after = observation
            else:
                action_id = None
                outcome = {"ok": False, "reason": failure}
                after = observation
        else:
            action_id = response["actionId"]
            try:
                outcome, after = world.act(action_id)
            except Exception as world_error:
                outcome = {"ok": False, "reason": f"simulator action refused: {world_error}"}
                after = observation
        action_applied_at = utc_now()
        step = {
            "turn": turn,
            "persistentTurn": base_turns + turn,
            "decisionStartedAt": decision_started_at,
            "actionAppliedAt": action_applied_at,
            "decisionWallSeconds": round(time.monotonic() - decision_started_mono, 6),
            "requestSha256": digest(request),
            "response": response,
            "actionId": action_id,
            "outcome": outcome,
            "beforeStateDigest": observation["stateDigest"],
            "afterStateDigest": after["stateDigest"],
            "delta": summarize_delta(observation, after),
        }
        if request.get("playerCodec"):
            step["playerCodec"] = request["playerCodec"]
        if response is None and deterministic_repair is not None:
            step["deterministicRepair"] = deterministic_repair
        if player_failure is not None:
            step["playerFailure"] = player_failure
        steps.append(step)
        observation = after
        accepted_count = sum(
            1
            for item in steps
            if item["outcome"].get("ok") and not item.get("deterministicRepair")
        )
        write_slot_metadata(slot_info, {
            "updatedAt": action_applied_at,
            "stateDigest": observation["stateDigest"],
            "totalTurns": base_turns + len(steps),
            "totalAcceptedTurns": base_accepted + accepted_count,
            "activeSession": {
                "episodeId": episode_id,
                "startedAt": session_started_at,
                "paceSeconds": args.pace_seconds,
                "turnsObserved": len(steps),
            },
        })
        if not outcome.get("ok"):
            break
    verification = world.verify() if isinstance(world, SpaceWorld) else {"valid": True, "kind": "native-state-validation-on-every-step"}
    observed_at = utc_now()
    wall_elapsed = round(time.monotonic() - session_started_mono, 6)
    player = {"kind": args.player, "id": args.model or (args.organ.name if args.organ else "bounded-hash-player")}
    if args.player == "model":
        player["codec"] = MODEL_PLAYER_CODEC_SCHEMA
    episode = {
        "schema": EPISODE_SCHEMA,
        "episodeId": episode_id,
        "world": world_name,
        "worldRootSha256": hashlib.sha256(str(roots[world_name].resolve()).encode()).hexdigest(),
        "seed": slot_info["seed"],
        "player": player,
        "save": {
            "schema": SAVE_SLOT_SCHEMA,
            "owner": slot_info["owner"],
            "slot": slot_info["slot"],
            "resumed": world_resumed,
            "startingStateDigest": starting_state_digest,
            "finalStateDigest": observation["stateDigest"],
            "priorMetadataDigestMatched": prior_metadata_digest_matched,
            "recoveredInterruptedSession": recovered_session,
        },
        "session": {
            "startedAt": session_started_at,
            "endedAt": observed_at,
            "wallElapsedSeconds": wall_elapsed,
            "minimumTurnCadenceSeconds": args.pace_seconds,
            "paceMode": "HUMAN_PACED" if args.pace_seconds > 0 else "UNPACED_TEST",
            "requestedMinutes": args.session_minutes,
        },
        "turnLimit": args.turns,
        "steps": steps,
        "finalStateDigest": observation["stateDigest"],
        "verification": verification,
        "sourceClass": (
            "PERSISTENT_SIMULATOR_PLAY"
            if args.pace_seconds > 0
            else "UNPACED_SIMULATOR_TEST"
        ),
        "completionClaimed": False,
        "modelWeightsChanged": False,
        "observedAt": observed_at,
        "authority": {"training": False, "promotion": False, "canon": False},
    }
    episode["receiptSha256"] = digest(episode)
    direct_ledger = data_home / "experience" / "simulator-direct-experience.jsonl"
    trajectory_records = data_home / "experience" / "simulator-trajectories.jsonl"
    projections = data_home / "experience" / "training-projections.jsonl"
    capability_gaps = data_home / "experience" / "simulator-capability-gaps.jsonl"
    append_jsonl(direct_ledger, episode)
    all_ok = bool(steps) and all(step["outcome"].get("ok") for step in steps)
    gap_records = [
        capability_gap_record(episode, step)
        for step in steps
        if step.get("deterministicRepair") or step.get("playerFailure")
    ]
    for gap in gap_records:
        append_jsonl(capability_gaps, gap)
    signal = "INCONCLUSIVE" if all_ok and not gap_records else "HARMFUL"
    projection_receipt = project_episode_turns(
        episode, gap_records, args, trajectory_records, projections, run_dir
    )
    (run_dir / "episode-receipt.json").write_bytes(canonical_bytes({**episode, "projection": projection_receipt}) + b"\n")
    accepted_count = sum(
        1
        for item in steps
        if item["outcome"].get("ok") and not item.get("deterministicRepair")
    )
    write_slot_metadata(slot_info, {
        "updatedAt": observed_at,
        "stateDigest": observation["stateDigest"],
        "totalSessions": int(slot_info["metadata"].get("totalSessions", 0)) + 1,
        "totalTurns": base_turns + len(steps),
        "totalAcceptedTurns": base_accepted + accepted_count,
        "totalInterruptedSessions": base_interrupted + (1 if recovered_session else 0),
        "totalWallSeconds": round(
            float(slot_info["metadata"].get("totalWallSeconds", 0.0)) + wall_elapsed, 6
        ),
        "activeSession": None,
        "lastSession": {
            "episodeId": episode_id,
            "receiptSha256": episode["receiptSha256"],
            "startedAt": session_started_at,
            "endedAt": observed_at,
            "turnsObserved": len(steps),
            "acceptedWalmiTurns": accepted_count,
            "wallElapsedSeconds": wall_elapsed,
            "recoveredInterruptedSession": recovered_session,
        },
    })
    if args.keep_workdir:
        archive = None
        expanded_files = len([item for item in run_dir.rglob("*") if item.is_file()])
        expanded_bytes = sum(item.stat().st_size for item in run_dir.rglob("*") if item.is_file())
    else:
        archive, expanded_files, expanded_bytes = compact_run(run_dir)
    return {
        "episodeId": episode_id,
        "world": world_name,
        "saveOwner": slot_info["owner"],
        "saveSlot": slot_info["slot"],
        "savePath": str(slot_info["slotDir"]),
        "resumed": world_resumed,
        "turns": len(steps),
        "accepted": all_ok and not gap_records,
        "worldContinued": all_ok,
        "outcomeSignal": signal,
        "receiptSha256": episode["receiptSha256"],
        "finalStateDigest": episode["finalStateDigest"],
        "projection": projection_receipt,
        "archive": str(archive) if archive else None,
        "workdir": str(run_dir) if args.keep_workdir else None,
        "expandedFilesCompacted": expanded_files if archive else 0,
        "expandedBytesCompacted": expanded_bytes if archive else 0,
        "capabilityGapIds": [gap["id"] for gap in gap_records],
        "deterministicRepairsApplied": sum(1 for gap in gap_records if gap["repair"]["worldActionApplied"]),
        "wallElapsedSeconds": wall_elapsed,
        "paceMode": episode["session"]["paceMode"],
    }


def play_world(world_name: str, args: argparse.Namespace, data_home: Path, roots: dict[str, Path]) -> dict:
    runs_root = data_home / "experience" / "simulator-runs"
    before = {path.name for path in runs_root.iterdir()} if runs_root.is_dir() else set()
    try:
        return _play_world(world_name, args, data_home, roots)
    except Exception:
        if runs_root.is_dir():
            for path in runs_root.iterdir():
                if path.name not in before and path.is_dir() and not any(path.iterdir()):
                    path.rmdir()
        raise


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Let WALMI or a candidate organ play the three local AXM simulator worlds.")
    parser.add_argument("--world", choices=["all", *DEFAULT_ROOTS], default="all")
    parser.add_argument("--turns", type=int, default=3)
    parser.add_argument("--seed", help="seed for a new slot; an existing slot resumes its recorded seed")
    parser.add_argument("--player", choices=["deterministic", "organ", "model"], default="deterministic")
    parser.add_argument("--organ", type=Path)
    parser.add_argument("--model")
    parser.add_argument("--waldo-bin", type=Path)
    parser.add_argument("--data-home", type=Path, default=default_data_home())
    parser.add_argument("--theme-root", type=Path, default=DEFAULT_ROOTS["theme-park"])
    parser.add_argument("--space-root", type=Path, default=DEFAULT_ROOTS["factual-space"])
    parser.add_argument("--living-root", type=Path, default=DEFAULT_ROOTS["living-city"])
    parser.add_argument("--save-owner", default="walmi", help="player-owned save namespace")
    parser.add_argument("--save-slot", default="main", help="named save within each selected simulator")
    parser.add_argument("--new-save", action="store_true", help="create a new slot and refuse any overwrite")
    parser.add_argument(
        "--pace-seconds",
        type=float,
        default=30.0,
        help="minimum wall-clock interval between decisions; zero is an explicitly unpaced test",
    )
    parser.add_argument(
        "--session-minutes",
        type=float,
        help="optional wall-clock session limit in addition to the turn limit",
    )
    parser.add_argument(
        "--repair-policy",
        choices=["stop", "visible-fallback"],
        default="stop",
        help="stop on player failure, or visibly apply a deterministic bounded replacement",
    )
    parser.add_argument("--keep-workdir", action="store_true", help="retain expanded run files instead of a verified ZIP")
    args = parser.parse_args()
    if not 1 <= args.turns <= MAX_TURNS:
        parser.error(f"--turns must be between 1 and {MAX_TURNS}")
    if not 0 <= args.pace_seconds <= 3600:
        parser.error("--pace-seconds must be between 0 and 3600")
    if args.session_minutes is not None and not 0.1 <= args.session_minutes <= 1440:
        parser.error("--session-minutes must be between 0.1 and 1440")
    try:
        args.save_owner = safe_identity(args.save_owner, "save owner")
        args.save_slot = safe_identity(args.save_slot, "save slot")
    except ValueError as error:
        parser.error(str(error))
    return args


def supports_waldo_projection(
    waldo_bin: Path | None, environment: dict[str, str]
) -> bool:
    if waldo_bin is None or not waldo_bin.is_file():
        return False
    result = subprocess.run(
        [str(waldo_bin), "mirror", "experience", "--help"],
        cwd=waldo_bin.parent,
        env=environment,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        timeout=15,
        check=False,
    )
    return result.returncode == 0 and b"project-trajectory" in result.stdout


def main() -> int:
    args = parse_args()
    args.waldo_bin = infer_waldo_bin(args.waldo_bin, args.data_home)
    args.waldo_environment = waldo_environment(args.data_home, args.waldo_bin)
    args.waldo_projection_supported = supports_waldo_projection(
        args.waldo_bin, args.waldo_environment
    )
    roots = {"theme-park": args.theme_root.resolve(), "factual-space": args.space_root.resolve(), "living-city": args.living_root.resolve()}
    selected = list(DEFAULT_ROOTS) if args.world == "all" else [args.world]
    for name in selected:
        if not roots[name].is_dir():
            raise SystemExit(f"simulator root does not exist: {roots[name]}")
    data_home = args.data_home.resolve()
    data_home.mkdir(parents=True, exist_ok=True)
    results = []
    failures = []
    for world_name in selected:
        try:
            results.append(play_world(world_name, args, data_home, roots))
        except Exception as error:
            failures.append({"world": world_name, "error": str(error)})
    report = {
        "schema": "axm.walmi.simulator-bridge-run/v1",
        "status": "PASS" if not failures else "PARTIAL_OR_FAILED",
        "dataHome": str(data_home),
        "results": results,
        "failures": failures,
        "modelWeightsChanged": False,
        "trainingInvoked": False,
    }
    print(json.dumps(report, indent=2, ensure_ascii=False))
    return 0 if not failures else 1


if __name__ == "__main__":
    raise SystemExit(main())
