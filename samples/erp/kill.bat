@echo off
setlocal enabledelayedexpansion
title Kill BitCode Engine
echo.
echo  ========================================
echo   Kill BitCode Engine
echo  ========================================
echo.

cd /d "%~dp0"

:: Read port from bitcode.yaml
set PORT=8080
for /f "tokens=2 delims=: " %%a in ('findstr /r "^port:" bitcode.yaml 2^>nul') do set PORT=%%a

echo [INFO] Looking for process on port %PORT%...

:: Find and kill process using the port
set FOUND=0
for /f "tokens=5" %%p in ('netstat -aon ^| findstr ":%PORT% " ^| findstr "LISTENING"') do (
    if %%p neq 0 (
        echo [KILL] Stopping PID %%p on port %PORT%...
        taskkill /F /PID %%p >nul 2>&1
        if !errorlevel! equ 0 (
            echo [OK]   Process %%p killed.
        ) else (
            echo [WARN] Could not kill PID %%p ^(may need admin rights^).
        )
        set FOUND=1
    )
)

if !FOUND! equ 0 (
    echo [INFO] No process found on port %PORT%.
) else (
    echo.
    echo [DONE] Port %PORT% is now free.
)

echo.
pause
