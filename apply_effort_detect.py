# -*- coding: utf-8 -*-
"""Copy [effort-detect] turnstate sources into a cockpit-cliproxy CLIProxyAPI tree."""
from __future__ import annotations

import os
import shutil
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
SRC = HERE / "pkg-turnstate"

FILES = [
    "effort_detect.go",
    "effort_detect_test.go",
    "transport.go",
    "cache.go",
    "cache_test.go",
]


def resolve_dest() -> Path:
    env = os.environ.get("COCKPIT_TURNSTATE_DIR", "").strip()
    candidates = []
    if env:
        candidates.append(
            Path(env)
            / "sidecars"
            / "cockpit-cliproxy"
            / "third_party"
            / "CLIProxyAPI"
            / "internal"
            / "turnstate"
        )
    # common local layouts
    root = Path.cwd()
    candidates.extend(
        [
            root
            / "sidecars"
            / "cockpit-cliproxy"
            / "third_party"
            / "CLIProxyAPI"
            / "internal"
            / "turnstate",
            Path(r"D:\ProjectX\cockpit-tools-turnstate")
            / "sidecars"
            / "cockpit-cliproxy"
            / "third_party"
            / "CLIProxyAPI"
            / "internal"
            / "turnstate",
        ]
    )
    for c in candidates:
        if c.is_dir():
            return c
    raise SystemExit(
        "ERROR: turnstate destination not found. Set COCKPIT_TURNSTATE_DIR to your "
        "cockpit-tools-turnstate checkout (the folder that contains sidecars/cockpit-cliproxy)."
    )


def main() -> int:
    if not SRC.is_dir():
        print("ERROR: missing pkg-turnstate next to this script:", SRC)
        return 1
    dest = resolve_dest()
    dest.mkdir(parents=True, exist_ok=True)
    for name in FILES:
        s = SRC / name
        if not s.is_file():
            print("ERROR: missing source", s)
            return 1
        d = dest / name
        shutil.copy2(s, d)
        print("copied", name, "->", d)
    print("OK apply_effort_detect")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
