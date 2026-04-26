# windowctl

**windowctl** is a cross-platform command-line interface (CLI) tool for managing desktop windows and monitors in a consistent, scriptable way.

## Overview

Windowctl exposes a unified interface across macOS, Windows, and Linux for:

- Listing windows and monitors
- Filtering windows by title or application name
- Moving windows across monitors
- Applying structured, declarative layouts
- Focusing specific windows

It is distributed both as a **CLI tool** and as a **Go library**, enabling programmatic use in other tools and scripts.

## Installation

### npm (recommended)

```sh
npx windowctl <command>
```

```sh
npm install -g windowctl
```

### Go

```sh
go install github.com/muthuishere/windowctl/src/cmd/windowctl@latest
```

### Binary (GitHub Releases)

Download the appropriate binary for your platform from the [Releases](../../releases) page.

## Usage

### List Windows

```sh
windowctl windows list
windowctl windows list --title "chrome"
windowctl windows list --app "Firefox" --json
```

### List Monitors

```sh
windowctl monitors list
```

### Move a Window

```sh
# Move to monitor 2, top-right quadrant (zone 2B)
windowctl move --title "chrome" --monitor 2 --zone 2B

# Move to the left half of the current monitor (zone 1A)
windowctl move --title "Terminal" --zone 1A

# Move to a split zone (first third of screen)
windowctl move --title "Slack" --zone 3:1

# Move using absolute coordinates
windowctl move --title "chrome" --x 0 --y 0 --w 960 --h 1080
```

### Focus a Window

```sh
windowctl focus --title "jira"
```

### Apply a Layout

```sh
windowctl apply layout.yaml
```

## Zones

### Predefined (Enum) Zones

| Zone | Description      |
|------|------------------|
| 1A   | Left half        |
| 1B   | Right half       |
| 2A   | Top-left quarter |
| 2B   | Top-right quarter|
| 2C   | Bottom-left quarter|
| 2D   | Bottom-right quarter|

### Split Zones (`N:M`)

Divide the screen into N equal columns and place the window in column M.

```
2:1  →  left half
2:2  →  right half
3:1  →  first third
3:2  →  middle third
3:3  →  last third
```

## Platform Support

| Platform | Status          | Backend                        |
|----------|-----------------|--------------------------------|
| macOS    | Supported       | CoreGraphics + Accessibility API |
| Windows  | Supported       | Win32 (user32.dll)             |
| Linux    | Best-effort     | X11 / wmctrl fallback          |
| Wayland  | Limited         | Best-effort only               |

## Building

This project uses [Taskfile](https://taskfile.dev) for build orchestration.

```sh
task build      # Build CLI binary
task test       # Run tests
task lint       # Run linter
task snapshot   # Build snapshot release
task release    # Publish release via GoReleaser
```

## Go Library

windowctl can be used as a Go library:

```go
import windowctl "github.com/muthuishere/windowctl/src"

windows, err := windowctl.ListWindows(windowctl.Filter{Title: "chrome"})
monitors, err := windowctl.ListMonitors()
err = windowctl.Move(match, target)
err = windowctl.Focus(match)
```

> The repository keeps all Go source under `src/`; the package itself is
> still named `windowctl`, so consumers use the import alias above and
> call `windowctl.ListWindows` etc.

## License

MIT
