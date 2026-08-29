#!/usr/bin/env python3
"""Cross-platform acceptance lane for the sealed standalone WALMI source tree."""

from __future__ import annotations

import os
from pathlib import Path
import subprocess
import sys
import tempfile


ROOT = Path(__file__).resolve().parents[1]
GO_PACKAGES = (
    "./cmd/waldo-axm-mirror",
    "./cmd/workshop",
    "./composes",
    "./internal/axmmirror",
    "./internal/cli",
    "./internal/inference",
    "./internal/model",
    "./internal/modelweights",
    "./internal/training",
)
PYTHON_SUITES = (
    "portable/walmi-pc-package/test_walmi_simulator_bridge.py",
    "portable/walmi-pc-package/test_walmi_autolearn.py",
    "portable/walmi-pc-package/test_walmi_device_pack_builder.py",
    "portable/walmi-pc-package/test_walmi_portable_bootstrap.py",
)


def run(command: list[str], *, environment: dict[str, str] | None = None) -> None:
    print("+", " ".join(command), flush=True)
    subprocess.run(command, cwd=ROOT, env=environment, check=True)


def check_go_format() -> None:
    listed = subprocess.run(
        ["git", "ls-files", "--", "*.go"],
        cwd=ROOT,
        check=True,
        stdout=subprocess.PIPE,
        text=True,
    ).stdout.splitlines()
    failed: list[str] = []
    for relative in listed:
        source = (ROOT / relative).read_bytes().replace(b"\r\n", b"\n")
        formatted = subprocess.run(
            ["gofmt"],
            cwd=ROOT,
            input=source,
            check=True,
            stdout=subprocess.PIPE,
        ).stdout
        if formatted != source:
            failed.append(relative)
    if failed:
        for relative in failed:
            print(f"gofmt required: {relative}", file=sys.stderr)
        raise RuntimeError(f"{len(failed)} Go files require formatting")
    print(f"gofmt: PASS ({len(listed)} files, line endings normalized in memory)")


def main() -> int:
    run([sys.executable, "tools/seal_source_tree.py", "--verify"])
    check_go_format()
    go_environment = os.environ.copy()
    existing_go_flags = go_environment.get("GOFLAGS", "").strip()
    go_environment["GOFLAGS"] = f"{existing_go_flags} -mod=readonly".strip()
    run(["go", "vet", *GO_PACKAGES], environment=go_environment)
    run(["go", "test", *GO_PACKAGES], environment=go_environment)
    with tempfile.TemporaryDirectory(prefix="walmi-wasm-") as temporary:
        environment = go_environment.copy()
        environment["GOOS"] = "js"
        environment["GOARCH"] = "wasm"
        output = str(Path(temporary) / "walmi-browser.wasm")
        run(
            ["go", "build", "-trimpath", "-o", output, "./cmd/walmi-browser-wasm"],
            environment=environment,
        )
    for suite in PYTHON_SUITES:
        run([sys.executable, suite])
    run([sys.executable, "tools/seal_source_tree.py", "--verify"])
    print("WALMI_CROSS_PLATFORM_VERIFICATION_PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
