@echo off
title BitCode Engine - ERP Sample
echo.
echo  ========================================
echo   BitCode Engine - ERP Sample
echo  ========================================
echo.

cd /d "%~dp0"

:: Check if Go is installed
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed or not in PATH.
    echo         Download from https://go.dev/dl/
    pause
    exit /b 1
)

:: Check engine directory
if not exist "..\..\engine\go.mod" (
    echo [ERROR] Engine not found at ..\..\engine
    echo         Make sure you're running from samples\erp\
    pause
    exit /b 1
)

:: Install bitcode CLI
echo [1/2] Installing bitcode CLI...
go install -C ..\..\engine ./cmd/bitcode/
if %errorlevel% neq 0 (
    echo [ERROR] Failed to install bitcode CLI.
    pause
    exit /b 1
)
echo       Done.

echo.
echo  ----------------------------------------
echo   Endpoints:
echo     App:        http://localhost:8989/app
echo     Health:     http://localhost:8989/health
echo     Admin:      http://localhost:8989/admin
echo     WebSocket:  ws://localhost:8989/ws
echo     Auth:       POST /auth/register, /auth/login
echo     CRM API:    /api/contacts, /api/leads
echo     HRM API:    /api/employees, /api/departments
echo.
echo   Press Ctrl+C to stop.
echo  ----------------------------------------
echo.

:: Start with bitcode dev (auto-detects engine repo, uses Air if available)
echo [2/2] Starting bitcode dev...
echo.
bitcode dev
