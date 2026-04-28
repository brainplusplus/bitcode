@echo off
setlocal

echo [BitCode] Building web components...
cd /d "%~dp0packages\components"
call npm install --silent
if errorlevel 1 (
    echo [BitCode] ERROR: npm install failed for components
    exit /b 1
)

echo [BitCode] Starting desktop app...
cd /d "%~dp0packages\tauri"
call npm run dev:desktop
