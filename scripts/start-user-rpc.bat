batch
@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\user\rpc"

:: 关键：用 cmd /c 而不是 /k，关闭窗口就自动终止 air 进程
start "user-rpc" cmd /c "cd /d %TARGET_DIR% && air"