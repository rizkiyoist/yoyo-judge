<#
.SYNOPSIS
  Run the yoyo-judge backend and frontend locally for testing, in two
  separate windows, without doing a full production build first.

.DESCRIPTION
  - Backend: `go run .` on -Port (default 5000).
  - Frontend: `npm run dev` (Vite dev server, http://localhost:5173), which
    proxies /yoyojudge/* to the backend per frontend/vite.config.ts.

  Port 5000 (not 8081, the production default baked into router.go) is used
  here because 8081 falls inside a Windows-reserved TCP port-exclusion
  range on at least one dev machine - check with:
    netsh interface ipv4 show excludedportrange protocol=tcp
  - and 5000 is what vite.config.ts's dev proxy already targets, so no
  extra config is needed for the common case. If 5000 is also blocked on
  your machine, pass a different -Port (numbers you've already confirmed
  free: 9000) AND edit the proxy target in frontend/vite.config.ts to match
  - the two must agree, this script doesn't patch that file for you.

.PARAMETER Port
  Backend port. Must match the target in frontend/vite.config.ts's dev
  proxy (currently hardcoded to http://localhost:5000).

.PARAMETER NoGoogleWarning
  Suppress the reminder that Google login won't work locally unless
  GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/GOOGLE_REDIRECT_URL are set - the
  "Demo / email login" section on the login page works regardless.

.EXAMPLE
  ./dev.ps1
  ./dev.ps1 -Port 5050
#>
param(
    [int]$Port = 5000,
    [switch]$NoGoogleWarning
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$staticDir = Join-Path $root 'bin\static'
$frontendDir = Join-Path $root 'frontend'

# static.go's //go:embed all:bin/static requires this directory to exist
# (with at least one file) at compile time, even for `go run` here where
# the embedded copy is never actually served - Vite serves the frontend in
# dev mode instead. A real production build (./build.ps1) overwrites this
# with the real frontend output; this placeholder only exists so a fresh
# clone doesn't fail to compile before that's ever been run.
if (-not (Test-Path $staticDir) -or -not (Get-ChildItem $staticDir -Recurse -File -ErrorAction SilentlyContinue)) {
    Write-Host "==> Creating placeholder ./bin/static (go:embed requires it; unused in dev mode)" -ForegroundColor Yellow
    New-Item -ItemType Directory -Force -Path $staticDir | Out-Null
    Set-Content -Path (Join-Path $staticDir 'index.html') -Value '<!-- placeholder for go:embed; run ./build.ps1 for a real build -->'
}

if (-not (Test-Path (Join-Path $frontendDir 'node_modules'))) {
    Write-Host "==> Installing frontend dependencies (first run only)" -ForegroundColor Cyan
    Push-Location $frontendDir
    try {
        npm install
        if ($LASTEXITCODE -ne 0) { throw "npm install failed" }
    }
    finally {
        Pop-Location
    }
}

Write-Host "==> Starting backend: go run . (PORT=$Port)" -ForegroundColor Cyan
# FRONTEND_URL must be set explicitly here: without it, server/oauth.go's
# frontendBaseURL() infers the post-login redirect target from the
# backend's own request host (localhost:$Port), not the Vite dev server -
# so a Google login would redirect back into the backend's embedded
# ./bin/static (server/oauth.go serves its own SPA fallback there) instead
# of the actual dev frontend at :5173. That embedded copy is whatever the
# last `./build.ps1` produced - typically a *production*-configured build (its
# own baked-in API base URL points at the real deployed backend, not this
# local one) - so a session token minted by this local backend would look
# invalid there, and login would appear to hang forever on "Finishing
# sign-in...". Pointing FRONTEND_URL at :5173 avoids all of that.
Start-Process powershell -ArgumentList @(
    '-NoExit', '-Command',
    "Set-Location '$root'; `$env:PORT='$Port'; `$env:FRONTEND_URL='http://localhost:5173'; Write-Host 'yoyo-judge backend (dev)' -ForegroundColor Cyan; go run ."
) -WindowStyle Normal

Write-Host "==> Starting frontend: npm run dev" -ForegroundColor Cyan
Start-Process powershell -ArgumentList @(
    '-NoExit', '-Command',
    "Set-Location '$frontendDir'; Write-Host 'yoyo-judge frontend (dev)' -ForegroundColor Cyan; npm run dev"
) -WindowStyle Normal

Write-Host ""
Write-Host "Backend health check: http://localhost:$Port/yoyojudge/healthz" -ForegroundColor Green
Write-Host "Frontend (open this): http://localhost:5173/" -ForegroundColor Green
Write-Host ""
if (-not $NoGoogleWarning) {
    Write-Host "Note: Google login won't work locally unless GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET /" -ForegroundColor DarkYellow
    Write-Host "GOOGLE_REDIRECT_URL are set as env vars before running this. Use the 'Demo / email login'" -ForegroundColor DarkYellow
    Write-Host "section on the login page instead." -ForegroundColor DarkYellow
    Write-Host ""
}
Write-Host "Two new PowerShell windows were opened for backend/frontend logs -" -ForegroundColor DarkGray
Write-Host "close them (or Ctrl+C inside each) to stop." -ForegroundColor DarkGray
