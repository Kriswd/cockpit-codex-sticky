# 详细部署文档（中文）

仓库：https://github.com/Kriswd/cockpit-effort-detect

本仓库同时提供：

| 优先级 | 能力 | 日志标签 | 像什么 |
| --- | --- | --- | --- |
| **主** | Turn-State 粘性：缓存/回注 `X-Codex-Turn-State` | `[turn-state]` | 类似 CPA 的会话粘性 |
| 附 | 推理档位 / reasoning token / 516 指纹检测 | `[effort-detect]` | 观测用，不改路由 |

客户若只想验证「和 CPA 类似的粘性」，按本文 **第 1～4 节** 即可；第 5 节起是附带检测能力。

---

## 1. Turn-State 是什么效果

上游 Codex 有时会在响应里带回约 **292 字节** 的 `X-Codex-Turn-State`。  
CPA 一类代理会把它记下来，并在同会话后续请求里再带回去，减少账号/路由乱跳。

本补丁给 Cockpit 的 `cockpit-cliproxy` 加上同样思路：

1. 响应里出现合法 turn-state → **记住**（按 账号 + 模型 + 会话 键缓存，默认约 30 分钟）
2. 后续同键请求若客户端没带该头 → **自动注入**
3. 日志打 `[turn-state]`，方便确认有没有生效

### 做得到

- 多轮 / 续写场景下更稳的会话粘性（与 CPA 目标同类）
- 用日志证明「记住了 / 注入了」

### 做不到

- 不能禁止上游把 `gpt-6-astra` 静默换成 `gpt-5.6-luna`
- 不附带 DSH 推理档位 UI（那是别的插件）
- 不附带预编译安装包（需本机 Go 编译替换 exe）

---

## 2. 环境要求

1. Windows 10/11  
2. 已安装 Cockpit Tools，且 Codex API 走 `cockpit-cliproxy.exe`（常见 `http://127.0.0.1:61227/v1`）  
3. Go（建议 1.22+，`go version` 可用）  
4. Python 3（部署脚本用）  
5. **可编译的 sidecar 源码树**，且已有：

```text
<COCKPIT_TURNSTATE_DIR>\sidecars\cockpit-cliproxy\third_party\CLIProxyAPI\internal\turnstate
```

以及可在该 sidecar 根目录 `go build`。

6. 安装目录中有待替换二进制：

```text
<COCKPIT_TOOLS_DIR>\cockpit-cliproxy.exe
```

没有源码树则无法部署——本仓库只提供 `pkg-turnstate` 补丁文件与脚本，不含完整 Cockpit 源码。

---

## 3. 部署步骤（主流程）

### 3.1 克隆

```powershell
git clone https://github.com/Kriswd/cockpit-effort-detect.git
cd cockpit-effort-detect
```

### 3.2 设置路径

```powershell
$env:COCKPIT_TURNSTATE_DIR = "D:\path\to\cockpit-tools-turnstate"
$env:COCKPIT_TOOLS_DIR = "D:\path\to\Cockpit Tools"
```

自检：

```powershell
Test-Path "$env:COCKPIT_TURNSTATE_DIR\sidecars\cockpit-cliproxy\third_party\CLIProxyAPI\internal\turnstate"
Test-Path "$env:COCKPIT_TOOLS_DIR\cockpit-cliproxy.exe"
go version
python --version
```

### 3.3 一键编译替换

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Deploy-EffortDetect.ps1
```

脚本会把 `pkg-turnstate\`（含 **cache / transport / effort_detect**）拷进源码树、`go build`、备份并替换安装目录里的 `cockpit-cliproxy.exe`。

也可：

```powershell
python .\apply_effort_detect.py
python .\build_and_replace.py
```

若路径已写死在本机习惯目录，也可参考 `Reinstall-Cockpit-TurnState.bat`（需按客户路径改 SRC / INSTALL）。

### 3.4 重启 Cockpit Tools

完全退出再打开，确认新的 `cockpit-cliproxy.exe` 在跑。

> 客户端升级后 exe 常被覆盖，需重新执行 3.3。

---

## 4. 主验收：`[turn-state]`

### 4.1 怎么测

1. 客户端（DSH / Codex 等）走本地 Cockpit API  
2. 同一会话连续发 **至少 2～3 轮**（续写、工具调用、多步任务更易触发）  
3. 打开日志：

```text
%USERPROFILE%\.antigravity_cockpit\logs\codex-api.log.*
```

搜索：`[turn-state]`

### 4.2 成功时大致会看到

（措辞以你编译进二进制的中文日志为准，大意如下）

- 第一次从上游 **记住** turn-state（长度 292）  
- 后续请求 **注入** 已缓存的头  
- 或说明为何跳过（长度不对、无会话键等）

### 4.3 快速搜索

```powershell
$log = Get-ChildItem "$env:USERPROFILE\.antigravity_cockpit\logs\codex-api.log.*" |
  Sort-Object LastWriteTime -Descending | Select-Object -First 1
Select-String -Path $log.FullName -Pattern '\[turn-state\]' | Select-Object -Last 30
```

### 4.4 主验收清单

- [ ] 部署脚本成功，Cockpit 已重启  
- [ ] 同会话多轮请求已发出  
- [ ] 日志出现 `[turn-state]` 记住 / 注入类记录  
- [ ] 已知：粘性 ≠ 换模修复  

---

## 5. 附带能力：`[effort-detect]`

同一补丁二进制还会打出检测日志，便于判断「档位有没有带上、推理 token、是否像 516 截断」。

搜索：`[effort-detect]`

示例：

```text
[effort-detect] 请求档位=xhigh 模型=gpt-6-astra 账号=codex_xxxx
[effort-detect] 请求档位=xhigh 模型=gpt-6-astra 推理token=311 截断指纹=未命中 ...
```

说明：

- **请求档位**：客户端发出的 effort  
- **推理token**：上游 usage；SSE 需等流结束  
- **截断指纹**：是否像 `518*n-2` 截断  

这只是观测，**不会**改模型路由。

可选组件 `Install-Codex516Guard.ps1` 是独立 516 续写代理，默认不接管 `61227`，仅在你把客户端指过去时生效。

---

## 6. 常见问题

### Q1：和 CPA 插件是不是一回事？

**目标同类**（turn-state 粘性），实现落在 Cockpit 的 Go sidecar 上，不是往 CPA 里装插件。客户需要的是 Cockpit + 可编译 cliproxy 源码，而不是 CPA 安装包。

### Q2：只有 Turn-State、不想要 effort-detect？

当前 `transport.go` 两者绑在同一 RoundTrip。若只要粘性，需自行改源码去掉 `logEffortDetect` / `attachEffortDetect` 后再编译（进阶）。默认推荐两个都留着，日志量可接受。

### Q3：编译 / 找不到 turnstate 目录

检查 `COCKPIT_TURNSTATE_DIR` 是否指向含 `sidecars\cockpit-cliproxy\...` 的树，且该树本身原先就能 `go build`。

### Q4：替换 exe 失败

先退出 Cockpit Tools，再跑部署脚本。

### Q5：有 `[effort-detect]` 但没有 `[turn-state]`

可能本轮上游未返回合法 292 字节头，或会话键为空。换多轮工具任务再试，并把相关日志段（可打码）留下。

### Q6：粘性有了，模型还是 luna

粘性只保证「更可能同一路由/账号上下文」，不保证模型 ID。换模需换号、换上游或接受现状；可用 ModelTrace / `response.model` 另验。

---

## 7. 目录说明

```text
cockpit-effort-detect/
  README.md
  DEPLOY.zh-CN.md              # 本文（Turn-State 为主）
  Reinstall-Cockpit-TurnState.bat  # 路径模板，部署前请改
  pkg-turnstate/
    cache.go                   # Turn-State 缓存（主）
    transport.go               # 注入/捕获 + effort-detect 挂钩
    effort_detect.go           # 附带检测
    *_test.go
  apply_effort_detect.py
  build_and_replace.py
  Deploy-EffortDetect.ps1
  Install-Codex516Guard.ps1    # 可选
  RUN-ALL-ON-WINDOWS.ps1
```

问题反馈请带：系统、`go version`、两个环境变量路径、部署输出、以及含 `[turn-state]` / `requestId` 的日志片段（打码邮箱与 token）。
