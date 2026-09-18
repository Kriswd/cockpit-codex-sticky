# Deploy [effort-detect] into custom cockpit-cliproxy and replace installed exe.
# Prerequisites:
#   - Go toolchain
#   - A cockpit-cliproxy source tree with internal/turnstate (set COCKPIT_TURNSTATE_DIR)
#   - Optional: COCKPIT_TOOLS_DIR pointing at the installed "Cockpit Tools" folder
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

if (-not $env:COCKPIT_TURNSTATE_DIR -or -not (Test-Path $env:COCKPIT_TURNSTATE_DIR)) {
  Write-Host "Set COCKPIT_TURNSTATE_DIR to your cockpit-tools-turnstate checkout, e.g.:"
  Write-Host '  $env:COCKPIT_TURNSTATE_DIR = "D:\path\to\cockpit-tools-turnstate"'
  if (Test-Path "D:\ProjectX\cockpit-tools-turnstate") {
    $env:COCKPIT_TURNSTATE_DIR = "D:\ProjectX\cockpit-tools-turnstate"
    Write-Host "Using fallback: $env:COCKPIT_TURNSTATE_DIR"
  } else {
    throw "COCKPIT_TURNSTATE_DIR is required"
  }
}

Write-Host "=== Part A: apply sources ==="
python "$Root\apply_effort_detect.py"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "=== Part A: build + replace ==="
python "$Root\build_and_replace.py"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$Note = @"
# effort-detect rebuild note
# Keep internal/turnstate/effort_detect.go (+ transport RoundTrip hooks) when rebuilding cockpit-cliproxy.
# Log tag: [effort-detect]
# Log file: %USERPROFILE%\.antigravity_cockpit\logs\codex-api.log.*
"@
Set-Content -Path (Join-Path $Root "EFFORT-DETECT-REBUILD-NOTE.txt") -Value $Note -Encoding UTF8
Write-Host "wrote EFFORT-DETECT-REBUILD-NOTE.txt"
Write-Host "=== Part A complete. Restart Cockpit Tools to load the new cockpit-cliproxy.exe. ==="
