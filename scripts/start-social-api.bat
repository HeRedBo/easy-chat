batch
@echo off
set "ROOT=%~dp0.."
set "TARGET_DIR=%ROOT%\apps\social\api"

start "social-api" cmd /c "cd /d %TARGET_DIR% && air"