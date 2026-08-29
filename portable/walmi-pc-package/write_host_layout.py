from __future__ import annotations

import shutil
from pathlib import Path

host = Path("dist/host")
for name in ["models", "checkpoints", "experience", "rollback", "state", "workspace", "assets"]:
    folder = host / name
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "README.txt").write_text(
        f"WALMI local {name} storage. This directory is part of the portable host.\n",
        encoding="utf-8",
    )

(host / "START_WALMI_PC.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%CD%\runtime\node;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
if not exist "%CD%\state\tmp" mkdir "%CD%\state\tmp"
if not exist "%CD%\state\pip-cache" mkdir "%CD%\state\pip-cache"
if not exist "%CD%\state\torch-cache" mkdir "%CD%\state\torch-cache"
if not exist "%CD%\state\model-cache" mkdir "%CD%\state\model-cache"
if not exist "%CD%\state\pycache" mkdir "%CD%\state\pycache"
set "TEMP=%CD%\state\tmp"
set "TMP=%CD%\state\tmp"
set "PIP_CACHE_DIR=%CD%\state\pip-cache"
set "TORCH_HOME=%CD%\state\torch-cache"
set "HF_HOME=%CD%\state\model-cache"
set "XDG_CACHE_HOME=%CD%\state\model-cache"
set "PYTHONPYCACHEPREFIX=%CD%\state\pycache"
set "WALDO_CONFIG=%CD%\state\waldo-config.json"
runtime\python\python.exe tools\walmi_portable_bootstrap.py
if errorlevel 1 exit /b 1
echo WALMI PC FULL STACK v0.2
echo Local WALDO + Mirror + Hermes research stack, weight-learning runtime, Workspace Hand and Asset Hands loaded from this folder.
echo Workshop is NOT required.
echo.
bin\waldo.exe advisor WALMI
if errorlevel 1 cmd /k "echo WALMI shell ready. Try bin\waldo.exe --help, bin\waldo.exe mirror --help, WALMI_WORKSPACE_HAND.cmd or WALMI_ASSET_HAND.cmd"
''',
    encoding="ascii",
)
(host / "WALMI_LOCAL_SHELL.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%CD%\runtime\node;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
if not exist "%CD%\state\tmp" mkdir "%CD%\state\tmp"
if not exist "%CD%\state\pip-cache" mkdir "%CD%\state\pip-cache"
if not exist "%CD%\state\torch-cache" mkdir "%CD%\state\torch-cache"
if not exist "%CD%\state\model-cache" mkdir "%CD%\state\model-cache"
if not exist "%CD%\state\pycache" mkdir "%CD%\state\pycache"
set "TEMP=%CD%\state\tmp"
set "TMP=%CD%\state\tmp"
set "PIP_CACHE_DIR=%CD%\state\pip-cache"
set "TORCH_HOME=%CD%\state\torch-cache"
set "HF_HOME=%CD%\state\model-cache"
set "XDG_CACHE_HOME=%CD%\state\model-cache"
set "PYTHONPYCACHEPREFIX=%CD%\state\pycache"
set "WALDO_CONFIG=%CD%\state\waldo-config.json"
runtime\python\python.exe tools\walmi_portable_bootstrap.py
if errorlevel 1 exit /b 1
cmd /k "echo WALMI local shell ready. Workshop is optional, not a dependency."
''',
    encoding="ascii",
)
(host / "WALMI_WORKSPACE_HAND.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
if not exist "%CD%\state\tmp" mkdir "%CD%\state\tmp"
set "TEMP=%CD%\state\tmp"
set "TMP=%CD%\state\tmp"
set "PYTHONPYCACHEPREFIX=%CD%\state\pycache"
set "WALDO_CONFIG=%CD%\state\waldo-config.json"
runtime\python\python.exe tools\walmi_portable_bootstrap.py
if errorlevel 1 exit /b 1
if "%~2"=="" (
  echo Usage: WALMI_WORKSPACE_HAND.cmd ^<workspace-folder^> ^<request.json^> [response.json]
  exit /b 2
)
if "%~3"=="" (
  runtime\python\python.exe tools\walmi_workspace_hand.py "%~1" "%~2"
) else (
  runtime\python\python.exe tools\walmi_workspace_hand.py "%~1" "%~2" "%~3"
)
''',
    encoding="ascii",
)
(host / "WALMI_ASSET_HAND.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
if not exist "%CD%\state\tmp" mkdir "%CD%\state\tmp"
set "TEMP=%CD%\state\tmp"
set "TMP=%CD%\state\tmp"
set "WALDO_CONFIG=%CD%\state\waldo-config.json"
runtime\python\python.exe tools\walmi_portable_bootstrap.py
if errorlevel 1 exit /b 1
if "%~1"=="" (
  echo Usage: WALMI_ASSET_HAND.cmd ^<request.json^> [response.json]
  exit /b 2
)
if "%~2"=="" (
  runtime\node\node.exe tools\walmi_asset_hand_runner.js "%~1"
) else (
  runtime\node\node.exe tools\walmi_asset_hand_runner.js "%~1" "%~2"
)
''',
    encoding="ascii",
)
(host / "WALMI_AUTOLEARN.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%CD%\runtime\node;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
if not exist "%CD%\state\tmp" mkdir "%CD%\state\tmp"
set "TEMP=%CD%\state\tmp"
set "TMP=%CD%\state\tmp"
set "PIP_CACHE_DIR=%CD%\state\pip-cache"
set "TORCH_HOME=%CD%\state\torch-cache"
set "HF_HOME=%CD%\state\model-cache"
set "XDG_CACHE_HOME=%CD%\state\model-cache"
set "PYTHONPYCACHEPREFIX=%CD%\state\pycache"
set "WALDO_CONFIG=%CD%\state\waldo-config.json"
runtime\python\python.exe tools\walmi_portable_bootstrap.py
if errorlevel 1 exit /b 1
runtime\python\python.exe tools\walmi_autolearn.py %*
''',
    encoding="ascii",
)
(host / "WALMI_EXPERIENCE_OBSERVE.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
if "%~1"=="" (
  echo Usage: WALMI_EXPERIENCE_OBSERVE.cmd ^<outcome.json^>
  exit /b 2
)
set "OUTCOME=%~1"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%CD%\runtime\node;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
if not exist "%CD%\state\tmp" mkdir "%CD%\state\tmp"
set "TEMP=%CD%\state\tmp"
set "TMP=%CD%\state\tmp"
set "PIP_CACHE_DIR=%CD%\state\pip-cache"
set "TORCH_HOME=%CD%\state\torch-cache"
set "HF_HOME=%CD%\state\model-cache"
set "XDG_CACHE_HOME=%CD%\state\model-cache"
set "PYTHONPYCACHEPREFIX=%CD%\state\pycache"
set "WALDO_CONFIG=%CD%\state\waldo-config.json"
runtime\python\python.exe tools\walmi_portable_bootstrap.py
if errorlevel 1 exit /b 1
bin\waldo.exe mirror experience observe "%OUTCOME%" --ledger "%CD%\experience\direct-experience.jsonl" --learn-to "%CD%\experience\training-projections.jsonl" --hermes-to "%CD%\experience\hermes-memory.jsonl"
if errorlevel 1 exit /b 1
runtime\python\python.exe tools\walmi_autolearn.py
''',
    encoding="ascii",
)
(host / "README_FIRST.txt").write_text(
    "WALMI WINDOWS FULL HOST v0.2\n\n"
    "Run START_WALMI_PC.cmd.\n\n"
    "Every mutable WALDO path is bootstrapped inside this extracted host, so putting the host on D: keeps indexes, "
    "lookaside storage, staging, caches, models, checkpoints and experience on D:. Moving the extracted folder repairs "
    "those paths on the next launch.\n\n"
    "WALMI_EXPERIENCE_OBSERVE.cmd records a completed outcome, preserves the direct ledger, derives either a positive "
    "response target or an outcome-conditioned reflection, and invokes the local growth controller. The controller waits "
    "for 8 new projections by default, then re-ingests the accumulated derived corpus and persists a fresh WALDO checkpoint. "
    "Use WALMI_AUTOLEARN.cmd --status to inspect the threshold. Weight change is recorded separately from behavior quality.\n\n"
    "The host is standalone: Workshop is not required. It includes WALDO, AXM Mirror, Mirror Review, "
    "portable Python/PyTorch, a portable Node runtime, the WALMI Workspace Hand, the existing WALMI inner-asset path, "
    "and a pinned local copy of AXM Asset Fabric/Asset Hands capability code.\n\n"
    "Workspace access is opt-in per folder. If no folder is attached WALMI still boots normally. "
    "Writes are candidate-only, hash-bound, transactional, symlink-refusing, .git-refusing and rollback-on-error.\n\n"
    "Asset generation is local. Native Blender/Godot/Unity/Unreal/FreeCAD/FFmpeg-style bridges remain optional and only work "
    "when the target program/tool exists and the hand's own permission checks pass.\n\n"
    "MODEL_WEIGHT_SCAN.json states whether starting weights existed. No weights are fabricated.\n",
    encoding="utf-8",
)

(host / "tools").mkdir(parents=True, exist_ok=True)
shutil.copy2(
    "portable/walmi-pc-package/walmi_portable_bootstrap.py",
    host / "tools/walmi_portable_bootstrap.py",
)
shutil.copy2(
    "portable/walmi-pc-package/walmi_autolearn.py",
    host / "tools/walmi_autolearn.py",
)

for name in ["MODEL_WEIGHT_SCAN.json", "ASSET_CAPABILITY_RECEIPT.json", "WORKSPACE_HAND_SMOKE.json", "AUTOLEARN_EXPERIENCE_SMOKE.json"]:
    source = Path("dist/inventory") / name
    if source.is_file():
        shutil.copy2(source, host / name)
if Path("dist/WINDOWS_RUNTIME_PATCH_RECEIPT.json").is_file():
    shutil.copy2("dist/WINDOWS_RUNTIME_PATCH_RECEIPT.json", host / "WINDOWS_RUNTIME_PATCH_RECEIPT.json")
for name in ["LICENSE", "NOTICE"]:
    shutil.copy2(name, host / name)

print("WALMI v0.2 host layout written")
