# Progress Summary

_Last updated: 2026-09-02 (owner vs. transferable head judge; whole-contest
lock replaces the earlier per-stage lock; division/player deletion;
eval-score scale fixed to real 0-10, verified against the source xlsx)_

## Resuming on a new machine

Repo state as of this note: `main` — everything described below is already
committed and pushed, so `git clone`/`git pull` is all that's needed for the
code itself. What **won't** come with a fresh clone (all gitignored):

- `env.json` (copy from `env.json.example`) — now carries real secrets: a
  `"google"` section with `client_id`/`client_secret`/`redirect_url` for
  Google OAuth, loaded by `envjson.go` at startup (see "Google credentials
  via env.json" below). The old `"mysql"` section was removed — nothing
  reads it, superseded by SQLite (see below). `dbconfig.yml` is likewise
  unused now, only relevant if you touch the old `migration/` folder.
- `bin/`, `dist/` — build output. Run `./build.ps1` (repo root, PowerShell)
  to regenerate both the Windows/Linux backend binaries and the embedded
  frontend `dist/`. **`go build .`/`go run .` will fail on a fresh clone**
  until the frontend step of `build.ps1` has populated `dist/` at least
  once, since `static.go` embeds it at compile time — or use `./dev.ps1`
  (see below), which creates a placeholder automatically.
- `frontend/node_modules/` — run `npm install` in `frontend/` first (or use
  `./dev.ps1`, which does this for you).
- `yoyojudge.db` (SQLite file, path overridable via `DATABASE_PATH`) — the
  actual persisted app data (contests, users, sessions, scores). Not in git
  (correctly — it's real data, not code). Auto-created + seeded with demo
  data on first run if missing; **don't let a redeploy silently create a
  fresh empty one** in place of an existing one with real data on it.
- Production `cert.pem`/`key.pem` live only on the deploy server
  (`103.134.154.210`, `rizkiyoist.duckdns.org`), not in this repo or
  expected on a dev machine — see the acme.sh steps under "Real production
  deployment" below if that ever needs to be redone from scratch.

**Local dev gotcha (Windows):** the default port `8081` (and other ports —
checked `8090`) can fall inside a Windows-reserved TCP exclusion range (seen
on one dev machine: `8056–8155`, from Hyper-V/WSL2), causing `listen tcp
:8081: bind: An attempt was made to access a socket in a way forbidden by
its access permissions.` Check with `netsh interface ipv4 show
excludedportrange protocol=tcp` and run with e.g. `PORT=9000
./bin/yoyo-judge.exe` instead — this is local-machine-specific, not a code
bug, and doesn't affect the actual deploy server.

## Goal

Convert the IYYF (International Yo-Yo Federation) scoring calculation Excel
workbook (`IYYF-SCORE-CALC-FINAL-2017.xlsx` / prelim variant, both under
`docs/`) into a web-based yoyo judging/scoring system, written in Go.

## Current approach

**Superseded (2026-08-16):** the scoring math is now reimplemented natively
in Go under `library/calc`, instead of writing judge inputs into the `.xlsx`
workbook and letting Excel's own formulas compute the derived scores. The
original `.xls` reference files couldn't be read directly (the import
library only supports `.xlsx`), so they were converted with LibreOffice
headless (`soffice --headless --convert-to xlsx`) to inspect their actual
formulas (not just cached values) sheet by sheet, and those formulas were
ported to Go. The old excelize-based writer/reader code
(`library/writer`, `library/reader`) and cell-reference constants
(`library/calc/const.go`) are still in the tree but are no longer the
intended path forward — they were the previous approach described below.

<details>
<summary>Previous approach (superseded)</summary>

The original strategy was to keep the workbook as the source of truth for
the scoring math, and treat it like a "backend calculator": the Go app
writes judge inputs into the known cells of the workbook (via `excelize`),
lets Excel's own formulas compute the derived scores, and (eventually) reads
the computed results back out. This avoided reimplementing IYYF's scoring
rules by hand, at the cost of coupling the app to the workbook's sheet/cell
layout and to a single shared file (no real concurrency story).

</details>

## What's implemented

### Native scoring engine (`library/calc`) — new, 2026-08-16

Reverse-engineered from the actual formulas in both reference workbooks
(`docs/IYYF-SCORE-CALC-FINAL-2017.xls` and `...-PRELIM-2017.xls`, ver. 2.3,
2017-8-3 Hironori Mii), across the `SET-UP`, `RAW-TEx`, `RAW-TEvPEv`,
`ADJ-CLICK`, `ADJ-GIVEN`, and `FINAL-SCORE` sheets:

- `stage.go` — `ScoringStage` (`StageFinal` / `StagePrelim`) and the two
  stages' default configuration:
  - FINAL: 8 evaluation categories (EXE/CTL/TDV/SEM → T.Ev, MU1/MU2/BDY/SHW →
    P.Ev), each averaged across the 6 eval judges and **halved** before
    summing; deductions Stop(1)/Discard(3)/Cut(5).
  - PRELIM: 4 evaluation categories (EXE/CTL → T.Ev, MU1/BDY → P.Ev), summed
    **without halving**; deductions Stop(1)/Discard(3)/Detach(5).
  - Both stages: 6 clicker judges, clicker value 60.
  - `NewContest(stage, players)` builds a `Contest` pre-populated with a
    stage's defaults.
- `input.go` — input types: `ClickerScore` (plus/minus + `Net()`),
  `EvalCategory`, `Deduction`, `PlayerInput` (one player's raw judge input),
  `Contest` (stage config + all players).
- `rules.go` — `(*Contest) Calculate() []PlayerResult`, reproducing the
  workbook's formula chain:
  1. **T.Ex (technical execution)**: each clicker judge's net click count
     (plus − minus) is scaled as a fraction of *that judge's* best net count
     across the whole field, times the clicker value (60); T.Ex is the
     average of the 6 scaled judge scores. (`ADJ-CLICK` H/J/L/N/P/R →
     `FINAL-SCORE` D)
  2. **Evaluation categories**: each category's raw score is averaged across
     the 6 eval judges, then halved if the stage calls for it. Category
     scores roll up into T.Ev/P.Ev group totals. (`ADJ-GIVEN` →
     `FINAL-SCORE` E-N)
  3. **Evaluation total** = T.Ex + T.Ev total + P.Ev total. (`FINAL-SCORE` O)
  4. **Deductions**: count × per-occurrence value, summed. (`ADJ-CLICK`
     C-E → `FINAL-SCORE` P-R)
  5. **Final score** = evaluation total − deduction total. (`FINAL-SCORE` S)
  6. **Place**: standard competition ranking, descending by final score —
     ties share a place and the next place skips accordingly (matches
     Excel's `RANK(score, all_scores, 0)`). (`FINAL-SCORE` T)
- `rules_test.go` — unit tests with hand-verified expected values (cross-checked
  against the formula chain in Python) covering the FINAL stage happy path,
  tied-place ranking, and the PRELIM stage's no-halving behavior.

Not yet done: no HTTP layer wires this up, and `handler/input.go`'s structs
(`SetUp`, `Player`, `RawTex`, `RawPev`) aren't yet mapped to
`calc.PlayerInput`/`calc.Contest`.

### Excel integration (`library/writer`, `library/reader`) — superseded, kept for reference
- `library/calc/const.go` — cell-reference constants mapping the 4 relevant
  sheets in the workbook to named fields:
  - `SET-UP` — contest name, division, stage, and the 6 clicker (technical)
    judges + 6 evaluation judges
  - `PLAYER` — player name
  - `Raw-TEx` — technical execution: stop/discard/cut counts and each clicker
    judge's plus/minus counts
  - `Raw-TEvPEv` — evaluation scores: each of the 6 eval judges' Exe / Ctl /
    Tdv / Sem / Mu1 / Mu2 / Bdy / Shw sub-scores
- `library/writer/setup.go`, `player.go`, `tex.go`, `pev.go` — each opens the
  `.xlsx` file, writes one struct's fields into the corresponding named cells
  via excelize, and saves. These are the "input" side: pushing judge/contest
  data into the workbook.
- `library/reader/xls-reader.go` — minimal read-only proof of concept that
  prints column A of the `SET-UP` sheet. Not yet used to extract computed
  scores.
- `library/calc/input.go` now holds the native calc engine's input types
  instead (see above) — it's no longer an empty stub tied to this section.

### Data shapes (`handler/input.go`)
Plain structs describing the full expected input shape per the v2.3 IYYF
spec (2017-08-03): `SetUp`, `OptionalSetUpFinal`, `OptionalSetUpPrelim`
(stub), `Player`, `RawTex`, `RawPev`. These look like the intended
request/domain models but aren't yet wired to the writer layer or exposed
over HTTP.

### App scaffold
- `main.go` — loads config, opens MySQL via GORM, builds the router.
- `router.go` — `mux` router + CORS, only a health-check `"/"` route wired up;
  commented-out placeholder for a real route group.
- `config/` — Viper-based JSON config loader (`env.json`) and MySQL/GORM
  connector.
- `domain/service/generic.go` — generic GORM repository (`Repository[T]`)
  with Create/Update/Delete/FindOneBy/FindBy/Count — reusable CRUD base for
  future domain models.
- `library/helper/limit_offset.go` — pagination helper used by the generic
  repository.
- `domain/model/user.go`, `user_social.go` — `users` and `user_socials`
  tables (the latter for social login tokens).
- `controller/auth/` — `Usecase` interface with a `LoginGoogle()` stub (not
  implemented).
- `request/user.go` — empty package stub.
- `migration/` — SQL migrations (via `sql-migrate` style `-- +migrate Up/Down`)
  creating `users` and `user_socials` tables, including the FK and unique
  constraints for one social account per user per provider.

### Vue frontend (`frontend/`) — new

A Vue 3 + Vite + Pinia + TypeScript SPA, built against a **mocked API**
(no real Go HTTP backend exists yet — see "What's not implemented" below).
Full plan/rationale: see the design in this conversation; summary here for
future reference.

- **Workflow modeled**: a head judge logs in, creates a contest, adds
  divisions (each markable Prelim/Final/both), invites other judges
  (searchable by name/email) into clicker (J-A1-6) or evaluator (J-B1-6)
  slots per division+stage, manages the player roster; invited judges log
  in, enter their own scores, and can view the full results — including
  every other judge's raw input, not just the computed total.
- `src/lib/scoring.ts` (+ `scoring.test.ts`) — a 1:1 TypeScript port of
  `library/calc/stage.go` + `rules.go`, tested against the exact same
  expected values as `library/calc/rules_test.go`, via Vitest
  (`npm run test` in `frontend/`). This keeps the client-side results
  computation provably in sync with the Go engine.
- `src/types.ts` — shared domain types (`User`, `Contest`, `Division`,
  `JudgeAssignment`, `Player`, `PlayerRawScores`, `PlayerResult`) — the
  contract a real backend should eventually satisfy.
- `src/api/client.ts` — the `ScoringApi` interface (login, contests,
  divisions, judge invites, players, score submission, results). This is
  the seam: swapping in a real HTTP client later means implementing this
  one interface, no view changes needed.
- `src/api/mock.ts` — in-memory + `localStorage`-backed implementation,
  seeded with a demo contest ("Indonesia National Yoyo Championships",
  division "3A", both stages, 1 head judge + 6 clicker + 6 eval seeded
  users with real Indonesian names, 4 players, and
  pre-filled FINAL-stage scores). `login()` just matches by email (no real
  auth). `getResults()` builds a `calc.Contest`-shaped object from stored
  raw inputs and runs it through `lib/scoring.ts`, so results are genuinely
  computed, not fake numbers. **Note:** the seed pre-authenticates a
  browser as the head judge by default (`sessionUserId` set at seed time),
  so a fresh session lands on the contest list, not `/login`, until you log
  out — intentional for demo convenience, worth revisiting for a real login.
- `src/stores/auth.ts`, `src/stores/contests.ts` — Pinia stores for the
  cross-cutting concerns (current user/session, the contest list); other
  views call the API directly for view-scoped data (players, assignments,
  raw scores, results).
- `src/router/index.ts` — routes for login, contest list, contest/division
  editing, judge management, player roster, score entry, and results, with
  an auth guard redirecting to `/login` when logged out.
- `src/views/*.vue` — one view per route; role gating (head-judge-only
  actions, judge-assignment-only score entry) is enforced client-side only,
  matching the mock's lack of real auth.
- Verified by hand with a throwaway Playwright driver (not committed):
  login → contest list → results leaderboard (computed, ranks correctly
  reflect seeded deductions) → judge management (invite/list/remove) →
  score entry as an invited clicker judge (edit persists to `localStorage`
  and round-trips through the UI). No console errors observed.
- `build.ps1` (repo root) — builds both the Go backend and the frontend in
  one step; `-Only backend` / `-Only frontend` to build just one. See
  "Single-binary deploy" below for how this evolved.

Superseded by the real backend below — the frontend now talks to it by
default (`VITE_USE_MOCK=true` still selects the mock for offline work).

### Go HTTP backend (`server/`) — new, 2026-08-16

Implements `frontend/src/api/client.ts`'s `ScoringApi` contract for real,
backed by an **in-memory, mutex-protected store** rather than MySQL/GORM —
so the site is testable with zero setup (no MySQL/Docker required). Real
persistence can replace just the store layer later without touching the
HTTP layer.

- `server/types.go` — `User`, `Contest`, `Division`, `JudgeAssignment`,
  `Player`, `ClickerInput`, `MajorDeductions`, `PlayerRawScores`, mirroring
  `frontend/src/types.ts` field-for-field (JSON tags in camelCase).
  `PlayerResultResponse` wraps `library/calc.PlayerResult` with the
  `playerId` the frontend expects, **without** re-implementing the scoring
  math — unlike the frontend's necessary TS port, the Go backend calls
  `library/calc` directly.
- `server/store.go` — nested contests → divisions → players/assignments/scores,
  seeded with the **identical demo data** as `frontend/src/api/mock.ts`
  (head judge Galih Kurniawan, `galih@example.com`; 6 clicker + 6 eval
  judges — names kept in sync between the two seed files by hand, e.g.
  clicker slots 1-3 are Paketu Dennis/Levian Saputra/Boris Chietra, eval
  slots 1-2 are Reynold Andika/Hendra Kusumah; contest "Indonesia National
  Yoyo Championships" (year 2026), division "3A", both stages, **10
  players**, FINAL-stage scores pre-filled) — the same login flow already
  verified against the mock works unchanged. `toPlayerInput` converts
  stored raw scores into `calc.PlayerInput`. **Keep the two seed files'
  judge-name arrays and player lists in sync by hand** — nothing enforces
  this automatically.
- `Contest.Year` (int) added end-to-end: Go `server/types.go`/`store.go`/
  `handlers.go`, frontend `types.ts`, `api/client.ts`/`mock.ts`/`http.ts`,
  `stores/contests.ts`. Surfaced as a year input next to the contest-name
  field on the "Create a contest" form and a colored `.badge-year` pill
  next to the contest name everywhere it's shown.
- `server/auth.go` — bearer auth: `POST /api/auth/login` returns the user's
  id as an opaque token; protected routes require `Authorization: Bearer
  <userId>`. No passwords — matches the mock's security level, a known gap.
  `GET /api/users/search` is deliberately public (no auth) since the login
  screen needs it before a session exists.
- `server/handlers.go` — one handler per `ScoringApi` method under `/api`
  (auth, contests, divisions, judges, players, scores, results); results
  are computed live via `calc.NewContest(stage, inputs).Calculate()`.
- Wired into the existing app: `router.go` mounts `server.Mount(router,
  store)` and adds `http://localhost:5173` (Vite's dev port) to the CORS
  allowlist; `main.go` no longer calls `config.InitSQL` — it constructs a
  seeded `server.NewStore()` instead (DB wiring can come back once real
  persistence is built).
- `frontend/src/api/http.ts` — real `ScoringApi` implementation using
  `fetch` against `VITE_API_BASE_URL` (default `http://localhost:5000/api`),
  storing the login token in `localStorage`. `frontend/src/api/index.ts` now
  defaults to this; `VITE_USE_MOCK=true` still selects `mock.ts`.
- Verified: `curl` smoke tests (login, `/me`, list contests, get results)
  confirmed against the identical numbers seen from the mock, and the full
  browser flow (login → contest list → results leaderboard) re-verified
  with a throwaway Playwright driver against the real backend — 0 console
  errors.
- **Bug found and fixed along the way:** `GET /users/search` was
  originally gated behind the same auth middleware as everything else, but
  the login screen calls it *before* a session exists (to list demo users
  to pick from) — made it public to match `mock.ts`'s behavior.
- **Unrelated pre-existing bug found and fixed:** `config/sql.go`'s
  `InitSQL` read `key+".pass"` and `key+".name"`, but `env.json` uses
  `"password"` and `"database"` — so `dbPass`/`dbName` were always empty,
  producing a DSN with no password and no database name. This is why
  `yoyo-judge.exe` failed to connect while `sql-migrate` (which reads
  `dbconfig.yml`'s literal DSN string directly) worked fine with the "same"
  config — they were never actually reading the same values. Fixed the key
  names; currently unused since `main.go` doesn't call `InitSQL` anymore,
  but ready for when real DB persistence is wired back in.

### Frontend UX polish — new, 2026-08-16

- **Head-judge score override** moved to its own route/view
  (`ScoreOverrideView.vue`, `/.../override`), reached via a button on the
  normal scoring page (`ScoreEntryView.vue`). Regular scoring now only ever
  shows the logged-in judge's own assigned slot(s) — no inline override
  controls. The override page lets the head judge pick any clicker/eval
  judge from a dropdown and edit their raw scores directly.
- **Results table** (`ResultsView.vue`) restructured to mirror the xls
  workbook's RESULT sheet columns (TE, then each raw category under its
  descriptive label, Categories Total, E.Total, Stop/Discard/Cut-or-Detach,
  Final Score, Details), with the TE/T.Ev columns colored blue
  (`.col-tex`) and the P.Ev columns colored amber (`.col-pev`) via CSS
  custom properties so they adapt to the light/dark theme — a quick visual
  split between "technical" and "performance" scoring. Final Score stays
  bold. `.card` now has `overflow-x: auto` so wide tables scroll inside
  their card instead of visually overflowing/clipping its border.
- **Contest list** (`ContestListView.vue`): "Contests" → "All Contests";
  Players/Input Score/Result Detail column buttons now just say the stage
  name ("Prelim"/"Final") since the column header already says what they
  are; Input Score and Result Detail are separate columns; Top 3 column
  shows only the FINAL stage's leaderboard (prelim only as a fallback for
  divisions with no final stage); judge lists split into Clicker (TEx) /
  Evaluation (PEv) columns, bolding the head judge's name if they also
  appear as an assigned judge; new **"Download Results"** button builds a
  self-contained HTML file (via `Blob` + a temporary `<a download>`) with
  the FINAL-stage results table for every division in the contest and
  triggers a browser download — no server round-trip beyond fetching each
  division's results.
- **Judge management** (`JudgeManagementView.vue`): "Current assignments"
  has Prelim/Final tabs (shared state with the invite form, so an invite
  lands on whichever tab is active) and splits Clicker/Evaluation judges
  into two side-by-side tables instead of one combined table with a Role
  column; head judge's name bolded there too if assigned as a judge.
- Removed the 🪀 emoji from the nav bar brand (just "yoyo-judge" now).
- Fixed a spacing bug on the contest list: the per-division table sat
  flush against the contest name/year/buttons row because the `<h2>` had
  `margin: 0` (needed so the year badge could sit inline with it) with
  nothing replacing the lost margin — added `margin-bottom` back on the
  wrapping row.

### Single-binary deploy: embedded frontend + Linux cross-build — new, 2026-08-21

Modeled on the sibling project `../orkestrator-v2`'s approach (its
`backend/static.go` + `main.go`'s `r.NoRoute` handler), so `bin/` is now a
true "copy it to the server and run it" deploy — no separate static host,
docroot, or SPA-fallback config needed on the ops side.

- `static.go` (repo root) — `//go:embed all:dist` embeds a `dist/` folder
  (populated by `build.ps1` from `frontend/dist` before compiling) directly
  into the Go binary. **`dist/` is gitignored** (like `bin/`) — a fresh
  clone needs `./build.ps1` (or at least its frontend step) run once before
  `go build .`/`go run .` will succeed, since the embed path must exist at
  compile time.
- `router.go` — after mounting `/api/*` (`server.Mount`) and `/healthz`
  (moved off `/`, which now serves the frontend), a catch-all
  `router.PathPrefix("/").Handler(...)` serves the embedded `dist/` via
  `http.FileServer`, with manual SPA fallback: a request path containing a
  `.` is served as a static asset, anything else gets `index.html` so Vue
  Router's history-mode client-side routing can take over. Registered last
  so it doesn't shadow the API routes. (Both `/api` and this catch-all
  later gained a `BASE_PATH` prefix — see "BASE_PATH deploy mode" below.)
- `frontend/vite.config.ts` — `base` changed from `'./'` (relative) back to
  `'/'` (absolute root). This was a real bug caught while verifying the
  embedded setup: with a relative base, a hard refresh on a nested route
  like `/contests/abc/edit` resolves `./assets/*.js` relative to that
  path (`/contests/abc/assets/*.js`) instead of the real root, 404ing —
  confirmed with `curl` before and after the fix. Absolute paths are the
  right call now that the app is always served from one origin's root by
  the embedded server; the earlier "relocatable to any subpath" goal is
  moot for this deploy model (the still-copied `bin/static/` standalone
  folder is the fallback for anyone who does want to host it separately,
  but they'd need to rebuild with a matching `base` anyway, same as they
  already need to rebuild with a matching `VITE_API_BASE_URL`).
- `build.ps1` — now cross-compiles **both** `bin/yoyo-judge.exe`
  (windows/amd64) and `bin/yoyo-judge-linux-amd64` (linux/amd64,
  `CGO_ENABLED=0`, statically linked, verified as a real ELF binary) from
  the same source tree via `GOOS`/`GOARCH`. Build order changed: frontend
  now builds *first* (`Build-Frontend`), copying into repo-root `dist/` for
  embedding, then `bin/static/` as before; `Build-Backend` fails fast with
  a clear message if `dist/` doesn't exist yet, instead of a cryptic
  `//go:embed` compiler error.
- Verified end-to-end against the compiled `yoyo-judge.exe` itself (not the
  Vite dev server): login → contest list → results (deep nested route) →
  hard browser reload on that same deep URL — all served correctly from
  `:5000`, 0 console errors, confirmed with a throwaway Playwright script
  before deleting it.

### BASE_PATH deploy mode — new, 2026-08-21

Also mirroring `../orkestrator-v2` (its `config.BasePath`/`.env`
`BASE_PATH`): a second deployment shape for when you don't control the
config of whatever's already serving the target domain on port 80/443 (so
no reverse-proxy `location` block can be added). Both shapes now coexist,
selected by whether `BASE_PATH` is set at runtime:

- **Same-origin (default)**: `yoyo-judge.exe`/`yoyo-judge-linux-amd64`
  serves both the API and the embedded frontend on one port, entirely
  under a shared prefix (default `/yoyojudge`, matching orkestrator-v2's
  `/orkestrator-v2` convention) — e.g. `http://server:5000/yoyojudge`.
  Works standalone or behind a reverse proxy that preserves the full path.
- **Cross-origin (split)**: the frontend build (`bin/static/` or the
  `dist/` that gets embedded) is instead copied directly into an existing
  web server's docroot at a matching subpath (e.g.
  `/var/www/html/yoyojudge/`) — no config change needed there, it just
  serves whatever files exist. The backend keeps running standalone on its
  own exposed port, with its routes prefixed the same way, and the
  frontend is built with an absolute `VITE_API_BASE_URL` pointing at that
  port so its (now cross-origin) API calls reach it directly.

Changes:
- `router.go`'s new `basePath()` reads `BASE_PATH` from the environment,
  defaulting to `/yoyojudge` (trailing slash trimmed). Every route —
  `/healthz`, `server.Mount`'s `/api/*`, and the embedded-frontend
  catch-all — is now registered under this prefix. The catch-all also
  registers an exact-match route at the bare prefix (no trailing slash,
  e.g. `/yoyojudge`) in addition to `PathPrefix(bp + "/")`, since a mux
  `PathPrefix` alone wouldn't match a request with nothing after the
  prefix.
- `server/handlers.go`'s `Mount` now takes a `basePath string` param,
  mounting `/api` under `basePath + "/api"` instead of a hardcoded `/api`.
- `frontend/vite.config.ts` — production builds now set `base` to
  `${VITE_BASE_PATH}/` (default `/yoyojudge/`); the dev server keeps
  `base: '/'` (simplest for local dev) but proxies that same prefix to
  `http://localhost:5000` so API calls made during `npm run dev` still
  reach the backend without needing `VITE_API_BASE_URL` set.
- `frontend/src/api/http.ts` — the default `BASE_URL` (used when
  `VITE_API_BASE_URL` isn't set) is now a **relative**, same-origin path:
  `${VITE_BASE_PATH}/api` (default `/yoyojudge/api`) instead of the
  previously hardcoded `http://localhost:5000/api`. This is what makes the
  embedded same-origin mode work regardless of what `BASE_PATH` is set to,
  without baking in a hostname; `VITE_API_BASE_URL` remains the override
  for the cross-origin split-deploy shape.
- `build.ps1` gained `-BasePath` and `-ApiBaseUrl` params, setting
  `VITE_BASE_PATH`/`VITE_API_BASE_URL` for the frontend build step (restored
  afterward so they don't leak into the calling shell).
- Verified with `curl` against the compiled binary: `/yoyojudge/`,
  `/yoyojudge` (no trailing slash), `/yoyojudge/assets/*.js`,
  `/yoyojudge/api/users/search`, `/yoyojudge/healthz`, and a deep SPA
  route all return 200; the old unprefixed `/api/...` and `/` now 404 (as
  expected — everything lives under the prefix now). Re-verified the full
  browser flow (login → deep route → **hard reload on that deep,
  prefixed URL**) against the compiled binary with 0 console errors, and
  separately verified `npm run dev` still works end-to-end through the new
  proxy rule.

### Real production deployment: TLS on the backend, configurable port — new, 2026-08-21

What actually shipped to `rizkiyoist.duckdns.org`, after a long back-and-forth
figuring out which of the two shapes above the real target needed, plus two
real bugs found along the way. Worth reading in full before touching the
deploy again.

**The actual deployed shape is the cross-origin split** (frontend on the
existing domain's nginx docroot, backend standalone on its own port),
matching `../orkestrator-v2`'s pattern — but with one addition orkestrator
doesn't have:

- `router.go` can now serve **HTTPS directly**: `TLS_CERT_FILE`/
  `TLS_KEY_FILE` env vars, or (simpler, no env vars needed at all)
  `cert.pem`/`key.pem` sitting next to the binary are picked up
  automatically (`firstExisting()`). Falls back to plain HTTP if neither is
  present.
- The listen port is now `PORT`-configurable (`port()`), defaulting to
  `8081` — not orkestrator's `8080`, so the two can run on the same box
  without colliding (they're on the same server: both resolve to
  `103.134.154.210`, both actually served under
  `rizkiyoist.duckdns.org/{orkestrator-v2,yoyojudge}`, confirmed with curl).
- `build.ps1`'s `-ApiBaseUrl` **default** is now the real production value:
  `https://rizkiyoist.duckdns.org:8081/yoyojudge/api` — a plain
  `.\build.ps1` with no flags produces the correct deployable build. This
  replaced an earlier design where the default was a same-origin relative
  path (correct for the embedded single-binary shape, wrong for what's
  actually deployed) — that default kept silently reverting the API URL to
  the wrong thing on rebuild, which was the single biggest source of
  wasted back-and-forth in this session. Lesson: **when there's one real
  deployment target, bake its actual values in as the default** — don't
  make the correct build depend on remembering a flag.

**Why HTTPS on the backend is unavoidable here (not a preference):**
`rizkiyoist.duckdns.org` 301-redirects `http://` → `https://` (confirmed
with curl — this is real, not assumed). That forces the page to load over
HTTPS, and browsers hard-block an HTTPS page from calling a plain `http://`
API (mixed content) — no exception for "same server, different port".
Initially assumed `../orkestrator-v2` proved this pattern safe without TLS
(its README bakes in a plain `http://103.134.154.210:8080/...` API URL);
directly verified that `../orkestrator-v2` is served from the **same**
domain, with the **same** forced-HTTPS redirect — meaning it has this exact
same latent bug, just never hit/noticed. Confirmed live in a real browser
(not just curl) at the user's request. **Not fixed there** — a separate,
independent fix would be needed for that app (its own TLS support + its
own cert), explicitly out of scope here per the user's request not to touch
that repo.

**Real bug found, not a CORS bug despite how it presented:** after getting
the cert working, the API URL was initially built using the server's raw
IP (`https://103.134.154.210:8081/...`). The browser reported this as a
CORS failure — but the actual cause was a **TLS certificate hostname
mismatch**: the cert was issued for `rizkiyoist.duckdns.org`, not for the
IP, so a real browser's certificate validation rejects the IP connection
outright, which surfaces as a blocked cross-origin request. `curl -k`
(skip cert validation) masked this during earlier verification, so it
looked fine there — plain `curl` without `-k` reproduced the real failure
immediately once tried. Fixed by using the domain name (not the IP) in the
API URL, on the same port; a cert only validates against the hostname it
was actually issued for, regardless of what IP that hostname resolves to.

**Cert setup used**: `acme.sh` with DuckDNS's DNS-01 challenge (no port
binding, so no conflict with nginx already on 80/443):
```
curl https://get.acme.sh | sh -s email=...
export DuckDNS_Token="..."
~/.acme.sh/acme.sh --issue --dns dns_duckdns -d rizkiyoist.duckdns.org
~/.acme.sh/acme.sh --install-cert -d rizkiyoist.duckdns.org \
  --key-file ~/yoyojudge/key.pem --fullchain-file ~/yoyojudge/cert.pem
```
Auto-renews via `acme.sh`'s own cron job; the backend just needs restarting
after a renewal to pick up the refreshed files.

Not yet done at the time: no real persistence (everything resets on backend
restart); no real authentication; the reload-into-404-on-a-deep-route issue
is inherent to serving the frontend as static files from nginx (no
SPA-fallback rule there) and is unfixed — would need either an nginx
`try_files` rule or switching the frontend to hash-based routing.
**Superseded by the next section** — both persistence and auth are now
real.

### SQLite persistence + Google OAuth login — new, 2026-09-02

Replaces the in-memory store with a SQLite-backed one and adds real Google
sign-in, closing out the two biggest items from "what's not implemented"
above. CGO-free via `github.com/glebarez/sqlite` (no CGO/gcc toolchain
needed to build, unlike `mattn/go-sqlite3` — matters for the Windows→Linux
cross-compile `build.ps1` already does).

- `server/db_models.go` — GORM models: `DBUser`, `DBSession`, `DBContest`,
  `DBDivision`, `DBJudgeAssignment`, `DBPlayer`, `DBPlayerRawScore`. Raw
  judge scores (clicker inputs, eval scores, deductions) are stored as JSON
  blobs per player+stage rather than fully normalized columns.
- `server/db.go` — `OpenDB()` opens the SQLite file at `DATABASE_PATH`
  (default `yoyojudge.db`, created next to the binary), enables WAL mode +
  `synchronous=NORMAL`, and `AutoMigrate`s all the models — no separate
  migration step needed, unlike the old `sql-migrate`/`migration/` path.
  `SeedIfEmpty()` runs the same demo-data seed as before (identical judge
  names/contest/players to keep continuity with earlier manual testing),
  but only when the `users` table is empty — safe to call on every startup.
- `server/store.go` — rewritten from in-memory maps to GORM queries
  end-to-end; the `sync.Mutex` is gone (SQLite handles its own
  concurrency). Session tokens are now random 64-char hex strings stored in
  a real `sessions` table, not "the token is just the user ID" as before.
- `server/auth.go` — `bearerUser` now looks up the sessions table; logout
  deletes the session row instead of being a no-op.
- `server/oauth.go` — full Google OAuth2 authorization-code flow
  (`golang.org/x/oauth2` + `google.Endpoint`): `/auth/google` redirects to
  Google, `/auth/google/callback` exchanges the code, fetches userinfo,
  upserts the user (matched by Google `sub` first, falling back to email),
  creates a session, and redirects to the frontend with `?token=...`.
  CSRF-style `state` values are single-use, in-memory, 5-minute expiry.
  Requires three env vars to activate (`isGoogleConfigured()` checks all
  three); without them, `/auth/google*` returns 503 and the email/demo
  login path (still present) keeps working:
  - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` — from a Google Cloud OAuth
    client.
  - `GOOGLE_REDIRECT_URL` — must exactly match what's registered in Google
    Cloud, e.g. `https://rizkiyoist.duckdns.org:8081/yoyojudge/api/auth/google/callback`.
  - `FRONTEND_URL` (optional) — where to redirect after login, e.g.
    `https://rizkiyoist.duckdns.org/yoyojudge`. If unset, it's inferred
    from the incoming request's scheme/host/`BASE_PATH`, which is correct
    for the embedded single-binary deploy but **must be set explicitly for
    the cross-origin split deploy** (the actual production shape), since
    there the backend's own host isn't the frontend's host.
- Frontend: `AuthCallbackView.vue` (new, `/auth/callback` public route)
  reads `?token=`, stores it, resets the auth store, and redirects home.
  `LoginView.vue` now leads with a "Continue with Google" button (hidden
  when `VITE_USE_MOCK=true`, since the mock has no OAuth backend); the old
  email/demo-user login is still there, collapsed under a `<details>`.
- **Local dev / new-machine gotcha**: `yoyojudge.db` is gitignored and
  local to wherever the backend last ran — a fresh clone or a new machine
  starts with an empty, freshly-seeded DB, not whatever data existed
  elsewhere. On the actual deploy server, treat `yoyojudge.db` as real data
  to back up, not a build artifact to regenerate.

### Google credentials via env.json — new, 2026-09-02

`envjson.go` (repo root) — `loadEnvJSON()`, called first thing in
`main()`, reads `env.json`'s new `"google"` section
(`client_id`/`client_secret`/`redirect_url`) and calls `os.Setenv` for
`GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`GOOGLE_REDIRECT_URL` — but only
for whichever of those three isn't already set in the real environment
(`setEnvIfUnset`), so a real deployment's env vars always win and this is
purely a local-dev convenience (no more remembering to `export` three vars
every session). `env.json`'s old unused `"mysql"` section was removed
(nothing read it — see "SQLite persistence" above) while touching this
file. `env.json.example` mirrors the new shape. Verified: ran the compiled
binary with a temp `env.json` containing fake credentials and confirmed
`/api/auth/google`'s redirect carried that exact `client_id`.

**Google Cloud Console gotchas hit while setting this up for real** (worth
reading before touching OAuth config again):
- "Authorized JavaScript origins" and "Authorized redirect URIs" are two
  separate fields with different rules — origins must be bare
  `scheme://host:port` (no path, no trailing slash: `http://localhost:5000`),
  while redirect URIs need the full exact callback path
  (`http://localhost:5000/yoyojudge/api/auth/google/callback`). This app's
  server-side redirect flow doesn't strictly need an origin entry at all;
  the actual failure mode when the redirect URI is missing/wrong is Google
  showing "this app doesn't comply with Google's OAuth 2.0 policy" (or the
  more classic `Error 400: redirect_uri_mismatch`) with the offending
  `redirect_uri` printed in the request details — that value is exactly
  what needs to be added, verbatim, to "Authorized redirect URIs".
- Google's console warns it can take 5 minutes to a few hours to propagate
  a saved change — a retry immediately after saving isn't a reliable test
  that a fix didn't work.

### dev.ps1: one-command local dev runner — new, 2026-09-02

`dev.ps1` (repo root) — `./dev.ps1` opens two PowerShell windows: backend
(`go run .`, `PORT` defaulting to **5000**, not the production default
8081) and frontend (`npm run dev`, Vite on `:5173`, which already proxies
`/yoyojudge/*` to the backend per `vite.config.ts`). Ensures `./dist`
exists first (a placeholder file if missing — `go:embed` needs *something*
there even though dev mode never serves it, Vite serves the frontend
instead) and runs `npm install` if `frontend/node_modules` is missing.

- **Port 5000, not 8081**: 8081 (and other ports — `8090` was also tried)
  fell inside a Windows-reserved TCP exclusion range on the dev machine
  this was built on (`8056–8155`, likely from Hyper-V/WSL2) — confirmed
  with `netsh interface ipv4 show excludedportrange protocol=tcp`, which
  is worth checking first on a new machine before assuming 5000 is safe
  there too. The failure mode is `listen tcp :8081: bind: An attempt was
  made to access a socket in a way forbidden by its access permissions` —
  a Windows-level block, not a Go or app bug.
- **`FRONTEND_URL` is set explicitly** to `http://localhost:5173` for the
  backend process. This was a real bug hit while testing: without it,
  `server/oauth.go`'s `frontendBaseURL()` infers the post-Google-login
  redirect from the backend's own request host (`localhost:$Port`), so the
  browser lands on the backend's embedded `./dist` instead of the actual
  dev frontend at `:5173`. That embedded `dist/` is whatever the last
  `./build.ps1` produced — typically a **production**-configured build
  whose baked-in API base URL points at the real deployed backend, not the
  local one — so a session token minted by the local backend looks invalid
  there, and the login flow hangs forever on `AuthCallbackView.vue`'s
  "Finishing sign-in…" screen (confirmed by finding
  `rizkiyoist.duckdns.org:8081/yoyojudge/api` baked into
  `dist/assets/*.js` while debugging). Always test at `http://localhost:5173`,
  not `http://localhost:5000/yoyojudge`, for this reason.
- Google login still needs real credentials (via `env.json` or env vars,
  see above) to work at all locally; the script prints a reminder pointing
  at the "Demo / email login" section on the login page as a fallback.

### Invite a judge by email who hasn't signed in yet — new, 2026-09-02

Previously, inviting a judge required them to already exist as a `User` row
— in practice, only the seeded demo accounts or someone who'd already
logged in at least once via Google. Now the head judge can invite by email
directly even if nobody with that email has ever signed in, and the invite
is already waiting for them the first time they do.

- `server/store.go`'s `findOrCreateUserByEmail(email)` — looks up a user by
  email (case-insensitive), or creates a **placeholder** `DBUser` with only
  `Email` set (blank first/last name, no Google ID). `usersByIDs(ids
  []string)` — new batch lookup, resolves a list of user IDs to `User`s in
  one query.
- `server/handlers.go`'s `handleInviteJudge` now accepts an optional
  `email` field alongside the existing `userId` — if `userId` is empty and
  `email` is set, it resolves (or creates) the user via
  `findOrCreateUserByEmail` first. `handleSearchUsers` (mounted at the
  existing public `/users/search` route) now also accepts an `ids`
  (comma-separated) query param as an alternative to `q`, for the batch
  lookup above — kept on the same route rather than adding a new one, since
  it's the same trust level as the free-text search already there.
- `server/store.go`'s `findOrCreateUserByGoogle` — **bug fixed alongside
  this**: it already matched an existing user by email and linked the
  Google ID, but never filled in the name, so a placeholder invited by
  email would stay nameless forever even after actually signing in with
  Google. Now backfills `first_name`/`last_name` from the Google profile
  whenever both are currently blank, and never overwrites a name that's
  already set (verified with a throwaway Go test exercising both cases,
  deleted after passing — see this section's history for the exact
  scenario if it needs re-verifying).
- Frontend: `ScoringApi.inviteJudge`'s identity parameter changed from a
  bare `userId: string` to `{ userId: string } | { email: string }` — a
  breaking signature change to `client.ts`, `http.ts`, and `mock.ts`
  together, plus the one call site (`JudgeManagementView.vue`). New
  `ScoringApi.getUsers(ids: string[])` (implemented in both `http.ts` and
  `mock.ts`) replaced a pre-existing hack in **four** views
  (`JudgeManagementView.vue`, `ScoreOverrideView.vue`, `ResultsView.vue`,
  `ContestListView.vue`) that all called `api.searchUsers('example.com')`
  to bulk-resolve every judge's name for display — that only ever worked
  because every seeded demo user happens to share that email domain, and
  would have silently broken (showing raw user IDs) for anyone invited by
  a real email through this new feature. All four now resolve exactly the
  user IDs they actually reference (assignments + contest owner) via
  `getUsers`. Each view's name-formatting helper (`userLabel`/`judgeName`/
  `judgeLabel`) also now falls back to `"<email> (not signed in yet)"`
  instead of a blank string when first/last name are both empty.
- `LoginView.vue`'s own `searchUsers('example.com')` call (for the "seeded
  demo users" quick-login list) was deliberately left alone — that's
  actually listing demo accounts specifically, not resolving arbitrary
  known IDs, so the same fix doesn't apply there.
- Verified end-to-end against the compiled binary: invited a fabricated
  email, confirmed a blank-named `DBUser` was created; invited the same
  email again and confirmed the *same* user ID was reused (not a
  duplicate); resolved that ID via the new `ids=` search param and got the
  right email back with blank names.

### Contest visibility, real score-write authorization, head-judge stage locking — new, 2026-09-02

**Superseded (later the same day) — see "Owner vs. head judge, whole-contest
lock..." below.** The per-division-stage lock described in this section was
replaced with a single whole-contest lock once the actual intended
semantics came out in conversation ("a locked contest can't be changed:
division, score, player, judge list" — much broader than "lock one stage's
scores"). The visibility change and the score-write-authorization gap fix
below are still accurate and unchanged; only the lock model changed.

Previously `listContestsForUser` filtered the contest list to just the ones
a user owned or was invited to, but critically **no score-submission
endpoint checked who was calling it at all** — any authenticated user could
POST a clicker/eval/deduction score for any player/slot in any division,
regardless of assignment. That gap was purely accidental (the frontend's
`isOwner`/`myAssignments` checks were the only thing shaping the UI, never
enforced server-side) but became a real problem once every judge can browse
every contest. Fixed both, plus added the ability for a head judge to freeze
a stage's scores.

- **Visibility**: `server/store.go`'s `listAllContests()` replaces
  `listContestsForUser` — every signed-in user now sees every contest,
  regardless of ownership or invitation. `ScoringApi.listContests()` lost
  its now-meaningless `userId` parameter across `client.ts`, `http.ts`,
  `mock.ts`, `stores/contests.ts`, and `ContestListView.vue`.
- **Score-write authorization** (`server/store.go`): new
  `authorizeSlotScoreWrite(divisionID, stage, role, slot, userID)` (for
  clicker/eval scores — one specific slot) and
  `authorizeDeductionsWrite(divisionID, stage, userID)` (for major
  deductions — any clicker judge for that division+stage, matching
  `ScoreEntryView.vue`'s UI, where any of the 6 clicker judges can enter
  them). Both let the **contest owner** through unconditionally — that's
  what makes `ScoreOverrideView.vue`'s "act as any judge" flow keep working,
  and is also the exemption that lets the owner edit through a lock (see
  below). `handleSubmitClickerScore`/`handleSubmitDeductions`/
  `handleSubmitEvalScore` in `server/handlers.go` now call these before
  writing, returning `403` (wrong/no assignment) or `423 Locked` (see
  below) instead of silently succeeding.
- **Head-judge stage locking (superseded, see below)**: `DBDivision` gained a `LockedStages` JSON
  column (`server/db_models.go`, auto-migrated — no manual migration step
  needed, GORM's `AutoMigrate` adds the column to existing `yoyojudge.db`
  files on next startup); `Division.lockedStages: ScoringStage[]` mirrors
  it on the frontend (`types.ts`). New endpoint `PATCH
  /divisions/{divisionId}/lock` (`handleSetDivisionLock`, owner-only —
  `403` otherwise) toggles one stage's lock via
  `setDivisionStageLock`/`isStageLocked` in `store.go`. Locking blocks
  every judge but the owner from submitting scores for that stage (`423`);
  the owner can still edit/override through it, and can unlock at any time
  — matches "lock it so *other* judges can't change it," not "freeze it for
  everyone including me."
  - Frontend: `ScoringApi.setDivisionStageLock(divisionId, stage, locked)`
    added to `client.ts`/`http.ts`/`mock.ts` (mock also mirrors the
    owner-bypass lock check in its `submitClickerScore`/`submitDeductions`/
    `submitEvalScore`, via a new `assertScoreWritable` — not the finer
    per-slot assignment check, since the mock has no real per-user security
    boundary to enforce in the first place).
  - `ScoreEntryView.vue` (the page ordinary judges use): shows a
    "Lock scores"/"Unlock scores" toggle button and a 🔒 banner to the
    owner, and to everyone else a fieldset-disabled (all inputs
    `disabled`) view with a "locked by the head judge" message once a
    stage is locked — a client-side courtesy to avoid a round trip that's
    going to 423 anyway; the real enforcement is server-side.
- Verified end-to-end against the compiled binary: an uninvited user could
  list a contest they weren't part of (visibility) but got `403` submitting
  a score for it; after being invited to clicker slot 1, submitting to slot
  1 succeeded (`204`) and slot 2 was rejected (`403`); after the owner
  locked the stage, the assigned judge's submit got `423` while the owner's
  own submit still succeeded (`204`); a non-owner's attempt to toggle the
  lock got `403`; after the owner unlocked it, the judge could submit
  again; a clicker judge (not the exact slot being tested) successfully
  submitted deductions, confirming the "any clicker judge" rule.
- **Known remaining gap at the time, since partly closed** (see next
  section): `handleAddDivision`, `handleUpdateDivisionStages`,
  `handleInviteJudge`, and `handleRemoveJudgeAssignment` had no server-side
  owner check at all, same as before — only client-side `isOwner` gating.

### Owner-only invite/remove judge, plus a cross-contest deletion fix — new, 2026-09-02

**Superseded (later the same day)**: "owner-only" here became "owner **or**
head judge" once head judge became a real, separately-transferable role
(see "Owner vs. head judge..." below) — the cross-contest deletion fix
itself is unaffected and still stands as originally described.

Closed the invite/remove half of the gap flagged just above (division
management — `handleAddDivision`/`handleUpdateDivisionStages` — is still
open, see "What's not implemented yet").

- `handleInviteJudge` and `handleRemoveJudgeAssignment`
  (`server/handlers.go`) now call `isContestOwner(contestID,
  currentUser(r).ID)` and reject with `403` for anyone but the contest's
  owner — same "head judge" concept used everywhere else in this app
  (there's no separate head-judge role distinct from the contest owner).
- **Real bug found and fixed while adding the remove-judge check, not just
  a missing-auth gap**: `DELETE /contests/{contestId}/judges/{assignmentId}`
  only ever validated that the caller owned *the contest ID named in the
  URL* — it never checked that the assignment being deleted actually
  belonged to that contest. Since a URL's `contestId` is attacker-supplied,
  the owner of *any* contest could have deleted a judge assignment from
  *any other* contest by pairing their own (owned, so it passes the check)
  contest ID with a stolen assignment ID from elsewhere. Fixed by scoping
  the delete itself to `WHERE id = ? AND contest_id = ?`
  (`removeJudgeAssignment` in `store.go` now takes both and reports
  whether a row actually matched, `404` if not) rather than trusting the
  two URL segments to agree.
- Verified end-to-end against the compiled binary with two contests owned
  by two different users: the non-owner's invite attempt on the other's
  contest got `403`; the real owner's invite succeeded (`201`); the
  cross-contest deletion attack (own contest ID + the other contest's real
  assignment ID) got `404` and the assignment was confirmed still present
  afterward; the real owner's own deletion, same assignment, correct
  contest ID, succeeded (`204`).

### Owner vs. head judge, whole-contest lock, division/player deletion, eval-score scale fix — new, 2026-09-02

A large role/permission rework, done as three rounds of clarifying
questions (worth reading the actual conversation if this section is ever
confusing — the model went through a couple of real course-corrections):
first "owner" and "head judge" turned out to be two separate, transferable
concepts, not one; then "lock a stage's scores" turned out to mean "lock
the *entire contest*, everything, for *everyone including the head judge*
until unlocked" — much broader than the per-stage lock built earlier the
same day (see the two superseded sections above). Also closed out division
deletion (didn't exist before at all), player removal (ditto), and a
real scoring-input bug caught by cross-checking against the actual xlsx.

**Owner vs. head judge — now genuinely separate roles:**
- `DBContest` gained `HeadJudgeUserID` (`server/db_models.go`,
  auto-migrated) — defaults to the owner at contest creation
  (`createContest`), but is transferable independently. `OwnerUserID`
  never changes once set; `Contest.headJudgeUserId`/`ownerUserId` both
  surface on the frontend (`types.ts`). Verified (see below) that
  `AutoMigrate` adds this column cleanly to a database created before it
  existed, and that `dbContestToContest`'s fallback (empty
  `HeadJudgeUserID` → treat as the owner) keeps pre-existing contests
  working exactly as before.
- New endpoint `PATCH /contests/{contestId}/head-judge` (body
  `{"userId"}`, `handleTransferHeadJudge`/`transferHeadJudge` in
  `store.go`) hands head-judge privileges to any user who is either the
  contest's owner or already holds a judge assignment somewhere in the
  contest — `400` for anyone else, so a stranger can't be handed control
  of a contest they have nothing to do with. Callable by the owner *or*
  the current head judge (so head-judgeship can be passed along a chain,
  not just handed back by the owner).
- **The exact permission split**, matching the role table worked out in
  conversation:
  - **Owner or head judge** (`isOwnerOrHeadJudge`/`authorizeContestWrite`
    in `store.go`): invite/remove judges, add/remove divisions, add/remove
    players, transfer head-judge status.
  - **Head judge only** (`isHeadJudge`): lock/unlock the contest, and
    override any judge's score (submitting a clicker/eval score for a slot
    you're not assigned to, or a deduction without being assigned at all —
    this is what makes `ScoreOverrideView.vue`'s "act as any judge" work).
    The owner does **not** get this once head-judge status has been
    transferred away — a deliberate consequence of the role table as
    written, called out explicitly during the conversation rather than
    assumed.
  - **Normal judge** (anyone with a `JudgeAssignment`): submit a
    clicker/eval score for their own assigned slot; submit major
    deductions for that division+stage — **broadened to any assigned
    judge, clicker or evaluator**, not just clickers (see "eval-score
    scale" note below for why: one player has one shared deduction value
    regardless of how many judges, of either role, are watching them).
    This also meant fixing four frontend views
    (`ScoreEntryView`/`ScoreOverrideView`) to actually expose deduction
    inputs to eval-only judges, not just clicker judges, since the old UI
    only ever showed deductions inside the clicker judge's table.
- `JudgeManagementView.vue` gained a "Head judge" card: shows who currently
  holds it (bolded/badged distinctly from the owner in the assignment
  tables — `isOwnerUserId`/`isHeadJudgeUserId`), and a dropdown + button to
  transfer it to the owner or any currently-invited judge.

**Whole-contest lock — replaces the per-division-stage lock built earlier
today:**
- `DBContest.Locked bool` (auto-migrated, defaults `false`). Freezes
  **everything** in the contest — every division, stage, player, judge,
  and score — against writes from **anyone, including the head judge
  themself**; the lock/unlock toggle is the one call that's deliberately
  exempt (`handleSetContestLock` doesn't route through
  `authorizeContestWrite`'s own-lock check), or a locked contest could
  never be unlocked. Confirmed explicitly in conversation: this is *not*
  "the head judge can still edit through the lock" (that was the original,
  smaller-scoped design) — locking really does mean nobody touches
  anything until it's unlocked again.
- New endpoint `PATCH /contests/{contestId}/lock` (body `{"locked"}`,
  `handleSetContestLock`) — head-judge-only, `403` otherwise.
- `store.go`'s `authorizeSlotScoreWrite`/`authorizeDeductionsWrite`/
  `authorizeContestWrite` all check `isContestLocked` **first**, before any
  role check, so a lock genuinely blocks the head judge too — the earlier
  per-stage design's "head judge is exempt" branch is gone.
- Frontend: `Contest.locked` (`types.ts`), `ScoringApi.setContestLocked`
  replaces the removed `setDivisionStageLock` throughout
  `client.ts`/`http.ts`/`mock.ts` (mock's `assertContestWritable`/
  `assertSlotScoreWritable`/`assertDeductionsWritable` mirror the same
  "locked blocks everyone, no head-judge exemption" rule). `🔒 This contest
  is locked` banners (disabling the relevant inputs/buttons) now appear on
  `ScoreEntryView`, `ScoreOverrideView`, `ContestEditView`, and
  `PlayerRosterView`; the actual lock/unlock control lives on
  `JudgeManagementView.vue` (head-judge-only).

**Division and player deletion — didn't exist before at all:**
- `DELETE /contests/{contestId}/divisions/{divisionId}`
  (`handleDeleteDivision`/`deleteDivision`) — owner or head judge, blocked
  by the contest lock like everything else, and refuses (`409`) if the
  division still has any players, with a message telling the caller to
  remove them first. Also cleans up the division's judge assignments and
  any raw-score rows so nothing is left orphaned.
- `DELETE /divisions/{divisionId}/players/{playerId}`
  (`handleRemovePlayer`/`removePlayer`) — same authorization, deletes the
  player and their raw scores for both stages.
- Frontend: `ContestEditView.vue` now shows a player count per division and
  a disabled-with-tooltip "Delete" button when that count is `>0`;
  `PlayerRosterView.vue` gained a "Remove" button per player (with a
  confirm dialog warning their scores go too).

**Eval-score input scale bug — 0-10 whole numbers, not 0-5/0-10 in 0.5
steps:** reported directly ("when adding pev score it should be in
increment of 1, from 1 to 10" → clarified to 0-10 after some back-and-forth
about whether this touched the FINAL stage's calc engine at all). It did
touch it, sort of — but only a UI/metadata bug, not the arithmetic:
- **What was actually wrong**: `library/calc/stage.go`'s `FinalCategories()`
  had `MaxValue: 5` (PRELIM's already used the correct `MaxValue: 10`).
  `MaxValue` only ever feeds the frontend input widget's `max` attribute —
  `rules.go`'s `Calculate()` never reads it — so this was purely "the UI
  wrongly capped FINAL category inputs at 5 and let 0.5 steps through,"
  not a bug in how scores were computed from whatever was typed in.
- **Independently verified against the real workbook**, not just taken on
  the user's word: opened `IYYF-SCORE-CALC-FINAL-2017.xlsx` with excelize,
  wrote the exact input values from `rules_test.go`'s hand-verified FINAL
  test case directly into the real sheet cells, and used excelize's
  `CalcCellValue` to force genuine formula recalculation (not just read
  cached values) — every output (T.Ex, T.Ev/P.Ev totals, E.Total, final
  score, place) matched the Go engine exactly. Separately, the
  `FINAL-SCORE` sheet's own header (`T.Ev total (/20)`, 4 categories) only
  arithmetically works out if the raw per-category input goes up to 10 (10
  ÷ 2 × 4 = 20); the old 0-5 assumption would cap the group at 10, not 20 —
  independent confirmation from the workbook's own structure, not just the
  formula chain.
- **Fix**: `FinalCategories()`'s `MaxValue` changed `5` → `10` (all 8
  categories) in both `library/calc/stage.go` and its frontend TS mirror
  `frontend/src/lib/scoring.ts` — the `Halve: true` div-by-2 step is
  untouched and correct as-is now that the input feeding it is the right
  scale. `ScoreEntryView.vue`/`ScoreOverrideView.vue`'s eval score inputs
  changed `step="0.5"` → `step="1"` (min was already `0`, max was already
  bound to `cat.maxValue`, so fixing that one number fixed both stages'
  widgets at once). No calc-engine or test-assertion changes were needed —
  `rules_test.go`/`scoring.test.ts`'s numeric assertions were never wrong,
  only what "scale" the numbers going in were supposed to represent.
- **Major deduction entry vs. the source workbook, for context**: in the
  original `.xls`, Stop/Discard/Cut live on the `Raw-TEx` sheet — one
  shared cell per player, same sheet as the clicker judges' input, not the
  eval sheet — so historically it was implicitly "a clicker judge's job."
  The app already matched the "one shared value per player" part; letting
  *any* assigned judge (not just clickers) submit it, per this round's
  role-table work above, is a deliberate product decision diverging from
  that old convention, not a correctness fix.
- Verified: `go test ./library/calc/...` and the frontend `vitest` suite
  both still pass unchanged (confirming no calc-engine behavior actually
  changed); a throwaway Go test simulating a pre-migration `db_contests`
  table (missing `HeadJudgeUserID`/`Locked`) confirmed `AutoMigrate` adds
  both columns cleanly against a real SQLite file, deleted after passing.
  Full end-to-end role/lock/deletion flow re-verified against the compiled
  binary: default head judge = owner; transfer moved lock/override
  privilege but left `ownerUserId` untouched; the *old* owner got `403`
  trying to lock after transfer while the *new* head judge succeeded;
  locking blocked the head judge's own score submission (`423`) until they
  unlocked; division deletion was blocked (`409`) with a player present and
  succeeded (`204`) after removing them; transferring head-judge status to
  an uninvited stranger was rejected (`400`).

## What's not implemented yet

- `handler/input.go`'s structs (`SetUp`, `Player`, `RawTex`, `RawPev`) aren't
  mapped to `calc.PlayerInput`/`calc.Contest` or connected to `server/`,
  `request/`, or any controller/usecase layer. These predate `server/` and
  look increasingly redundant with `server/db_models.go` + `server/types.go`
  now that real persistence exists — worth revisiting whether they're still
  needed at all rather than mapping them.
- `OptionalSetUpPrelim` in `handler/input.go` is still an empty stub.
- The unused `users`/`user_socials` MySQL/GORM models (`domain/model/`),
  the `controller/auth` Google-login stub (superseded by `server/oauth.go`),
  and `config/`'s MySQL connector are all now dead code relative to the
  SQLite path above — candidates for deletion once confirmed nothing else
  depends on them.
- The old excelize-based writer/reader path (`library/writer`,
  `library/reader`, `library/calc/const.go`) is superseded but still in the
  tree — worth deleting once nothing needs it, along with the working-copy
  `IYYF-SCORE-CALC-FINAL-2017.xlsx`/`.xlsxbak` files at the repo root.
- Only `library/calc` (Go) and `frontend/src/lib/scoring.ts` (TS) have
  tests; `server/` and the rest of the repo don't.

## Suggested next steps

1. **Done (2026-09-02):** ~~decide on real persistence~~ — SQLite shipped
   (see "SQLite persistence + Google OAuth login" above), superseding the
   MySQL/`sql-migrate` plan this item used to describe.
2. **Done (2026-09-02):** ~~flesh out real authentication~~ — Google OAuth
   shipped alongside SQLite persistence. Still only Google as a provider,
   and no password-based account option.
3. Decide the fate of `handler/input.go`'s structs, the `domain/model/`
   `users`/`user_socials` GORM models, `controller/auth`, and `config/`'s
   MySQL connector — all look like dead code now that `server/` has its own
   types/models/auth backed by SQLite. Likely just deletable, but confirm
   nothing still imports them first.
4. Once the native calc engine + backend are trusted end-to-end, remove the
   superseded `library/writer`/`library/reader`/`library/calc/const.go` code
   and the root-level `.xlsx`/`.xlsxbak` working copies.
5. Add backend tests (`server/` currently has none) — now more valuable
   than before since real persistence/auth logic exists to break.
