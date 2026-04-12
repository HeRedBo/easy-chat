@echo off
chcp 65001
setlocal enabledelayedexpansion
title 项目启动

set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%~dp0..\..\\"
cd /d "%PROJECT_ROOT%"

set "PID_FILE=%SCRIPT_DIR%run.pid"
del "%PID_FILE%" 2>nul

:: 启动普通模块 rpc + api
:startModule
set "m=%~1"
echo 启动 %m% ...

start "%m%-rpc" cmd /k "echo %%pid%%>>"%PID_FILE%" && cd /d "%cd%\apps\%m%\rpc" && air"
start "%m%-api" cmd /k "echo %%pid%%>>"%PID_FILE%" && cd /d "%cd%\apps\%m%\api" && air"

goto :eof

:: 启动 task mq
:startTask
echo 启动 task ...
start "task-mq" cmd /k "echo %%pid%%>>"%PID_FILE%" && cd /d "%cd%\apps\task\mq" && air"
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
echo ✅ 启动完成（已记录PID）
pause >nul
exit /b