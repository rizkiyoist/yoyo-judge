# Progress Summary

_Last updated: 2026-08-16_

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

## What's not implemented yet

- No HTTP handlers/routes for any of: setting up a contest, entering
  players, entering judge scores, or reading back computed results.
- No persistence for contests/players/scores — `library/calc` is pure
  in-memory calculation; nothing maps its types to/from GORM models or the
  database yet.
- `handler/input.go`'s structs (`SetUp`, `Player`, `RawTex`, `RawPev`) aren't
  yet mapped to `calc.PlayerInput`/`calc.Contest`, and aren't connected to
  `request/` or any controller/usecase layer.
- `OptionalSetUpPrelim` in `handler/input.go` is still an empty stub.
- Google login (`controller/auth`) is a stub with a `TODO`.
- The old excelize-based writer/reader path (`library/writer`,
  `library/reader`, `library/calc/const.go`) is superseded but still in the
  tree — worth deleting once nothing needs it, along with the working-copy
  `IYYF-SCORE-CALC-FINAL-2017.xlsx`/`.xlsxbak` files at the repo root.
- Only the new `library/calc` package has tests; nothing else in the repo
  does.

## Suggested next steps

1. Map `handler/input.go` structs (or a request-layer equivalent) → 
   `calc.PlayerInput`/`calc.Contest`, so incoming judge/setup data has a
   clear path into the scoring engine.
2. Add HTTP handlers/routes in `router.go` for the setup → player → scoring
   → results flow, replacing the commented-out placeholder.
3. Decide on persistence: do contests/players/raw scores get stored in MySQL
   (via the existing generic `Repository[T]`), with `calc.Calculate()` run
   on demand or on write?
4. Once the native engine is trusted end-to-end, remove the superseded
   `library/writer`/`library/reader`/`library/calc/const.go` code and the
   root-level `.xlsx`/`.xlsxbak` working copies.
5. Flesh out `controller/auth` (Google login) if user accounts are needed
   before scoring features.
