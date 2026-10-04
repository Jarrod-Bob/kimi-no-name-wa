# Build the frontend, then embed it in the Go binary.
Push-Location web
# npm ci only on a fresh checkout; later builds reuse node_modules.
if (-not (Test-Path node_modules)) {
    npm ci
    if (-not $?) { Pop-Location; throw "npm ci failed" }
}
npm run build
if (-not $?) { Pop-Location; throw "frontend build failed" }
Pop-Location
if (-not (Test-Path internal\web\dist\.gitkeep)) { New-Item -ItemType File internal\web\dist\.gitkeep | Out-Null }
$env:CGO_ENABLED = "0"
go build -o kimi-no-name-wa.exe ./cmd/kimi-no-name-wa
if (-not $?) { throw "go build failed" }
Write-Host "built kimi-no-name-wa.exe"
