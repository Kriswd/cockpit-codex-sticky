# cockpit-effort-detect

Community patch for **Cockpit Tools** `cockpit-cliproxy`: sticky `X-Codex-Turn-State` helpers plus `[effort-detect]` logging.

It records the requested reasoning effort and, when present, `reasoning_tokens` / 516-style truncation fingerprints from upstream usage. Useful when DSH / local Codex API traffic may be silently truncated or feel "dumbed down".

> This is an unofficial patch. Cockpit client updates can overwrite `cockpit-cliproxy.exe` — re-run the deploy script (or your rebuild bat) after upgrading.

## What you get

- `pkg-turnstate/` — Go sources for `internal/turnstate`
  - `effort_detect.go` — parse effort + usage (including SSE tail)
  - `transport.go` — RoundTrip hooks that emit `[effort-detect]` lines
  - `cache.go` — turn-state cache helpers
- `apply_effort_detect.py` — copy sources into your CLIProxyAPI tree
- `build_and_replace.py` — `go build` and replace installed `cockpit-cliproxy.exe`
- `Deploy-EffortDetect.ps1` / `RUN-ALL-ON-WINDOWS.ps1` — one-shot Windows deploy
- `Install-Codex516Guard.ps1` — optional companion install for a standalone 516-guard proxy

## Requirements

- Windows (scripts are PowerShell / Python)
- Go toolchain
- A Cockpit Tools install that uses the Go sidecar `cockpit-cliproxy.exe`
- A source tree that contains  
  `sidecars/cockpit-cliproxy/third_party/CLIProxyAPI/internal/turnstate`  
  (set `COCKPIT_TURNSTATE_DIR` to that checkout root)

## Quick start

```powershell
git clone https://github.com/<OWNER>/cockpit-effort-detect.git
cd cockpit-effort-detect

# Point at YOUR patched / vendored cockpit-cliproxy source tree
$env:COCKPIT_TURNSTATE_DIR = "D:\path\to\cockpit-tools-turnstate"
# Optional: installed Cockpit Tools folder (contains cockpit-cliproxy.exe)
$env:COCKPIT_TOOLS_DIR = "D:\path\to\Cockpit Tools"

powershell -NoProfile -ExecutionPolicy Bypass -File .\Deploy-EffortDetect.ps1
```

Restart **Cockpit Tools**, send one request through the local Codex API (e.g. DSH → `http://127.0.0.1:61227/v1`), then check:

`%USERPROFILE%\.antigravity_cockpit\logs\codex-api.log.*`

Look for lines like:

```text
[effort-detect] 请求档位=xhigh 模型=gpt-6-astra 账号=codex_xx
[effort-detect] 请求档位=xhigh 模型=gpt-6-astra 推理token=311 截断指纹=未命中 账号=codex_xx
```

## Notes / limits

- Detects **requested effort**, **reasoning token counts**, and the **516 truncation fingerprint**. It does **not** by itself prevent upstream model swaps (e.g. astra served as luna).
- Streaming responses need the SSE-tail parser (included); older head-only sniffers often showed `推理token=-`.
- Optional `codex-516-guard` can continue/fold truncated streams; it is separate from DSH→61227 unless you point clients at the guard port.

## License

Sources are provided for testing and interoperability with Cockpit Tools / CLIProxyAPI-based sidecars. Respect upstream project licenses when redistributing binaries.
