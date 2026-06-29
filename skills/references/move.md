---
name: window-ctl-skill
---

# Move — recipes

Covers `windowctl move` — placing a window into a zone, a split, or
absolute / monitor-relative coordinates.

**Pre-flight:** Always run `windowctl windows list --title "<filter>"
--json` first (see `windows.md` WIN-L-5). On 0 matches the CLI
returns `no window matched filter --title="..."`; on N matches it
silently picks the first.

**Mutex & validation:**
- `--zone` and `--x/--y/--w/--h` cannot be combined; the CLI exits
  non-zero (`--zone and --x/--y/--w/--h are mutually exclusive`).
- `--w` and `--h` must be `> 0` in coord mode (`--w and --h must be > 0
  in coord mode`).
- `--monitor 0` and `--monitor -1` are rejected (`--monitor must be >= 1
  (use 1, 2, 3, ...; omit to auto-resolve)`). Omit the flag to
  auto-resolve.
- `--monitor 99` (any out-of-range positive ID) → `monitor: invalid
  monitor ID 99`, exit 1.

**Coordinate interpretation (the most common footgun):**
- WITH `--monitor <id>` → `--x/--y` are RELATIVE to that monitor's
  origin. Window lands at `(monitor.X + x, monitor.Y + y)`.
- WITHOUT `--monitor` → `--x/--y` are ABSOLUTE in the global virtual
  desktop.

**Matcher:** `--title` is case-insensitive substring; `--app` is
case-insensitive exact match. Empty values are rejected. At least
one is required.

**OS clamps:** When the OS or app refuses the requested geometry
(Chrome's minimum window size, sandbox limits, fullscreen-mode),
`move` exits non-zero with:
```
windowctl: requested 960x1080 at (0,0), OS clamped to 960x798 at (1196,217)
  (likely a minimum-window-size constraint)
```
The window did move, but not where asked. **Trust the
`Width`/`Height` in this message; the `(X, Y)` may be stale** —
re-read with `windows list --json` for the authoritative position.
Do NOT auto-retry; surface the error verbatim and ask the user.

**User-visible formatting (family default):**
- On clean exit 0, `windowctl move` prints nothing. Confirm to the
  user with: *"Moved <app> to <zone or coords> on monitor <id |
  current>."*
- On clamp (exit 1), surface the CLI message and re-state the
  authoritative position from a follow-up `windows list`.

**macOS:** First `move` ever from a given parent process returns
`Accessibility permission denied` until trust is granted. Route to
`permissions.md` then retry. If a move reports `window <id> is gone
from the AX tree`, retry with `WCTL_AX_DEBUG=1` and surface the
stderr dump.

---

## MOV-Z-1: Move into an enum zone (current monitor)

**When to use:** "put chrome on the left half", "snap terminal to
the right half", "place editor in the top-right quarter".

**Command:**
```bash
windowctl move --title "<filter>" --zone <1A|1B|2A|2B|2C|2D>
```

Zone meanings live in `zones.md`.

**Expected response:** Empty stdout, exit 0.

**Common errors:**
- `zone: invalid enum "..."` → user phrasing didn't map to the enum.
  Re-check `zones.md`; consider a split (MOV-S-1).
- `no window matched filter --title="..."` → re-run WIN-L-5 and ask.
- `Accessibility permission denied` (macOS) → route to
  `permissions.md`.
- Clamp message → see "OS clamps" above.

**User-visible formatting:** *"Moved <app> to the <zone-description>
of your current monitor."*

---

## MOV-Z-2: Move into an enum zone on a specific monitor

**When to use:** "put slack on monitor 2, top half", "chrome
bottom-right of the external".

**Call sequence:**
1. `windowctl monitors list --json` (cached).
2. Resolve target monitor `ID` (see `monitors.md` MON-R-*).
3. `windowctl move --title "<filter>" --monitor <id> --zone <z>`.

**Expected response:** Empty stdout, exit 0.

**Common errors:**
- `monitor: invalid monitor ID <id>` → ID out of range (or `0`).
  Re-list monitors; the set may have changed.
- `--monitor must be >= 1 ...` → user passed `0` or a negative.
- Same AX / no-match / clamp errors as MOV-Z-1.

**User-visible formatting:** *"Moved <app> to <zone> of monitor
<id> (<WxH>)."*

---

## MOV-S-1: Move into an N-way split column

**When to use:** "first third of my screen", "middle column of a
3-way split", "second of four", "split into thirds and put chrome
in the middle".

**Command:**
```bash
windowctl move --title "<filter>" [--monitor <id>] --zone N:M
```

Constraints: `N >= 1`, `1 <= M <= N`.

**Expected response:** Empty stdout, exit 0. Window spans the
column's full monitor height.

**Common errors:**
- `zone: invalid split "0:1": N must be a positive integer`.
- `zone: invalid split "3:5": M must satisfy 1 <= M <= N`.
- `monitor: invalid monitor ID <id>`.
- AX denied / no match / clamp.

**User-visible formatting:** *"Moved <app> to column <M> of <N> on
<monitor>."*

---

## MOV-C-1: Move with absolute coordinates

**When to use:** Pixel-exact placement that doesn't fit the zone
grammar — recording setups, fixed-position overlays, replicating an
exact prior placement.

**Command:**
```bash
windowctl move --title "<filter>" --x <px> --y <px> --w <px> --h <px>
```

Coordinates are **absolute** in the global virtual desktop. To find
the desktop bounds, sum the monitor rects from
`monitors list --json`.

**Expected response:** Empty stdout, exit 0.

**Common errors:**
- `--w and --h must be > 0 in coord mode` (exit 2).
- Window lands off-screen → user supplied coords outside any monitor
  rect. Verify against `monitors list`.
- Clamp message — see "OS clamps" above.

**User-visible formatting:** *"Moved <app> to absolute (X, Y) at
WxH."*

---

## MOV-C-2: Move with monitor-relative coordinates

**When to use:** "the right half of monitor 2 minus a margin", "200
px in from the top-left of the external display" — when the user
thinks in per-monitor coordinates.

**Command:**
```bash
windowctl move --title "<filter>" --monitor <id> --x <px> --y <px> --w <px> --h <px>
```

With `--monitor`, `(x, y)` is relative to that monitor's origin —
`(0, 0)` is the monitor's top-left. The CLI computes
`(monitor.X + x, monitor.Y + y)` internally.

**Expected response:** Empty stdout, exit 0.

**Common errors:**
- Same `--w/--h must be > 0` validation as MOV-C-1.
- `x + w` > monitor width → off-screen risk.
- Forgetting `--monitor` and passing per-monitor coords as absolute
  is the #1 cause of "the window vanished".

**User-visible formatting:** *"Moved <app> to (X, Y, WxH) on monitor
<id>."*

---

## MOV-Z-3: Move using auto-resolved monitor

**When to use:** Default for any zone command without a monitor hint
("put chrome on the right half") — the user wants the window's
current monitor.

**Command:**
```bash
windowctl move --title "<filter>" --zone <z>
```

**Expected response:** Empty stdout, exit 0. The CLI auto-resolves
the monitor containing the majority of the window's current visible
area.

**Common errors:** Same as MOV-Z-1.

**User-visible formatting:** *"Moved <app> to <zone>."* (Don't say
"on monitor X" — the user didn't specify one.)

---

## MOV-F-1: Move "this window" (the focused one)

**When to use:** "snap this to the right half", "send the current
window to monitor 2", "move it" with no explicit app name.

**Call sequence:**
1. `windowctl windows list --json`.
2. Pick the window with `Focused: true` (see `windows.md` WIN-L-6).
3. `windowctl move --title "<focused.Title>" --zone <z>` (or
   whatever placement the user asked for).

**Expected response:** Empty stdout, exit 0.

**Common errors:**
- Zero focused windows (Spaces transition). Re-poll once before
  giving up.
- Title contains characters that look like glob/quotes — use
  double-quotes around the substring; if still ambiguous, also
  pass `--app "<focused.App>"` to disambiguate.

**User-visible formatting:** *"Snapped <App> to <zone>."*
