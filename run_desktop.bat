@echo off
setlocal

where node >nul 2>&1 || (echo [BitCode] ERROR: Node.js not found. Install from https://nodejs.org/ & exit /b 1)
where npm >nul 2>&1 || (echo [BitCode] ERROR: npm not found. Install Node.js from https://nodejs.org/ & exit /b 1)
where cargo >nul 2>&1 || (echo [BitCode] ERROR: Rust/Cargo not found. Install from https://rustup.rs/ & exit /b 1)

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
