$ErrorActionPreference = "Stop"

$root = $PSScriptRoot

if (-not (Get-Command node -ErrorAction SilentlyContinue)) { Write-Error "[BitCode] ERROR: Node.js not found. Install from https://nodejs.org/"; exit 1 }
if (-not (Get-Command npm -ErrorAction SilentlyContinue)) { Write-Error "[BitCode] ERROR: npm not found. Install Node.js from https://nodejs.org/"; exit 1 }
if (-not (Get-Command cargo -ErrorAction SilentlyContinue)) { Write-Error "[BitCode] ERROR: Rust/Cargo not found. Install from https://rustup.rs/"; exit 1 }

Write-Host "[BitCode] Building web components..."
Set-Location "$root\packages\components"
npm install --silent
if ($LASTEXITCODE -ne 0) { throw "npm install failed for components" }

Write-Host "[BitCode] Starting desktop app..."
Set-Location "$root\packages\tauri"
npm run dev:desktop
