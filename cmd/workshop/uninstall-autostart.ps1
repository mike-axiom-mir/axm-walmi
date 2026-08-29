param(
    [string]$InstallRoot = "$env:LOCALAPPDATA\AXM\LocalWorkshop"
)

$ErrorActionPreference = "Stop"
$TaskName = "AXM Local Workshop"

$Task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($Task) {
    Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
}

$Exe = Join-Path $InstallRoot "axm-workshop.exe"
if (Test-Path $Exe) {
    Remove-Item -Force $Exe
}

Write-Host "AXM Local Workshop auto-start removed."
Write-Host "Workshop data was NOT deleted."
