# Master deploy for Part A + Part B. Run on <YOUR-PC>.
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Write-Host "Running from $Root"
& powershell -NoProfile -ExecutionPolicy Bypass -File "$Root\Deploy-EffortDetect.ps1"
& powershell -NoProfile -ExecutionPolicy Bypass -File "$Root\Install-Codex516Guard.ps1"
Write-Host "ALL DONE. Restart Cockpit, then one DSH astra Xhigh request; search [effort-detect] in codex-api.log.*"
