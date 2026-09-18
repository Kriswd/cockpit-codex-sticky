# 详细部署文档（中文）

本文说明如何把 **cockpit-effort-detect** 部署到 Windows 上的 Cockpit Tools，并验收 `[effort-detect]` 日志是否生效。

仓库：https://github.com/Kriswd/cockpit-effort-detect

---

## 1. 这个补丁能做什么 / 不能做什么

### 能做

- 在 `cockpit-cliproxy` 里增加 **`[effort-detect]`** 日志：
  - 客户端请求的推理档位（如 `low` / `medium` / `high` / `xhigh` / `max`）
  - 上游返回的 `reasoning_tokens`（含 SSE 流式尾部 usage）
  - 是否命中 **516 截断指纹**（`reasoning_tokens == 518*n - 2`，例如 516、1034、1552）
- 附带 turn-state 相关 helper（`cache.go` 等），用于粘性会话场景（需你的源码树已接入 turnstate 传输层）

### 不能做（请先和管理预期对齐）

| 能力 | 本仓库是否包含 |
| --- | --- |
| DSH 界面里给模型选推理档位（dsh-effort-config） | 否 |
| 检测「请求 astra、实际 luna」这类换模 | 否（可用 ModelTrace 等另测） |
| 强制上游一定给你指定模型 | 否 |
| 现成编译好的 `cockpit-cliproxy.exe` 安装包 | 否（需本机用 Go 编译） |
| FlClash / 系统代理出网稳定性调优 | 否（网络侧配置，与本补丁无关） |

**结论：** 部署成功后，客户能得到和「本机补丁生效后」一样的 **检测日志能力**；不会自动拥有你本机整套周边工具链。

---

## 2. 环境要求

请逐项确认：

1. **Windows 10/11**
2. 已安装并会用 **Cockpit Tools**，且本机 Codex API 走 sidecar：`cockpit-cliproxy.exe`（常见监听 `http://127.0.0.1:61227/v1`）
3. **Go 工具链**（`go version` 能跑；建议 Go 1.22+）
4. **Python 3**（部署脚本用）
5. **一份可编译的 cliproxy 源码树**，且已存在目录：

```text
<COCKPIT_TURNSTATE_DIR>\sidecars\cockpit-cliproxy\third_party\CLIProxyAPI\internal\turnstate
```

以及 sidecar 根目录可 `go build`：

```text
<COCKPIT_TURNSTATE_DIR>\sidecars\cockpit-cliproxy\
```

> 若只有官方安装目录、没有上述源码树，**无法**仅靠本仓库完成替换。需要先准备/同步一份带 `internal/turnstate` 的 CLIProxyAPI / Cockpit sidecar 源码（与你平时编译 `cockpit-cliproxy.exe` 的那份一致）。

6. 已安装的 Cockpit Tools 目录里有待替换的二进制，例如：

```text
<COCKPIT_TOOLS_DIR>\cockpit-cliproxy.exe
```

---

## 3. 部署步骤

### 3.1 获取本仓库

```powershell
git clone https://github.com/Kriswd/cockpit-effort-detect.git
cd cockpit-effort-detect
```

### 3.2 设置路径环境变量

把下面两个路径改成客户自己的真实路径：

```powershell
# 必填：含 sidecars\cockpit-cliproxy 的源码根
$env:COCKPIT_TURNSTATE_DIR = "D:\path\to\cockpit-tools-turnstate"

# 建议填写：已安装的 Cockpit Tools 目录（里面有 cockpit-cliproxy.exe）
$env:COCKPIT_TOOLS_DIR = "D:\path\to\Cockpit Tools"
```

自检：

```powershell
Test-Path "$env:COCKPIT_TURNSTATE_DIR\sidecars\cockpit-cliproxy\third_party\CLIProxyAPI\internal\turnstate"
Test-Path "$env:COCKPIT_TOOLS_DIR\cockpit-cliproxy.exe"
go version
python --version
```

两项 `Test-Path` 都应返回 `True`。

### 3.3 一键部署

在仓库根目录执行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Deploy-EffortDetect.ps1
```

脚本会：

1. 把 `pkg-turnstate\*` 拷进源码树的 `internal\turnstate`
2. `go test` / `go build` 生成新的 `cockpit-cliproxy.exe`
3. 停掉正在跑的 `cockpit-cliproxy` 进程
4. 备份旧 exe，再覆盖安装目录里的 `cockpit-cliproxy.exe`

也可以分步：

```powershell
python .\apply_effort_detect.py
python .\build_and_replace.py
```

### 3.4 重启 Cockpit Tools

完全退出并重新打开 **Cockpit Tools**，确认 Codex API / sidecar 重新拉起（任务管理器里应重新出现 `cockpit-cliproxy.exe`）。

> Cockpit 客户端升级后，官方安装包常会 **覆盖** `cockpit-cliproxy.exe`。升级后请重新执行 3.3。

---

## 4. 验收（必须做）

### 4.1 发一笔会经过本地 API 的请求

任选一种：

- DSH / 其他客户端，Base URL 指向 Cockpit 本地 Codex API，例如 `http://127.0.0.1:61227/v1`
- 或对本地 API 直接打一条 chat completions

建议带上明确档位，例如 `reasoning_effort=xhigh`（若客户端 UI 可选则选极高/Xhigh）。

### 4.2 查日志

日志目录：

```text
%USERPROFILE%\.antigravity_cockpit\logs\
```

文件名类似：`codex-api.log.YYYY-MM-DD`

在最新日志中搜索：`[effort-detect]`

成功时大致会看到：

```text
[effort-detect] 请求档位=xhigh 模型=gpt-6-astra 账号=codex_xxxx
[effort-detect] 请求档位=xhigh 模型=gpt-6-astra 推理token=311 截断指纹=未命中 input=... output=... total=... 账号=codex_xxxx
```

说明：

- **请求档位**：客户端发出去的档位（证明有没有带上）
- **推理token**：上游 usage 里的 reasoning tokens；流式响应需等流结束后才会出现在第二条日志
- **截断指纹**：`未命中` 或命中 516 系列；命中不代表「换模」，只说明推理长度像被截断过

若只有请求档位、`推理token=-`：

- 可能请求还在飞、或非流式/usage 字段缺失
- 本补丁已包含 SSE **尾部**解析；若仍长期为 `-`，把该次 `requestId` 前后日志贴出排查

### 4.3 可选：PowerShell 快速搜今天的日志

```powershell
$log = Get-ChildItem "$env:USERPROFILE\.antigravity_cockpit\logs\codex-api.log.*" |
  Sort-Object LastWriteTime -Descending |
  Select-Object -First 1
Select-String -Path $log.FullName -Pattern '\[effort-detect\]' | Select-Object -Last 20
```

---

## 5. 可选：codex-516-guard

`Install-Codex516Guard.ps1` 用于安装/拉起一个 **独立** 的 516 截断续写代理（常见监听 `127.0.0.1:8787`）。

注意：

- 它 **不会** 自动接管 DSH → `61227` 的流量
- 只有把客户端指到 guard 端口时才会生效
- 与 `[effort-detect]` 日志是两件不同的事：一个是检测，一个是（可选）缓解截断

没有 516 问题的客户可以跳过。

---

## 6. 常见问题

### Q1：`turnstate destination not found` / `sidecar root missing`

未设对 `COCKPIT_TURNSTATE_DIR`，或源码树没有 `internal\turnstate`。先保证平时就能在该树里 `go build` 出 cliproxy。

### Q2：编译失败

- 确认在 `sidecars\cockpit-cliproxy` 下官方/现有工程本身能编译
- 本补丁只覆盖 `internal\turnstate` 下若干文件；若你的树结构不同，需手动对齐 import 路径后再编

### Q3：替换 exe 时文件被占用

脚本会尝试结束 `cockpit-cliproxy`；若仍失败，先退出 Cockpit Tools 再跑 `build_and_replace.py`。

### Q4：重启后日志里没有 `[effort-detect]`

1. 确认实际运行的是刚替换的 exe（看进程路径、文件修改时间）
2. 确认请求确实打到该 sidecar（端口、API Key）
3. Cockpit 是否又拉起了另一份内置二进制

### Q5：客户感觉「还是蠢 / 还是慢」

本补丁 **只负责看得见档位与截断指纹**。若 `response.model` 或指纹显示上游不是你以为的模型，那是账号/上游路由问题，不是本补丁能「改回」的。

### Q6：和官方 Codex 客户端比更不稳（TLS handshake EOF）

多半是出网代理路径问题（例如系统 HTTP 代理 + TUN 双层），与本补丁无关。可对照：官方 Codex 是否也走同一 HTTP 代理；必要时让 cliproxy 与官方客户端走同一出网方式。

---

## 7. 推荐验收清单（给客户打勾）

- [ ] `go version` / `python --version` 正常
- [ ] `COCKPIT_TURNSTATE_DIR`、`COCKPIT_TOOLS_DIR` 路径正确
- [ ] `Deploy-EffortDetect.ps1` 成功结束
- [ ] 已重启 Cockpit Tools
- [ ] 发过至少 1 次经本地 API 的请求
- [ ] 日志中出现带 **请求档位** 的 `[effort-detect]`
- [ ] 流结束后出现带 **推理token** 的 `[effort-detect]`（或明确记录 usage 缺失）
- [ ] 已知：本补丁不解决换模，不附带 DSH 档位插件

---

## 8. 目录说明

```text
cockpit-effort-detect/
  README.md                 # 英文简介
  DEPLOY.zh-CN.md           # 本中文详细文档
  LICENSE
  pkg-turnstate/            # 打进 internal/turnstate 的 Go 源码
  apply_effort_detect.py    # 拷贝源码
  build_and_replace.py      # 编译并替换 exe
  Deploy-EffortDetect.ps1   # 一键部署
  RUN-ALL-ON-WINDOWS.ps1    # 可选总入口
  Install-Codex516Guard.ps1 # 可选 516-guard
```

有问题请带着：操作系统、`go version`、两个环境变量路径、部署脚本完整输出、以及一段含 `requestId` 的 `codex-api` 日志（可打码账号邮箱）。
