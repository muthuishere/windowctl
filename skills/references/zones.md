---
name: window-ctl-skill
---

# Zones — cheatsheet

`windowctl move --zone <z>` accepts two zone grammars: a fixed enum and
an N-way split. Both compute against the resolved monitor's rect.

## Enum zones (halves + quarters)

| Zone | Description       | Rect math (W, H = monitor size)            |
|------|-------------------|--------------------------------------------|
| `1A` | Left half         | `(0, 0, W/2, H)`                           |
| `1B` | Right half        | `(W/2, 0, W - W/2, H)`                     |
| `2A` | Top-left quarter  | `(0, 0, W/2, H/2)`                         |
| `2B` | Top-right quarter | `(W/2, 0, W - W/2, H/2)`                   |
| `2C` | Bottom-left       | `(0, H/2, W/2, H - H/2)`                   |
| `2D` | Bottom-right      | `(W/2, H/2, W - W/2, H - H/2)`             |

Zones are case-insensitive (`2b` works). Anything outside this set
returns `zone: invalid enum "..."` and exits non-zero.

## Split zones (`N:M`)

`N:M` divides the monitor width into `N` equal vertical columns and
places the window in column `M` (1-indexed). Every column spans the
full monitor height.

| Split | Description                   |
|-------|-------------------------------|
| `2:1` | Left half (same as `1A`)      |
| `2:2` | Right half (same as `1B`)     |
| `3:1` | First third                   |
| `3:2` | Middle third                  |
| `3:3` | Last third                    |
| `4:2` | Second column of a 4-way split|

Math: `cellW = W/N`, `x = monitor.x + (M-1)*cellW`, `y = monitor.y`,
`w = cellW`, `h = H`.

Constraints: `N >= 1`, `1 <= M <= N`. Anything else returns
`zone: invalid split "..."` and exits non-zero.

## Picking enum vs split

- "left half / right half" → `1A` / `1B` (or `2:1` / `2:2`).
- "top-left quarter" → `2A` (no split equivalent — splits are vertical
  columns only, no horizontal halving).
- "first third / middle third / last third" → `3:1` / `3:2` / `3:3`.
- "third column of a 4-way split" → `4:3`.
- Half-then-half ("right half, top quarter") → `2B`.

## What `windowctl` does NOT support

- **Horizontal splits beyond quarters.** `2A..2D` are the only
  half-height zones. There's no `3A..3D` for thirds-tall.
- **Free-form grids.** `4:3` is column 3 of a 4-way split, not row 3
  of a 4-row grid.
- **Percentages or fractions** (`50%`, `1/2`) — use the enum or split
  forms.

For anything outside this grammar, fall back to absolute coords
(`--x --y --w --h`) — see `move.md` recipe `MOV-C-1`.
