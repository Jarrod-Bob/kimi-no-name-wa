@echo off
rem Build the frontend, then embed it in the Go binary. Windows, cmd.exe.
rem Runs regardless of PowerShell execution policy -- this is a batch
rem script, not a PowerShell script.
setlocal

pushd web
rem npm ci only on a fresh checkout; later builds reuse node_modules.
if not exist node_modules (
    call npm ci
    if errorlevel 1 (
        popd
        echo npm ci failed
        exit /b 1
    )
)
call npm run build
if errorlevel 1 (
    popd
    echo frontend build failed
    exit /b 1
)
popd

if not exist internal\web\dist mkdir internal\web\dist
if not exist internal\web\dist\.gitkeep type nul > internal\web\dist\.gitkeep

set CGO_ENABLED=0
go build -o kimi-no-name-wa.exe ./cmd/kimi-no-name-wa
if errorlevel 1 (
    echo go build failed
    exit /b 1
)

echo built kimi-no-name-wa.exe
