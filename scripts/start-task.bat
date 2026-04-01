@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\task\mq"

:: 新开 cmd 窗口
start "im-ws" cmd /c "cd /d %TARGET_DIR% && air"