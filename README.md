# cockpit-codex-sticky

通俗名：**Cockpit Codex 会话粘性**

给 Cockpit Tools 的本地 Codex 反代（cockpit-cliproxy）加上类似 CPA 的能力：把同一次对话「粘」在同一账号/路由上（X-Codex-Turn-State），日志里搜 [turn-state]。  
顺带提供 [effort-detect]：看推理档位、推理 token、有没有 516 截断。

中文详细部署：[DEPLOY.zh-CN.md](./DEPLOY.zh-CN.md)

仓库：https://github.com/Kriswd/cockpit-codex-sticky  
（由 cockpit-effort-detect 更名而来，旧链接会跳转）

> 非官方补丁。Cockpit 升级后若覆盖了 cockpit-cliproxy.exe，请重新部署。

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
git clone https://github.com/Kriswd/cockpit-codex-sticky.git
cd cockpit-codex-sticky

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
