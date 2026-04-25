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

# Install bitcode CLI
Write-Host "[1/2] Installing bitcode CLI..."
go install -C ../../engine ./cmd/bitcode/
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Failed to install bitcode CLI."
    Read-Host "Press Enter to exit"
    exit 1
}
Write-Host "      Done."

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

# Start with bitcode dev (auto-detects engine repo, uses Air if available)
Write-Host "[2/2] Starting bitcode dev..."
Write-Host ""
bitcode dev
