<#
.SYNOPSIS
  Build the yoyo-judge backend (Go) and frontend (Vue/Vite).

  The frontend is embedded into the backend binary (see static.go's
  //go:embed all:bin/static), so Build-Frontend must run before
  Build-Backend - the backend build reads whatever's currently in
  ./bin/static. That same folder doubles as the plain copy for anyone
  hosting the frontend separately (the cross-origin split deploy shape) -
  one build output, not two. The result is one binary per OS (Windows +
  Linux) that serves the API and the frontend on the same port; nothing
  else needs to be deployed or configured to serve static files
  separately for the embedded shape.

.PARAMETER Only
  Build only "backend" or only "frontend". Omit to build both. Note:
  "-Only backend" embeds whatever is currently in ./bin/static (from a
  previous full build), not a fresh frontend build.

.PARAMETER BasePath
  URL prefix the frontend build is served under, e.g. "/yoyojudge" (the
  default, matching router.go's basePath() default). Only affects the
  frontend's production build (asset paths + default relative API prefix);
  the backend reads its own prefix from the BASE_PATH environment variable
  at runtime, independent of this build step.

.PARAMETER ApiBaseUrl
  Absolute URL to call the API at. Defaults to the real production
  deployment target: the frontend is hosted separately (nginx docroot on
  the rizkiyoist.duckdns.org domain, TLS terminated there) from the backend
  (standalone process on :8081, with its own cert - see router.go's
  firstExisting() cert.pem/key.pem convention - since it's called directly,
  cross-origin, and must therefore also be HTTPS or the browser blocks it
  as mixed content). The API URL uses the *domain*, not the server's raw
  IP, even though the backend also happens to be reachable by IP - the
  installed certificate is issued for the domain name only, so connecting
  via the IP fails real TLS hostname validation in a browser (this was a
  real bug: it looked like a CORS error, but curl without -k reproduced
  the actual cause). Override only for a different deployment target.

.EXAMPLE
  ./build.ps1
  ./build.ps1 -Only backend
  ./build.ps1 -Only frontend
  ./build.ps1 -ApiBaseUrl https://some-other-host:8081/yoyojudge/api
#>
param(
    [ValidateSet('backend', 'frontend')]
    [string]$Only,
    [string]$BasePath,
    [string]$ApiBaseUrl = 'https://rizkiyoist.duckdns.org:8081/yoyojudge/api'
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$binDir = Join-Path $root 'bin'
$staticDir = Join-Path $binDir 'static'

function Build-Backend {
    Write-Host "==> Building backend (Go)" -ForegroundColor Cyan
    if (-not (Test-Path $staticDir)) {
        throw "$staticDir does not exist - run Build-Frontend (or ./build.ps1 without -Only backend) first, since static.go embeds it into the binary."
    }
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    Push-Location $root
    try {
        go build -o (Join-Path $binDir 'yoyo-judge.exe') .
        if ($LASTEXITCODE -ne 0) { throw "go build (windows) failed" }
        Write-Host "Backend built: $binDir\yoyo-judge.exe (windows/amd64)" -ForegroundColor Green

        # Cross-compile a Linux binary too, so bin/ is deployable to either
        # OS. CGO_ENABLED=0 since nothing in this module needs cgo, which
        # keeps this a static binary with no cross-compiler toolchain needed.
        $prevGOOS = $env:GOOS
        $prevGOARCH = $env:GOARCH
        $prevCGO = $env:CGO_ENABLED
        try {
            $env:GOOS = 'linux'
            $env:GOARCH = 'amd64'
            $env:CGO_ENABLED = '0'
            go build -o (Join-Path $binDir 'yoyo-judge-linux-amd64') .
            if ($LASTEXITCODE -ne 0) { throw "go build (linux) failed" }
        }
        finally {
            $env:GOOS = $prevGOOS
            $env:GOARCH = $prevGOARCH
            $env:CGO_ENABLED = $prevCGO
        }
        Write-Host "Backend built: $binDir\yoyo-judge-linux-amd64 (linux/amd64)" -ForegroundColor Green
    }
    finally {
        Pop-Location
    }
}

function Build-Frontend {
    Write-Host "==> Building frontend (Vue/Vite)" -ForegroundColor Cyan
    $frontendDir = Join-Path $root 'frontend'
    Push-Location $frontendDir
    try {
        if (-not (Test-Path (Join-Path $frontendDir 'node_modules'))) {
            npm install
            if ($LASTEXITCODE -ne 0) { throw "npm install failed" }
        }

        $prevBasePath = $env:VITE_BASE_PATH
        $prevApiBaseUrl = $env:VITE_API_BASE_URL
        try {
            if ($BasePath) { $env:VITE_BASE_PATH = $BasePath }
            if ($ApiBaseUrl) { $env:VITE_API_BASE_URL = $ApiBaseUrl }
            npm run build
            if ($LASTEXITCODE -ne 0) { throw "npm run build failed" }
        }
        finally {
            $env:VITE_BASE_PATH = $prevBasePath
            $env:VITE_API_BASE_URL = $prevApiBaseUrl
        }
    }
    finally {
        Pop-Location
    }

    # Copy into ./bin/static - this is what static.go's //go:embed
    # all:bin/static pulls into the backend binary at compile time, and
    # also what anyone hosting the frontend separately (the cross-origin
    # split deploy shape) copies to their web server's docroot. One build
    # output serves both purposes.
    if (Test-Path $staticDir) {
        Remove-Item -Recurse -Force $staticDir
    }
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    Copy-Item -Recurse (Join-Path $frontendDir 'dist') $staticDir
    Write-Host "Frontend copied: $staticDir" -ForegroundColor Green
}

switch ($Only) {
    'backend' { Build-Backend }
    'frontend' { Build-Frontend }
    default {
        # Frontend must build first: static.go embeds ./bin/static into
        # the backend binary at compile time.
        Build-Frontend
        Build-Backend
    }
}
