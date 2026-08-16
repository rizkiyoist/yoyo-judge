<#
.SYNOPSIS
  Build the yoyo-judge backend (Go) and frontend (Vue/Vite).

.PARAMETER Only
  Build only "backend" or only "frontend". Omit to build both.

.EXAMPLE
  ./build.ps1
  ./build.ps1 -Only backend
  ./build.ps1 -Only frontend
#>
param(
    [ValidateSet('backend', 'frontend')]
    [string]$Only
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$binDir = Join-Path $root 'bin'

function Build-Backend {
    Write-Host "==> Building backend (Go)" -ForegroundColor Cyan
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    Push-Location $root
    try {
        go build -o (Join-Path $binDir 'yoyo-judge.exe') .
    }
    finally {
        Pop-Location
    }
    Write-Host "Backend built: $binDir\yoyo-judge.exe" -ForegroundColor Green
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
        npm run build
        if ($LASTEXITCODE -ne 0) { throw "npm run build failed" }
    }
    finally {
        Pop-Location
    }

    # Copy the built dist/ next to the backend binary so bin/ is one
    # self-contained, relocatable folder — copy it anywhere and it still
    # works (dist/ uses relative asset paths, see vite.config.ts base: './').
    $staticDir = Join-Path $binDir 'static'
    if (Test-Path $staticDir) {
        Remove-Item -Recurse -Force $staticDir
    }
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    Copy-Item -Recurse (Join-Path $frontendDir 'dist') $staticDir

    Write-Host "Frontend built: $staticDir" -ForegroundColor Green
}

switch ($Only) {
    'backend' { Build-Backend }
    'frontend' { Build-Frontend }
    default {
        Build-Backend
        Build-Frontend
    }
}
