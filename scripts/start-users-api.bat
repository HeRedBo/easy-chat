@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\user\api"

:: 新开 cmd 窗口
start "users-api" cmd /k "cd /d %TARGET_DIR% && air"