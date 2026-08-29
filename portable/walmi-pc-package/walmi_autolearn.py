from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path


LEARNING_SCHEMA = "axm.waldo.mirror-experience-learning/v0.38"
TRAJECTORY_SCHEMA = "axm.waldo.experience-trajectory-training/v0.51"
STATE_SCHEMA = "walmi.autolearn-state/v1"
RECEIPT_SCHEMA = "walmi.autolearn-receipt/v1"
MODEL_NAME = re.compile(r"^[a-z0-9][a-z0-9._-]{0,63}$")
VALID_ROLES = {"system", "user", "assistant", "tool"}
MENU_INDEX_PROMPT_PREFIX = "WALMI_MENU_INDEX_V1\n"
MENU_INDEX_CURRICULUM_REPLAYS = 4
CURRICULUM_REPLAY_MARKER = "\nDERIVED_REPLAY="
TRAINING_VIEW_REVISION = 2
NEUTRAL_MENU_SELF_CHOICE = "SIMULATOR_MENU_INDEX_NEUTRAL_SELF_CHOICE"


def canonical_bytes(value: object) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")


def digest(value: object) -> str:
    return hashlib.sha256(canonical_bytes(value)).hexdigest()


def write_json_atomic(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.tmp")
    with temporary.open("w", encoding="utf-8", newline="\n") as stream:
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def write_jsonl_atomic(path: Path, rows: list[dict[str, object]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.tmp")
    with temporary.open("w", encoding="utf-8", newline="\n") as stream:
        for row in rows:
            stream.write(json.dumps(row, sort_keys=True, separators=(",", ":")))
            stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def append_receipt(path: Path, value: dict[str, object]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    line = canonical_bytes(value) + b"\n"
    with path.open("ab") as stream:
        stream.write(line)
        stream.flush()
        os.fsync(stream.fileno())


def require_text(value: object, label: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"{label} must be non-empty text")
    return value


def normalize_projection(value: object, line_number: int) -> dict[str, object]:
    if not isinstance(value, dict):
        raise ValueError(f"projection line {line_number} is not an object")
    schema = value.get("schema")
    if schema == LEARNING_SCHEMA:
        if value.get("trainingReady") is not True:
            raise ValueError(f"projection line {line_number} is not training-ready")
        prompt = require_text(value.get("prompt"), f"projection line {line_number} prompt")
        target = require_text(
            value.get("targetResponse"),
            f"projection line {line_number} targetResponse",
        )
        source = require_text(
            value.get("learningRecordSha256"),
            f"projection line {line_number} learningRecordSha256",
        )
        messages = [
            {"role": "user", "content": prompt},
            {"role": "assistant", "content": target},
        ]
        target_kind = "OBSERVED_POSITIVE_RESPONSE"
    elif schema == TRAJECTORY_SCHEMA:
        roles = value.get("supervisedRoles")
        if roles != ["assistant"]:
            raise ValueError(
                f"projection line {line_number} must supervise only the assistant lesson"
            )
        raw_messages = value.get("messages")
        if not isinstance(raw_messages, list) or len(raw_messages) < 3:
            raise ValueError(f"projection line {line_number} has no complete trajectory")
        messages = []
        for position, message in enumerate(raw_messages, 1):
            if not isinstance(message, dict) or set(message) - {"role", "content", "context"}:
                raise ValueError(
                    f"projection line {line_number} message {position} is not strict"
                )
            role = message.get("role")
            if role not in VALID_ROLES:
                raise ValueError(
                    f"projection line {line_number} message {position} has invalid role"
                )
            normalized_message: dict[str, object] = {
                "role": role,
                "content": require_text(
                    message.get("content"),
                    f"projection line {line_number} message {position} content",
                ),
            }
            if message.get("context"):
                normalized_message["context"] = require_text(
                    message.get("context"),
                    f"projection line {line_number} message {position} context",
                )
            messages.append(normalized_message)
        if messages[-1]["role"] != "assistant":
            raise ValueError(
                f"projection line {line_number} trajectory must end in its assistant lesson"
            )
        source = require_text(
            value.get("sourceReceiptSha256"),
            f"projection line {line_number} sourceReceiptSha256",
        )
        target_kind = require_text(
            value.get("targetKind"), f"projection line {line_number} targetKind"
        )
        if target_kind != "OUTCOME_CONDITIONED_REFLECTION":
            raise ValueError(f"projection line {line_number} has the wrong reflection target")
    else:
        raise ValueError(f"projection line {line_number} has unsupported schema {schema!r}")

    normalized: dict[str, object] = {
        "messages": messages,
        "sourceProjectionSha256": source,
        "sourceSchema": schema,
        "targetKind": target_kind,
        "outcomeSignal": value.get("outcomeSignal", ""),
    }
    normalized["id"] = "experience-" + digest(normalized)[:24]
    return normalized


def load_projections(path: Path) -> tuple[list[dict[str, object]], list[str]]:
    rows: list[dict[str, object]] = []
    digests: list[str] = []
    seen: set[str] = set()
    with path.open("r", encoding="utf-8") as stream:
        for line_number, line in enumerate(stream, 1):
            if not line.strip():
                continue
            try:
                raw = json.loads(line)
            except json.JSONDecodeError as error:
                raise ValueError(f"projection line {line_number}: {error}") from error
            row = normalize_projection(raw, line_number)
            row_digest = digest(row)
            if row_digest in seen:
                continue
            seen.add(row_digest)
            rows.append(row)
            digests.append(row_digest)
    if not rows:
        raise ValueError(f"projection ledger {path} contains no learning records")
    return rows, digests


def curriculum_category(row: dict[str, object]) -> str:
    messages = row.get("messages")
    if not isinstance(messages, list) or len(messages) < 3:
        return "BASE_EXPERIENCE"
    prompt = messages[0].get("content") if isinstance(messages[0], dict) else None
    evidence_message = messages[-2] if isinstance(messages[-2], dict) else None
    evidence = evidence_message.get("content") if evidence_message else None
    target = messages[-1].get("content") if isinstance(messages[-1], dict) else None
    is_menu_digit = (
        row.get("sourceSchema") == TRAJECTORY_SCHEMA
        and isinstance(prompt, str)
        and prompt.startswith(MENU_INDEX_PROMPT_PREFIX)
        and evidence_message.get("role") == "tool"
        and isinstance(evidence, str)
        and isinstance(target, str)
        and re.fullmatch(r"[0-7]", target) is not None
    )
    if is_menu_digit and row.get("outcomeSignal") == "INCONCLUSIVE":
        # Preserve the observed turn in the append-only experience ledger, but
        # do not teach the model that its own neutral choice was correct merely
        # because the simulator accepted it. This prevents self-imitation from
        # collapsing the neural menu policy onto a favorite digit.
        return NEUTRAL_MENU_SELF_CHOICE
    if (
        is_menu_digit
        and row.get("outcomeSignal") == "HARMFUL"
        and "player failure:" in evidence
    ):
        return "SIMULATOR_MENU_INDEX_CONTRACT_GAP"
    return "BASE_EXPERIENCE"


def expand_training_view(
    rows: list[dict[str, object]],
) -> tuple[list[dict[str, object]], int, int]:
    expanded: list[dict[str, object]] = []
    curriculum_replays = 0
    neutral_self_choices_excluded = 0
    for row in rows:
        category = curriculum_category(row)
        if category == NEUTRAL_MENU_SELF_CHOICE:
            neutral_self_choices_excluded += 1
            continue
        repeats = MENU_INDEX_CURRICULUM_REPLAYS if category != "BASE_EXPERIENCE" else 1
        for replay in range(1, repeats + 1):
            messages = [dict(message) for message in row["messages"]]
            projected = {
                **row,
                "messages": messages,
                "curriculumCategory": category,
                "curriculumReplay": replay,
            }
            if replay > 1:
                projected["id"] = f"{row['id']}-curriculum-{replay}"
                evidence = messages[-2]
                evidence["content"] = (
                    f"{evidence['content']}"
                    f"{CURRICULUM_REPLAY_MARKER}{replay}/{repeats}"
                )
                curriculum_replays += 1
            expanded.append(projected)
    return expanded, curriculum_replays, neutral_self_choices_excluded


def current_weight_sha256(model_root: Path, model_name: str) -> str:
    bom_path = model_root / model_name / "MODEL-BOM.json"
    if not bom_path.is_file():
        return ""
    try:
        bom = json.loads(bom_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return ""
    current = bom.get("current_run_id")
    for run in bom.get("runs", []):
        if not isinstance(run, dict) or run.get("id") != current:
            continue
        for artifact in run.get("artifacts", []):
            if isinstance(artifact, dict) and artifact.get("role") == "weights":
                value = artifact.get("sha256")
                return value if isinstance(value, str) else ""
    return ""


def current_run_corpus_records(model_root: Path, model_name: str) -> int:
    model_directory = (model_root / model_name).resolve()
    bom_path = model_directory / "MODEL-BOM.json"
    try:
        bom = json.loads(bom_path.read_text(encoding="utf-8"))
        current = bom.get("current_run_id")
        run = next(
            item
            for item in bom.get("runs", [])
            if isinstance(item, dict) and item.get("id") == current
        )
        relative_run_bom = require_text(run.get("run_bom"), "current run BOM path")
        run_bom_path = (model_directory / relative_run_bom).resolve()
        run_bom_path.relative_to(model_directory)
        run_bom = json.loads(run_bom_path.read_text(encoding="utf-8"))
        records = run_bom["corpus_bom"]["totals"]["docs"]
    except (OSError, json.JSONDecodeError, KeyError, StopIteration, TypeError, ValueError) as error:
        raise RuntimeError(f"cannot verify current training corpus records: {error}") from error
    if not isinstance(records, int) or isinstance(records, bool) or records < 1:
        raise RuntimeError("current training corpus has no positive integer record count")
    return records


def run_checked(arguments: list[str], host: Path) -> None:
    print("WALMI_AUTOLEARN_EXEC", subprocess.list2cmdline(arguments), flush=True)
    completed = subprocess.run(arguments, cwd=host, check=False)
    if completed.returncode != 0:
        raise RuntimeError(
            f"command failed with exit code {completed.returncode}: {arguments[0]}"
        )


def compose(generation: int, corpus: str, epochs: int, learning_rate: float) -> dict[str, object]:
    return {
        "kind": "waldo-model-compose",
        "schema": 1,
        "interaction": {"template": "user-assistant-v1"},
        "architecture": {
            "family": "decoder-transformer",
            "context_tokens": 512,
            "vocabulary_size": 259,
            "hidden_size": 384,
            "intermediate_size": 1024,
            "layers": 6,
            "attention_heads": 6,
            "key_value_heads": 2,
            "tie_embeddings": True,
            "parameter_dtype": "bfloat16",
            "tokenizer": {"name": "byte", "revision": "builtin-byte-schema-1"},
        },
        "stages": [
            {
                "name": f"experience-{generation:06d}",
                "type": "alignment",
                "objective": "assistant-response-modeling",
                "conversation": {
                    "template": "user-assistant-v1",
                    "supervised_roles": ["assistant"],
                },
                "corpora": [corpus],
                "parameters": {
                    "profile": "causal-pretrain-shuffled",
                    "epochs": epochs,
                    "batch_size": 1,
                    "sequence_length": 512,
                    "learning_rate": learning_rate,
                    "seed": generation,
                },
            }
        ],
    }


def generation_corpus(generation: int) -> str:
    if generation < 1:
        raise ValueError("experience generation must be positive")
    return f"experience/walmi-generation-{generation:06d}"


def optimizer_view_changed(
    prior_generation: int,
    prior_snapshot_sha256: str,
    current_snapshot_sha256: str,
    training_view_revision_changed: bool,
) -> bool:
    if prior_generation < 1 or training_view_revision_changed:
        return True
    if not prior_snapshot_sha256:
        return True
    return prior_snapshot_sha256 != current_snapshot_sha256


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Train a clean local WALMI model when enough new experience projections accumulate."
    )
    parser.add_argument("--projection", type=Path)
    parser.add_argument("--model", default="walmi")
    parser.add_argument("--threshold", type=int, default=int(os.environ.get("WALMI_AUTOLEARN_THRESHOLD", "8")))
    parser.add_argument("--epochs", type=int, default=1)
    parser.add_argument("--learning-rate", type=float, default=0.0003)
    parser.add_argument("--status", action="store_true")
    return parser.parse_args()


def main() -> int:
    arguments = parse_args()
    if not MODEL_NAME.fullmatch(arguments.model):
        raise SystemExit("--model must be a valid WALDO model name")
    if arguments.threshold < 1 or arguments.epochs < 1:
        raise SystemExit("--threshold and --epochs must be positive")
    if not 0 < arguments.learning_rate <= 1:
        raise SystemExit("--learning-rate must be in (0, 1]")

    script_host = Path(__file__).resolve().parent.parent
    host = Path(os.environ.get("WALMI_HOME", script_host)).resolve()
    if host != script_host:
        raise SystemExit(f"WALMI_HOME {host} does not match portable host {script_host}")
    data_home = Path(os.environ.get("WALMI_DATA_HOME", host)).resolve()
    data_home.mkdir(parents=True, exist_ok=True)
    waldo = host / "bin" / "waldo.exe"
    experience = data_home / "experience"
    projection_path = (arguments.projection or experience / "training-projections.jsonl").resolve()
    state_path = experience / "autolearn-state.json"
    receipts_path = experience / "autolearn-receipts.jsonl"
    lock_path = experience / ".autolearn.lock"
    training_root = experience / "training"
    snapshot_path = training_root / "conversations.jsonl"
    profile_path = training_root / "chat-messages-profile.yaml"
    compose_path = training_root / "experience-compose.json"
    model_root = data_home / "models"

    if not projection_path.is_file():
        raise SystemExit(f"projection ledger does not exist: {projection_path}")
    experience.mkdir(parents=True, exist_ok=True)
    try:
        lock_descriptor = os.open(lock_path, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
    except FileExistsError:
        raise SystemExit(f"another WALMI autolearn cycle holds {lock_path}")
    os.close(lock_descriptor)

    try:
        rows, projection_digests = load_projections(projection_path)
        state: dict[str, object] = {
            "schema": STATE_SCHEMA,
            "model": arguments.model,
            "generation": 0,
            "trainedProjectionDigests": [],
        }
        if state_path.is_file():
            loaded = json.loads(state_path.read_text(encoding="utf-8"))
            if not isinstance(loaded, dict) or loaded.get("schema") != STATE_SCHEMA:
                raise ValueError(f"invalid autolearn state {state_path}")
            state.update(loaded)
        if state.get("model") != arguments.model:
            raise ValueError(
                f"autolearn state belongs to model {state.get('model')!r}, not {arguments.model!r}"
            )
        trained = state.get("trainedProjectionDigests", [])
        if not isinstance(trained, list) or any(not isinstance(item, str) for item in trained):
            raise ValueError("autolearn state has invalid trained projection digests")
        trained_set = set(trained)
        new_digests = [item for item in projection_digests if item not in trained_set]
        prior_generation = int(state.get("generation", 0))
        generation = prior_generation + 1
        prior_training_view_revision = int(state.get("lastTrainingViewRevision", 0))
        training_view_changed = (
            prior_generation > 0
            and prior_training_view_revision != TRAINING_VIEW_REVISION
        )
        threshold_ready = len(new_digests) >= arguments.threshold

        waiting_receipt: dict[str, object] = {
            "schema": RECEIPT_SCHEMA,
            "state": "WAITING_FOR_EXPERIENCE_THRESHOLD",
            "model": arguments.model,
            "threshold": arguments.threshold,
            "availableRecords": len(rows),
            "newRecords": len(new_digests),
            "trainingViewRevision": TRAINING_VIEW_REVISION,
            "trainingViewChanged": training_view_changed,
            "directExperienceRewritten": False,
            "trainingInvoked": False,
            "weightsChanged": False,
            "qualityImproved": None,
        }
        if arguments.status or not (threshold_ready or training_view_changed):
            if arguments.status and training_view_changed:
                waiting_receipt["state"] = "DERIVED_TRAINING_VIEW_REFRESH_READY"
            elif arguments.status and threshold_ready:
                waiting_receipt["state"] = "EXPERIENCE_THRESHOLD_READY"
            waiting_receipt["receiptSha256"] = digest(waiting_receipt)
            if not arguments.status:
                append_receipt(receipts_path, waiting_receipt)
            print(json.dumps(waiting_receipt, indent=2, sort_keys=True))
            return 0

        training_rows, curriculum_replays, neutral_self_choices_excluded = (
            expand_training_view(rows)
        )
        if not training_rows:
            raise RuntimeError("derived training view contains no eligible learning rows")
        prior_snapshot_sha256 = str(state.get("lastCorpusSnapshotSHA256", ""))
        if (
            not prior_snapshot_sha256
            and prior_generation > 0
            and snapshot_path.is_file()
        ):
            prior_snapshot_sha256 = hashlib.sha256(snapshot_path.read_bytes()).hexdigest()
        write_jsonl_atomic(snapshot_path, training_rows)
        corpus_snapshot_sha256 = hashlib.sha256(snapshot_path.read_bytes()).hexdigest()
        if not optimizer_view_changed(
            prior_generation,
            prior_snapshot_sha256,
            corpus_snapshot_sha256,
            training_view_changed,
        ):
            state.update(
                {
                    "schema": STATE_SCHEMA,
                    "model": arguments.model,
                    "trainedProjectionDigests": projection_digests,
                    "lastProjectionRecords": len(rows),
                    "lastDerivedTrainingRows": len(training_rows),
                    "lastTrainingViewRevision": TRAINING_VIEW_REVISION,
                    "lastNeutralSelfChoiceRowsExcluded": neutral_self_choices_excluded,
                    "lastCorpusSnapshotSHA256": corpus_snapshot_sha256,
                }
            )
            write_json_atomic(state_path, state)
            unchanged_receipt: dict[str, object] = {
                "schema": RECEIPT_SCHEMA,
                "state": "DIRECT_EXPERIENCE_RECORDED_DERIVED_VIEW_UNCHANGED",
                "model": arguments.model,
                "generation": prior_generation,
                "threshold": arguments.threshold,
                "availableRecords": len(rows),
                "newRecords": len(new_digests),
                "trainingTrigger": "EXPERIENCE_THRESHOLD_WITH_UNCHANGED_DERIVED_VIEW",
                "trainingViewRevision": TRAINING_VIEW_REVISION,
                "uniqueProjectionRecords": len(rows),
                "derivedTrainingRows": len(training_rows),
                "curriculumReplayRows": curriculum_replays,
                "neutralSelfChoiceRowsExcluded": neutral_self_choices_excluded,
                "corpusSnapshotSHA256": corpus_snapshot_sha256,
                "directExperienceRewritten": False,
                "trainingInvoked": False,
                "weightsChanged": False,
                "qualityImproved": None,
                "qualityState": "UNCHANGED_DERIVED_VIEW_NO_OPTIMIZER_CLAIM",
            }
            unchanged_receipt["receiptSha256"] = digest(unchanged_receipt)
            append_receipt(receipts_path, unchanged_receipt)
            print(json.dumps(unchanged_receipt, indent=2, sort_keys=True))
            return 0
        profile_path.write_text(
            "format: jsonl\n"
            "type: chat-messages\n"
            "fields:\n"
            "  id: id\n"
            "  meta:\n"
            "    source_schema: sourceSchema\n"
            "    source_projection_sha256: sourceProjectionSha256\n"
            "    target_kind: targetKind\n"
            "    outcome_signal: outcomeSignal\n"
            "    curriculum_category: curriculumCategory\n"
            "    curriculum_replay: curriculumReplay\n"
            "messages:\n"
            "  role: messages[].role\n"
            "  content: messages[].content\n",
            encoding="utf-8",
            newline="\n",
        )
        corpus = generation_corpus(generation)
        index_destination = data_home / "state" / "index" / Path(corpus)
        ingest_arguments = [
            str(waldo), "index", "ingest", str(snapshot_path), str(index_destination),
            "--title", "WALMI Accumulated Direct Experience",
            "--description", "Derived response targets and outcome-conditioned reflections; the append-only direct experience ledger remains separate.",
            "--license", "LicenseRef-WALMI-Private-Experience",
            "--source", "https://local.invalid/walmi/direct-experience",
            "--source-name", "walmi-direct-experience",
            "--source-category", "other",
            "--language", "en",
            "--input-profile", str(profile_path),
        ]
        if (index_destination / f"{index_destination.name}.yaml").is_file():
            ingest_arguments.append("--update")
        run_checked(ingest_arguments, host)

        write_json_atomic(
            compose_path,
            compose(generation, corpus, arguments.epochs, arguments.learning_rate),
        )
        before_weight = current_weight_sha256(model_root, arguments.model)
        run_checked(
            [str(waldo), "model", "train", arguments.model, str(compose_path)],
            data_home / "state" / "index",
        )
        after_weight = current_weight_sha256(model_root, arguments.model)
        weights_changed = bool(after_weight) and after_weight != before_weight
        if not weights_changed:
            raise RuntimeError("training completed without a new current weight artifact hash")
        retained_training_records = current_run_corpus_records(model_root, arguments.model)
        if retained_training_records != len(training_rows):
            raise RuntimeError(
                "WALDO retained "
                f"{retained_training_records} of {len(training_rows)} derived training rows"
            )

        state.update(
            {
                "schema": STATE_SCHEMA,
                "model": arguments.model,
                "generation": generation,
                "trainedProjectionDigests": projection_digests,
                "lastWeightSHA256": after_weight,
                "lastProjectionRecords": len(rows),
                "lastDerivedTrainingRows": len(training_rows),
                "lastRetainedTrainingRecords": retained_training_records,
                "lastTrainingViewRevision": TRAINING_VIEW_REVISION,
                "lastNeutralSelfChoiceRowsExcluded": neutral_self_choices_excluded,
                "lastCorpusSnapshotSHA256": corpus_snapshot_sha256,
            }
        )
        write_json_atomic(state_path, state)
        receipt: dict[str, object] = {
            "schema": RECEIPT_SCHEMA,
            "state": "EXPERIENCE_GROWTH_CHECKPOINT_PERSISTED",
            "model": arguments.model,
            "generation": generation,
            "threshold": arguments.threshold,
            "availableRecords": len(rows),
            "newRecords": len(new_digests),
            "trainingTrigger": (
                "DERIVED_TRAINING_VIEW_REFRESH"
                if training_view_changed and not threshold_ready
                else "EXPERIENCE_THRESHOLD"
            ),
            "trainingViewRevision": TRAINING_VIEW_REVISION,
            "uniqueProjectionRecords": len(rows),
            "derivedTrainingRows": len(training_rows),
            "retainedTrainingRecords": retained_training_records,
            "curriculumReplayRows": curriculum_replays,
            "neutralSelfChoiceRowsExcluded": neutral_self_choices_excluded,
            "corpus": corpus,
            "corpusSnapshotSHA256": corpus_snapshot_sha256,
            "previousWeightSHA256": before_weight,
            "currentWeightSHA256": after_weight,
            "directExperienceRewritten": False,
            "trainingInvoked": True,
            "weightsChanged": True,
            "qualityImproved": None,
            "qualityState": "UNKNOWN_UNTIL_HELD_OUT_BEHAVIOR_EVALUATION",
        }
        receipt["receiptSha256"] = digest(receipt)
        append_receipt(receipts_path, receipt)
        print(json.dumps(receipt, indent=2, sort_keys=True))
        return 0
    finally:
        try:
            lock_path.unlink()
        except FileNotFoundError:
            pass


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, RuntimeError, json.JSONDecodeError) as error:
        print(f"WALMI_AUTOLEARN_HOLD {error}", file=sys.stderr)
        raise SystemExit(1)
