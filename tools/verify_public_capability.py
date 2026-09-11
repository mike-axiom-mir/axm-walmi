#!/usr/bin/env python3
"""Verify WALMI's bounded public capability declaration against source truth."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import stat
import sys

SCHEMA = "axm.walmi.public-capability-verification/v0.1"
CAPABILITY_ID = "axm.walmi.deterministic-inner-asset-forge"

EXPECTED_MARKER = {
    "schema": "axm.discovery-public/v1",
    "repo": "mike-axiom-mir/axm-walmi",
    "display_name": "AXM WALMI",
    "public": True,
}

EXPECTED_RECORD = {
    "authority": {
        "automaticInstall": False,
        "automaticSelection": False,
        "canon": False,
        "discoveryOnly": True,
        "execution": False,
        "merge": False,
    },
    "consumers": [],
    "contracts": {
        "input": "axm.waldo-witness.inner-asset-recipe/v0.1",
        "receipt": "axm.waldo-witness.inner-asset-candidate/v0.1",
        "verification": "axm.waldo-witness.inner-asset-validation/v0.1",
    },
    "entrypoints": {
        "command": "waldo-axm-mirror forge-asset <inner-asset-recipe.json> <candidate.axmasset>",
        "discoveryCommand": "waldo-axm-mirror verify-asset <candidate.axmasset>",
        "library": "internal/axmmirror",
    },
    "id": CAPABILITY_ID,
    "license": "Apache-2.0",
    "providers": ["mike-axiom-mir/axm-walmi"],
    "runtime": {
        "dependencies": [],
        "kind": "go",
        "minimumVersion": "1.25",
        "networkRequired": False,
    },
    "schema": "axm.public-capability/v1",
    "source": {
        "descriptor": "internal/axmmirror/innerasset.go",
        "license": "LICENSE",
        "metadata": "go.mod",
    },
    "status": "EXPERIMENTAL",
    "summary": "Deterministically forge and verify bounded candidate-only local sprite assets as portable .axmasset bundles.",
    "version": "0.1.0",
}

SOURCE_PATHS = (
    ".axm/discovery-public.json",
    "registry/capabilities.jsonl",
    "cmd/waldo-axm-mirror/main.go",
    "internal/axmmirror/innerasset.go",
    "internal/axmmirror/innerasset_bundle.go",
    "LICENSE",
    "go.mod",
)


def _reject_duplicate_pairs(pairs: list[tuple[str, object]]) -> dict[str, object]:
    result: dict[str, object] = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def _load_json(data: bytes, label: str) -> object:
    try:
        text = data.decode("utf-8", errors="strict")
        return json.loads(text, object_pairs_hook=_reject_duplicate_pairs)
    except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
        raise ValueError(f"{label}: invalid unambiguous UTF-8 JSON: {exc}") from exc


def _regular_file(root: Path, relative: str, *, max_bytes: int = 2 * 1024 * 1024) -> tuple[Path, bytes]:
    path = root / relative
    try:
        info = path.lstat()
    except OSError as exc:
        raise ValueError(f"missing required source: {relative}") from exc
    if not stat.S_ISREG(info.st_mode) or path.is_symlink():
        raise ValueError(f"required source must be a regular non-symlink file: {relative}")
    if info.st_size > max_bytes:
        raise ValueError(f"required source exceeds verification bound: {relative}")
    return path, path.read_bytes()


def _sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _require(text: str, needle: str, label: str) -> None:
    if needle not in text:
        raise ValueError(f"{label} no longer proves declared capability: missing {needle!r}")


def verify(root: Path) -> dict[str, object]:
    root = root.resolve()
    evidence: dict[str, dict[str, object]] = {}
    raw: dict[str, bytes] = {}
    for relative in SOURCE_PATHS:
        _path, data = _regular_file(root, relative)
        raw[relative] = data
        evidence[relative] = {"bytes": len(data), "sha256": _sha256(data)}

    marker = _load_json(raw[".axm/discovery-public.json"], "public marker")
    if marker != EXPECTED_MARKER:
        raise ValueError("public marker drifted from the explicit WALMI public-safe declaration")

    registry_text = raw["registry/capabilities.jsonl"].decode("utf-8", errors="strict")
    lines = [line for line in registry_text.splitlines() if line.strip()]
    if len(lines) != 1:
        raise ValueError("public capability registry must contain exactly one nonblank record")
    record = _load_json((lines[0] + "\n").encode(), "public capability record")
    if record != EXPECTED_RECORD:
        raise ValueError("public inner-asset capability declaration drifted from its bounded contract")

    main_go = raw["cmd/waldo-axm-mirror/main.go"].decode("utf-8", errors="strict")
    for command in (
        'waldo-axm-mirror forge-asset <inner-asset-recipe.json> <candidate.axmasset>',
        'waldo-axm-mirror verify-asset <candidate.axmasset>',
    ):
        _require(main_go, command, "CLI source")

    inner_asset = raw["internal/axmmirror/innerasset.go"].decode("utf-8", errors="strict")
    for needle in (
        'InnerAssetRecipeSchema     = "axm.waldo-witness.inner-asset-recipe/v0.1"',
        'InnerAssetCandidateSchema  = "axm.waldo-witness.inner-asset-candidate/v0.1"',
        'InnerAssetValidationSchema = "axm.waldo-witness.inner-asset-validation/v0.1"',
        'type InnerAssetCandidate struct {',
        'CandidateOnly',
        'HumanApproved',
        'Canonical',
        'AutomaticPromotion',
        'inner asset recipe must carry closed authority',
    ):
        _require(inner_asset, needle, "inner-asset source")

    bundle = raw["internal/axmmirror/innerasset_bundle.go"].decode("utf-8", errors="strict")
    for needle in (
        "EncodeInnerAssetBundle",
        "VerifyInnerAssetBundle",
        "verifyInnerAssetBuildFiles",
        "verifyInnerAssetDeterminism",
        'entries["candidate.json"]',
        'entries["validation.json"]',
    ):
        _require(bundle, needle, "portable bundle source")

    go_mod = raw["go.mod"].decode("utf-8", errors="strict")
    _require(go_mod, "module github.com/openwaldo/waldo", "go.mod")
    _require(go_mod, "\ngo 1.25.0\n", "go.mod")

    license_text = raw["LICENSE"].decode("utf-8", errors="strict")
    _require(license_text, "Apache License", "LICENSE")
    _require(license_text, "Version 2.0", "LICENSE")

    registry_sha = evidence["registry/capabilities.jsonl"]["sha256"]
    aggregate = hashlib.sha256()
    for relative in sorted(evidence):
        item = evidence[relative]
        aggregate.update(f"{relative}\0{item['bytes']}\0{item['sha256']}\n".encode("utf-8"))

    receipt: dict[str, object] = {
        "schema": SCHEMA,
        "state": "VERIFIED_DECLARATION",
        "repository": "mike-axiom-mir/axm-walmi",
        "capability_id": CAPABILITY_ID,
        "registry_sha256": registry_sha,
        "source_evidence": evidence,
        "source_evidence_sha256": aggregate.hexdigest(),
        "truth": {
            "declaration_is_runtime_proof": False,
            "provider_tests_required_for_runtime_claim": True,
            "visual_quality_proven": False,
            "human_approval_proven": False,
        },
        "authority": {
            "automatic_install": False,
            "automatic_selection": False,
            "canon": False,
            "execution": False,
            "merge": False,
            "promotion": False,
            "source_mutation": False,
        },
    }
    canonical = json.dumps(receipt, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode("utf-8")
    receipt["receipt_sha256"] = _sha256(canonical)
    return receipt


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=str(Path(__file__).resolve().parents[1]))
    parser.add_argument("--receipt")
    return parser


def _write_receipt(path: str | None, payload: dict[str, object]) -> None:
    if not path:
        return
    output = Path(path)
    if output.exists() and (output.is_symlink() or not output.is_file()):
        raise ValueError("receipt destination must be absent or a regular non-symlink file")
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8", newline="\n")


def main(argv: list[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        receipt = verify(Path(args.root))
        _write_receipt(args.receipt, receipt)
        print(json.dumps(receipt, ensure_ascii=False, indent=2, sort_keys=True))
        return 0
    except (OSError, ValueError) as exc:
        hold = {
            "schema": SCHEMA,
            "state": "HOLD",
            "error": str(exc),
            "authority": {
                "automatic_install": False,
                "automatic_selection": False,
                "canon": False,
                "execution": False,
                "merge": False,
                "promotion": False,
                "source_mutation": False,
            },
        }
        try:
            _write_receipt(args.receipt, hold)
        except (OSError, ValueError):
            pass
        print(f"walmi-public-capability: HOLD: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
