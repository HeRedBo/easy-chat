batch
@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\im\api"

start "im-rpc" cmd /c "cd /d %TARGET_DIR% && air"
