---
title: Layouts in one call
description: Apply a whole desk arrangement with windowctl batch.
---

`batch` reads a JSON array from stdin or `--file` and applies every entry.

```sh
windowctl batch --file work-mode.json --json
cat work-mode.json | windowctl batch
```

```json
[
  { "app": "Google Chrome", "monitor": 2, "zone": "2B" },
  { "app": "Code",          "monitor": 1, "zone": "1A" },
  { "title": "Terminal",    "monitor": 1, "zone": "1B" },
  { "app": "Slack",         "monitor": 2, "zone": "2C" }
]
```

Each entry takes the same targeting as `move`: `app` or `title` to select,
`monitor` plus `zone`, or explicit `x`/`y`/`w`/`h`.

## Partial success is reported, not hidden

Every entry reports its own outcome. A layout where three windows moved and one
was clamped tells you which one and why, and the exit code reflects it. That is
the difference between a layout tool you can put in a login script and one you
have to watch.

## Why this instead of a loop

A loop of `move` calls re-enumerates every window on every iteration and gives
you N separate exit codes to reason about. `batch` enumerates once and hands
back one structured result — which is also what makes it the right primitive for
an agent to call.
