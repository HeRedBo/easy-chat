batch
@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\social\rpc"

start "social-rpc" cmd /c "cd /d %TARGET_DIR% && air"
