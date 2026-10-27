@echo off
REM One-click startup for Windows (Docker Desktop + Docker Compose).
setlocal EnableExtensions
cd /d "%~dp0"

set "BUILD_ARG="
if "%~1"=="" goto :arguments_ready
if /I "%~1"=="--build" (
  set "BUILD_ARG=--build"
  goto :arguments_ready
)
if /I "%~1"=="--rebuild" (
  set "BUILD_ARG=--build"
  goto :arguments_ready
)
if /I "%~1"=="--help" goto :usage
if /I "%~1"=="-h" goto :usage
echo Unknown argument: %~1
goto :usage_error

:arguments_ready

where docker >nul 2>&1
if errorlevel 1 (
  echo Docker was not found. Install and start Docker Desktop, then try again.
  exit /b 1
)

docker info >nul 2>&1
if errorlevel 1 (
  echo Docker is not running. Start Docker Desktop, then try again.
  exit /b 1
)

docker compose version >nul 2>&1
if not errorlevel 1 (
  set "COMPOSE=docker compose"
) else (
  where docker-compose >nul 2>&1
  if errorlevel 1 (
    echo Docker Compose was not found. Update Docker Desktop, then try again.
    exit /b 1
  )
  set "COMPOSE=docker-compose"
)

if not exist ".env" (
  echo Notice: .env was not found. Default configuration will be used.
  echo         To enable cloud models, copy .env.example to .env and configure it.
)

if defined BUILD_ARG (
  echo Rebuilding and starting FinTech Knowledge Base...
) else (
  echo Starting Finance Knowledge Base...
)
call %COMPOSE% up -d %BUILD_ARG%
if errorlevel 1 exit /b 1

echo Waiting for the application to become healthy...
for /L %%I in (1,1,60) do (
  powershell -NoProfile -Command "try { $r = Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 http://127.0.0.1:8201/health; if ($r.StatusCode -eq 200) { exit 0 } } catch {}; exit 1"
  if not errorlevel 1 goto :ready
  timeout /t 2 /nobreak >nul
)

echo.
echo The application did not become healthy within 120 seconds.
call %COMPOSE% ps
echo.
echo View logs with: %COMPOSE% logs --tail=100 app
exit /b 1

:ready
echo.
echo Started successfully.
echo Knowledge Base: http://127.0.0.1:8201
echo Health Check:   http://127.0.0.1:8201/health
echo Elasticsearch:  http://127.0.0.1:9202
echo.
call %COMPOSE% ps
exit /b 0

:usage
echo Usage: start.bat [--build]
echo   No argument: start existing images without forcing a rebuild.
echo   --build:     rebuild images after code or Dockerfile changes.
exit /b 0

:usage_error
echo Usage: start.bat [--build]
exit /b 2
