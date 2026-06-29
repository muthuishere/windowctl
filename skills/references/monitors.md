---
name: window-ctl-skill
---

# Monitors — recipes

Covers `windowctl monitors list` and the heuristics for mapping
natural-language monitor descriptions ("the laptop", "the external",
"where my cursor is", "where I'm typing") to the integer IDs the
`move` / `resize` commands expect.

**Output schema (JSON):**
```json
[{ "ID": 1, "X": 0, "Y": 0, "Width": 1920, "Height": 1080,
   "Primary": true, "Active": false, "Focused": false },
 { "ID": 2, "X": 1920, "Y": 0, "Width": 1920, "Height": 1080,
   "Primary": false, "Active": true, "Focused": true }]
```

**ID model:**
- IDs are **1-indexed** (`1, 2, 3, ...`), assigned by ascending
  `(X, Y)` origin. The leftmost display is always `1`, regardless of
  which is primary, regardless of system enumeration order. Stable
  across reboots and re-plugs.
- `--monitor 0` is invalid for moves. The per-window `Monitor: 0`
  in `windows list` means "off every display" (off-screen / hidden).

**Coordinate model:**
- `X`, `Y` are the monitor's origin in the **global virtual desktop**
  — not necessarily `(0, 0)` for monitor 1 (depends on the
  arrangement set in System Settings → Displays).
- A `--monitor <id>` flag on `move` makes coordinate args relative to
  that monitor's `(X, Y)`; without it, coords are absolute.

**ACTIVE vs FOCUSED:**
- `Active: true` — cursor is currently over this monitor.
- `Focused: true` — frontmost window's centroid is on this monitor.
- They are independent: you can mouse over one display while typing
  into a window on another.

**User-visible formatting (family default):**
- For lists: one line per monitor with the highlight flags.
  `1: 1920x1080  origin (0, 0)     [primary]`
  `2: 1920x1080  origin (1920, 0)  [active, focused]`
  `3: 1024x640   origin (3840, 30)`

---

## MON-L-1: List every monitor

**When to use:** "list monitors", "what displays do I have", "show my
screens".

**Command:**
```bash
windowctl monitors list --json
```

**Expected response:** JSON array per the schema above.

**Common errors:** None expected.

**User-visible formatting:** As above. Cache the result in session
context (`monitors`) — refresh only if the user mentions plugging /
unplugging a display, or if a subsequent move's clamp error suggests
the layout shifted.

---

## MON-R-1: Resolve "the laptop / built-in display"

**When to use:** Any heuristic phrase referring to the built-in
display: "the laptop", "the macbook screen", "the built-in".

**Call sequence:**
1. `windowctl monitors list --json`.
2. Pick `Primary: true` AND smallest area among primaries. Built-in
   laptop displays are almost always primary; on docked-with-lid-
   closed setups the primary may be external — fall back to "the
   smallest non-zero monitor with origin (0, 0)" if no primary
   qualifies.

**Expected response:** A single monitor `ID`. Pass it as `--monitor
<id>` to `move`.

**Common errors:** Multiple primaries on Linux/X11 — pick the lowest
`ID`.

**Synthesis:** *"Placing on your built-in display (1920x1080)."*

---

## MON-R-2: Resolve "the external / second monitor"

**When to use:** "the external", "second monitor", "the big one",
"the 4K", "send to my other screen".

**Call sequence:**
1. `windowctl monitors list --json`.
2. Filter to `Primary: false`.
   - "the external" with one non-primary → use it.
   - "the big one" → largest area (`Width * Height`) among
     non-primary.
   - "the 4K" → first non-primary with `Width >= 3840`.
   - With multiple non-primaries and no further hint, ask which by
     listing `ID: WxH @ (X, Y)`.
3. Pass the resolved `ID` as `--monitor <id>`.

**Expected response:** A single monitor `ID`.

**Common errors:** 0 non-primary monitors → tell the user *"only one
display is attached"* and ask whether they meant the primary.

**Synthesis:** *"Placing on the external (3840x2160)."*

---

## MON-R-3: Resolve "the current monitor" (auto-resolve)

**When to use:** Default for any move command without a monitor hint
("put chrome on the right half") — the user wants the window's
current monitor, not monitor 1.

**Call sequence:** None — omit `--monitor` from the `move` invocation.
`windowctl` auto-resolves to the monitor containing the majority of
the window's visible area.

**Expected response:** N/A — nothing to call here.

**Common errors:** None.

**Synthesis:** Don't mention the resolution. Just act.

---

## MON-R-4: Resolve "the active monitor / where my cursor is"

**When to use:** "the monitor I'm pointing at", "where my cursor
is", "the active screen".

**Call sequence:**
1. `windowctl monitors list --json`.
2. Pick the entry with `Active: true`.

**Expected response:** Exactly one `ID` (cursor lives on one
monitor at a time).

**Common errors:** None — `Active` is set deterministically per
poll.

**Synthesis:** *"Active monitor is <id> (<WxH>)."*

---

## MON-R-5: Resolve "the focused monitor / where I'm typing"

**When to use:** "the screen with my current window", "where I'm
typing", "where the focused app is", "this monitor" (when the
user is interacting with a window, not just hovering).

**Call sequence:**
1. `windowctl monitors list --json`.
2. Pick the entry with `Focused: true`.

**Expected response:** Exactly one `ID`. Independent from
`Active` — the cursor may be elsewhere.

**Common errors:** Zero `Focused` monitors when no window is
frontmost (rare — fullscreen Mission Control / Spaces transitions).
Re-poll after a moment, or fall back to `Primary: true`.

**Synthesis:** *"You're on monitor <id>."*
