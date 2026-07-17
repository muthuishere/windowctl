# Design

## Coordinate contract (authoritative)

All points/rects in the public API are **global virtual-desktop points** — the space
`ListMonitors`, `Move`, `MouseMove/Click`, and `CaptureRect` already share. On the test
Mac the three monitors span `x ∈ [-1728, 3840]` (a laptop retina panel at negative x, two
1920×1080 externals), proving the space is a signed union, not per-display.

- **Retina / HiDPI:** capture is resampled so **1 image pixel == 1 global point**. A pixel
  read at image `(px,py)` of a window/region captured with origin `(ox,oy)` is the global
  point `(ox+px, oy+py)` — feed it straight to `mouse click`. This is the load-bearing
  invariant the whole visual loop rests on.
- **Monitor-relative rule:** when `--monitor N` is given, x/y are offset by monitor N's
  origin; absolute otherwise. `find` and `clipboard` return/consume absolute global points.

## `find --text` — Vision OCR → click point

macOS **Vision** `VNRecognizeTextRequest` (accurate mode) runs on a captured PNG. Pure
CGO against the Vision + CoreImage frameworks — no new Go deps, no subprocess.

**The coordinate transform is the crux.** Vision returns each observation's `boundingBox`
as a normalized rect in `[0,1]` with a **bottom-left origin** (Apple/Vision convention),
which must be flipped to the top-left image space and then offset by the capture origin:

```
img_x = bbox.minX * imgW
img_y = (1 - bbox.maxY) * imgH          // flip bottom-left → top-left
click_global_x = capture_ox + img_x + (bbox.width  * imgW)/2
click_global_y = capture_oy + img_y + (bbox.height * imgH)/2
```

Because the capture is point-normalized (1px==1pt), `imgW/imgH` are already point
dimensions, so the click point is directly in the global click space. Matching is
case-insensitive substring by default (consistent with `--title`); each match returns
`{text, confidence, bounds(global), click(global)}`. Default scope is a monitor (default
= focused monitor); `--title/--app` scopes to a window's rect; `--x/--y/--w/--h` to a
region. `--json` emits the list; text output prints `click_x click_y  confidence  text`.

Sources: normalized bottom-left bounding boxes — bendodson "Detecting text with
VNRecognizeTextRequest", machinethink "How to display Vision bounding boxes",
Apple `VNImageRectForNormalizedRect` docs.

## Typing hardening

Two root causes reproduced live, two fixes:

1. **Zero inter-event gap in the type path.** `wctl_type_chunk` posts key-down then key-up
   with no delay; the chord path already sleeps 10 ms between down and up. Fix: add the
   same small gap in the type path (down→up), keeping the existing 10 ms between 20-char
   chunks. Best practice: pace `CGEventKeyboardSetUnicodeString` events to avoid dropped
   input (kulman, isamert; Apple forums). This is *mitigation*, not a cure — which is why
   `clipboard` + paste is offered as the reliable alternative.
2. **First keystrokes dropped after a focus change.** The focus guard verifies the window
   is *key* but not that its text view is first-responder-ready. Fix: a short, bounded
   settle (`WCTL_FOCUS_SETTLE_MS`, default ~120 ms, env-overridable, 0 to disable) after
   focus verification, before injecting. Kept in the shared public package so it is
   unit-tested with a fake clock, not the adapter.

`clipboard set` + `key --combo cmd+v` (or the caller's own paste) sidesteps synthetic
keycodes/layout entirely and is documented as the preferred way to insert non-trivial text.

## scroll / drag

- `scroll` → `CGEventCreateScrollWheelEvent` (line units), optional move-to-point first;
  `--amount` signed, `--horizontal` for the x axis.
- `drag` → mouse-down at `from`, a few interpolated `mouseDragged` events, mouse-up at
  `to`; button selectable. Interpolation matters so apps register the drag.

## Window-state verbs

Via the existing AX bridge (`AXUIElementSetAttributeValue` / `AXUIElementPerformAction`):
- `minimize` → set `kAXMinimizedAttribute = true`; `maximize` → AX zoom / set frame to the
  window's monitor visible frame; `fullscreen` → toggle `kAXFullScreenAttribute` (fallback
  to the green-button `AXPressAction`); `close` → `AXPressAction` on `kAXCloseButton`.
  All resolve the target window the same way Move/Focus do and return
  `ErrAccessibilityDenied` without trust.

## Interface / layering

New `Adapter` methods: `FindText(rect, query) []Match`, `Clipboard() (string,error)` /
`SetClipboard(string)`, `Scroll(x,y,dx,dy)`, `Drag(from,to,button)`, and
`WindowState(id, op)`. Everything expressible as composition (region/window/monitor
resolution, normalized→point transform, substring matching, the point-normalization
guarantee, post-focus settle) stays in the public package; adapters receive fully
resolved rects/points and a parsed op — matching the project's stated invariant.

## Non-darwin

Each new adapter method returns `ErrNotImplemented` on linux/windows for now (clipboard
may shell out later). The public logic (matching, transforms, settle, receipts) is
OS-agnostic and covered by unit tests that run on all three CI platforms.
