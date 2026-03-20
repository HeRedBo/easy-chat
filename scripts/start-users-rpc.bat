batch
@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\user\rpc"

start "users-rpc" cmd /k "cd /d %TARGET_DIR% && air"