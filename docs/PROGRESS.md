# Progress Summary

_Last updated: 2026-08-16 (real backend added)_

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
- `build.ps1` (repo root) — builds both the Go backend (`bin/yoyo-judge.exe`)
  and the frontend (`frontend/dist`) in one step; `-Only backend` /
  `-Only frontend` to build just one.

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
  judges with real Indonesian names, e.g. `agus.a1@example.com` .. 
  `lestari.b6@example.com`; contest "Indonesia National Yoyo
  Championships", division "3A", both stages, 4 players, FINAL-stage
  scores pre-filled) — the same login flow already verified against the
  mock works unchanged. `toPlayerInput` converts stored raw scores into
  `calc.PlayerInput`.
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

Not yet done: no real persistence (everything resets on backend restart);
no real authentication; `handler/input.go`'s structs and the `users`/
`user_socials` GORM models aren't connected to any of this yet.

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

1. Decide on real persistence: contests/divisions/judge assignments/players/
   raw scores stored in MySQL (the generic `Repository[T]` in
   `domain/service/generic.go` is already there for this), replacing
   `server/store.go`'s in-memory maps — the HTTP handler layer shouldn't
   need to change.
2. Flesh out real authentication (replacing both the Google login stub and
   `server/auth.go`'s trivial bearer scheme) — sessions/JWTs plus actual
   credential verification.
3. Map `handler/input.go`'s structs onto `server/`'s types where they fit,
   or retire whichever turns out redundant.
4. Once the native calc engine + backend are trusted end-to-end, remove the
   superseded `library/writer`/`library/reader`/`library/calc/const.go` code
   and the root-level `.xlsx`/`.xlsxbak` working copies.
