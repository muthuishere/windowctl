# Spec 13 — screenshot

- **Status:** Implemented
- **Source:** FR-SHOT-01, §8.2, requirements §11
- **Summary:** Capture any monitor, region, or window to a point-normalized PNG whose pixel grid IS the global coordinate grid, so agents can read a pixel position off the image and click it.

## Requirement: Target selection

`windowctl screenshot` accepts at most one target:

- `--title <s> | --app <s>` — the matched window's current bounds (FR-MOV-04 matching rules; mutually exclusive with `--monitor`/region).
- `--x --y --w --h` — an explicit region; monitor-relative when `--monitor` is also given, absolute virtual-desktop points otherwise (the FR-MOV-02 rule).
- `--monitor <n>` alone — that monitor's full bounds.
- nothing — the **focused** monitor, falling back to primary, falling back to the first.

#### Scenario: default capture follows the user's attention

- **WHEN** `windowctl screenshot` runs with no target flags
- **THEN** the monitor whose `Focused` flag is set is captured, because "screenshot" with no arguments means "what I'm working on", not "monitor 1".

#### Scenario: region spanning displays

- **WHEN** an absolute region straddles two monitors
- **THEN** the capture composites both (CGWindowListCreateImage on macOS operates on the global virtual desktop; BitBlt on Windows reads the virtual screen).

## Requirement: Point normalization (the visual-loop invariant)

The written PNG is resampled so its pixel dimensions equal the captured rect's point dimensions — on a 2x retina display the compositor's pixels are downscaled 2:1.

**Why this is load-bearing:** the consuming agent finds a UI element at image pixel (px, py) and clicks at global (rectX + px, rectY + py). Without normalization every consumer would need to know each display's scale factor. The captured rect origin is always reported (stdout text and `--json`) precisely to enable this translation.

#### Scenario: retina display

- **WHEN** a 1920×1080-point monitor backed by a 3840×2160 framebuffer is captured
- **THEN** the PNG is 1920×1080 and `--json` reports `Width:1920, Height:1080` plus the global origin.

## Requirement: Screen Recording permission (macOS)

Capture requires the Screen Recording TCC permission — a **separate** grant from Accessibility. `CaptureRect` preflights (`CGPreflightScreenCaptureAccess`) and returns `ErrScreenCaptureDenied` (whose message points at `windowctl permissions --screen`) rather than silently writing a wallpaper-only frame. `windowctl permissions --screen` requests (may prompt once); `--screen --status [--json]` checks silently, mirroring spec 11's read-only contract; JSON shape `{"granted": bool}`. Not required on linux/windows (`granted: true`).

## Implementation notes

- darwin: native CGO — `CGWindowListCreateImage` → optional `CGBitmapContext` downscale → ImageIO PNG. The API family is compile-time obsoleted in the macOS 15 SDK; the file compiles with `-mmacosx-version-min=12.0` (CFLAGS only, so link warnings don't appear) which downgrades obsoletion to deprecation. **Verified working at runtime on macOS 26.** If Apple ever removes the symbol at runtime, the replacement is ScreenCaptureKit via C blocks.
- windows: GDI `BitBlt` (with `CAPTUREBLT`) into a top-down 32bpp DIB section, BGRA→RGBA, stdlib `png.Encode`.
- linux: shells out to ImageMagick `import -window root -crop`, falling back to `scrot -a`; `ErrNotImplemented`-wrapped error when neither exists.
