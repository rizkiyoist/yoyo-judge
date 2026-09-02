# yoyo-judge

Web-based IYYF scoring system. Go backend + Vue 3 frontend, single-binary deploy.

## Dev

```bash
# Backend (runs on :8081)
go run .

# Frontend dev server (runs on :5173, proxies API to :8081)
cd frontend && npm install && npm run dev
```

Open `http://localhost:5173/yoyojudge`. Log in with any seeded demo user (e.g. `galih@example.com`).

The database (`yoyojudge.db`) is created automatically on first run and seeded with a demo contest.

## Build (production binary)

```powershell
.\build.ps1
```

Produces `bin/yoyo-judge-linux-amd64` (and `.exe`). The frontend is embedded — copy the binary to the server and run it.

## Deploy

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
