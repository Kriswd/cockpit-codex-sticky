# cockpit-effort-detect

Cockpit Tools `cockpit-cliproxy` 社区补丁，一个仓库两件事：

1. **主功能（类似 CPA）Turn-State 粘性** — 缓存并回注 `X-Codex-Turn-State`（292 字节），同一会话尽量粘在同一账号/路由；日志 `[turn-state]`
2. **附带** `[effort-detect]` — 记录请求推理档位、`reasoning_tokens`、516 截断指纹

**中文详细部署：[DEPLOY.zh-CN.md](./DEPLOY.zh-CN.md)**（推荐客户按此操作）

仓库：https://github.com/Kriswd/cockpit-effort-detect

> 非官方补丁。Cockpit 客户端升级可能覆盖 `cockpit-cliproxy.exe`，升级后请重新部署。

## 主效果：Turn-State（你要分享给别人测的那个）

CPA / CLIProxyAPI 一类网关会用上游返回的 `X-Codex-Turn-State` 做会话粘性。Cockpit 本地 sidecar 默认不一定记住它。本补丁在 RoundTrip 外层：

- 从上游响应里 **记住** 合法的 292 字节 turn-state
- 同一 `账号 + 模型 + 会话` 后续请求自动 **回注** 该头
- 日志里用通俗中文打 `[turn-state]`（记住 / 注入 / 跳过等）

用来减轻「同一线程乱跳账号、粘性丢失」类问题。  
它 **不能** 强制上游一定给你 astra 而不是 luna。

## 附带：effort-detect

部署同一套二进制后，还会在 `codex-api.log` 里看到 `[effort-detect]`（档位 / 推理 token / 516 指纹）。详见部署文档后半部分。

## 快速开始

```powershell
git clone https://github.com/Kriswd/cockpit-effort-detect.git
cd cockpit-effort-detect

$env:COCKPIT_TURNSTATE_DIR = "D:\path\to\cockpit-tools-turnstate"
$env:COCKPIT_TOOLS_DIR = "D:\path\to\Cockpit Tools"

powershell -NoProfile -ExecutionPolicy Bypass -File .\Deploy-EffortDetect.ps1
# 或（路径写死时）用桌面同款逻辑：
# .\Reinstall-Cockpit-TurnState.bat
```

重启 Cockpit Tools，经本地 API 多轮对话后，在：

`%USERPROFILE%\.antigravity_cockpit\logs\codex-api.log.*`

中搜索 **`[turn-state]`**（主验收）和可选的 **`[effort-detect]`**。

## License

MIT。再分发二进制时请同时遵守 Cockpit Tools / CLIProxyAPI 等上游许可。
