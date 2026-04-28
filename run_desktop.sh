#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "[BitCode] Building web components..."
cd "$SCRIPT_DIR/packages/components"
npm install --silent

echo "[BitCode] Starting desktop app..."
cd "$SCRIPT_DIR/packages/tauri"
npm run dev:desktop
