$ErrorActionPreference = "Stop"

$root = $PSScriptRoot

Write-Host "[BitCode] Building web components..."
Set-Location "$root\packages\components"
npm install --silent
if ($LASTEXITCODE -ne 0) { throw "npm install failed for components" }

Write-Host "[BitCode] Starting desktop app..."
Set-Location "$root\packages\tauri"
npm run dev:desktop
