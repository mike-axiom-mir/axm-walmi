#!/usr/bin/env python3
"""Export the current WALMI source worktree into a clean standalone directory."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess


EXCLUDED_PREFIXES = (
    ".git/",
    ".walmi-intake/",
    "WALMI_PC_FULL_STACK_v0_2/",
    "dist/",
)


def git_output(root: Path, *arguments: str) -> bytes:
    return subprocess.check_output(["git", *arguments], cwd=root)


def source_files(root: Path) -> list[str]:
    tracked = git_output(root, "ls-files", "-z").split(b"\0")
    untracked = git_output(root, "ls-files", "--others", "--exclude-standard", "-z").split(b"\0")
    paths = set()
    for encoded in (*tracked, *untracked):
        if not encoded:
            continue
        relative = encoded.decode("utf-8", errors="surrogateescape").replace("\\", "/")
        if relative == ".walmi-intake" or relative in {"WALMI_PC_FULL_STACK_v0_2", "dist"}:
            continue
        if relative.startswith(EXCLUDED_PREFIXES):
            continue
        paths.add(relative)
    return sorted(paths)


def sha256(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path)
    parser.add_argument("--source", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    source = args.source.resolve()
    output = args.output.resolve()
    if output.exists() and any(output.iterdir()):
        raise SystemExit(f"refusing non-empty output directory: {output}")
    output.mkdir(parents=True, exist_ok=True)

    inventory = []
    for relative in source_files(source):
        source_path = (source / relative).resolve()
        source_path.relative_to(source)
        if source_path.is_symlink():
            raise SystemExit(f"refusing linked source path: {relative}")
        if not source_path.is_file():
            continue
        target_path = output / relative
        target_path.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source_path, target_path)
        inventory.append({
            "path": relative,
            "bytes": target_path.stat().st_size,
            "sha256": sha256(target_path),
        })

    commit = git_output(source, "rev-parse", "HEAD").decode().strip()
    branch = git_output(source, "branch", "--show-current").decode().strip()
    dirty = bool(git_output(source, "status", "--porcelain=v1", "-uno").strip())
    provenance = {
        "schema": "axm.walmi.standalone-source-provenance/v1",
        "source": "openwaldo/waldo local WALMI integration worktree",
        "sourceCommit": commit,
        "sourceBranch": branch,
        "worktreeChangesIncluded": dirty,
        "copyMode": "FULL_FILES_NO_LINKS",
        "excludedRuntimeOrPrivateRoots": list(EXCLUDED_PREFIXES),
        "sourceFileCount": len(inventory),
        "sourceBytes": sum(item["bytes"] for item in inventory),
        "files": inventory,
    }
    (output / "WALMI-SOURCE-PROVENANCE.json").write_text(
        json.dumps(provenance, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
        newline="\n",
    )
    print(json.dumps({
        "status": "PASS",
        "output": str(output),
        "sourceFileCount": provenance["sourceFileCount"],
        "sourceBytes": provenance["sourceBytes"],
        "copyMode": provenance["copyMode"],
    }, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
