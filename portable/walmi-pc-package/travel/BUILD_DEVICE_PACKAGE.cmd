@echo off
setlocal
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0WALMI_CARTRIDGE.ps1" -Action build-device %*
exit /b %ERRORLEVEL%
