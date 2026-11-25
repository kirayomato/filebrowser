@echo off
setlocal

set "ROOT=%~dp0"
set "OUT=%ROOT%dist"
set "GOOS=android"
set "GOARCH=arm64"
set "CGO_ENABLED=0"
set "GOPROXY=https://goproxy.cn,direct"

echo [1/2] Building frontend...
cd /d "%ROOT%frontend"
if errorlevel 1 goto :fail
call pnpm install --frozen-lockfile
if errorlevel 1 goto :fail
call pnpm run build
if errorlevel 1 goto :fail

echo [2/2] Building android/arm64 binary...
cd /d "%ROOT%"
if not exist "%OUT%" mkdir "%OUT%"
if errorlevel 1 goto :fail
go build -ldflags="-s -w" -o "%OUT%\filebrowser-android-arm64" .
if errorlevel 1 goto :fail

echo Done: %OUT%\filebrowser-android-arm64
exit /b 0

:fail
echo Build failed.
exit /b 1
