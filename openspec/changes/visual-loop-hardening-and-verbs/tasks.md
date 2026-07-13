# Tasks

## 1. Core interface + types
- [ ] 1.1 Add `Match`(OCR), `Scroll`, `Drag`, `Clipboard`/`SetClipboard`, `FindText`,
  `WindowState` to `internal/core` Adapter + supporting types (`TextMatch`, `WindowOp`).
- [ ] 1.2 Non-darwin adapters return `ErrNotImplemented` (compiles on all 3 OSes).

## 2. find --text (Vision OCR) — priority
- [ ] 2.1 CGO `wctl_ocr_rect` in darwin `automation.go`: capture→Vision→matches with
  normalized bottom-left→top-left transform, point-normalized.
- [ ] 2.2 Public `FindText(scope, query)` in root `find.go`: scope resolution
  (monitor/window/region), substring match, confidence sort.
- [ ] 2.3 CLI `windowctl find --text` (+ `--json`, scope flags).
- [ ] 2.4 Live test: OCR the TextEdit proof + a real UI, click a found point.

## 3. clipboard get/set
- [ ] 3.1 CGO NSPasteboard read/write.
- [ ] 3.2 Public `Clipboard()/SetClipboard()`, CLI `clipboard get|set`.
- [ ] 3.3 Live round-trip test.

## 4. scroll + drag
- [ ] 4.1 CGO scroll-wheel + interpolated drag.
- [ ] 4.2 Public `Scroll`/`Drag` (monitor-relative rule), CLI verbs.
- [ ] 4.3 Live test.

## 5. window-state verbs
- [ ] 5.1 CGO AX minimize/maximize/fullscreen/close.
- [ ] 5.2 Public helpers + CLI verbs.
- [ ] 5.3 Live test on a throwaway window.

## 6. Typing hardening
- [ ] 6.1 Pace type events (down→up gap) in darwin `wctl_type_chunk`.
- [ ] 6.2 Bounded post-focus settle in `ensureFocused` (`WCTL_FOCUS_SETTLE_MS`), fake-clock unit test.

## 7. Proofs
- [ ] 7.1 `scripts/smoke-visual-loop.sh` (+ `task smoke-visual-loop`): find --text→click→
  insert→screenshot-verify on TextEdit, no hardcoded pixels; emits a receipt line.
- [ ] 7.2 Visual write-rail proof: fill a YouTube comment box via find --text, screenshot-
  verify, DO NOT submit, clear the box. Document head-to-head vs DOM recipes.

## 8. Complementarity
- [ ] 8.1 Verify sysauto's windowctl subcommands still work; list them in the report.
- [ ] 8.2 Document meetbot primitives (launch/wait/focus/find/click/type).
- [ ] 8.3 Receipts: JSON line per visual-loop flow.

## 9. Tests + docs + release sync
- [ ] 9.1 Unit tests (transform math, matching, settle) green on `go test ./...`.
- [ ] 9.2 Update `skills/` docs + `docs/specs` for new verbs.
- [ ] 9.3 npm dist: verify launcher + staging path unaffected; version bump on release.
- [ ] 9.4 Huddle verify; report to CEO; open PR.
