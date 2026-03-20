@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\user\api"

:: 新开 cmd 窗口
start "user-api" cmd /k "cd /d %TARGET_DIR% && air"