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

### Copy to the server

No CI — the server has no git access, so this is just two commands run by hand from the repo root after `.\build.ps1`. You'll be prompted for your normal SSH credentials/passphrase each time; nothing here changes how you authenticate.

Binary:

```bash
scp bin/yoyo-judge-linux-amd64 rizki@103.134.154.210:/home/rizki/yoyojudge/yoyo-judge-linux-amd64
```

Frontend — fully replaces the docroot's contents (same as deleting and re-pasting by hand):

```bash
ssh rizki@103.134.154.210 "rm -rf /var/www/html/yoyojudge/*" && scp -r bin/static/. rizki@103.134.154.210:/var/www/html/yoyojudge/
```

Then SSH in as usual to restart the backend in its screen session — neither command touches `yoyojudge.db`, `env.json`, or `cert.pem`/`key.pem`, since those all live outside both destination paths.

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
