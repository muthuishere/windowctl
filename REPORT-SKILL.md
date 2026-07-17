# REPORT-SKILL — auto-understand the GUI (text-targeted verbs)

**Branch:** `organs/harden` → PR against `main` (public repo `muthuishere/windowctl`), owner review, **not merged**.
**Goal:** make windowctl auto-understand the GUI so agents drive it by the **words on screen**, never by hand-computed coordinates.

## What shipped

### 1. Composite, text-targeted verbs (screenshot → OCR → resolve → act, in one call)

New public API (`visualverbs.go`) + CLI (`cmd/windowctl/visualverbs.go`), wired into dispatch + usage:

| Verb | Library | Does |
|------|---------|------|
| `click --text <s> [scope] [--right/--middle/--double] [--json]` | `ClickText(ClickTextOptions)` | OCR the scope, click the highest-confidence match for the text. Window scope is **raised first** so OCR reads the intended window. Returns the clicked `TextMatch`. |
| `exists --text <s> [scope] [--json]` | `TextExists(FindOptions)` | Read-only gate. **Exit 0 = present, 1 = absent** for shell `until`/`&&` loops. Never clicks. |
| `read [scope] [--text <s>] [--json]` | `ReadScreen(FindOptions)` | Dump recognized text in **reading order** (top→bottom, left→right within a 12pt row band). Empty `--text` = every line. |

Scope flags (`--title`/`--app`/`--monitor`/`--x --y --w --h`) are **identical** to `find`/`screenshot` — the coordinate contract is shared, not reimplemented. All live in the OS-agnostic public layer, composing the existing spec-13/14 primitives; **zero new adapter surface**.

### 2. Coordinate-space unification (the live trap, closed)

Two spaces exist: **absolute global** (window bounds, capture rect, monitor origins) and **monitor-relative** (raw `--x/--y` under `--monitor`). `FindText` already returns click points in **global** space (adapter adds the capture-rect origin); the composite verbs feed those straight to `MouseClick` with **no monitor** — also global. One space end-to-end, so the relative-vs-global confusion **cannot** occur through these verbs. Proven by `TestClickTextClicksTopMatchInGlobalSpace` (match at global (2400,900) on monitor 2 → clicked at exactly (2400,900), no offset).

### 3. Focus-guard positive test

Added `TestTypeIntoVerifiesFocusBeforeTyping` — the positive counterpart to the existing "refuses when focus never lands". The mock now models a **real focus transfer** (`focusSetsMonitorFocused`) and records the focused monitor at keystroke time (`typedFocusedMonitor`), so the test proves focus had transferred to the target **before** the first keystroke — not merely that text was typed.

### 4. SKILL.md rewrite — text-targeted verbs FIRST

- New leading Core Rule: **"Driving a GUI? Target ON-SCREEN TEXT, not coordinates"** — `click`/`exists`/`read` are first-choice; raw `--x/--y` is the last resort (bare icons, fixed offsets).
- New leading section in `references/visual-write.md` documenting the three verbs and the full zero-pixel write loop, with `find`+`mouse click` demoted to "when you need the raw point".
- Description triggers, Process routing, and Families table updated. Skill self-check (`tests/all.sh`) green.

### 5. Spec alignment

New slice `docs/specs/16-text-targeted-verbs.md` (indexed in `docs/specs/README.md`) with the coordinate-unification invariant, per-verb requirements, and Given/When/Then scenarios mapped to the headless tests.

## Tests — all headless, no live desktop input

`go test ./...` green. New tests (`visualverbs_test.go`) drive the mock adapter with OCR fixtures:

- `TestClickTextClicksTopMatchInGlobalSpace` — coordinate unification
- `TestClickTextRaisesWindowScopeFirst` — window raised + OCR scoped to its bounds
- `TestClickTextNoMatchErrorsAndClicksNothing` — miss = `ErrNoMatch`, no click
- `TestClickTextRequiresText`, `TestTextExists`, `TestTextExistsRequiresText`
- `TestReadScreenReadingOrder`
- `TestTypeIntoVerifiesFocusBeforeTyping`

`go vet ./...` clean; new files gofmt-clean. **No live clicking/typing was performed on this desktop** (standing rule honored) — the geometry/OCR-parsing logic is unit-tested against fixtures only.

## Also on the branch

`scripts/smoke-visual-loop.sh` carries an owner-authored hardening (clipboard-verified assertion + unique window titles) — a real-window CI smoke, **not run in this session**. Included in the commit as coherent with the branch.

## Not done (deliberate)

- No merge — PR is for owner review.
- No live-input smoke for the new verbs (out of scope by the standing no-live-input rule); a future CI smoke could exercise `click`/`exists`/`read` on the macOS runner.
