$Host.UI.RawUI.WindowTitle = "BitCode Engine - ERP Sample"

Write-Host ""
Write-Host "  ========================================"
Write-Host "   BitCode Engine - ERP Sample"
Write-Host "  ========================================"
Write-Host ""

Set-Location $PSScriptRoot

# Check Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] Go is not installed or not in PATH."
    Write-Host "        Download from https://go.dev/dl/"
    Read-Host "Press Enter to exit"
    exit 1
}

# Check engine directory
if (-not (Test-Path "../../engine/go.mod")) {
    Write-Host "[ERROR] Engine not found at ../../engine"
    Write-Host "        Make sure you're running from samples/erp/"
    Read-Host "Press Enter to exit"
    exit 1
}

# Tidy dependencies
Write-Host "[1/2] Installing dependencies..."
Push-Location ../../engine
go mod tidy 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Failed to install dependencies."
    Pop-Location
    Read-Host "Press Enter to exit"
    exit 1
}
Write-Host "      Done."
Pop-Location

Write-Host ""
Write-Host "  ----------------------------------------"
Write-Host "   Endpoints:"
Write-Host "     App:        http://localhost:8989/app"
Write-Host "     Health:     http://localhost:8989/health"
Write-Host "     Admin:      http://localhost:8989/admin"
Write-Host "     WebSocket:  ws://localhost:8989/ws"
Write-Host "     Auth:       POST /auth/register, /auth/login"
Write-Host "     CRM API:    /api/contacts, /api/leads"
Write-Host "     HRM API:    /api/employees, /api/departments"
Write-Host ""
Write-Host "   Press Ctrl+C to stop."
Write-Host "  ----------------------------------------"
Write-Host ""

# Try air (hot-reload), fallback to manual build+run
if (Get-Command air -ErrorAction SilentlyContinue) {
    Write-Host "[2/2] Starting with Air (hot-reload enabled)"
    Write-Host "      Watching: *.go, *.json, *.html, *.yaml"
    Write-Host ""
    air
} else {
    Write-Host "[2/2] Building and starting (no hot-reload)"
    Write-Host "      Install Air for hot-reload: go install github.com/air-verse/air@latest"
    Write-Host ""

    if (-not (Test-Path "tmp")) { New-Item -ItemType Directory -Path "tmp" | Out-Null }

    go build -C ../../engine -o ../samples/erp/tmp/engine.exe cmd/engine/main.go
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] Build failed."
        Read-Host "Press Enter to exit"
        exit 1
    }

    & ./tmp/engine.exe
}
