# Frontend layout & behavior spec

Source of truth for the redesigned UI. When the app under `frontend/src/` disagrees with this doc, treat it as a bug in one of them and reconcile before shipping.

## Global chrome

- **Top bar** on every page: `yoyo-judge` wordmark (no icon, plain text) on the left, then right-aligned: view-mode button, theme button, user name, "Log out". Sticky to top.
- **Theme toggle**: 🌙 in light mode, ☀️ in dark mode. Persists in `localStorage`. Palette matches `frontend/src/style.css` (accent `#7c3aed` light / `#a78bfa` dark).
- **View-mode toggle**: cycles Auto → Mobile → Desktop (icons: 🪄 / 📱 / 🖥️). Default Auto follows the `(max-width: 720px)` media query. `data-mobile-view` attribute on `<html>` bumps root font-size from 15px to 19px, narrows the content column to 640px, enlarges tap targets to ≥44px.
- **No emojis in status labels** (badges/pills). Say "Locked" and "Hidden", not "🔒 Locked" / "🙈 Hidden".

## Main page — contest list

- H1 "All Contests" plus a one-line description.
- **Create contest form** at the top: dashed-border card labeled "CREATE A CONTEST" with `Contest name` text input, `Year` number input, primary `Create` button.
- **Contest rows** are collapsible cards (accordion). Header shows:
  - Contest name + `badge-year` (accent-tinted) + status pills (`Locked` / `Hidden`, plain text)
  - One-line meta below (e.g. "3 divisions · Head judge: Rizki")
  - Right-side actions: `Edit`, `Download Results`, `Lock` / `Unlock`, primary `Open →` (navigates to the workspace)
- **Expanded body** shows one row per division-per-stage — Prelim and Final are separate rows, not combined:
  - Left: `<div-name>` heading + pill (`Prelim` blue / `Final` accent) + mini top-3 table with columns `Player` (left) and `Final Score` (right-aligned). Max width 320px so score stays close to name on wide screens.
  - Right: `Result Detail` button (disabled + no table when there are no results yet — "No results yet." italic instead).

## Contest workspace — shared shell

Every contest sub-page shares this shell (in this order):
1. Top bar (global).
2. `← Back to contests` crumb link (inside the standard `.wrap`, no other content on this line).
3. **Tab bar** — sticky, sits directly beneath the top bar via `top: var(--topbar-h)`. Tabs, in order: **Divisions · Judges · Players · Input Score · Result Detail**. Horizontally scrollable on mobile.
4. Page content in the standard `.wrap` (`max-width: 1080px`, 32px side padding).

Sticky offset: measure the top bar with JS on load, resize, and view-mode change; write `--topbar-h` on `:root`. Do not hardcode `top: 63px` — it breaks when mobile-view scales the font.

## Workspace pages

Each workspace page **must mirror the layout of its production Vue view** — same headings, cards, forms, and tables. The only permitted addition is the tab bar. Do not invent buttons/features not in the current production code.

- **Divisions** — matches `ContestEditView.vue`.
  - Heading: `<contest> - Divisions`, right-aligned `Go to Judges →` link.
  - "Add a division" card: `Division name` field, `Prelim` / `Final` checkboxes, primary `Add division` button.
  - "Divisions" card: table with columns `Name | Prelim | Final | Players | (Delete)`.
- **Judges** — matches `JudgeManagementView.vue`.
  - Heading with per-division picker buttons on the right (primary = selected).
  - "Invite a judge" card: `Role` select (Clicker Judge (TEx) / Evaluation Judge (PEv)), `Stage` select, "Will be assigned slot N." hint, search input.
  - "Current assignments" card: per-stage picker buttons, then two side-by-side sub-tables `Clicker Judges (TEx)` and `Evaluation Judges (PEv)`; head judge row shows a `Head Judge` badge next to the name. Then a `Major Deduction Judge` sub-section with its own remove/assign controls.
  - "Head judge" card: transfer dropdown + `Make head judge` button.
- **Players** — matches `PlayerRosterView.vue` (per-division). Division picker on the right of the heading. "Add a player" card (`#N` badge + Name field + primary `Add player`). "Roster" card with `# | Name | (Remove)` table.
- **Input Score** — matches `ScoreEntryView.vue`.
  - Division+stage picker buttons on the right of the heading.
  - `Head judge: override any judge's score` button.
  - `save-status` pill (idle / pending / saved variants).
  - `Clicker judge - slot N` card with `# | Player | + | -`.
  - `Evaluation judge - slot N` card with `# | Player | <5 categories, max shown in header>`.
  - `Major deduction judge` card with `# | Player | Stop | Discard | Cut`.
- **Result Detail** — matches `ResultsView.vue`.
  - Division+stage picker on the right of the heading.
  - Wide results table with grouped column-header tints: `col-tex` (T.Ex), `col-tev` / `col-pev` (evaluation categories), `col-total` (E.Total, Final Score), `col-deduction` (Stop / Discard / Cut). Expandable per-row `Details` button.
  - This page's `.wrap` and `.tabs-inner` expand to `min(1440px, calc(100% - 24px))` so on wide screens the table fits without a horizontal scrollbar; narrow screens still scroll via `.results-scroll { overflow-x: auto }`.

## Table conventions

- Action buttons in the last column (`Delete`, `Remove`, `Details`) are **right-aligned** — the containing `<td>` gets `text-align: right` so buttons hug the right edge instead of drifting to the middle of a wide column.
- Header cells: uppercase, 0.05em letter-spacing, muted color. Body cells: 10px 12px padding.

## Score input behavior (`.score-input-wrap`)

Every numeric input on the Input Score page is a `.score-input-wrap` containing:
- `<input class="score-input" data-group data-row data-col data-min data-max data-step>`
- `.score-steppers` with two `.score-step` buttons (▲ up, ▼ down; `data-dir="up"` / `"down"`).

**Spreadsheet-style keyboard navigation** (matches `frontend/src/components/ScoreNumberInput.vue`):
- `ArrowUp` → focus same column, previous row.
- `ArrowDown` / `Enter` → same column, next row.
- `ArrowLeft` → same row, previous column.
- `ArrowRight` → same row, next column.
- `focus` selects the entire value so typing replaces rather than appends.
- Navigation is scoped to the current `data-group` — arrow keys never cross from the clicker table into the eval table.

**Stepper increments**:
- Default `step = 1` (integer). All production `ScoreNumberInput` usages in `ScoreEntryView` accept this default — do NOT introduce fractional steps in the redesign.
- Clamp to `data-min` / `data-max` after every step. Eval categories are `min=0, max=cat.maxValue`. Clicker `+/-` and major-deduction fields are `min=0`, no max.
- Clicking a stepper keeps focus on the input so keyboard entry can continue immediately.

## Real data — do not invent

Everything below is verified from source. Do not substitute made-up values.

### Categories (`frontend/src/lib/scoring.ts`)

Final (8 categories, all `maxValue: 10`, `halve: true`):

| Code | Group | Descriptive label (Results view only) |
| ---- | ----- | ------------------------------------- |
| EXE  | T.Ev  | Execution        |
| CTL  | T.Ev  | Control          |
| TDV  | T.Ev  | Trick Diversity  |
| SEM  | T.Ev  | Space Use/Emp.   |
| MU1  | P.Ev  | Choreography     |
| MU2  | P.Ev  | Construction     |
| BDY  | P.Ev  | Body Control     |
| SHW  | P.Ev  | Showmanship      |

Prelim (4 categories, all `maxValue: 10`, `halve: false`):

| Code | Group | Descriptive label (Results view only) |
| ---- | ----- | ------------------------------------- |
| EXE  | T.Ev  | Cleanliness  |
| CTL  | T.Ev  | Execution    |
| MU1  | P.Ev  | Music Use    |
| BDY  | P.Ev  | Body Control |

**Score Entry** column headers use the raw code with the `(max N)` suffix: `EXE (max 10)`. **Result Detail** column headers use the descriptive label from the tables above.

### Deductions (`frontend/src/lib/scoring.ts`)

- Final: `Stop (1)`, `Discard (3)`, `Cut (5)`.
- Prelim: `Stop (1)`, `Discard (3)`, `Detach (5)`.

The Results view computes the third-deduction column label with `stage === 'prelim' ? 'Detach' : 'Cut'`.

### Input defaults

- `DEFAULT_EVAL_SCORE = 5` (`ScoreEntryView.vue:16`) — every eval-judge input shows `5` when no raw score has been entered for that (player, category, judge slot).
- Clicker `+` / `−` default to `0` (falls back through `p.clickers[slot]?.plus ?? 0` / `?.minus ?? 0`).
- Major-deduction `Stop` / `Discard` / `Cut` (or `Detach`) default to `0`.
- `DEFAULT_CLICKER_VALUE = 60` (`scoring.ts:59`) — this is the max clicker-scaled value used by the scoring formula, NOT a UI default.

### Save-status pill wording (`ScoreEntryView.vue`)

- `save-status--idle` → text `Ready`.
- `save-status--pending` → text `Saving…`.
- `save-status--saved` → text `All changes saved`.

### Lock-warning copy (verbatim)

- Divisions view: `🔒 This contest is locked - unlock it from the Judges page before editing divisions.`
- Players view: `🔒 This contest is locked - unlock it from the Judges page before changing the roster.`
- Score Entry: `🔒 This contest is locked - go to the Judges page to unlock it before any scores can change.`
- Judges view: `🔒 This contest is locked. Unlock it below to make changes.` (head judge) / `... Only the head judge can unlock it.` (other judges).

The 🔒 is retained inside the warning banner (it's part of the current production copy). Status pills/badges in the redesign still use plain text (`Locked`, `Hidden`) — the banner is separate.

## Icons

Icons are monotone outline SVGs — Lucide-style paths inlined in **`frontend/src/components/Icon.vue`**. No emoji, no colored/pictorial icons anywhere in the chrome. Palette (add new ones to the component, don't scatter inline `<svg>` in views):

- `sun`, `moon` — theme toggle in `App.vue`
- `wand`, `smartphone`, `monitor` — view-mode cycle in `App.vue`
- `chevron-right` — accordion caret on the contest list (CSS rotates it 90° when the card opens)
- `chevron-up`, `chevron-down` — reserved for future collapsibles
- `trash` — destructive row action (Delete/Remove) in Divisions, Judges, Players

Stroke uses `currentColor`, so any container's text color recolors the icon automatically (danger red on hover, muted grey when disabled, etc.).

### Destructive row actions (`button.danger.icon-only`)

Delete / Remove in tables are icon-only trash buttons — no text label, `aria-label` carries the accessible name, `title` gives sighted hover copy (e.g. `Remove player`). Two shared CSS rules do the heavy lifting in `style.css`:

- `button.icon-only, button.chrome-btn` — 34×34 square with centered SVG.
- `td:has(> button.danger:only-child) { text-align: right; width: 1px; white-space: nowrap }` — the last column collapses to just the button so the trash doesn't waste a third of the row. Its empty header cell (`th:empty:last-child`) collapses the same way.

## Vue port — where the spec lives in the real app

The spec has been ported into `frontend/src/`:

- `App.vue` — top bar with plain `yoyo-judge` brand, view-mode cycle (🪄/📱/🖥️), theme toggle (🌙/☀️), user name, Log out. Measures its own height on mount / resize / theme+view-mode change and publishes `--topbar-h` on `<html>` so the workspace tab bar sticks flush against it in mobile-view.
- `composables/theme.ts` — `theme` (persisted `light`/`dark`), `viewMode` (persisted `auto`/`mobile`/`desktop`), and `cycleViewMode()`. `auto` resolves via `(max-width: 720px)`.
- `style.css` — component styles (`.contest-card` accordion, `.division-stage-block`, `.top3-mini`, `.new-contest`, `.plain-pill`, `.pill-stage`, `.tabs-bar`/`.tab`, `.workspace-crumb`, `:root[data-mobile-view]` overrides).
- `components/ContestWorkspaceLayout.vue` — the shared shell for every workspace page: `← Back to contests` + sticky tab bar (Divisions · Judges · Players · Input Score · Result Detail) + `<slot/>`. Players/Input-Score/Result-Detail tab targets fall back to the first available division/stage when the current route lacks those params.
- `views/ContestListView.vue` — rewritten to the accordion layout. Preserves all existing behaviors (create/edit/lock/hide/download) and the mounted data-loading (judges + results). Auto-opens the first contest on load.
- `views/ContestEditView.vue`, `views/JudgeManagementView.vue`, `views/PlayerRosterView.vue`, `views/ScoreEntryView.vue`, `views/ResultsView.vue` — wrapped in `<ContestWorkspaceLayout>` with the appropriate `active-tab`. Bodies are unchanged.

Score-input arrow-key nav + steppers stay in `components/ScoreNumberInput.vue` — the redesign didn't need to touch that component. `ScoreEntryView.vue` still passes `:group`/`:row`/`:col` per production semantics.
