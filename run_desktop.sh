#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

command -v node >/dev/null 2>&1 || { echo "[BitCode] ERROR: Node.js not found. Install from https://nodejs.org/"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "[BitCode] ERROR: npm not found. Install Node.js from https://nodejs.org/"; exit 1; }
command -v cargo >/dev/null 2>&1 || { echo "[BitCode] ERROR: Rust/Cargo not found. Install from https://rustup.rs/"; exit 1; }

echo "[BitCode] Building web components..."
cd "$SCRIPT_DIR/packages/components"
npm install --silent

echo "[BitCode] Starting desktop app..."
cd "$SCRIPT_DIR/packages/tauri"
npm run dev:desktop
