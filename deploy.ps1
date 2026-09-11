<#
.SYNOPSIS
  Build and deploy yoyo-judge to the production server in one command.

  Authentication is by SSH key (~/.ssh/id_ed25519, already authorized on
  the server), so nothing here prompts for a password. Every ssh/scp call
  passes -o BatchMode=yes deliberately: if the key ever stops working the
  script fails immediately with a clear error instead of silently blocking
  on a password prompt you can't see.

  What it does, in order:
    1. .\build.ps1            (skip with -SkipBuild)
    2. stops the backend      - `sudo systemctl stop yoyojudge`; copying
                                over a running binary fails with ETXTBSY,
                                "text file busy"
    3. copies the binary      -> $AppDir
    4. replaces the docroot   -> $Docroot, then chmod o+rX so nginx can
                                read it back (otherwise the site 403s)

  The backend is left stopped at the end - restarting is a manual step, by
  design. Pass -Restart to have the script bring it back up instead.

  Stopping, unlike restarting, is not optional: the copy in step 3 fails
  while the old process still holds the binary open. It must also go
  through systemctl rather than kill/pkill - the unit is Restart=always,
  so a killed process comes straight back and takes the binary with it.

  Never touched: yoyojudge.db, env.json, cert.pem/key.pem. They live in
  $AppDir alongside the binary, and only the binary itself is overwritten.

.PARAMETER SkipBuild
  Deploy whatever is already in .\bin - don't rebuild first.

.PARAMETER Restart
  Also start the backend again once the copy is done. Off by default: the
  frontend is live as soon as the copy finishes, but the API stays down
  until you run `sudo systemctl start yoyojudge` yourself.

.PARAMETER DryRun
  Print every command that would run, locally and remotely, and connect to
  nothing.

.EXAMPLE
  .\deploy.ps1
  .\deploy.ps1 -SkipBuild
  .\deploy.ps1 -Restart
  .\deploy.ps1 -DryRun
#>
param(
    [switch]$SkipBuild,
    [switch]$Restart,
    [switch]$DryRun,
    [string]$Server = 'rizki@103.134.154.210',
    [string]$AppDir = '/home/rizki/yoyojudge',
    [string]$Docroot = '/var/www/html/yoyojudge',
    [string]$Service = 'yoyojudge',
    [string]$Binary = 'yoyo-judge-linux-amd64'
)

$ErrorActionPreference = 'Stop'

# BatchMode: fail fast instead of waiting on an invisible password prompt.
$SshOpts = @('-o', 'BatchMode=yes', '-o', 'ConnectTimeout=15')

function Invoke-Remote {
    param([string]$Command, [string]$Describe)
    Write-Host "==> $Describe" -ForegroundColor Cyan
    if ($DryRun) {
        Write-Host "     ssh $Server `"$Command`"" -ForegroundColor DarkGray
        return
    }
    & ssh @SshOpts $Server $Command
    if ($LASTEXITCODE -ne 0) { throw "$Describe failed (ssh exit $LASTEXITCODE)" }
}

function Invoke-Copy {
    param([string]$From, [string]$To, [string]$Describe, [switch]$Recurse)
    Write-Host "==> $Describe" -ForegroundColor Cyan
    $scpArgs = @($SshOpts)
    if ($Recurse) { $scpArgs += '-r' }
    $scpArgs += @($From, "${Server}:$To")
    if ($DryRun) {
        Write-Host "     scp $($scpArgs -join ' ')" -ForegroundColor DarkGray
        return
    }
    & scp @scpArgs
    if ($LASTEXITCODE -ne 0) { throw "$Describe failed (scp exit $LASTEXITCODE)" }
}

# Run from the repo root so scp gets relative local paths. A Windows
# absolute path (C:\...) contains a colon, which scp reads as a host
# separator and mangles into "no such host: C".
Push-Location $PSScriptRoot
try {
    if (-not $SkipBuild) {
        Write-Host "==> Building" -ForegroundColor Cyan
        if ($DryRun) {
            Write-Host "     .\build.ps1" -ForegroundColor DarkGray
        }
        else {
            & (Join-Path $PSScriptRoot 'build.ps1')
            if ($LASTEXITCODE -ne 0) { throw 'build.ps1 failed' }
        }
    }

    if (-not $DryRun) {
        foreach ($required in @("bin/$Binary", 'bin/static/index.html')) {
            if (-not (Test-Path $required)) {
                throw "$required is missing - run .\build.ps1 first (or drop -SkipBuild)."
            }
        }
    }

    # systemctl stop is synchronous and suppresses Restart=always, so the
    # binary's file handle really is released once this returns. The
    # is-active check is a guard against a unit stuck in "deactivating".
    Invoke-Remote -Describe 'Stopping backend' -Command @"
sudo systemctl stop $Service
if systemctl is-active --quiet $Service; then
  echo 'service did not stop - refusing to overwrite the binary'
  exit 1
fi
exit 0
"@

    Invoke-Copy -Describe 'Copying backend binary' -From "bin/$Binary" -To "$AppDir/$Binary"
    Invoke-Remote -Describe 'Clearing frontend docroot' -Command "rm -rf $Docroot/*"
    Invoke-Copy -Describe 'Copying frontend' -Recurse -From 'bin/static/.' -To "$Docroot/"

    # rm -rf + scp recreate the docroot under this user's umask, which can
    # leave it unreadable to nginx's user. o+rX opens directories and files
    # back up without making anything executable that wasn't already.
    Invoke-Remote -Describe 'Fixing docroot permissions' -Command "chmod -R o+rX $Docroot"

    if (-not $Restart) {
        Write-Host "==> Backend left stopped - start it yourself:" -ForegroundColor Yellow
        Write-Host "      ssh $Server `"sudo systemctl start $Service`"" -ForegroundColor DarkGray
    }
    else {
        Invoke-Remote -Describe 'Starting backend' -Command @"
sudo systemctl start $Service
sleep 1.5
if ! systemctl is-active --quiet $Service; then
  echo 'service failed to start - check: journalctl -u $Service -n 30'
  exit 1
fi
systemctl show -p MainPID --value $Service | xargs -I{} echo 'backend running (pid {})'
"@
    }

    Write-Host ''
    if ($DryRun) {
        Write-Host 'Dry run complete - nothing was deployed.' -ForegroundColor Yellow
    }
    else {
        Write-Host 'Deployed.' -ForegroundColor Green
    }
}
finally {
    Pop-Location
}
