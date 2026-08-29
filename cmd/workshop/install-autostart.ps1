param(
    [string]$InstallRoot = "$env:LOCALAPPDATA\AXM\LocalWorkshop"
)

$ErrorActionPreference = "Stop"
$TaskName = "AXM Local Workshop"
$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$Exe = Join-Path $InstallRoot "axm-workshop.exe"

New-Item -ItemType Directory -Force -Path $InstallRoot | Out-Null

Write-Host "Building AXM Local Workshop..."
Push-Location $RepoRoot
try {
    & go build -o $Exe ./cmd/workshop
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
}
finally {
    Pop-Location
}

$Action = New-ScheduledTaskAction -Execute $Exe -WorkingDirectory $InstallRoot
$Trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
$Settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) -ExecutionTimeLimit ([TimeSpan]::Zero)

Register-ScheduledTask -TaskName $TaskName -Action $Action -Trigger $Trigger -Settings $Settings -Description "Starts the local AXM AI Workshop and Waldo native bridge at user logon." -Force | Out-Null
Start-ScheduledTask -TaskName $TaskName

Write-Host "AXM Local Workshop installed and started."
Write-Host "Open: http://127.0.0.1:7788"
Write-Host "Task: $TaskName"
Write-Host "Binary: $Exe"
