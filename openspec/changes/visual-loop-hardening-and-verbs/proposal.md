# Proposal: Visual-loop hardening + on-screen OCR and control verbs

## Why

`windowctl` already has the visual loop (screenshot → mouse → type/press). Two gaps
block it from being a *reliable, self-directing* machine-control organ for the deemwar
CEO-embodiment fleet:

1. **It cannot see what to click.** An agent must hand-compute pixel coordinates from a
   screenshot. There is no way to say "click the *Submit* button" — the loop is blind to
   on-screen text.
2. **Synthetic typing is flaky and can wedge the target.** Live testing on a 3-monitor
   Retina Mac reproduced two real failure modes: (a) keystrokes fired immediately after a
   focus/first-responder change are dropped, and (b) after many rapid back-to-back
   `CGEvent` injections the target app's text view stops accepting keyboard input entirely
   (mouse still works, events post `rc=0`, a fresh app instance recovers). The type path
   posts key-down/up with **zero** inter-event delay, unlike the chord path (10 ms).
   Best practice for `CGEventKeyboardSetUnicodeString` is to pace events — insert small
   delays between keystrokes to avoid dropped input (kulman, isamert; Apple forums).

## What changes

New CLI verbs + library functions, all behind the existing three-layer architecture
(cmd → public package → per-OS adapter), unit-testable logic in the public package:

- **`windowctl find --text <s>`** — on-screen OCR via the native macOS **Vision**
  framework (`VNRecognizeTextRequest`), pure CGO, **zero new deps**. Returns every match
  with confidence + a click point **in the same global point space as `mouse click`**, so
  `find --text 'Submit' | click` needs no hardcoded pixels. This completes the visual loop.
- **`windowctl clipboard get|set`** — read/write the system pasteboard. More reliable than
  synthetic typing and layout/IME-independent; the recommended path for inserting text.
- **`windowctl scroll`** — wheel scroll (direction + amount) at an optional point.
- **`windowctl drag`** — press-hold-move-release from one point to another (drag-drop,
  sliders, text selection).
- **Window-state verbs** — `windowctl minimize|maximize|fullscreen|close` alongside the
  existing move/resize/focus.
- **Typing hardening** — inter-event pacing in the darwin type path and a short, bounded
  post-focus settle in the shared focus guard, so the first keystrokes after a focus
  change land.
- **Coordinate contract, documented precisely** — one authoritative doc covering the
  global point space, the retina 1 px == 1 pt capture guarantee, the monitor-relative
  rule, the Vision normalized→point transform, and negative-origin (left-of-primary)
  multi-monitor arrangements.
- **`windowctl recipe save|list|run`** — persist and replay named, parameterized ($VAR)
  sequences of the visual primitives, scoped to a window match, stored in
  `~/.config/windowctl/recipes.json` with embedded seeds (`youtube-comment`, `x-compose`).
  Vision-first with coordinate + clipboard-paste fallbacks; **safe-by-default** — the final
  submit/publish step is `confirm`-gated and skipped unless `--confirm` is passed. This is
  the visual twin of a browser-bridge DOM recipe: windowctl = visual writes into ANY app,
  browser-bridge = DOM reads.
- **Receipts** — the visual-loop flow (and each new verb) can emit one JSON receipt line.

## Impact

- Affected specs: NEW `13-screenshot` sibling capabilities — `find-text-ocr`,
  `clipboard`, `scroll-drag`, `window-state`, plus MODIFIED `14-input-automation`
  (pacing + settle) and a new `coordinate-contract` reference.
- Affected code: `internal/core` (Adapter interface + types), `internal/adapter/darwin`
  (Vision OCR, scroll, drag, clipboard, window-state, type pacing), root public package
  (`find.go`, `clipboard.go`, `pointer.go`, window-state helpers, focus settle),
  `cmd/windowctl` (verb wiring), `skills/` docs, `npm/` (no code change; version bump on
  release).
- Linux/Windows: verbs are added to the interface; darwin is the reference implementation.
  Non-darwin returns `ErrNotImplemented` where a native path isn't wired yet, keeping the
  CI matrix green (compiles on all three).

## Non-goals

- Linux/Windows native OCR (Vision is macOS-only; Tesseract shell-out is a follow-up).
- Remote-viewer clipboard sync (noted as a follow-up, not built here).
- Replacing synthetic typing — clipboard is offered *alongside* it, not as a removal.
