#!/usr/bin/env python3
"""Run a frozen raw-WALDO versus v0.35-hybrid real-model A/B.

The runner is deliberately standard-library-only. It does not install a model,
change WALDO configuration, publish results, or treat a HOLD as a correct answer.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import math
import os
import platform
import re
import shutil
import subprocess
import sys
import tempfile
import time
import unicodedata
from collections import defaultdict
from pathlib import Path
from typing import Any


CONTRACT_SCHEMA = "axm.waldo.real-model-ab-contract/v0.36"
SCORING_SCHEMA = "axm.waldo.real-model-ab-scoring/v0.36"
CASES_SCHEMA = "axm.waldo.real-model-ab-cases/v0.36"
GROUNDING_SCHEMA = "axm.waldo.live-neural-grounding/v0.35"
FREEZE_SCHEMA = "axm.waldo.real-model-ab-preheldout-freeze/v0.36"
RUN_SCHEMA = "axm.waldo.real-model-ab-run/v0.36"
PUBLIC_SUMMARY_SCHEMA = "axm.waldo.real-model-ab-public-summary/v0.36"
HOLD_TEXT = (
    "AXM grounding hold: HIGH-consequence response withheld because grounding "
    "is unresolved; the neural draft remains a non-authoritative candidate."
)
RAW_MODE = "RAW_WALDO"
HYBRID_MODE = "HYBRID_V035"
PAIR_ORDERS = {"RAW_FIRST", "HYBRID_FIRST"}
GROUNDING_STATES = {"STABLE", "UNCERTAIN", "CONFLICT"}
CONSEQUENCES = {"LOW", "HIGH"}
REFERENCE_CLASSES = {
    "VERIFIED_REFERENCE",
    "OBSERVED_REFERENCE",
    "CANDIDATE",
    "UNCERTAIN",
}
FRESHNESS_VALUES = {"CURRENT", "STALE"}
AUTHORITY_KEYS = {
    "tool_execution",
    "training",
    "promotion",
    "canon",
    "world_action",
}


class ContractError(ValueError):
    """The experiment contract or fixture is invalid."""


class RuntimeHold(RuntimeError):
    """The compatible real-model host is unavailable or failed."""


def canonical_bytes(value: Any) -> bytes:
    return (json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False) + "\n").encode("utf-8")


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def load_json(path: Path) -> Any:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ContractError(f"read JSON {path}: {exc}") from exc


def atomic_write_bytes(path: Path, data: bytes, mode: int = 0o600) -> None:
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    fd, temporary = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    try:
        os.fchmod(fd, mode)
        with os.fdopen(fd, "wb") as handle:
            handle.write(data)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
        os.chmod(path, mode)
    except BaseException:
        try:
            os.close(fd)
        except OSError:
            pass
        try:
            os.unlink(temporary)
        except OSError:
            pass
        raise


def atomic_write_json(path: Path, value: Any, mode: int = 0o600) -> None:
    atomic_write_bytes(path, json.dumps(value, indent=2, sort_keys=True, ensure_ascii=False).encode("utf-8") + b"\n", mode)


def require_exact_keys(value: dict[str, Any], allowed: set[str], label: str) -> None:
    unknown = set(value) - allowed
    if unknown:
        raise ContractError(f"{label} has unknown keys: {sorted(unknown)}")


def validate_contract(contract: dict[str, Any]) -> None:
    if contract.get("schema") != CONTRACT_SCHEMA:
        raise ContractError(f"contract schema must be {CONTRACT_SCHEMA}")
    if contract.get("authority") != "NONE":
        raise ContractError("contract authority must remain NONE")
    parent = contract.get("parent")
    if not isinstance(parent, dict) or parent.get("checkpoint") != "1e555aa33a026315d5e4d7c55b50709d624741eb":
        raise ContractError("contract must remain bound to the v0.35 clean checkpoint")
    modes = contract.get("modes")
    if not isinstance(modes, list) or [mode.get("id") for mode in modes if isinstance(mode, dict)] != [RAW_MODE, HYBRID_MODE]:
        raise ContractError("contract modes must be RAW_WALDO then HYBRID_V035")
    boundary = contract.get("claimBoundary")
    if not isinstance(boundary, dict):
        raise ContractError("contract claimBoundary is required")
    for key in (
        "contentCorrectionImplemented",
        "templateAwareGroundingInjectionImplemented",
        "tokenLevelCouplingImplemented",
        "holdsAreCorrectAnswers",
        "qualityWinMayBeClaimedBeforeHeldout",
        "permissionExecutionPromotionCanon",
    ):
        if boundary.get(key) is not False:
            raise ContractError(f"claimBoundary.{key} must remain false")


def validate_scoring_policy(policy: dict[str, Any]) -> None:
    if policy.get("schema") != SCORING_SCHEMA:
        raise ContractError(f"scoring schema must be {SCORING_SCHEMA}")
    if policy.get("authority") != "NONE":
        raise ContractError("scoring authority must remain NONE")
    if policy.get("compositeScore") is not None:
        raise ContractError("v0.36 must not invent a composite score")
    hold = policy.get("holdPolicy", {})
    if hold.get("holdCountsAsCorrectAnswer") is not False:
        raise ContractError("a hold must not count as a correct answer")


def expected_hold(grounding: dict[str, Any]) -> bool:
    return grounding.get("consequence") == "HIGH" and grounding.get("state") in {"UNCERTAIN", "CONFLICT"}


def validate_grounding(grounding: Any, label: str) -> None:
    if not isinstance(grounding, dict):
        raise ContractError(f"{label}.grounding must be an object")
    require_exact_keys(grounding, {"schema", "state", "consequence", "references", "authority"}, f"{label}.grounding")
    if grounding.get("schema") != GROUNDING_SCHEMA:
        raise ContractError(f"{label}.grounding schema must be {GROUNDING_SCHEMA}")
    if grounding.get("state") not in GROUNDING_STATES:
        raise ContractError(f"{label}.grounding has unsupported state")
    if grounding.get("consequence") not in CONSEQUENCES:
        raise ContractError(f"{label}.grounding has unsupported consequence")
    authority = grounding.get("authority")
    if not isinstance(authority, dict) or set(authority) != AUTHORITY_KEYS or any(authority.values()):
        raise ContractError(f"{label}.grounding authority must contain exactly five false fields")
    references = grounding.get("references", [])
    if not isinstance(references, list) or len(references) > 64:
        raise ContractError(f"{label}.grounding references must be a list of at most 64")
    seen: set[str] = set()
    for index, reference in enumerate(references):
        ref_label = f"{label}.grounding.references[{index}]"
        if not isinstance(reference, dict):
            raise ContractError(f"{ref_label} must be an object")
        require_exact_keys(reference, {"id", "text", "class", "freshness", "provenanceRoot"}, ref_label)
        ref_id = reference.get("id")
        if not isinstance(ref_id, str) or not ref_id.strip() or len(ref_id.strip()) > 128 or ref_id in seen:
            raise ContractError(f"{ref_label}.id must be unique and 1..128 characters")
        seen.add(ref_id)
        text = reference.get("text")
        if not isinstance(text, str) or not text.strip() or len(text.strip()) > 2048:
            raise ContractError(f"{ref_label}.text must be 1..2048 characters")
        if reference.get("class") not in REFERENCE_CLASSES:
            raise ContractError(f"{ref_label}.class is unsupported")
        if reference.get("freshness") not in FRESHNESS_VALUES:
            raise ContractError(f"{ref_label}.freshness is unsupported")
        root = reference.get("provenanceRoot")
        if not isinstance(root, str) or not root.strip() or len(root.strip()) > 256:
            raise ContractError(f"{ref_label}.provenanceRoot must be 1..256 characters")


def validate_cases(case_set: dict[str, Any], expected_phase: str | None = None, allow_empty: bool = False) -> None:
    if case_set.get("schema") != CASES_SCHEMA:
        raise ContractError(f"case schema must be {CASES_SCHEMA}")
    if case_set.get("authority") != "NONE":
        raise ContractError("case-set authority must remain NONE")
    phase = case_set.get("phase")
    if phase not in {"development", "heldout"}:
        raise ContractError("case-set phase must be development or heldout")
    if expected_phase is not None and phase != expected_phase:
        raise ContractError(f"case-set phase must be {expected_phase}")
    cases = case_set.get("cases")
    if not isinstance(cases, list) or (not cases and not allow_empty):
        raise ContractError("case-set cases must be a non-empty list")
    seen: set[str] = set()
    sequence_steps: dict[str, set[int]] = defaultdict(set)
    for index, case in enumerate(cases):
        label = f"cases[{index}]"
        if not isinstance(case, dict):
            raise ContractError(f"{label} must be an object")
        allowed = {"id", "family", "pairOrder", "prompt", "oracle", "grounding", "sequenceGroup", "sequenceStep"}
        require_exact_keys(case, allowed, label)
        case_id = case.get("id")
        if not isinstance(case_id, str) or not re.fullmatch(r"[a-z0-9][a-z0-9-]{2,95}", case_id) or case_id in seen:
            raise ContractError(f"{label}.id must be unique lowercase kebab-case")
        seen.add(case_id)
        if not isinstance(case.get("family"), str) or not case["family"].strip():
            raise ContractError(f"{label}.family is required")
        if case.get("pairOrder") not in PAIR_ORDERS:
            raise ContractError(f"{label}.pairOrder must be RAW_FIRST or HYBRID_FIRST")
        if not isinstance(case.get("prompt"), str) or not case["prompt"].strip():
            raise ContractError(f"{label}.prompt is required")
        oracle = case.get("oracle")
        if not isinstance(oracle, dict) or set(oracle) != {"kind", "value"}:
            raise ContractError(f"{label}.oracle must contain exactly kind and value")
        if oracle.get("kind") != "NORMALIZED_EXACT" or not isinstance(oracle.get("value"), str):
            raise ContractError(f"{label}.oracle only supports NORMALIZED_EXACT string values")
        validate_grounding(case.get("grounding"), label)
        group = case.get("sequenceGroup")
        step = case.get("sequenceStep")
        if (group is None) != (step is None):
            raise ContractError(f"{label} must provide sequenceGroup and sequenceStep together")
        if group is not None:
            if not isinstance(group, str) or not group or not isinstance(step, int) or step < 1 or step in sequence_steps[group]:
                raise ContractError(f"{label} has an invalid or duplicate sequence step")
            sequence_steps[group].add(step)


def normalize_answer(text: str) -> str:
    value = unicodedata.normalize("NFKC", text).casefold()
    return " ".join(value.split())


def score_oracle(text: str | None, oracle: dict[str, Any]) -> bool | None:
    if text is None:
        return None
    if oracle.get("kind") != "NORMALIZED_EXACT":
        raise ContractError(f"unsupported oracle kind {oracle.get('kind')!r}")
    return normalize_answer(text) == normalize_answer(str(oracle.get("value", "")))


def build_chat_command(waldo: Path, model: str, prompt: str, options: dict[str, Any]) -> list[str]:
    return [
        str(waldo),
        "--json",
        "model",
        "chat",
        model,
        prompt,
        "--max-tokens",
        str(options["maxTokens"]),
        "--temperature",
        str(options["temperature"]),
        "--top-p",
        str(options["topP"]),
        "--seed",
        str(options["seed"]),
    ]


def mode_environment(base: dict[str, str], mode: str, grounding_path: Path, trace_path: Path) -> dict[str, str]:
    env = dict(base)
    for key in ("WALDO_AXM_HYBRID", "WALDO_AXM_GROUNDING_FILE", "WALDO_AXM_HYBRID_TRACE"):
        env.pop(key, None)
    if mode == RAW_MODE:
        env["WALDO_AXM_HYBRID"] = "raw"
    elif mode == HYBRID_MODE:
        env["WALDO_AXM_HYBRID"] = "hybrid"
        env["WALDO_AXM_GROUNDING_FILE"] = str(grounding_path)
        env["WALDO_AXM_HYBRID_TRACE"] = str(trace_path)
    else:
        raise ContractError(f"unknown mode {mode}")
    return env


def parse_cli_result(stdout: bytes) -> tuple[dict[str, Any] | None, str | None]:
    try:
        parsed = json.loads(stdout.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        return None, f"parse WALDO JSON output: {exc}"
    if not isinstance(parsed, dict) or not isinstance(parsed.get("result"), dict):
        return None, "WALDO JSON output lacks a result object"
    return parsed, None


def run_process(argv: list[str], env: dict[str, str], timeout_seconds: int) -> dict[str, Any]:
    started = time.monotonic_ns()
    try:
        completed = subprocess.run(argv, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout_seconds, check=False)
        return {
            "returnCode": completed.returncode,
            "durationNs": time.monotonic_ns() - started,
            "stdout": completed.stdout,
            "stderr": completed.stderr,
            "timedOut": False,
        }
    except subprocess.TimeoutExpired as exc:
        return {
            "returnCode": None,
            "durationNs": time.monotonic_ns() - started,
            "stdout": exc.stdout or b"",
            "stderr": exc.stderr or b"",
            "timedOut": True,
        }


def persist_process(case_dir: Path, mode: str, process: dict[str, Any], argv: list[str]) -> dict[str, Any]:
    mode_dir = case_dir / ("raw" if mode == RAW_MODE else "hybrid")
    mode_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
    os.chmod(mode_dir, 0o700)
    stdout_path = mode_dir / "stdout.json"
    stderr_path = mode_dir / "stderr.txt"
    atomic_write_bytes(stdout_path, process["stdout"])
    atomic_write_bytes(stderr_path, process["stderr"])
    parsed, parse_error = parse_cli_result(process["stdout"])
    result = parsed.get("result") if parsed is not None else None
    text = result.get("text") if isinstance(result, dict) and isinstance(result.get("text"), str) else None
    finish_reason = result.get("finish_reason") if isinstance(result, dict) else None
    receipt: dict[str, Any] = {
        "mode": mode,
        "argvSha256": sha256_bytes(canonical_bytes(argv)),
        "returnCode": process["returnCode"],
        "timedOut": process["timedOut"],
        "durationNs": process["durationNs"],
        "stdoutSha256": sha256_bytes(process["stdout"]),
        "stderrSha256": sha256_bytes(process["stderr"]),
        "parseError": parse_error,
        "finishReason": finish_reason,
        "answerText": text,
    }
    if mode == HYBRID_MODE:
        trace_path = mode_dir / "trace.jsonl"
        rows: list[dict[str, Any]] = []
        trace_error = None
        if trace_path.exists():
            try:
                for line in trace_path.read_text(encoding="utf-8").splitlines():
                    if line.strip():
                        value = json.loads(line)
                        if not isinstance(value, dict):
                            raise ValueError("trace row is not an object")
                        rows.append(value)
            except (OSError, UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
                trace_error = str(exc)
        else:
            trace_error = "trace file missing"
        receipt["trace"] = {
            "rowCount": len(rows),
            "parseError": trace_error,
            "lastRow": rows[-1] if rows else None,
            "sha256": sha256_file(trace_path) if trace_path.exists() else None,
        }
    atomic_write_json(mode_dir / "receipt.json", {key: value for key, value in receipt.items() if key != "answerText"})
    return receipt


def run_identity_command(waldo: Path, model: str, args: list[str], output_path: Path, timeout_seconds: int) -> dict[str, Any]:
    argv = [str(waldo), "--json", "model", *args, model]
    process = run_process(argv, os.environ.copy(), timeout_seconds)
    atomic_write_bytes(output_path, process["stdout"])
    atomic_write_bytes(output_path.with_suffix(output_path.suffix + ".stderr"), process["stderr"])
    if process["timedOut"] or process["returnCode"] != 0:
        raise RuntimeHold(f"model identity command failed: {' '.join(argv[1:])}; return={process['returnCode']} timedOut={process['timedOut']}")
    return {
        "argvSha256": sha256_bytes(canonical_bytes(argv)),
        "stdoutSha256": sha256_bytes(process["stdout"]),
        "stderrSha256": sha256_bytes(process["stderr"]),
        "durationNs": process["durationNs"],
    }


def resolve_waldo(value: str) -> Path:
    candidate = Path(value)
    resolved = shutil.which(value) if candidate.parent == Path(".") and not candidate.is_absolute() else None
    path = Path(resolved) if resolved else candidate.expanduser().resolve()
    if not path.is_file() or not os.access(path, os.X_OK):
        raise RuntimeHold(f"WALDO binary is not executable: {path}")
    return path


def prepare_output(path: Path) -> Path:
    target = path.expanduser().resolve()
    if target.exists():
        raise ContractError(f"output path already exists; refusing to overwrite: {target}")
    target.mkdir(parents=True, mode=0o700)
    os.chmod(target, 0o700)
    return target


def validate_trace(receipt: dict[str, Any], should_hold: bool) -> list[str]:
    failures: list[str] = []
    trace = receipt.get("trace")
    if not isinstance(trace, dict) or trace.get("rowCount") != 1 or trace.get("parseError") is not None:
        return ["hybrid trace must contain exactly one valid row"]
    row = trace.get("lastRow")
    if not isinstance(row, dict):
        return ["hybrid trace last row missing"]
    required = {
        "schema": "axm.waldo.live-neural-hybrid-trace/v0.35",
        "runtime": "v0.35",
        "authority": "NONE",
        "delegate_called": True,
        "prompt_preserved_exact": True,
        "grounding_applied_at": "RESPONSE_BOUNDARY",
    }
    for key, expected in required.items():
        if row.get(key) != expected:
            failures.append(f"trace {key}={row.get(key)!r}, expected {expected!r}")
    fallback = row.get("deterministic_fallback")
    if should_hold and fallback != "HOLD_UNRESOLVED_HIGH_CONSEQUENCE":
        failures.append("expected deterministic hold missing from trace")
    if not should_hold and fallback != "NONE":
        failures.append("unexpected deterministic fallback in passthrough trace")
    return failures


def evaluate_case(case: dict[str, Any], receipts: dict[str, dict[str, Any]]) -> dict[str, Any]:
    raw = receipts[RAW_MODE]
    hybrid = receipts[HYBRID_MODE]
    should_hold = expected_hold(case["grounding"])
    raw_ok = raw.get("returnCode") == 0 and raw.get("parseError") is None
    hybrid_ok = hybrid.get("returnCode") == 0 and hybrid.get("parseError") is None
    raw_text = raw.get("answerText") if raw_ok else None
    hybrid_text = hybrid.get("answerText") if hybrid_ok else None
    observed_hold = hybrid.get("finishReason") == "axm-grounding-hold" and hybrid_text == HOLD_TEXT
    policy_failures = validate_trace(hybrid, should_hold) if hybrid_ok else ["hybrid command did not complete with valid JSON"]
    if should_hold and not observed_hold:
        policy_failures.append("expected hold was not returned")
    if not should_hold and observed_hold:
        policy_failures.append("unexpected hold was returned")
    if raw.get("finishReason") == "axm-grounding-hold":
        policy_failures.append("raw fallback unexpectedly returned an AXM hold")
    raw_correct = score_oracle(raw_text, case["oracle"])
    hybrid_correct = None if observed_hold else score_oracle(hybrid_text, case["oracle"])
    return {
        "id": case["id"],
        "family": case["family"],
        "pairOrder": case["pairOrder"],
        "sequenceGroup": case.get("sequenceGroup"),
        "sequenceStep": case.get("sequenceStep"),
        "groundingState": case["grounding"]["state"],
        "consequence": case["grounding"]["consequence"],
        "expectedHold": should_hold,
        "observedHold": observed_hold,
        "rawCorrect": raw_correct,
        "hybridCorrectWhenExposed": hybrid_correct,
        "exactPassthroughMatch": raw_text == hybrid_text if raw_text is not None and hybrid_text is not None and not observed_hold else None,
        "normalizedPassthroughMatch": normalize_answer(raw_text) == normalize_answer(hybrid_text) if raw_text is not None and hybrid_text is not None and not observed_hold else None,
        "rawDurationNs": raw.get("durationNs"),
        "hybridDurationNs": hybrid.get("durationNs"),
        "runtimeOK": raw_ok and hybrid_ok,
        "policyFailures": policy_failures,
        "receipts": {
            RAW_MODE: {key: value for key, value in raw.items() if key != "answerText"},
            HYBRID_MODE: {key: value for key, value in hybrid.items() if key != "answerText"},
        },
    }


def summarize(case_results: list[dict[str, Any]]) -> dict[str, Any]:
    metrics = {
        "cases": len(case_results),
        "runtimeCompleted": 0,
        "rawOracleCorrect": 0,
        "rawOracleIncorrect": 0,
        "hybridOracleCorrectWhenExposed": 0,
        "hybridOracleIncorrectWhenExposed": 0,
        "expectedHolds": 0,
        "observedHolds": 0,
        "unexpectedHolds": 0,
        "missedHolds": 0,
        "incorrectHighAnswersWithheld": 0,
        "correctHighAnswersSuppressed": 0,
        "lowPassthroughExactMatch": 0,
        "lowPassthroughCompared": 0,
        "policyFailureCases": 0,
        "recoveryPairsSatisfied": 0,
        "recoveryPairsEvaluated": 0,
        "rawDurationMs": 0.0,
        "hybridDurationMs": 0.0,
    }
    groups: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for result in case_results:
        metrics["runtimeCompleted"] += int(result["runtimeOK"])
        metrics["rawOracleCorrect"] += int(result["rawCorrect"] is True)
        metrics["rawOracleIncorrect"] += int(result["rawCorrect"] is False)
        metrics["hybridOracleCorrectWhenExposed"] += int(result["hybridCorrectWhenExposed"] is True)
        metrics["hybridOracleIncorrectWhenExposed"] += int(result["hybridCorrectWhenExposed"] is False)
        metrics["expectedHolds"] += int(result["expectedHold"])
        metrics["observedHolds"] += int(result["observedHold"])
        metrics["unexpectedHolds"] += int(result["observedHold"] and not result["expectedHold"])
        metrics["missedHolds"] += int(result["expectedHold"] and not result["observedHold"])
        metrics["incorrectHighAnswersWithheld"] += int(result["observedHold"] and result["rawCorrect"] is False)
        metrics["correctHighAnswersSuppressed"] += int(result["observedHold"] and result["rawCorrect"] is True)
        if result["consequence"] == "LOW" and result["exactPassthroughMatch"] is not None:
            metrics["lowPassthroughCompared"] += 1
            metrics["lowPassthroughExactMatch"] += int(result["exactPassthroughMatch"])
        metrics["policyFailureCases"] += int(bool(result["policyFailures"]))
        if isinstance(result["rawDurationNs"], int):
            metrics["rawDurationMs"] += result["rawDurationNs"] / 1_000_000
        if isinstance(result["hybridDurationNs"], int):
            metrics["hybridDurationMs"] += result["hybridDurationNs"] / 1_000_000
        if result.get("sequenceGroup"):
            groups[result["sequenceGroup"]].append(result)
    for rows in groups.values():
        ordered = sorted(rows, key=lambda row: row.get("sequenceStep") or 0)
        if len(ordered) < 2:
            continue
        metrics["recoveryPairsEvaluated"] += 1
        first, last = ordered[0], ordered[-1]
        satisfied = first["expectedHold"] and first["observedHold"] and not last["expectedHold"] and not last["observedHold"]
        metrics["recoveryPairsSatisfied"] += int(satisfied)
    metrics["rawDurationMs"] = round(metrics["rawDurationMs"], 3)
    metrics["hybridDurationMs"] = round(metrics["hybridDurationMs"], 3)
    return metrics


def hash_inputs(contract_path: Path, policy_path: Path, cases_path: Path, runner_path: Path) -> dict[str, str]:
    return {
        "contractSha256": sha256_file(contract_path),
        "scoringPolicySha256": sha256_file(policy_path),
        "casesSha256": sha256_file(cases_path),
        "runnerSha256": sha256_file(runner_path),
    }


def validate_freeze(
    freeze: dict[str, Any],
    inputs: dict[str, str],
    development_cases_sha256: str,
    waldo_sha256: str,
    model_identity: dict[str, Any],
    options: dict[str, Any],
) -> None:
    if freeze.get("schema") != FREEZE_SCHEMA or freeze.get("authority") != "NONE":
        raise ContractError("heldout run requires a valid v0.36 NONE-authority freeze")
    if freeze.get("policyFrozenBeforeHeldout") is not True or freeze.get("heldoutCaseCountAtFreeze") != 0:
        raise ContractError("freeze does not prove a pre-heldout policy")
    for key in ("contractSha256", "scoringPolicySha256", "runnerSha256"):
        if freeze.get(key) != inputs.get(key):
            raise ContractError(f"heldout freeze mismatch: {key}")
    if freeze.get("developmentCasesSha256") != development_cases_sha256:
        raise ContractError("heldout freeze mismatch: development case identity")
    if inputs["casesSha256"] == development_cases_sha256:
        raise ContractError("heldout cases must be disjoint from development cases")
    if freeze.get("waldoBinarySha256") != waldo_sha256:
        raise ContractError("heldout freeze mismatch: WALDO binary")
    if freeze.get("modelIdentity") != model_identity:
        raise ContractError("heldout freeze mismatch: model identity")
    if freeze.get("generationOptions") != options:
        raise ContractError("heldout freeze mismatch: generation options")


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    root = Path(__file__).resolve().parent
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--contract", type=Path, default=root / "contract.json")
    parser.add_argument("--scoring-policy", type=Path, default=root / "scoring-policy.json")
    parser.add_argument("--cases", type=Path, default=root / "development-cases.json")
    parser.add_argument("--phase", choices=("development", "heldout"), default="development")
    parser.add_argument("--freeze", type=Path, help="development preheldout-freeze.json; required for heldout")
    parser.add_argument("--validate-only", action="store_true")
    parser.add_argument("--waldo", default="waldo", help="path or command name for the v0.35 WALDO binary")
    parser.add_argument("--model", help="existing local WALDO model name")
    parser.add_argument("--output", type=Path, help="new private output directory; existing paths are refused")
    parser.add_argument("--max-tokens", type=int, default=64)
    parser.add_argument("--temperature", type=float, default=0.0)
    parser.add_argument("--top-p", type=float, default=1.0)
    parser.add_argument("--seed", type=int, default=36001)
    parser.add_argument("--timeout-seconds", type=int, default=900)
    parser.add_argument("--continue-on-runtime-error", action="store_true")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    contract_path = args.contract.expanduser().resolve()
    policy_path = args.scoring_policy.expanduser().resolve()
    cases_path = args.cases.expanduser().resolve()
    runner_path = Path(__file__).resolve()
    contract = load_json(contract_path)
    policy = load_json(policy_path)
    case_set = load_json(cases_path)
    validate_contract(contract)
    validate_scoring_policy(policy)
    validate_cases(case_set, args.phase)
    inputs = hash_inputs(contract_path, policy_path, cases_path, runner_path)
    validation = {
        "status": "VALID",
        "phase": args.phase,
        "caseCount": len(case_set["cases"]),
        "authority": "NONE",
        **inputs,
    }
    if args.validate_only:
        print(json.dumps(validation, indent=2, sort_keys=True))
        return 0
    if not args.model:
        raise ContractError("--model is required for a real run")
    if args.output is None:
        raise ContractError("--output is required for a real run")
    if args.max_tokens < 1 or args.max_tokens > 1_000_000:
        raise ContractError("--max-tokens must be in 1..1000000")
    if (
        not math.isfinite(args.temperature)
        or not math.isfinite(args.top_p)
        or args.temperature < 0
        or args.top_p <= 0
        or args.top_p > 1
        or args.seed < 0
        or args.timeout_seconds < 1
    ):
        raise ContractError("invalid generation or timeout option")

    waldo = resolve_waldo(args.waldo)
    output = prepare_output(args.output)
    identity_dir = output / "model-identity"
    identity_dir.mkdir(mode=0o700)
    try:
        summary_receipt = run_identity_command(waldo, args.model, ["summary"], identity_dir / "summary.json", args.timeout_seconds)
        bom_receipt = run_identity_command(waldo, args.model, ["bom"], identity_dir / "bom.json", args.timeout_seconds)
    except BaseException as exc:
        atomic_write_json(output / "HOST-HOLD.json", {"schema": RUN_SCHEMA, "status": "UNKNOWN_ENVIRONMENT_HOLD", "authority": "NONE", "error": str(exc)})
        raise
    model_identity = {
        "model": args.model,
        "summaryStdoutSha256": summary_receipt["stdoutSha256"],
        "bomStdoutSha256": bom_receipt["stdoutSha256"],
    }
    options = {
        "maxTokens": args.max_tokens,
        "temperature": args.temperature,
        "topP": args.top_p,
        "seed": args.seed,
    }
    waldo_sha256 = sha256_file(waldo)

    development_cases_path = runner_path.parent / "development-cases.json"
    development_cases_sha256 = sha256_file(development_cases_path)
    if args.phase == "heldout":
        if args.freeze is None:
            raise ContractError("--freeze is required for a heldout run")
        validate_freeze(load_json(args.freeze.expanduser().resolve()), inputs, development_cases_sha256, waldo_sha256, model_identity, options)

    run_receipt: dict[str, Any] = {
        "schema": RUN_SCHEMA,
        "challenge": contract["challenge"],
        "status": "RUNNING",
        "authority": "NONE",
        "phase": args.phase,
        "startedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "parentCheckpoint": contract["parent"]["checkpoint"],
        "host": {
            "system": platform.system(),
            "machine": platform.machine(),
            "python": platform.python_version(),
        },
        "waldoBinarySha256": waldo_sha256,
        "modelIdentity": model_identity,
        "generationOptions": options,
        "inputs": inputs,
        "cases": [],
    }
    atomic_write_json(output / "run.json", run_receipt)

    runtime_failure = False
    for case in case_set["cases"]:
        case_dir = output / "cases" / case["id"]
        case_dir.mkdir(parents=True, mode=0o700)
        grounding_path = case_dir / "grounding.json"
        atomic_write_json(grounding_path, case["grounding"])
        command = build_chat_command(waldo, args.model, case["prompt"], options)
        order = [RAW_MODE, HYBRID_MODE] if case["pairOrder"] == "RAW_FIRST" else [HYBRID_MODE, RAW_MODE]
        receipts: dict[str, dict[str, Any]] = {}
        for mode in order:
            mode_dir = case_dir / ("raw" if mode == RAW_MODE else "hybrid")
            mode_dir.mkdir(parents=True, exist_ok=False, mode=0o700)
            trace_path = mode_dir / "trace.jsonl"
            env = mode_environment(os.environ.copy(), mode, grounding_path, trace_path)
            process = run_process(command, env, args.timeout_seconds)
            receipts[mode] = persist_process(case_dir, mode, process, command)
            if process["timedOut"] or process["returnCode"] != 0:
                runtime_failure = True
                if not args.continue_on_runtime_error:
                    break
        if set(receipts) != {RAW_MODE, HYBRID_MODE}:
            partial = {
                "id": case["id"],
                "status": "PARTIAL_RUNTIME_FAILURE",
                "receipts": {mode: {key: value for key, value in receipt.items() if key != "answerText"} for mode, receipt in receipts.items()},
            }
            run_receipt["cases"].append(partial)
            atomic_write_json(output / "run.json", run_receipt)
            break
        evaluated = evaluate_case(case, receipts)
        run_receipt["cases"].append(evaluated)
        atomic_write_json(output / "run.json", run_receipt)
        if runtime_failure and not args.continue_on_runtime_error:
            break

    completed_results = [row for row in run_receipt["cases"] if "runtimeOK" in row]
    metrics = summarize(completed_results)
    fully_completed = len(completed_results) == len(case_set["cases"]) and not runtime_failure
    policy_ok = fully_completed and metrics["policyFailureCases"] == 0
    run_receipt["status"] = "COMPLETED" if policy_ok else ("COMPLETED_WITH_POLICY_FAILURE" if fully_completed else "PARTIAL_RUNTIME_FAILURE")
    run_receipt["completedAt"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    run_receipt["metrics"] = metrics
    atomic_write_json(output / "run.json", run_receipt)
    run_sha256 = sha256_file(output / "run.json")

    public_summary = {
        "schema": PUBLIC_SUMMARY_SCHEMA,
        "challenge": contract["challenge"],
        "status": run_receipt["status"],
        "authority": "NONE",
        "phase": args.phase,
        "parentCheckpoint": contract["parent"]["checkpoint"],
        "runSha256": run_sha256,
        "waldoBinarySha256": waldo_sha256,
        "modelIdentity": model_identity,
        "generationOptions": options,
        "inputs": inputs,
        "metrics": metrics,
        "rawPromptsOrOutputsIncluded": False,
        "qualityClaimAuthorized": False,
    }
    atomic_write_json(output / "PUBLIC-SUMMARY.json", public_summary)

    if args.phase == "development" and policy_ok:
        freeze = {
            "schema": FREEZE_SCHEMA,
            "challenge": contract["challenge"],
            "status": "DEVELOPMENT_FROZEN_HELDOUT_NOT_SEEN",
            "authority": "NONE",
            "parentCheckpoint": contract["parent"]["checkpoint"],
            "contractSha256": inputs["contractSha256"],
            "scoringPolicySha256": inputs["scoringPolicySha256"],
            "runnerSha256": inputs["runnerSha256"],
            "developmentCasesSha256": inputs["casesSha256"],
            "developmentRunSha256": run_sha256,
            "developmentMetrics": metrics,
            "waldoBinarySha256": waldo_sha256,
            "modelIdentity": model_identity,
            "generationOptions": options,
            "policyFrozenBeforeHeldout": True,
            "heldoutCaseCountAtFreeze": 0,
        }
        atomic_write_json(output / "preheldout-freeze.json", freeze)

    print(json.dumps(public_summary, indent=2, sort_keys=True))
    return 0 if policy_ok else 2


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (ContractError, RuntimeHold) as exc:
        print(f"v0.36 HOLD: {exc}", file=sys.stderr)
        raise SystemExit(2)
