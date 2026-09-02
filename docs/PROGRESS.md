# Progress Summary

_Last updated: 2026-09-02 (added "resuming on a new machine" notes; no code changes this session)_

## Resuming on a new machine

Repo state as of this note: `main`, clean working tree, HEAD = `b5f85c8`
("Single-binary deploy support: embedded frontend, BASE_PATH, TLS, port
config") — everything described below is already committed and pushed, so
`git clone`/`git pull` is all that's needed for the code itself. What
**won't** come with a fresh clone (all gitignored):

- `env.json` (copy from `env.json.example`) and `dbconfig.yml` (copy from
  `dbconfig.yml.example`) — MySQL connection settings. Currently unused at
  runtime (`main.go` no longer calls `config.InitSQL`, see "Real production
  deployment" below), but `sql-migrate` still reads `dbconfig.yml` directly
  if you touch `migration/`.
- `bin/`, `dist/` — build output. Run `./build.ps1` (repo root, PowerShell)
  to regenerate both the Windows/Linux backend binaries and the embedded
  frontend `dist/`. **`go build .`/`go run .` will fail on a fresh clone**
  until the frontend step of `build.ps1` has populated `dist/` at least
  once, since `static.go` embeds it at compile time.
- `frontend/node_modules/` — run `npm install` in `frontend/` first.
- Production `cert.pem`/`key.pem` live only on the deploy server
  (`103.134.154.210`, `rizkiyoist.duckdns.org`), not in this repo or
  expected on a dev machine — see the acme.sh steps under "Real production
  deployment" below if that ever needs to be redone from scratch.

No other local-only state exists — the app has no real database yet
(everything is in-memory in `server/store.go`, reset on restart).

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

Not yet done: no real persistence (everything resets on backend restart);
no real authentication; `handler/input.go`'s structs and the `users`/
`user_socials` GORM models aren't connected to any of this yet; the
reload-into-404-on-a-deep-route issue is inherent to serving the frontend
as static files from nginx (no SPA-fallback rule there) and is unfixed —
would need either an nginx `try_files` rule or switching the frontend to
hash-based routing.

## What's not implemented yet

- No persistence for contests/players/scores — `server/store.go` is
  in-memory only; everything is lost on backend restart. Nothing maps to
  the `users`/`user_socials` GORM models or MySQL yet.
- `handler/input.go`'s structs (`SetUp`, `Player`, `RawTex`, `RawPev`) aren't
  mapped to `calc.PlayerInput`/`calc.Contest` or connected to `server/`,
  `request/`, or any controller/usecase layer.
- `OptionalSetUpPrelim` in `handler/input.go` is still an empty stub.
- Real authentication — both `controller/auth`'s Google login stub and
  `server/auth.go`'s bearer-token-is-the-userid scheme are stand-ins, not
  real auth.
- The old excelize-based writer/reader path (`library/writer`,
  `library/reader`, `library/calc/const.go`) is superseded but still in the
  tree — worth deleting once nothing needs it, along with the working-copy
  `IYYF-SCORE-CALC-FINAL-2017.xlsx`/`.xlsxbak` files at the repo root.
- Only `library/calc` (Go) and `frontend/src/lib/scoring.ts` (TS) have
  tests; `server/` and the rest of the repo don't.

## Suggested next steps

1. **TODO, deliberately deferred:** decide on real persistence. Candidate:
   **SQLite** instead of MySQL — `sql-migrate` (already used for
   `migration/`) supports a `sqlite3` dialect natively, so the existing
   migration tooling carries over; the two current migrations (`users`,
   `user_socials`) would need their MySQL-specific syntax (`ENUM`,
   `AUTO_INCREMENT`, `ON UPDATE CURRENT_TIMESTAMP`) rewritten for SQLite,
   and new migrations added for contests/divisions/judge
   assignments/players/raw scores (none of which exist as tables yet
   regardless of which DB is chosen). `config/sql.go` would need a
   `gorm.io/driver/sqlite` code path alongside the MySQL one. The generic
   `Repository[T]` in `domain/service/generic.go` is already there to build
   on. Holding off on this until after getting real users testing the
   in-memory-backed site, per user's call — revisit once that feedback is
   in. Replacing `server/store.go`'s in-memory maps with real queries
   shouldn't require changing the HTTP handler layer's shape.
2. Flesh out real authentication (replacing both the Google login stub and
   `server/auth.go`'s trivial bearer scheme) — sessions/JWTs plus actual
   credential verification.
3. Map `handler/input.go`'s structs onto `server/`'s types where they fit,
   or retire whichever turns out redundant.
4. Once the native calc engine + backend are trusted end-to-end, remove the
   superseded `library/writer`/`library/reader`/`library/calc/const.go` code
   and the root-level `.xlsx`/`.xlsxbak` working copies.
