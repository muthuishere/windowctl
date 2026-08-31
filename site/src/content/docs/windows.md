---
title: Listing windows
description: Discover what is open, in a table or as JSON.
---

```sh
windowctl windows list
windowctl windows list --app "Google Chrome"
windowctl windows list --title term --json
```

```json
[
  {
    "ID": 24815, "App": "Google Chrome", "Title": "windowctl — GitHub",
    "Bounds": { "X": 1920, "Y": 25, "W": 1920, "H": 1055 },
    "Monitor": 2, "Focused": true
  }
]
```

## Filter semantics

| Flag      | Match                                                        |
|-----------|--------------------------------------------------------------|
| `--title` | case-insensitive **substring**                               |
| `--app`   | case-insensitive **exact** match on the OS-reported app name |

The OS-reported name is not always the name on the icon: VS Code reports
`Code`, Chrome reports `Google Chrome`. List first, filter second.

An empty filter value is rejected (`--title cannot be empty`) rather than
silently matching everything — the difference matters when the next command in
your script moves whatever came back.

## Menu-bar chrome is filtered out

Status items and menu-bar extras are windows too, and listing them is noise.
They are excluded from `windows list`.

## No permission needed

Listing never triggers the macOS Accessibility prompt. Only `move`, `resize`,
`focus` and `batch` do.
