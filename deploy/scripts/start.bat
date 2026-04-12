@echo off
chcp 65001
setlocal enabledelayedexpansion
title 项目启动

set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%~dp0..\..\\"
cd /d "%PROJECT_ROOT%"

:: 启动普通模块 rpc + api
:startModule
set "m=%~1"
echo 启动 %m% ...

start "%m%-rpc" cmd /k "cd /d "%cd%\apps\%m%\rpc" && air"
start "%m%-api" cmd /k "cd /d "%cd%\apps\%m%\api" && air"

goto :eof

:: 启动 task mq
:startTask
echo 启动 task ...
start "task-mq" cmd /k "cd /d "%cd%\apps\task\mq" && air"
goto :eof

:: 主逻辑
if "%~1"=="" (
    call :startModule user
    call :startModule im
    call :startModule social
    call :startTask
) else (
    :loop
    if "%~1"=="" goto end
    if /i "%~1"=="task" (call :startTask) else (call :startModule %~1)
    shift
    goto loop
)

:end
echo.
echo ✅ 启动完成
pause >nul
exit /b