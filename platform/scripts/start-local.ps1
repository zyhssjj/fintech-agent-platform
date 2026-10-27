$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

function Require-Command([string]$name) {
  if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
    throw "Required command not found: $name"
  }
}

function Ensure-EnvFile([string]$directory) {
  $target = Join-Path $directory ".env"
  if (-not (Test-Path $target)) {
    Copy-Item (Join-Path $directory ".env.example") $target
    Write-Host "Created $target. Review it before enabling external email or AI." -ForegroundColor Yellow
  }
}

function Start-ProjectProcess([string]$name, [string]$directory, [string]$command) {
  Write-Host "Starting $name..." -ForegroundColor Cyan
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$directory'; $command" | Out-Null
}

Require-Command docker
Require-Command go
Require-Command npm
Require-Command python

Ensure-EnvFile (Join-Path $root "backend")
Ensure-EnvFile (Join-Path $root "frontend")
Ensure-EnvFile (Join-Path $root "ai")

Push-Location $root
try {
  docker compose -f deploy/docker-compose.yml up -d

  $frontend = Join-Path $root "frontend"
  if (-not (Test-Path (Join-Path $frontend "node_modules"))) {
    Push-Location $frontend
    try { npm ci } finally { Pop-Location }
  }

  $ai = Join-Path $root "ai"
  $python = Join-Path $ai ".venv\Scripts\python.exe"
  if (-not (Test-Path $python)) {
    python -m venv (Join-Path $ai ".venv")
    & $python -m pip install -r (Join-Path $ai "requirements.txt")
  }

  Start-ProjectProcess "backend" (Join-Path $root "backend") "go run ./cmd/server"
  Start-ProjectProcess "AI service" $ai ".\.venv\Scripts\python.exe -m app.main"
  Start-ProjectProcess "frontend" $frontend "npm run dev"
} finally {
  Pop-Location
}

Write-Host "Services are starting. Open http://localhost:3000 after the three windows report ready." -ForegroundColor Green
