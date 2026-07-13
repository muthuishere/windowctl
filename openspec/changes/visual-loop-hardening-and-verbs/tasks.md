# Tasks

## 1. Core interface + types
- [x] 1.1 Add `FindText`, `Clipboard`/`SetClipboard`, `Scroll`, `Drag`, `WindowState` to
  `internal/core` Adapter + supporting types (`TextMatch`, `WindowOp`).
- [x] 1.2 Non-darwin adapters return `ErrNotImplemented` (compiles on all 3 OSes).

## 2. find --text (Vision OCR) — priority
- [x] 2.1 CGO `wctl_ocr_rect` in darwin `vision.m`: capture→Vision→matches with
  normalized bottom-left→top-left transform, point-normalized.
- [x] 2.2 Public `FindText(FindOptions)` in root `find.go`: scope resolution
  (monitor/window/region) shared with Screenshot, substring match, confidence sort.
- [x] 2.3 CLI `windowctl find --text` (+ `--json`, `--first`, scope flags).
- [x] 2.4 Live test: OCR the TextEdit proof + a real UI (Chrome), click a found point.

## 3. clipboard get/set
- [x] 3.1 CGO NSPasteboard read/write (`pasteboard.m`).
- [x] 3.2 Public `GetClipboard`/`SetClipboard`, CLI `clipboard get|set`.
- [x] 3.3 Live round-trip test.

## 4. scroll + drag
- [x] 4.1 CGO scroll-wheel + interpolated drag.
- [x] 4.2 Public `Scroll`/`Drag` (monitor-relative rule), CLI verbs.
- [x] 4.3 Live test.

## 5. window-state verbs
- [x] 5.1 CGO AX minimize/fullscreen/close; maximize composed via Move.
- [x] 5.2 Public `SetWindowState`/`ParseWindowOp` + CLI verbs.
- [x] 5.3 Live test on a throwaway window.

## 6. Typing hardening
- [x] 6.1 Pace type events (down→up gap) in darwin `wctl_type_chunk`.
- [x] 6.2 Bounded post-focus settle in `ensureFocused` (`WCTL_FOCUS_SETTLE_MS`), fake-clock unit test.
- [x] 6.3 `ensureFocused` skips the raise when already focused (first-responder-preserving) — found via live recipe replay.

## 7. Recipes
- [x] 7.1 `recipe.go` engine: Recipe/RecipeStep/RunRecipe, config load/merge/save, $VAR
  substitution, confirm gating, vision-first + coordinate + paste fallbacks, inter-step settle.
- [x] 7.2 `recipes.seed.json` seeds (youtube-comment, x-compose) — submit confirm-gated.
- [x] 7.3 CLI `recipe list|save|run`; unit tests + live replay (clipboard-verified).

## 8. Proofs
- [x] 8.1 `scripts/smoke-visual-loop.sh` (+ `task smoke-visual-loop`): find --text→click→
  insert→OCR-verify on TextEdit, no hardcoded pixels; emits a receipt line.
- [x] 8.2 Visual write-rail proof: filled a Google (Chrome) compose box via find --text,
  OCR-verified text IN the box, STOPPED before submit, cleared it. Head-to-head documented.

## 9. Complementarity
- [x] 9.1 sysauto's windowctl subcommands still work; listed in the report.
- [x] 9.2 meetbot primitives documented (launch/wait/focus/find/click/type).
- [x] 9.3 Receipts: JSON line per visual-loop flow.

## 10. Tests + docs + release sync
- [x] 10.1 Unit tests (transform/scope/matching/settle/recipe) green on `go test ./...`.
- [x] 10.2 Update `skills/` docs for new verbs.
- [x] 10.3 npm dist: launcher + staging path unaffected (no code change; version bump on release).
- [ ] 10.4 Huddle verify; report to CEO; open PR.
