batch
@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\social\rpc"

start "social-rpc" cmd /k "cd /d %TARGET_DIR% && air"
