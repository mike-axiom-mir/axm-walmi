$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$go = (Get-Command go -ErrorAction Stop).Source
$python = (Get-Command python -ErrorAction Stop).Source
$nativePackages = @(
    "./cmd/waldo-axm-mirror",
    "./cmd/workshop",
    "./composes",
    "./internal/axmmirror",
    "./internal/cli",
    "./internal/inference",
    "./internal/model",
    "./internal/modelweights",
    "./internal/training"
)
$pythonTests = @(
    "portable/walmi-pc-package/test_walmi_simulator_bridge.py",
    "portable/walmi-pc-package/test_walmi_autolearn.py",
    "portable/walmi-pc-package/test_walmi_device_pack_builder.py",
    "portable/walmi-pc-package/test_walmi_portable_bootstrap.py"
)

Push-Location $repoRoot
try {
    & $go test @nativePackages
    if ($LASTEXITCODE -ne 0) { throw "native WALMI Go tests failed" }

    $oldGoos = [Environment]::GetEnvironmentVariable("GOOS", "Process")
    $oldGoarch = [Environment]::GetEnvironmentVariable("GOARCH", "Process")
    $wasmOutput = Join-Path ([IO.Path]::GetTempPath()) ("walmi-browser-" + [guid]::NewGuid().ToString("N") + ".wasm")
    try {
        $env:GOOS = "js"
        $env:GOARCH = "wasm"
        & $go build -o $wasmOutput ./cmd/walmi-browser-wasm
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $wasmOutput -PathType Leaf)) {
            throw "browser WebAssembly cross-build failed"
        }
    }
    finally {
        [Environment]::SetEnvironmentVariable("GOOS", $oldGoos, "Process")
        [Environment]::SetEnvironmentVariable("GOARCH", $oldGoarch, "Process")
        if (Test-Path -LiteralPath $wasmOutput -PathType Leaf) {
            Remove-Item -LiteralPath $wasmOutput -Force
        }
    }

    $env:PYTHONDONTWRITEBYTECODE = "1"
    foreach ($test in $pythonTests) {
        & $python $test
        if ($LASTEXITCODE -ne 0) { throw "Python test failed: $test" }
    }

    & $python tools/seal_source_tree.py --verify
    if ($LASTEXITCODE -ne 0) { throw "source manifest verification failed" }
    Write-Output "WALMI_WINDOWS_VERIFICATION_PASS"
}
finally {
    Pop-Location
}
