param(
    [ValidateSet("start", "prepare", "check", "autolearn", "observe", "play-simulators", "build-device")]
    [string]$Action = "start",
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Arguments
)

$ErrorActionPreference = "Stop"
$cartridgeRoot = [System.IO.Path]::GetFullPath($PSScriptRoot)
$manifestPath = Join-Path $cartridgeRoot "walmi-cartridge.json"
if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
    throw "WALMI cartridge manifest is missing: $manifestPath"
}

$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
$archive = Join-Path $cartridgeRoot $manifest.runtime.archive
if (-not (Test-Path -LiteralPath $archive -PathType Leaf)) {
    throw "WALMI runtime pack is missing: $archive"
}
if (-not (Test-Path -LiteralPath "D:\" -PathType Container)) {
    throw "WALMI requires the D: drive for its runtime cache."
}

$cacheBase = "D:\WALMI_RUNTIME_CACHE"
$runtimeKey = $manifest.runtime.sha256.Substring(0, 20)
$runtimeRoot = Join-Path $cacheBase ("runtime-" + $runtimeKey)
$readyPath = Join-Path $runtimeRoot ".walmi-runtime-ready.json"
$required = @(
    "bin\waldo.exe",
    "runtime\python\python.exe",
    "tools\walmi_portable_bootstrap.py",
    "tools\walmi_autolearn.py"
    "tools\walmi_simulator_bridge.py"
    "tools\walmi_node_world_adapter.mjs"
)

function Test-WalmiRuntimeReady {
    if (-not (Test-Path -LiteralPath $readyPath -PathType Leaf)) { return $false }
    try {
        $ready = Get-Content -LiteralPath $readyPath -Raw | ConvertFrom-Json
    } catch {
        return $false
    }
    if ($ready.runtimeSha256 -ne $manifest.runtime.sha256) { return $false }
    foreach ($relative in $required) {
        if (-not (Test-Path -LiteralPath (Join-Path $runtimeRoot $relative) -PathType Leaf)) {
            return $false
        }
    }
    return $true
}

if (-not (Test-WalmiRuntimeReady)) {
    $actualHash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $manifest.runtime.sha256) {
        throw "WALMI runtime pack failed its SHA-256 check."
    }
    New-Item -ItemType Directory -Path $cacheBase -Force | Out-Null
    if (Test-Path -LiteralPath $runtimeRoot) {
        throw "An incomplete WALMI runtime cache already exists: $runtimeRoot"
    }
    $staging = Join-Path $cacheBase (".extract-" + $runtimeKey + "-" + $PID)
    if (Test-Path -LiteralPath $staging) {
        throw "WALMI extraction staging path already exists: $staging"
    }
    New-Item -ItemType Directory -Path $staging | Out-Null
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $stagingRoot = [System.IO.Path]::GetFullPath($staging).TrimEnd([System.IO.Path]::DirectorySeparatorChar) + [System.IO.Path]::DirectorySeparatorChar
    $zip = [System.IO.Compression.ZipFile]::OpenRead($archive)
    try {
        foreach ($entry in $zip.Entries) {
            $entryPath = $entry.FullName.Replace('/', [System.IO.Path]::DirectorySeparatorChar)
            $resolved = [System.IO.Path]::GetFullPath([System.IO.Path]::Combine($stagingRoot, $entryPath))
            if (-not $resolved.StartsWith($stagingRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
                throw "Unsafe WALMI runtime entry: $($entry.FullName)"
            }
        }
    } finally {
        $zip.Dispose()
    }
    [System.IO.Compression.ZipFile]::ExtractToDirectory($archive, $staging)
    foreach ($relative in $required) {
        if (-not (Test-Path -LiteralPath (Join-Path $staging $relative) -PathType Leaf)) {
            throw "Prepared WALMI runtime is missing $relative"
        }
    }
    $ready = [ordered]@{
        schema = "walmi.runtime-cache/v1"
        runtimeSha256 = $manifest.runtime.sha256
        preparedFrom = $manifest.name
    } | ConvertTo-Json
    Set-Content -LiteralPath (Join-Path $staging ".walmi-runtime-ready.json") -Value $ready -Encoding UTF8
    Move-Item -LiteralPath $staging -Destination $runtimeRoot
    Write-Host "WALMI runtime prepared on D:."
} else {
    Write-Host "WALMI runtime ready on D:."
}

$dataRoot = if ($env:WALMI_CARTRIDGE_DATA_HOME) {
    [System.IO.Path]::GetFullPath($env:WALMI_CARTRIDGE_DATA_HOME)
} else {
    Join-Path $cartridgeRoot "WALMI_DATA"
}
$stateRoot = Join-Path $dataRoot "state"
foreach ($path in @(
    $dataRoot,
    $stateRoot,
    (Join-Path $stateRoot "tmp"),
    (Join-Path $stateRoot "pip-cache"),
    (Join-Path $stateRoot "torch-cache"),
    (Join-Path $stateRoot "model-cache"),
    (Join-Path $stateRoot "pycache"),
    (Join-Path $dataRoot "experience"),
    (Join-Path $dataRoot "models"),
    (Join-Path $dataRoot "checkpoints"),
    (Join-Path $dataRoot "rollback")
)) {
    New-Item -ItemType Directory -Path $path -Force | Out-Null
}

$python = Join-Path $runtimeRoot "runtime\python\python.exe"
$waldo = Join-Path $runtimeRoot "bin\waldo.exe"
$env:WALMI_HOME = $runtimeRoot
$env:WALMI_DATA_HOME = $dataRoot
$env:PYTHONHOME = Join-Path $runtimeRoot "runtime\python"
$env:PATH = (Join-Path $runtimeRoot "runtime\python") + ";" + (Join-Path $runtimeRoot "runtime\python\Scripts") + ";" + (Join-Path $runtimeRoot "runtime\node") + ";" + $env:PATH
$env:TEMP = Join-Path $stateRoot "tmp"
$env:TMP = $env:TEMP
$env:PIP_CACHE_DIR = Join-Path $stateRoot "pip-cache"
$env:TORCH_HOME = Join-Path $stateRoot "torch-cache"
$env:HF_HOME = Join-Path $stateRoot "model-cache"
$env:XDG_CACHE_HOME = $env:HF_HOME
$env:PYTHONPYCACHEPREFIX = Join-Path $stateRoot "pycache"
$env:WALDO_CONFIG = Join-Path $stateRoot "waldo-config.json"

& $python (Join-Path $runtimeRoot "tools\walmi_portable_bootstrap.py")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

switch ($Action) {
    "prepare" {
        Write-Host "WALMI cartridge prepared."
        exit 0
    }
    "check" {
        & $waldo --json status
        exit $LASTEXITCODE
    }
    "autolearn" {
        & $python (Join-Path $runtimeRoot "tools\walmi_autolearn.py") @Arguments
        exit $LASTEXITCODE
    }
    "observe" {
        if (-not $Arguments -or -not $Arguments[0]) {
            throw "Usage: WALMI_EXPERIENCE_OBSERVE.cmd <outcome.json>"
        }
        $outcome = [System.IO.Path]::GetFullPath($Arguments[0])
        & $waldo mirror experience observe $outcome `
            --ledger (Join-Path $dataRoot "experience\direct-experience.jsonl") `
            --learn-to (Join-Path $dataRoot "experience\training-projections.jsonl") `
            --hermes-to (Join-Path $dataRoot "experience\hermes-memory.jsonl")
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        & $python (Join-Path $runtimeRoot "tools\walmi_autolearn.py")
        exit $LASTEXITCODE
    }
    "play-simulators" {
        $playArguments = @($Arguments)
        if (-not ($playArguments -contains "--player")) {
            $playArguments = @("--player", "model", "--model", "walmi") + $playArguments
        }
        & $python (Join-Path $runtimeRoot "tools\walmi_simulator_bridge.py") `
            --data-home $dataRoot `
            --waldo-bin $waldo `
            @playArguments
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        & $python (Join-Path $runtimeRoot "tools\walmi_autolearn.py")
        exit $LASTEXITCODE
    }
    "build-device" {
        & $python (Join-Path $runtimeRoot "tools\walmi_device_pack_builder.py") @Arguments
        exit $LASTEXITCODE
    }
    default {
        Write-Host "WALMI portable cartridge"
        Write-Host "Personal data: $dataRoot"
        & $waldo advisor WALMI
        exit $LASTEXITCODE
    }
}
