#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

echo ""
echo "  ========================================"
echo "   BitCode Engine - ERP Sample"
echo "  ========================================"
echo ""

# Check Go
if ! command -v go &> /dev/null; then
    echo "[ERROR] Go is not installed or not in PATH."
    echo "        Download from https://go.dev/dl/"
    exit 1
fi

# Check engine directory
if [ ! -f "../../engine/go.mod" ]; then
    echo "[ERROR] Engine not found at ../../engine"
    echo "        Make sure you're running from samples/erp/"
    exit 1
fi

# Install bitcode CLI
echo "[1/2] Installing bitcode CLI..."
go install -C ../../engine ./cmd/bitcode/
echo "      Done."

echo ""
echo "  ----------------------------------------"
echo "   Endpoints:"
echo "     App:        http://localhost:8989/app"
echo "     Health:     http://localhost:8989/health"
echo "     Admin:      http://localhost:8989/admin"
echo "     WebSocket:  ws://localhost:8989/ws"
echo "     Auth:       POST /auth/register, /auth/login"
echo "     CRM API:    /api/contacts, /api/leads"
echo "     HRM API:    /api/employees, /api/departments"
echo ""
echo "   Press Ctrl+C to stop."
echo "  ----------------------------------------"
echo ""

# Start with bitcode dev (auto-detects engine repo, uses Air if available)
echo "[2/2] Starting bitcode dev..."
echo ""
bitcode dev
