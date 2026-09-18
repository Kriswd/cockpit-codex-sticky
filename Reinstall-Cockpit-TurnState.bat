@echo off
chcp 65001 >nul
setlocal EnableExtensions
title 重装 Cockpit Turn-State 版 cliproxy

rem ===== 客户请改这三处 =====
set "SRC_ROOT=%COCKPIT_TURNSTATE_DIR%"
if "%SRC_ROOT%"=="" set "SRC_ROOT=D:\path\to\cockpit-tools-turnstate"
set "INSTALL_DIR=%COCKPIT_TOOLS_DIR%"
if "%INSTALL_DIR%"=="" set "INSTALL_DIR=D:\path\to\Cockpit Tools"
rem =========================

set "SIDECAR=%SRC_ROOT%\sidecars\cockpit-cliproxy"
set "DIST=%SRC_ROOT%\dist\cockpit-cliproxy.exe"
set "INSTALL_EXE=%INSTALL_DIR%\cockpit-cliproxy.exe"
set "APP_EXE=%INSTALL_DIR%\cockpit-tools.exe"
set "PATCH_ROOT=%~dp0"

echo.
echo === Cockpit Turn-State 一键重装 ===
echo 源码: %SRC_ROOT%
echo 安装: %INSTALL_EXE%
echo 补丁: %PATCH_ROOT%
echo.

if not exist "%SIDECAR%\go.mod" if not exist "%SIDECAR%\main.go" (
  echo [失败] 找不到 sidecar 源码：%SIDECAR%
  echo 请设置环境变量 COCKPIT_TURNSTATE_DIR，或编辑本 bat 里的 SRC_ROOT。
  pause
  exit /b 1
)
if not exist "%INSTALL_DIR%" (
  echo [失败] 找不到 Cockpit 安装目录：%INSTALL_DIR%
  pause
  exit /b 1
)

where go >nul 2>nul
if errorlevel 1 (
  echo [失败] 未找到 go，请先安装 Go 并加入 PATH。
  pause
  exit /b 1
)

where python >nul 2>nul
if not errorlevel 1 (
  echo [0/4] 套用 pkg-turnstate 补丁 ...
  set "COCKPIT_TURNSTATE_DIR=%SRC_ROOT%"
  set "COCKPIT_TOOLS_DIR=%INSTALL_DIR%"
  python "%PATCH_ROOT%apply_effort_detect.py"
  if errorlevel 1 (
    echo [失败] 套用补丁失败。
    pause
    exit /b 1
  )
)

echo [1/4] 停止 Cockpit / cliproxy ...
taskkill /F /IM cockpit-cliproxy.exe >nul 2>nul
taskkill /F /IM cockpit-tools.exe >nul 2>nul
timeout /t 2 /nobreak >nul

echo [2/4] 编译 turn-state 版 sidecar ...
if not exist "%SRC_ROOT%\dist" mkdir "%SRC_ROOT%\dist"
pushd "%SIDECAR%"
go build -trimpath -o "%DIST%" .
if errorlevel 1 (
  echo [失败] 编译失败。
  popd
  pause
  exit /b 1
)
popd

echo [3/4] 备份并替换二进制 ...
for /f %%i in ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd_HHmmss"') do set "TS=%%i"
if exist "%INSTALL_EXE%" copy /Y "%INSTALL_EXE%" "%INSTALL_DIR%\cockpit-cliproxy.exe.bak_before_turnstate_%TS%" >nul
copy /Y "%DIST%" "%INSTALL_EXE%" >nul
if errorlevel 1 (
  echo [失败] 替换失败，文件可能仍被占用。请手动关掉 Cockpit 再试。
  pause
  exit /b 1
)

echo [4/4] 尝试启动 Cockpit Tools ...
if exist "%APP_EXE%" (
  start "" "%APP_EXE%"
) else (
  echo [提示] 未找到 cockpit-tools.exe，请手动打开 Cockpit。
)

echo.
echo [完成] 已安装 Turn-State 版。
echo 主验证：Codex API 日志里搜  [turn-state]
echo 附带验证（可选）：搜  [effort-detect]
echo.
pause
