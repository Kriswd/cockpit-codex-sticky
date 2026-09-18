# cockpit-effort-detect

Cockpit Tools `cockpit-cliproxy` 社区补丁：`[effort-detect]` 推理档位 / `reasoning_tokens` / 516 截断指纹日志，以及 turn-state helper。

**中文详细部署请看：[DEPLOY.zh-CN.md](./DEPLOY.zh-CN.md)**（推荐客户直接按该文档操作）。

仓库：https://github.com/Kriswd/cockpit-effort-detect

> 非官方补丁。Cockpit 客户端升级可能覆盖 `cockpit-cliproxy.exe`，升级后需重新部署。

## 快速开始（摘要）

```powershell
git clone https://github.com/Kriswd/cockpit-effort-detect.git
cd cockpit-effort-detect

$env:COCKPIT_TURNSTATE_DIR = "D:\path\to\cockpit-tools-turnstate"
$env:COCKPIT_TOOLS_DIR = "D:\path\to\Cockpit Tools"

powershell -NoProfile -ExecutionPolicy Bypass -File .\Deploy-EffortDetect.ps1
```

重启 Cockpit Tools，经本地 Codex API（如 `http://127.0.0.1:61227/v1`）发一条请求，在：

`%USERPROFILE%\.antigravity_cockpit\logs\codex-api.log.*`

中搜索 `[effort-detect]`。

## 能力边界

- **有**：请求档位、推理 token、516 截断指纹日志（含 SSE 尾部解析）
- **无**：DSH 档位 UI 插件、换模指纹（ModelTrace）、强制指定上游模型、预编译安装包、代理/TUN 网络调优

完整说明、验收清单与排障见 [DEPLOY.zh-CN.md](./DEPLOY.zh-CN.md)。

## License

MIT。再分发二进制时请同时遵守 Cockpit Tools / CLIProxyAPI 等上游许可。
