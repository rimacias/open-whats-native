@echo off
REM Open-Whats Production Build Script for Windows
REM This script uses fyne-cross (which relies on Docker) to cross-compile the app
REM for Windows (.exe), macOS (.app), and Linux from a single Windows machine.

echo Building production packages for Open-Whats...

REM Check for fyne
where fyne >nul 2>nul
if %errorlevel% neq 0 (
    echo Installing Fyne CLI...
    go install fyne.io/fyne/v2/cmd/fyne@latest
)

REM Check for fyne-cross
where fyne-cross >nul 2>nul
if %errorlevel% neq 0 (
    echo Installing fyne-cross...
    go install github.com/fyne-io/fyne-cross@latest
)

REM Check for Docker
docker info >nul 2>nul
if %errorlevel% neq 0 (
    echo Error: Docker does not seem to be running. Please start Docker Desktop to cross-compile.
    exit /b 1
)

set APP_ID=com.openwhats.native
set TARGET_DIR=./cmd/open-whats/

echo -----------------------------------
echo Creating Windows executable...
fyne-cross windows -arch=amd64 -app-id=%APP_ID% -dir=%TARGET_DIR%
echo Windows build complete!

echo -----------------------------------
echo Creating macOS application...
fyne-cross darwin -arch=amd64,arm64 -app-id=%APP_ID% -dir=%TARGET_DIR%
echo macOS build complete!

echo -----------------------------------
echo Creating Linux tarball...
fyne-cross linux -arch=amd64 -app-id=%APP_ID% -dir=%TARGET_DIR%
echo Linux build complete!

echo -----------------------------------
echo All builds completed successfully!
echo Check the 'fyne-cross\bin\' directory for your binaries.
