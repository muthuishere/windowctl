---
title: Go library
description: The CLI is a thin shell over an importable package.
---

```go
import windowctl "github.com/muthuishere/windowctl"
```

```go
ws, err := windowctl.ListWindows(windowctl.Filter{App: "Google Chrome"})
ms, err := windowctl.ListMonitors()

err = windowctl.MoveZone(windowctl.Target{App: "Google Chrome"}, 2, "2B")
err = windowctl.MoveCoords(windowctl.Target{Title: "Terminal"}, 0, 0, 960, 1080)
err = windowctl.Resize(windowctl.Target{App: "Code"}, 900, 700)
err = windowctl.Focus(windowctl.Target{Title: "jira"})

ok, err := windowctl.CheckAccessibility()    // never prompts
err = windowctl.RequestAccessibility()       // prompts
```

Everything the CLI does, the library does — the command is argument parsing and
output formatting over these calls, nothing more. If you are writing Go, import
it; do not shell out to a binary you then have to ship and locate.

## Where the logic lives

The per-OS layer implements six primitives: list windows, list monitors, move,
focus, and the two permission calls. Everything expressible as a composition of
those — zone maths, N:M splits, majority-overlap monitor resolution, filter
matching, the relative-vs-absolute coordinate rule — lives in the portable
package. So the semantics are identical on macOS, Windows and Linux, and they
are unit tested once rather than three times.

## Module

Go 1.24. Dependencies: `golang.org/x/sys`, and `gorilla/websocket` for the
device layer. macOS builds use CGO against the system frameworks; Linux and
Windows builds are CGO-free.
