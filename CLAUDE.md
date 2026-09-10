# yoyo-judge — notes for Claude

## Frontend redesign spec

The UI redesign (accordion-list main page + tabbed contest workspace, dark/light + view-mode toggles, Excel-style score input) is specified in **[`docs/LAYOUT_SPEC.md`](docs/LAYOUT_SPEC.md)**. That doc covers page layouts, tab structure, button behaviors (including score-input arrow-key navigation and stepper increments), table alignment rules, and status-label conventions.

The redesign lives in the Vue app under `frontend/src/`:
- `components/ContestWorkspaceLayout.vue` (the tab shell wrapping the 5 workspace views)
- `components/Icon.vue` (monotone outline SVG icons — add new ones here, don't inline `<svg>` in views)
- `views/ContestListView.vue` (accordion list)
- `App.vue` (top bar with view-mode + theme + `--topbar-h` measurement)
- `style.css` (`.contest-card`, `.division-stage-block`, `.top3-mini`, `.tabs-bar`, `.plain-pill`, `button.icon-only`, etc.)

When making UI changes, read `docs/LAYOUT_SPEC.md` first and update it if the decision changes.

## Mock API for LAN previews

`frontend/src/api/mock.ts` is used when `VITE_USE_MOCK=true`. `crypto.randomUUID` requires a secure context, so the mock's `uid()` falls back to `Math.random().toString(16)` — keep that fallback so plain-HTTP LAN previews (`http://192.168.x.x:5174`) still boot.
