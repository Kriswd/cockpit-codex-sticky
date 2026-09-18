# Trial install of codex-516-guard on Windows. Does NOT change DSH→61227 routing.
$ErrorActionPreference = 'Stop'
$InstallDir = '%CODEX_516_GUARD_DIR%'
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

function Have-Cmd($name) {
  return [bool](Get-Command $name -ErrorAction SilentlyContinue)
}

Write-Host "=== checking python/uv ==="
$uv = Have-Cmd 'uv'
$py = Have-Cmd 'python'
Write-Host "uv=$uv python=$py"

if (-not $uv) {
  Write-Host "Installing uv via official script..."
  # Prefer official installer when available
  try {
    powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://astral.sh/uv/install.ps1 | iex"
  } catch {
    Write-Host "uv install script failed: $_"
  }
  $env:Path = [System.Environment]::GetEnvironmentVariable('Path','User') + ';' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + $env:Path
  $uv = Have-Cmd 'uv'
}

$HostBind = '127.0.0.1'
$Port = 8787
# Upstream: Cockpit local OpenAI-compatible API (Responses may or may not be exposed).
# Guard is Responses-oriented; default ChatGPT Codex upstream left as fallback.
$UpstreamCockpit = 'http://127.0.0.1:61227/v1'
# For Responses-compatible Codex path through Cockpit, try without /v1 if needed.
# Guard appends /responses to GUARD_UPSTREAM_BASE.

if ($uv) {
  Write-Host "uv tool install codex-516-guard"
  & uv tool install codex-516-guard
  if ($LASTEXITCODE -ne 0) { throw "uv tool install failed" }
  $exe = (Get-Command codex-516-guard -ErrorAction SilentlyContinue)
  if (-not $exe) {
    # common uv tools path on Windows
    $cand = Join-Path $env:USERPROFILE '.local\bin\codex-516-guard.exe'
    if (Test-Path $cand) { $exePath = $cand } else { $exePath = 'codex-516-guard' }
  } else { $exePath = $exe.Source }
} elseif ($py) {
  Write-Host "pip install --user codex-516-guard (needs Python>=3.12)"
  & python -m pip install --user 'codex-516-guard==0.2.8'
  $exePath = 'codex-516-guard'
} else {
  throw 'Neither uv nor python available'
}

# HOW-TO-USE
$how = @"
codex-516-guard trial (Windows)
===============================
Installed: $exePath
Listen:    http://${HostBind}:${Port}
Health:    http://${HostBind}:${Port}/healthz

What it does
------------
Detects Codex 516 truncation fingerprint:
  usage.output_tokens_details.reasoning_tokens == 518*n - 2
(n=1..6 in guard; observed ladder 516/1034/1552/...)
then auto-continues + folds rounds into one Responses stream.

API surface
-----------
This tool is a **Responses** (+ WebSocket responses) proxy for Codex CLI.
It is NOT a drop-in for DSH chat/completions.

Current DSH path (DO NOT break):
  DSH -> Cockpit http://localhost:61227/v1  (chat/completions)
  Effort/truncation visibility for DSH is covered by Part A [effort-detect] logs
  in %USERPROFILE%\.antigravity_cockpit\logs\codex-api.log.*

Trial with Codex CLI
--------------------
1. Start guard (see start script).
2. In ~/.codex/config.toml (top-level, before first [table]):
     openai_base_url = "http://127.0.0.1:8787/v1"
3. Optionally point guard upstream at Cockpit if Cockpit exposes Responses:
     codex-516-guard --host 127.0.0.1 --port 8787 --upstream http://127.0.0.1:61227/v1
   Default upstream (if omitted) is https://chatgpt.com/backend-api/codex
4. Revert: comment out openai_base_url and stop the guard. DSH/61227 unchanged.

Start / Stop
------------
  Start:  powershell -File %CODEX_516_GUARD_DIR%\Start-Guard.ps1
  Stop:   powershell -File %CODEX_516_GUARD_DIR%\Stop-Guard.ps1
  Health: curl http://127.0.0.1:8787/healthz

Limitations vs Sub2API
----------------------
- Sub2API-style relays often strip reasoning blocks; then continue/fold cannot work.
- Guard does not fold chat/completions; DSH stays on Part A logging only.
- Continuation adds real token cost (extra rounds).
- Unofficial workaround for openai/codex#30364.
"@
Set-Content -Path (Join-Path $InstallDir 'HOW-TO-USE.txt') -Value $how -Encoding UTF8

# Start/Stop scripts
$start = @"
`$ErrorActionPreference = 'Stop'
`$log = '%CODEX_516_GUARD_DIR%\guard.log'
`$err = '%CODEX_516_GUARD_DIR%\guard.err.log'
# Prefer Cockpit as upstream when Responses is available; otherwise default ChatGPT Codex.
`$args = @('--host','$HostBind','--port','$Port','--log-level','info')
# Uncomment to force Cockpit upstream:
# `$args += @('--upstream','$UpstreamCockpit')
Write-Host "Starting codex-516-guard on ${HostBind}:${Port}"
`$p = Start-Process -FilePath '$exePath' -ArgumentList `$args -WindowStyle Hidden -RedirectStandardOutput `$log -RedirectStandardError `$err -PassThru
Set-Content -Path '%CODEX_516_GUARD_DIR%\guard.pid' -Value `$p.Id
Start-Sleep -Seconds 2
try {
  (Invoke-WebRequest -Uri 'http://${HostBind}:${Port}/healthz' -UseBasicParsing -TimeoutSec 3).Content
} catch {
  Write-Host "health check failed: `$(`$_.Exception.Message)"
  Write-Host "see `$log / `$err"
}
"@
Set-Content -Path (Join-Path $InstallDir 'Start-Guard.ps1') -Value $start -Encoding UTF8

$stop = @"
`$ErrorActionPreference = 'Continue'
`$pidFile = '%CODEX_516_GUARD_DIR%\guard.pid'
if (Test-Path `$pidFile) {
  `$id = Get-Content `$pidFile | Select-Object -First 1
  if (`$id) { Stop-Process -Id ([int]`$id) -Force -ErrorAction SilentlyContinue; Write-Host "stopped pid=`$id" }
  Remove-Item `$pidFile -Force -ErrorAction SilentlyContinue
}
Get-CimInstance Win32_Process -ErrorAction SilentlyContinue | Where-Object {
  `$_.CommandLine -and (`$_.CommandLine -match 'codex-516-guard')
} | ForEach-Object {
  Write-Host "stopping leftover PID=`$(`$_.ProcessId)"
  Stop-Process -Id `$_.ProcessId -Force -ErrorAction SilentlyContinue
}
Write-Host 'guard stopped (DSH 61227 untouched)'
"@
Set-Content -Path (Join-Path $InstallDir 'Stop-Guard.ps1') -Value $stop -Encoding UTF8

Write-Host "Wrote $InstallDir\HOW-TO-USE.txt and Start/Stop scripts"
Write-Host "Starting trial service..."
& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $InstallDir 'Start-Guard.ps1')
Write-Host "=== Part B install done ==="
