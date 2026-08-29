from __future__ import annotations

import shutil
from pathlib import Path

host = Path("dist-v02/host")
for name in ["models", "checkpoints", "experience", "rollback", "state", "projects"]:
    folder = host / name
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "README.txt").write_text(f"WALMI local {name} storage.\n", encoding="utf-8")

(host / "START_WALMI_PC.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%PATH%"
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
echo WALMI PC FULL STACK v0.2
echo.
bin\waldo.exe mirror workspace --help
echo.
echo Read README_FIRST.txt before enabling project writes.
cmd /k
''', encoding="ascii")
(host / "WALMI_LOCAL_SHELL.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%PATH%"
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
cmd /k "echo WALMI local shell ready. Try bin\waldo.exe mirror workspace --help"
''', encoding="ascii")
(host / "RUN_WORKSPACE_HAND.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
if not exist "%CD%\state\tmp" mkdir "%CD%\state\tmp"
set "TEMP=%CD%\state\tmp"
set "TMP=%CD%\state\tmp"
set "PYTHONPYCACHEPREFIX=%CD%\state\pycache"
set "WALDO_CONFIG=%CD%\state\waldo-config.json"
echo Read-only example. Replace PROJECT_ROOT and MODEL_NAME.
bin\waldo.exe mirror workspace "PROJECT_ROOT" WALMI_WORKSPACE_REQUEST_EXAMPLE.json --model "MODEL_NAME"
echo Add --allow-write only after reviewing the selected root and request.
pause
''', encoding="ascii")
(host / "README_FIRST.txt").write_text(
    "WALMI WINDOWS FULL HOST v0.2\n\n"
    "Bundled: WALDO/Mirror/Hermes handoff stack, native Windows executables, portable Python 3.12, "
    "CPU PyTorch, browser fallback, and the bounded project Workspace Hand.\n\n"
    "1. Put or configure an existing local WALDO model. No model weights are fabricated or downloaded.\n"
    "2. Copy WALMI_WORKSPACE_REQUEST_EXAMPLE.json and change its prompt and exact verifier command.\n"
    "3. Run the workspace command without --allow-write first.\n"
    "4. Add --allow-write only for a selected project root you intend WALMI to modify.\n\n"
    "The Hand refuses root escapes, .git, symlinks, unhashed updates/deletes, oversized transactions, "
    "shell verification, install, integration, promotion, automatic learning, and CANON authority.\n",
    encoding="utf-8")
shutil.copy2("portable/walmi-pc-package/WALMI_WORKSPACE_REQUEST_EXAMPLE.json", host / "WALMI_WORKSPACE_REQUEST_EXAMPLE.json")
for name in ["LICENSE", "NOTICE"]:
    shutil.copy2(name, host / name)
print("WALMI v0.2 host layout written")
