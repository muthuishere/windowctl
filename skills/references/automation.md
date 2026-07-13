---
name: window-ctl-skill
---

# Automation — screenshot, mouse, keyboard, launch/wait

Covers the desktop-automation surface that lets an agent *see* the
screen and *act* on it: `screenshot`, `mouse`, `type`, `key`,
`launch`, `wait`. Together these close the **visual loop**:

> `screenshot` → (you read the PNG) → `mouse click` / `type` → `screenshot` again to confirm.

Everything here is a single native `windowctl` invocation (CGO /
CoreGraphics on macOS, Win32 on Windows, xdotool/ImageMagick on Linux).
The skill never touches Quartz, Win32, or X11 directly.

---

## The point-normalization guarantee (why the loop works)

`windowctl screenshot` writes a PNG whose **pixel grid equals the
global coordinate grid** — 1 image pixel == 1 coordinate point, even
on a retina/HiDPI display. The command reports the captured rect's
origin. So to click something you spotted in the image:

```
globalX = capturedRect.X + pixelXInImage
globalY = capturedRect.Y + pixelYInImage
```

Then `windowctl mouse click --x globalX --y globalY`. No scale factor,
ever. This is the single most important fact in this file.

---

## screenshot

```sh
windowctl screenshot [--out <path>] [--json] \
  ( --monitor <n> | --x <n> --y <n> --w <n> --h <n> | --title <s> | --app <s> )
```

- **No target** → the FOCUSED monitor (where the user is working),
  falling back to primary. Use this for "screenshot my screen".
- `--monitor <n>` → that whole display.
- `--x --y --w --h` → a region. Monitor-relative when `--monitor` is
  also given, absolute virtual-desktop points otherwise (same rule as
  `move`).
- `--title` / `--app` → the matched window's bounds (discover first,
  see Core Rules — exactly one window must match).
- `--out` defaults to `screenshot-<unixtime>.png` in the cwd.

`--json` shape (feed the origin back into `mouse`):
```json
{"Path":"shot.png","X":1920,"Y":0,"Width":2560,"Height":1440}
```

**macOS permission:** capture needs the **Screen Recording** TCC grant,
which is SEPARATE from Accessibility. If a capture returns
`Screen Recording permission denied`, route the user to
`windowctl permissions --screen` (check silently with
`windowctl permissions --screen --status`). See `permissions.md`.

---

## mouse

```sh
windowctl mouse move  --x <n> --y <n> [--monitor <n>]
windowctl mouse click [--x <n> --y <n>] [--monitor <n>] [--right|--middle] [--double]
windowctl mouse position [--json]
```

- Coords follow the monitor-relative rule (relative iff `--monitor`).
- `click` with no `--x/--y` clicks at the CURRENT cursor position.
- `--double` for double-click; `--right` / `--middle` for the other
  buttons.
- `mouse position` reports the cursor's global point — handy to record
  a spot the user points at, or to restore the cursor after acting.

**macOS:** mouse + keyboard synthesis needs Accessibility (same grant
as `move`/`focus`). A `mouse`/`type`/`key` call returning
`Accessibility permission denied` → `windowctl permissions`.

---

## type and key — and the focus guard

```sh
windowctl type --text "hello world" [--title <s> | --app <s>]
windowctl key  --combo "cmd+shift+s" [--title <s> | --app <s>]
```

`type` injects literal text (unicode, IME-safe). `key` presses ONE
chord: `+`-separated modifiers then a key.

- Modifiers: `cmd` (= Command on mac, Win/Super elsewhere — aliases
  `command`/`meta`/`super`/`win`), `ctrl`, `alt` (aliases `opt`/
  `option`), `shift`.
- Keys: any single character, `f1`..`f12`, or a named key: `enter`,
  `tab`, `esc`, `space`, `up`/`down`/`left`/`right`, `delete`,
  `backspace`, `home`, `end`, `pageup`, `pagedown`.
- Shift must be explicit: `cmd+shift+s`, not `cmd+S`.

### ALWAYS pass `--title`/`--app` when you know the target window.

Without a filter, `type`/`key` hit whatever holds the keyboard *at the
instant the event is delivered*. Right after a `launch`, a `focus`, or
a window-spawning chord like `cmd+n`, that is very often the PREVIOUS
window — so your keystrokes silently corrupt an unrelated document.

With `--title`/`--app`, the command focuses the match, **verifies**
the window actually became focused (polls up to 2s), and only then
types. If focus never lands it errors with *"input NOT sent"* instead
of typing into the wrong place. Treat the filtered form as the default
and the bare form as the rare exception (e.g. you JUST clicked into a
field yourself).

---

## launch and wait

```sh
windowctl launch --app "Google Chrome"
windowctl wait (--title <s> | --app <s>) [--timeout <ms>] [--json]
```

`launch` starts/foregrounds an app and returns immediately — it does
NOT wait for a window. `wait` polls until a window matches (default
timeout 10000ms).

**Standard "open then act" recipe:**
```sh
windowctl launch --app TextEdit
windowctl wait --app TextEdit --timeout 8000 --json   # blocks until the window exists
windowctl type --app TextEdit --text "now safe to type"
```

**Gotcha — `wait` matches PRE-EXISTING windows too.** If the app is
already running with a window, `wait` returns instantly on the old
one. When you specifically need the NEW window: snapshot
`windows list --json` before `launch`, then diff IDs after, instead of
relying on `wait`.

---

## Putting it together — a full visual loop

```sh
# 1. See the current screen
windowctl screenshot --out /tmp/before.png --json
#    → {"Path":"/tmp/before.png","X":0,"Y":0,"Width":1920,"Height":1080}

# 2. (agent reads /tmp/before.png, finds a button at pixel 840,470)

# 3. Click it — origin was (0,0) so global == pixel here
windowctl mouse click --x 840 --y 470

# 4. Type into the field that opened, targeting the window by name
windowctl type --app "Google Chrome" --text "search query"
windowctl key --combo enter --app "Google Chrome"

# 5. Confirm
windowctl screenshot --app "Google Chrome" --out /tmp/after.png
```

---

## Cross-platform notes (Parallels / multi-OS)

The same six commands exist on macOS, Windows, and Linux. Differences
to expect:

- **App names in `launch`:** macOS takes the display name (`"Google
  Chrome"`); Windows takes a name/path `start` can resolve; Linux
  takes a binary on `PATH` (`google-chrome`).
- **Permissions:** the macOS Accessibility + Screen Recording grants
  have no equivalent on Windows/Linux — `permissions` reports "not
  required" there.
- **`key` punctuation** is mapped by US-layout keycode position; on a
  non-US layout the produced glyph for punctuation chords may differ
  (letters, digits, named keys are stable).
- **Linux** needs `xdotool` (input) and `import`/`scrot` (capture)
  installed, and is X11-first (Wayland best-effort), mirroring the
  existing `move`/`focus` story.
