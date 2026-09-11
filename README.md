# yoyo-judge

Web-based IYYF scoring system. Go backend + Vue 3 frontend, single-binary deploy.

## Dev

```bash
# Backend (runs on :8081 by default - set PORT=... if that's blocked on your machine)
go run .

# Frontend dev server (runs on :5173, proxies API to :8081)
cd frontend && npm install && npm run dev
```

Or just run `.\dev.ps1` (PowerShell), which does both of the above in separate windows and points Google-login redirects at the dev server correctly.

Open `http://localhost:5173/` — not `/yoyojudge`; that prefix only applies to the production build and the backend's own API routes, the Vite dev server itself serves at root.

`go run .`/`go build .` need `./bin/static` to exist first (the frontend is embedded via `//go:embed`, see `static.go`) — either run `.\build.ps1` once, or use `.\dev.ps1`, which creates an empty placeholder for you automatically.

The database (`yoyojudge.db`) is created automatically on first run, empty — there's no seed/demo data by default. To log in on a fresh database you need Google OAuth configured (see below); there's no built-in way to create the first account otherwise. `server/db.go`'s `SeedIfEmpty` (demo contest + users) still exists but isn't called from `main.go` — uncomment that call there if you want it back for local testing.

### Google login

Needs a Google Cloud OAuth client. Either export the three env vars shown under Deploy below, or copy `env.json.example` to `env.json` and fill in its `"google"` section — `env.json` is only used to fill in whichever of the three isn't already set as a real env var, so real env vars always win. `env.json` is read from the current working directory at startup, wherever the binary/`go run .` is actually invoked from — it does **not** need to be copied into `server/` or any other subfolder.

## Build (production binary)

```powershell
.\build.ps1
```

Produces `bin/yoyo-judge-linux-amd64` (and `.exe`). The frontend is embedded — copy the binary to the server and run it.

## Deploy

Copy to the server: the binary (`bin/yoyo-judge-linux-amd64`), `cert.pem`/`key.pem` if using file-based TLS, and `env.json` if using that instead of real env vars for Google credentials. Everything else (frontend, config) is embedded or set via env vars. `yoyojudge.db` is created on first run — don't overwrite it on redeploy, it's your real data.

### One command

```powershell
.\deploy.ps1
```

That's the whole deploy: it builds, stops the backend, copies the binary and the frontend, and fixes docroot permissions. It never prompts, because authentication is by SSH key (see below). Useful switches: `-SkipBuild` (deploy what's already in `.\bin`), `-Restart`, `-DryRun` (print every local and remote command, connect to nothing).

**It leaves the backend stopped** - starting it again is yours to do (`sudo systemctl start yoyojudge`). Stopping, though, is not optional and the script always does it: `scp` fails with `ETXTBSY` ("text file busy") while the old process still holds the binary open. Pass `-Restart` if you'd rather the script bring it back up for you.

Nothing in the deploy touches `yoyojudge.db`, `env.json`, or `cert.pem`/`key.pem` — they sit in `/home/rizki/yoyojudge` next to the binary, and only the binary itself is overwritten.

Two details the script handles that are easy to get wrong by hand:

- **Stopping the backend through systemctl, not `kill`.** The unit is `Restart=always`, so a killed process comes straight back within seconds and re-takes the binary. `systemctl stop` is synchronous and suppresses the restart; the script then verifies the unit really is inactive before overwriting anything.
- **`chmod -R o+rX` on the docroot afterwards is required.** `rm -rf` + `scp -r` recreate it under your umask, which can leave it unreadable to nginx's user — the site then 403s even though every file is there.

### SSH key (why nothing prompts)

The server trusts `~/.ssh/id_ed25519`, so `ssh`/`scp` to `rizki@103.134.154.210` run unattended. The key has no passphrase, so this survives a reboot with no `ssh-agent` involved. Every call in `deploy.ps1` passes `-o BatchMode=yes` on purpose: if the key ever stops working, the script fails immediately instead of blocking on a password prompt you can't see.

To authorize the key again (new machine, or a rebuilt server), from PowerShell:

```powershell
ssh-keygen -t ed25519 -C "yoyo-judge-deploy"   # only if ~/.ssh/id_ed25519 doesn't exist yet; Enter at every prompt
type $env:USERPROFILE\.ssh\id_ed25519.pub | ssh rizki@103.134.154.210 "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
```

That asks for the password one last time. In `cmd.exe` it's `%USERPROFILE%` instead of `$env:USERPROFILE`.

### Running the backend (systemd)

The backend runs as a systemd service, not in a `screen` session. The unit lives in this repo at [`deploy/yoyojudge.service`](deploy/yoyojudge.service) and is installed at `/etc/systemd/system/yoyojudge.service`.

```bash
sudo systemctl start yoyojudge      # also: stop / restart
systemctl status yoyojudge
journalctl -u yoyojudge -f          # logs; replaces scrolling a screen buffer
```

It's `enable`d, so it starts on boot, and `Restart=always`, so it comes back within ~3s if it crashes. `WorkingDirectory=/home/rizki/yoyojudge` is load-bearing: `env.json`, `cert.pem` and `key.pem` are all read relative to it.

To reinstall the unit after editing it:

```powershell
scp deploy/yoyojudge.service rizki@103.134.154.210:/tmp/
ssh rizki@103.134.154.210 "sudo mv /tmp/yoyojudge.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl restart yoyojudge"
```

### Doing it by hand

If you'd rather not use the script, this is the same copy step as one line (stop the service first, and start it again afterwards):

```bash
scp bin/yoyo-judge-linux-amd64 rizki@103.134.154.210:/home/rizki/yoyojudge/yoyo-judge-linux-amd64 && ssh rizki@103.134.154.210 "rm -rf /var/www/html/yoyojudge/*" && scp -r bin/static/. rizki@103.134.154.210:/var/www/html/yoyojudge/ && ssh rizki@103.134.154.210 "chmod -R o+rX /var/www/html/yoyojudge"
```

### Backend configuration

The backend reads these at startup. In production they come from `env.json` in `/home/rizki/yoyojudge` (read relative to the working directory, which is why the unit sets `WorkingDirectory`); real environment variables always win over `env.json`. To set them as real env vars under systemd, add an `EnvironmentFile=` line to the unit rather than exporting them in a shell.

```bash
# Required — register these in Google Cloud Console first
export GOOGLE_CLIENT_ID="..."
export GOOGLE_CLIENT_SECRET="..."
export GOOGLE_REDIRECT_URL="https://yourdomain.com:8081/yoyojudge/api/auth/google/callback"
export FRONTEND_URL="https://yourdomain.com/yoyojudge"   # only if frontend is on a different port

# Optional overrides
export PORT=8081
export BASE_PATH=/yoyojudge
export DATABASE_PATH=/data/yoyojudge.db

# TLS: place cert.pem and key.pem next to the binary (or set TLS_CERT_FILE / TLS_KEY_FILE)

./yoyo-judge-linux-amd64
```

See `docs/PROGRESS.md` for architecture notes.
