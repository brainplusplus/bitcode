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

:: Tidy dependencies
echo [1/2] Installing dependencies...
cd ..\..\engine
go mod tidy >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Failed to install dependencies.
    pause
    exit /b 1
)
echo       Done.
cd /d "%~dp0"

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

:: Try air (hot-reload), fallback to manual build+run
where air >nul 2>&1
if %errorlevel% equ 0 (
    echo [2/2] Starting with Air (hot-reload enabled)
    echo       Watching: *.go, *.json, *.html, *.yaml
    echo.
    air
) else (
    echo [2/2] Building and starting (no hot-reload)
    echo       Install Air for hot-reload: go install github.com/air-verse/air@latest
    echo.
    cd ..\..\engine
    set CGO_ENABLED=0
    if exist bin\engine.exe del bin\engine.exe
    go build -o bin\engine.exe cmd\engine\main.go
    if %errorlevel% neq 0 (
        echo [ERROR] Build failed.
        pause
        exit /b 1
    )
    cd /d "%~dp0"
    ..\..\engine\bin\engine.exe
)
