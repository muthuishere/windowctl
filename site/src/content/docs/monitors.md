---
title: Monitors
description: 1-indexed displays, which one is active, and which window has focus.
---

```sh
$ windowctl monitors list
ID  NAME              BOUNDS                PRIMARY  ACTIVE
1   Built-in Display  0,0 1920x1200         true     false
2   DELL U2723QE      1920,0 3840x2160      false    true
```

- **IDs are 1-indexed, ordered by ascending (X, Y).** The leftmost display is
  always `1`. The ordering is stable across reboots and re-plugs, so a saved
  layout keeps meaning what it meant.
- **`PRIMARY`** is the OS's primary display.
- **`ACTIVE`** is the display the cursor is on right now.

`--monitor 0` is invalid: `0` is the off-screen sentinel used in window output,
not a display.

## Per-window monitor and focus

`windows list` stamps two derived fields:

- **`Monitor`** — the 1-indexed display under the window's centroid, or `0` when
  the window is off-screen.
- **`Focused`** — the frontmost window in z-order on the focused monitor.

Both are computed on the **unfiltered** window list, before your `--title` /
`--app` filter is applied. If they were computed after filtering, "is this
focused?" would change depending on what you asked for — which is exactly the
bug that made the rule explicit.

## Mapping human words to an ID

"The external", "the big one", "where my cursor is", "the laptop screen" are all
answerable from `monitors list --json` — resolve to an integer first, then pass
`--monitor`. Never hardcode a display ID into a script you intend to keep.
