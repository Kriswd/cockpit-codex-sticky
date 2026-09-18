# -*- coding: utf-8 -*-
"""Build cockpit-cliproxy.exe from sidecar module and replace installed binary."""
from __future__ import annotations

import os
import shutil
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

HINT = (
    "Set COCKPIT_TURNSTATE_DIR to the checkout that contains sidecars/cockpit-cliproxy, "
    "and optionally COCKPIT_TOOLS_DIR to the installed Cockpit Tools folder."
)


def resolve_sidecar() -> Path:
    env = os.environ.get("COCKPIT_TURNSTATE_DIR", "").strip()
    candidates = []
    if env:
        candidates.append(Path(env) / "sidecars" / "cockpit-cliproxy")
    candidates.append(Path(r"D:\ProjectX\cockpit-tools-turnstate\sidecars\cockpit-cliproxy"))
    for c in candidates:
        if c.is_dir():
            return c
    raise SystemExit("ERROR: sidecar root missing. " + HINT)


def resolve_installed(sidecar: Path) -> Path:
    env = os.environ.get("COCKPIT_TOOLS_DIR", "").strip()
    candidates = []
    if env:
        candidates.append(Path(env) / "cockpit-cliproxy.exe")
        candidates.append(Path(env))
    candidates.append(Path(r"D:\ProjectX\KrisAI\cockpit-tools\Cockpit Tools\cockpit-cliproxy.exe"))
    for c in candidates:
        if c.is_file() and c.name.lower() == "cockpit-cliproxy.exe":
            return c
        if c.is_dir():
            exe = c / "cockpit-cliproxy.exe"
            if exe.is_file():
                return exe
    return sidecar / "cockpit-cliproxy.exe"


def run(cmd, cwd=None, check=True, env=None):
    print("+", " ".join(str(c) for c in cmd), f"(cwd={cwd})")
    p = subprocess.run(cmd, cwd=cwd, text=True, capture_output=True, env=env)
    if p.stdout:
        sys.stdout.write(p.stdout)
    if p.stderr:
        sys.stderr.write(p.stderr)
    if check and p.returncode != 0:
        raise SystemExit(p.returncode)
    return p


def stop_cliproxy():
    ps = r"""
Get-Process -ErrorAction SilentlyContinue | Where-Object {
  $_.ProcessName -eq 'cockpit-cliproxy' -or $_.ProcessName -like 'cockpit-cliproxy*'
} | ForEach-Object {
  Write-Output ("Stopping PID=$($_.Id) Name=$($_.ProcessName)")
  Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
}
Get-CimInstance Win32_Process -ErrorAction SilentlyContinue | Where-Object {
  $_.ExecutablePath -and ($_.ExecutablePath -like '*cockpit-cliproxy.exe*')
} | ForEach-Object {
  Write-Output ("Stopping path PID=$($_.ProcessId)")
  Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 1
"""
    run(["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps], check=False)


def main() -> int:
    sidecar = resolve_sidecar()
    cliproxy_mod = sidecar / "third_party" / "CLIProxyAPI"
    turnstate = cliproxy_mod / "internal" / "turnstate"
    installed = resolve_installed(sidecar)

    if not turnstate.is_dir():
        print("ERROR: turnstate dir missing", turnstate)
        return 1

    print("sidecar", sidecar)
    print("turnstate", turnstate)
    print("install target", installed)

    test_cwd = cliproxy_mod if (cliproxy_mod / "go.mod").exists() else sidecar
    run(["go", "test", "./internal/turnstate", "-count=1"], cwd=str(test_cwd), check=False)

    out = sidecar / "cockpit-cliproxy.exe"
    env = os.environ.copy()
    env["CGO_ENABLED"] = "0"

    p = run(["go", "build", "-o", str(out), "."], cwd=str(sidecar), check=False, env=env)
    if p.returncode != 0 or not out.is_file():
        server = cliproxy_mod / "cmd" / "server"
        if (server / "main.go").exists():
            p2 = run(
                ["go", "build", "-o", str(out), "./cmd/server"],
                cwd=str(cliproxy_mod),
                check=False,
                env=env,
            )
            if p2.returncode != 0 or not out.is_file():
                print("ERROR: go build failed")
                return p2.returncode or 1
        else:
            print("ERROR: go build failed")
            return p.returncode or 1

    print("built", out, "size", out.stat().st_size)
    stop_cliproxy()
    time.sleep(1)

    installed.parent.mkdir(parents=True, exist_ok=True)
    if installed.exists():
        bak = installed.with_suffix(
            installed.suffix + f".bak-{datetime.now().strftime('%Y%m%d-%H%M%S')}"
        )
        shutil.copy2(installed, bak)
        print("backup", bak)
    shutil.copy2(out, installed)
    print("installed", installed)
    print("OK build_and_replace")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
