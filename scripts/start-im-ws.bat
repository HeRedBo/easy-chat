@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\im\ws"

:: 新开 cmd 窗口
start "im-ws" cmd /k "cd /d %TARGET_DIR% && air"