#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import tempfile

REQUEST_SCHEMA = "axm.walmi.workspace-request/v0.2"
RESPONSE_SCHEMA = "axm.walmi.workspace-response/v0.2"
WRITE_SCHEMA = "axm.walmi.workspace-write-request/v0.2"
RECEIPT_SCHEMA = "axm.walmi.workspace-write-receipt/v0.2"
MAX_ENTRIES = 2000
MAX_READ = 512 * 1024
MAX_OPS = 64
MAX_FILE = 512 * 1024
MAX_TOTAL = 2 * 1024 * 1024


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def json_digest(value: object) -> str:
    return digest(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8"))


def sha(value: str, name: str) -> str:
    if len(value) != 64 or value.lower() != value:
        raise ValueError(f"{name} must be a lowercase SHA-256 hex digest")
    try:
        bytes.fromhex(value)
    except ValueError as exc:
        raise ValueError(f"{name} must be a lowercase SHA-256 hex digest") from exc
    return value


def root_path(value: str) -> Path:
    raw = Path(value)
    if raw.is_symlink() or not raw.is_dir():
        raise ValueError("workspace root must be an existing non-symlink directory")
    return raw.resolve(strict=True)


def rel_path(value: str, *, allow_root: bool = False) -> str:
    if value == "" and allow_root:
        return ""
    if not isinstance(value, str) or not value or value.strip() != value or "\\" in value or ":" in value or "\x00" in value:
        raise ValueError("workspace path is invalid")
    if value.startswith("/"):
        raise ValueError("workspace path must be relative")
    parts = value.split("/")
    if any(part in ("", ".", "..") or part.casefold() == ".git" for part in parts):
        raise ValueError("workspace path contains a refused segment")
    return "/".join(parts)


def target(root: Path, relative: str, *, allow_root: bool = False) -> Path:
    relative = rel_path(relative, allow_root=allow_root)
    out = root if relative == "" else root.joinpath(*relative.split("/"))
    check = root
    for part in (() if relative == "" else relative.split("/")):
        check = check / part
        if check.exists() or check.is_symlink():
            if check.is_symlink():
                raise ValueError("symlink traversal is refused")
    return out


def atomic_write(path: Path, content: bytes, mode: int = 0o644) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp = tempfile.mkstemp(prefix=".walmi-workspace-", dir=path.parent)
    tmp_path = Path(tmp)
    try:
        with os.fdopen(fd, "wb") as handle:
            handle.write(content)
            handle.flush()
            os.fsync(handle.fileno())
        os.chmod(tmp_path, mode)
        os.replace(tmp_path, path)
    finally:
        try:
            tmp_path.unlink()
        except FileNotFoundError:
            pass


def preflight(root: Path, write: dict) -> tuple[dict, list[dict]]:
    if write.get("schema") != WRITE_SCHEMA:
        raise ValueError(f"workspace write schema must be {WRITE_SCHEMA}")
    candidate = str(write.get("candidate_id", ""))
    if not candidate or len(candidate) > 128 or any(not (c.isalnum() or c in "._-") for c in candidate):
        raise ValueError("candidate_id is invalid")
    source_sha = sha(str(write.get("source_sha256", "")), "source_sha256")
    operations = write.get("operations")
    if not isinstance(operations, list) or not 1 <= len(operations) <= MAX_OPS:
        raise ValueError(f"workspace write operations must contain 1-{MAX_OPS} entries")
    canonical = {"schema": WRITE_SCHEMA, "candidate_id": candidate, "source_sha256": source_sha, "operations": []}
    plans, seen, total = [], set(), 0
    for index, raw in enumerate(operations, 1):
        if not isinstance(raw, dict):
            raise ValueError(f"operation {index} must be an object")
        action = str(raw.get("action", "")).strip().lower()
        relative = rel_path(str(raw.get("path", "")))
        if relative.casefold() in seen:
            raise ValueError(f"duplicate workspace path {relative!r}")
        seen.add(relative.casefold())
        path = target(root, relative)
        exists = path.exists()
        if exists and (path.is_symlink() or not path.is_file()):
            raise ValueError(f"workspace path {relative!r} must address a regular non-symlink file")
        before = path.read_bytes() if exists else None
        expected = str(raw.get("expected_sha256", ""))
        content = str(raw.get("content", ""))
        content_sha = str(raw.get("content_sha256", ""))
        if action == "create":
            if exists:
                raise ValueError(f"create workspace path {relative!r} already exists")
            if expected:
                raise ValueError("create must not declare expected_sha256")
        elif action in ("update", "delete"):
            if not exists:
                raise ValueError(f"{action} workspace path {relative!r} does not exist")
            sha(expected, "expected_sha256")
            if digest(before or b"") != expected:
                raise ValueError(f"workspace path {relative!r} stale expected_sha256")
        else:
            raise ValueError(f"unsupported workspace action {action!r}")
        if action == "delete":
            if content or content_sha:
                raise ValueError("delete must not carry content")
        else:
            encoded = content.encode("utf-8")
            if len(encoded) > MAX_FILE:
                raise ValueError(f"workspace path {relative!r} exceeds {MAX_FILE} byte file limit")
            sha(content_sha, "content_sha256")
            if digest(encoded) != content_sha:
                raise ValueError(f"workspace path {relative!r} content_sha256 mismatch")
            total += len(encoded)
            if total > MAX_TOTAL:
                raise ValueError(f"workspace transaction exceeds {MAX_TOTAL} byte limit")
        mode = path.stat().st_mode & 0o777 if exists else 0o644
        op = {"action": action, "path": relative}
        if action != "delete":
            op.update(content=content, content_sha256=content_sha)
        if expected:
            op["expected_sha256"] = expected
        canonical["operations"].append(op)
        plans.append({"op": op, "path": path, "before": before, "mode": mode})
    return canonical, plans


def rollback(plans: list[dict]) -> None:
    for plan in reversed(plans):
        path, before = plan["path"], plan["before"]
        try:
            if before is None:
                path.unlink(missing_ok=True)
            else:
                atomic_write(path, before, plan["mode"])
        except OSError:
            pass


def apply(root: Path, write: dict) -> dict:
    canonical, plans = preflight(root, write)
    applied: list[dict] = []
    try:
        for plan in plans:
            op, path = plan["op"], plan["path"]
            target(root, op["path"])
            if op["action"] == "create":
                if path.exists():
                    raise ValueError("create target appeared after preflight")
            else:
                if not path.is_file() or digest(path.read_bytes()) != op["expected_sha256"]:
                    raise ValueError("target changed after preflight")
            if op["action"] == "delete":
                path.unlink()
            else:
                atomic_write(path, op["content"].encode("utf-8"), plan["mode"])
            applied.append(plan)
    except Exception:
        rollback(applied)
        raise
    rows, written = [], 0
    for plan in plans:
        op, before = plan["op"], plan["before"]
        row = {"action": op["action"], "path": op["path"], "bytes": 0}
        if before is not None:
            row["before_sha256"] = digest(before)
        if op["action"] != "delete":
            row["after_sha256"] = op["content_sha256"]
            row["bytes"] = len(op["content"].encode("utf-8"))
            written += row["bytes"]
        rows.append(row)
    receipt = {
        "schema": RECEIPT_SCHEMA, "state": "APPLIED", "candidate_id": canonical["candidate_id"],
        "source_sha256": canonical["source_sha256"], "request_sha256": json_digest(canonical),
        "operations": rows, "files_changed": len(rows), "bytes_written": written,
        "workspace_mutation": True, "writer_authored_code": False, "network_used": False,
        "installed": False, "promoted": False, "canon_changed": False,
    }
    receipt["receipt_sha256"] = json_digest(receipt)
    return receipt


def execute(root: Path, request: dict) -> dict:
    if request.get("schema") != REQUEST_SCHEMA:
        raise ValueError(f"workspace request schema must be {REQUEST_SCHEMA}")
    action = str(request.get("action", "")).strip().lower()
    response = {"schema": RESPONSE_SCHEMA, "state": "READY", "action": action, "available": True, "network_used": False, "authority": "WORKSPACE_ONLY"}
    if action == "status":
        response["state"] = "WORKSPACE_AVAILABLE"
    elif action == "tree":
        start_rel = rel_path(str(request.get("path", "")), allow_root=True)
        start = target(root, start_rel, allow_root=True)
        if not start.is_dir():
            raise ValueError("tree path must be a directory")
        limit = int(request.get("max_entries") or MAX_ENTRIES)
        limit = min(max(limit, 1), MAX_ENTRIES)
        rows, truncated = [], False
        for current, dirs, files in os.walk(start, topdown=True, followlinks=False):
            current_path = Path(current)
            dirs[:] = sorted(d for d in dirs if d.casefold() != ".git" and not (current_path / d).is_symlink())
            for name, kind in [(d, "directory") for d in dirs] + [(f, "file") for f in sorted(files) if not (current_path / f).is_symlink()]:
                path = current_path / name
                relative = path.relative_to(root).as_posix()
                if any(part.casefold() == ".git" for part in relative.split("/")):
                    continue
                if len(rows) >= limit:
                    truncated = True
                    break
                item = {"path": relative, "kind": kind}
                if kind == "file":
                    item["bytes"] = path.stat().st_size
                rows.append(item)
            if truncated:
                break
        response.update(state="TREE_READY", path=start_rel, kind="directory", entries=rows, truncated=truncated)
    elif action in ("read", "stat", "hash"):
        relative = rel_path(str(request.get("path", "")))
        path = target(root, relative)
        if action == "stat" and path.is_dir():
            response.update(state="STAT_READY", path=relative, kind="directory")
        else:
            if not path.is_file():
                raise ValueError(f"{action} path must be a regular file")
            data = path.read_bytes()
            response.update(state=f"{action.upper()}_READY", path=relative, kind="file", bytes=len(data), sha256=digest(data))
            if action == "read":
                limit = int(request.get("max_bytes") or MAX_READ)
                limit = min(max(limit, 1), MAX_READ)
                if len(data) > limit:
                    raise ValueError(f"read path exceeds {limit} byte request limit")
                try:
                    response.update(encoding="utf-8", content=data.decode("utf-8"))
                except UnicodeDecodeError:
                    response.update(encoding="base64", content=base64.b64encode(data).decode("ascii"))
    elif action == "apply":
        if not isinstance(request.get("write"), dict):
            raise ValueError("apply requires write")
        receipt = apply(root, request["write"])
        response.update(state="APPLIED", write_receipt=receipt)
    else:
        raise ValueError(f"unsupported workspace action {action!r}")
    return response


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("workspace")
    parser.add_argument("request")
    parser.add_argument("response", nargs="?")
    args = parser.parse_args()
    try:
        root = root_path(args.workspace)
        text = os.sys.stdin.read() if args.request == "-" else Path(args.request).read_text(encoding="utf-8")
        request = json.loads(text)
        response = execute(root, request)
        payload = json.dumps(response, indent=2, ensure_ascii=False) + "\n"
        if args.response:
            Path(args.response).write_text(payload, encoding="utf-8")
        else:
            print(payload, end="")
        return 0
    except Exception as exc:
        print(f"WALMI_WORKSPACE_HAND_ERROR: {exc}", file=os.sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
