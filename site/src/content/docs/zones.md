---
title: Zones & splits
description: The rectangle vocabulary — halves, quarters, and any N:M column split.
---

A zone is a name for a rectangle inside a monitor. You never compute pixels.

## Predefined zones

| Zone | Rectangle            |
|------|----------------------|
| `1A` | Left half            |
| `1B` | Right half           |
| `2A` | Top-left quarter     |
| `2B` | Top-right quarter    |
| `2C` | Bottom-left quarter  |
| `2D` | Bottom-right quarter |

## Split zones — `N:M`

Divide the screen into **N** equal columns, place the window in column **M**:

```
2:1  →  left half          3:1  →  first third
2:2  →  right half         3:2  →  middle third
                           3:3  →  last third
```

`4:3`, `5:2` and so on all work — N is not capped at three.

## Which monitor?

```sh
windowctl move --app "Slack" --zone 1A              # the monitor it is already on
windowctl move --app "Slack" --monitor 2 --zone 1A  # move to display 2, then snap
```

Omitting `--monitor` resolves the window's current monitor by **majority
overlap** — the display holding most of the window's area, not the one under its
top-left corner. That is the difference between "snap it where it is" and
"snap it somewhere surprising" for a window straddling two displays.

## Exact coordinates, when you need them

```sh
windowctl move --title "chrome" --x 0 --y 0 --w 960 --h 1080          # absolute
windowctl move --app "Code" --monitor 2 --x 100 --y 100 --w 800 --h 600  # relative to monitor 2
```

The rule: coordinates are **absolute** on their own, and **relative to the
monitor's origin** when `--monitor` is present. `--w`/`--h` must be > 0.
