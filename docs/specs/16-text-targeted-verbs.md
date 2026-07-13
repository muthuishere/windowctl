# Spec 16 — text-targeted-verbs

- **Status:** Implemented
- **Source:** FR-INP-01, FR-SHOT-01, FR-INP-02, requirements §11 (composes specs 13–14)
- **Summary:** Composite verbs that let an agent drive the GUI by the **words on screen** instead of hand-computed coordinates: `click --text` (OCR → click), `exists --text` (OCR → gate), `read` (OCR → dump). Each does screenshot → OCR → resolve → act in one call, entirely in the OS-agnostic public package by composing the spec-13/14 primitives — no new adapter surface.

## Requirement: Coordinate unification (the load-bearing invariant)

There are two coordinate spaces in windowctl: **absolute global** (virtual-desktop points — what `Window.Bounds`, the capture rect, and monitor origins use) and **monitor-relative** (raw `--x/--y` when `--monitor N` is set). Mixing them is the classic bug the text-targeted verbs exist to eliminate.

`FindText` already returns each match's `ClickX/ClickY` in absolute global points (the adapter adds the capture rect's origin). The composite verbs feed those straight to `MouseClick` with **no** monitor set — which `MouseClick` also treats as absolute global. The whole path stays in one space; the caller never reconciles the two.

#### Scenario: found label on a secondary monitor is clicked at its global point

- **WHEN** `click --text "Submit"` resolves a match whose global click point is (2400, 900) on monitor 2 (origin 1920,0)
- **THEN** the adapter receives exactly (2400, 900) — no monitor-relative offset is applied, subtracted, or double-counted.

## Requirement: `click --text` (OCR → click)

`click --text <s> [--title|--app] [--monitor N] [--x --y --w --h] [--right|--middle] [--double] [--json]`. Scope flags are identical to `find`/`screenshot`. The highest-confidence match for the (case-insensitive substring) text is clicked; the clicked `TextMatch` is returned/printed.

- When scoped to a window (`--title`/`--app`), that window is **raised first** so OCR reads the intended window (not an occluding one) and the click lands in it.
- No on-screen text matching the query is a hard error wrapping `ErrNoMatch` (message names the searched text), exits non-zero, and clicks nothing — safe to gate on.
- Reuses the spec-14 accessibility gating (macOS AX) via `MouseClick` and the spec-13 Screen Recording gating via the OCR capture.

#### Scenario: window scope is raised before OCR

- **WHEN** `click --text "OK" --app "Google Chrome"` runs
- **THEN** Chrome is focused/raised, OCR is scoped to Chrome's bounds, and the resulting click uses the match's global point.

## Requirement: `exists --text` (OCR → gate)

`exists --text <s> [scope…] [--json]` is `click`'s read-only sibling: same scoping, never clicks. Exit code is script-first — **0 when the text is present, 1 when absent** — so shell gates (`until windowctl exists …; do …`) work without parsing output.

#### Scenario: absent text exits non-zero

- **WHEN** `exists --text "Document Saved" --app TextEdit` runs and no such text is on screen
- **THEN** the command prints `not found` and exits 1, having issued no click.

## Requirement: `read` (OCR → dump in reading order)

`read [--title|--app] [--monitor N] [--x --y --w --h] [--text <s>] [--json]` dumps recognized text. Unlike `find` (confidence order, for click-targeting), `read` returns runs in **reading order** — top-to-bottom, then left-to-right within a row band (`readingRowBand`, 12pt) so a label slightly higher on the right doesn't jump ahead of one on the left. Empty `--text` reads everything; a non-empty `--text` filters to matching runs first.

#### Scenario: two labels on the same row sort left-to-right

- **WHEN** two runs share a row (Y within the band) and one sits further right
- **THEN** `read` emits the left run before the right run, and both before any run on a lower row.

## Tasks

| Task | Covers |
|------|--------|
| `task test` | Headless unit tests: coordinate-unification click, window-scope raise, no-match/empty-text guards, `exists` true/false, `read` reading order (`visualverbs_test.go`), plus the focus-guard positive path `TestTypeIntoVerifiesFocusBeforeTyping`. |
| `task build` | Compiles the `click`/`exists`/`read` CLI verbs (`cmd/windowctl/visualverbs.go`). |

No live-input smoke task exists for these verbs by design — they are validated headlessly with OCR fixtures against the mock adapter; real clicking/typing on a developer desktop is deliberately out of scope.
