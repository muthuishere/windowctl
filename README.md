# windowctl

**windowctl** is a cross-platform CLI **and** Go library for managing desktop windows and monitors on macOS, Windows, and Linux in a consistent, scriptable way.

## Overview

A unified interface across macOS, Windows, and Linux for:

- Listing windows and monitors (with which monitor is currently active and which holds the focused window)
- Filtering windows by title or application name
- Moving windows across monitors using zones or absolute/relative coordinates
- Resizing windows in place
- Focusing specific windows
- Bulk-placing many windows in one call from a JSON layout (`windowctl batch`)
- Granting macOS Accessibility permission interactively

## Installation

### npm (recommended)

```sh
npx @muthuishere/windowctl <command>
```

```sh
npm install -g @muthuishere/windowctl
```

The npm package is a thin JS launcher that resolves the matching `@muthuishere/windowctl-<os>-<arch>` sub-package (darwin-arm64, darwin-x64, linux-arm64, linux-x64, windows-x64) via `optionalDependencies`. No postinstall download, no GitHub Releases dependency at install time.

### Go

```sh
go install github.com/muthuishere/windowctl/cmd/windowctl@latest
```

### Binary (GitHub Releases)

Download the appropriate binary for your platform from the [Releases](../../releases) page.

## macOS Accessibility permission

Move and Focus on macOS require Accessibility (AX) permission for the parent process (your shell, your editor, etc.). The first call from a new parent prompts you in System Settings → Privacy & Security → Accessibility.

```sh
windowctl permissions                 # trigger the AX prompt (first time only)
windowctl permissions --status        # human-readable: "granted" / "denied"
windowctl permissions --status --json # → {"trusted": true|false} (exit 0 either way)
```

`windows list` and `monitors list` work without AX. Only Move/Focus/Resize need it.

## Usage

### List windows

```sh
windowctl windows list
windowctl windows list --title "chrome"
windowctl windows list --app "Firefox" --json
```

JSON shape (per window):

```json
{
  "ID": "227",
  "Title": "Reqsume - AI Resume Builder",
  "App": "Google Chrome",
  "PID": 4356,
  "Monitor": 2,
  "Focused": true,
  "Bounds": { "X": 1920, "Y": 25, "Width": 1920, "Height": 1055 }
}
```

- **`Monitor`** is the **1-indexed** ID of the display containing the window's centroid (matches `monitors list`). `0` means the window is off every display (off-screen / hidden).
- **`Focused`** is `true` for the frontmost window on the focused monitor — useful for "operate on whatever I'm looking at right now" scripts.
- **`Bounds`** uses `Width`/`Height` (not `W`/`H`) — same shape as `monitors list`.

Only real application windows are listed. macOS menu-bar widgets, status items, the Dock, Spotlight, Control Center, AltTab, and Window Server menubars (everything on a non-zero `kCGWindowLayer`) are filtered out so the list reflects what you can actually move and focus.

### List monitors

```sh
windowctl monitors list
```

```
ID  X     Y   WIDTH  HEIGHT  PRIMARY  ACTIVE  FOCUSED
1   0     0   1920   1080    true     true    false
2   1920  0   1920   1080    false    false   true
3   3840  30  1024   640     false    false   false
```

- **ID** is **1-indexed** and assigned by ascending `(X, Y)` origin so the leftmost display is always `1`. Stable across reboots and re-plugs (system enumeration order is not).
- **ACTIVE** = the cursor is currently over this monitor.
- **FOCUSED** = the frontmost window's centroid is on this monitor. Independent of ACTIVE — you can mouse over one display while typing into a window on another.

### Move a window

```sh
# Top-right quadrant of monitor 2
windowctl move --app "Google Chrome" --monitor 2 --zone 2B

# Left half of the monitor that contains the matched window
windowctl move --title "Terminal" --zone 1A

# First third of the screen (split zone)
windowctl move --title "Slack" --zone 3:1

# Absolute coordinates
windowctl move --title "chrome" --x 0 --y 0 --w 960 --h 1080

# Monitor-relative coordinates (--x/--y relative to monitor 2's origin)
windowctl move --app "Code" --monitor 2 --x 100 --y 100 --w 800 --h 600
```

`--monitor` is **1-indexed** (use `1`, `2`, `3`, ...). Omit to auto-resolve to the monitor containing the matched window. `--w` and `--h` must be `> 0` in coord mode.

When the OS clamps a move (e.g. Chrome refuses widths smaller than ~500px, or a window resists leaving the menu-bar inset), the call exits non-zero with the actual landed bounds:

```
windowctl: requested 512x640 at (3840,30), OS clamped to 576x615 at (3840,55) (likely a minimum-window-size constraint)
```

### Resize a window

Keeps the window's current `X`/`Y` and only changes its size.

```sh
windowctl resize --app "Google Chrome" --w 900 --h 700
```

### Focus a window

```sh
windowctl focus --title "jira"
windowctl focus --app "Google Chrome"
```

### Batch (bulk move)

Place many windows in one call by piping a JSON array of move specs. Each entry is the same shape as `move`'s flags. Entries run sequentially; one failure (e.g. an OS clamp on a single window) does **not** abort the rest.

```sh
windowctl batch < layout.json
windowctl batch --file layout.json
windowctl batch --json < layout.json   # structured per-entry results
```

`layout.json`:

```json
[
  { "app": "Ghostty",          "monitor": 2, "x": 0,   "y": 25,  "w": 1920, "h": 1055 },
  { "app": "Activity Monitor", "monitor": 1, "zone": "2A" },
  { "title": "Inbox",                        "x": 100, "y": 100, "w": 800,  "h": 600  }
]
```

Per entry: at least one of `title` / `app` is required; target is **either** `zone` **or** all four of `x`/`y`/`w`/`h` (not both). `monitor` is optional and 1-indexed.

Exit codes: `0` if every entry succeeded, `1` if any entry failed, `2` for invalid input (parse error, missing file). With `--json` the same per-entry results land on stdout as `{ "entry": {...}, "ok": true }` or `{ "entry": {...}, "error": "..." }`.

## Zones

### Predefined (Enum) Zones

| Zone | Description          |
|------|----------------------|
| 1A   | Left half            |
| 1B   | Right half           |
| 2A   | Top-left quarter     |
| 2B   | Top-right quarter    |
| 2C   | Bottom-left quarter  |
| 2D   | Bottom-right quarter |

### Split Zones (`N:M`)

Divide the screen into N equal columns and place the window in column M.

```
2:1  →  left half
2:2  →  right half
3:1  →  first third
3:2  →  middle third
3:3  →  last third
```

## Filter matching

- `--title` is a **case-insensitive substring** match.
- `--app` is a **case-insensitive exact** match against the OS-reported app name (e.g. VSCode reports `Code`, Chrome reports `Google Chrome`).

When multiple windows match, the first one is used.

## Platform support

| Platform | Status      | Backend                          |
|----------|-------------|----------------------------------|
| macOS    | Supported   | CoreGraphics + Accessibility API |
| Windows  | Supported   | Win32 (`user32.dll`)             |
| Linux    | Best-effort | `wmctrl` / `xrandr`              |
| Wayland  | Limited     | Best-effort only                 |

## Building

This project uses [Taskfile](https://taskfile.dev) for build orchestration.

```sh
task build              # Build CLI binary
task test               # Run unit tests
task lint               # go vet
task snapshot           # Cross-compile + stage npm binaries (no publish)
task smoke-darwin       # macOS real-window smoke (TextEdit)
task smoke-darwin-apps  # macOS per-app Move/Focus matrix (Chrome, Safari, VSCode, ...)
task smoke-windows      # Windows real-window smoke (Notepad)
task release -- 0.x.0   # Local release: bump → build → publish 6 npm pkgs → tag → GH release
```

## Go library

```go
import "github.com/muthuishere/windowctl"

windows, err := windowctl.ListWindows(windowctl.Filter{Title: "chrome"})
monitors, err := windowctl.ListMonitors()

match := windowctl.Match{App: "Google Chrome"}

// Move via zone (auto-resolves current monitor when monitorID is nil)
err = windowctl.MoveZone(match, nil, "2B")

// Move via coordinates (absolute when monitorID is nil; otherwise relative)
err = windowctl.MoveCoords(match, nil, windowctl.Rect{X: 0, Y: 0, W: 960, H: 1080})

// Resize in place
err = windowctl.Resize(match, 900, 700)

// Focus
err = windowctl.Focus(match)

// Bulk move — same semantics as `windowctl batch`. Optional fields
// (Monitor, X, Y, W, H) are `*int` so callers can distinguish "unset"
// from "zero"; the JSON form omits unset keys for the same reason.
mon := 2
x, y, w, h := 0, 25, 1920, 1055
results := windowctl.Batch([]windowctl.BatchEntry{
    {App: "Ghostty", Monitor: &mon, Zone: "2B"},
    {App: "Code",    X: &x, Y: &y, W: &w, H: &h},
})
for _, r := range results {
    if r.Err != nil { /* per-entry diagnostic */ }
}

// macOS AX prompt / status
err = windowctl.RequestAccessibility()
trusted := windowctl.CheckAccessibility()
```

## Debugging

Set `WCTL_AX_DEBUG=1` to dump the macOS Accessibility window-resolution walk to stderr. Useful when Move/Focus reports `window <id> is gone from the AX tree` — the dump shows the per-PID AX window list and which match rule (title / geometry / single-window) was used.

```sh
WCTL_AX_DEBUG=1 windowctl move --app "Google Chrome" --x 100 --y 100 --w 800 --h 600
```

## Natural-language wrapper

If you want to drive `windowctl` from an agent in plain English (*"split chrome left, slack right"*, *"send vscode to my external"*, *"save this layout as work-mode"*) instead of writing flag combinations, there is a companion agent skill:

```sh
npx skills add muthuishere-agent-skills/window-control
```

Repo: [github.com/muthuishere-agent-skills/window-control](https://github.com/muthuishere-agent-skills/window-control). It's the routing + recipe layer; the actual window operations still go through this CLI.

## License

MIT
