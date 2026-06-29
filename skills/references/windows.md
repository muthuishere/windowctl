---
name: window-ctl-skill
---

# Windows — recipes

Covers `windowctl windows list` across macOS, Windows, and Linux.

The list is now filtered server-side to real application windows
only (macOS `kCGWindowLayer == 0`); a typical session returns ~5–15
entries. Menubar widgets, status items, the Dock, Spotlight, Control
Center, AltTab, and Window Server menubars are excluded.

**Filter semantics (load-bearing):**
- `--title <s>` — case-insensitive **substring** match against the
  window title. Empty string is rejected (`--title cannot be empty`,
  exit 2).
- `--app <s>` — case-insensitive **exact** match against the
  application name. Empty string is rejected.

**Output schema (JSON):**
```json
[{ "ID": "227", "Title": "Reqsume - AI Resume Builder",
   "App": "Google Chrome", "PID": 4356,
   "Monitor": 2, "Focused": true,
   "Bounds": { "X": 1920, "Y": 25, "Width": 1920, "Height": 1055 } }]
```

- `Monitor` is the **1-indexed** ID of the display containing the
  window's centroid (matches `monitors list`). `0` means the window
  is off every display (off-screen / minimized).
- `Focused: true` for the frontmost window on the focused monitor —
  this is how you ask "which window is the user currently using?"
- `Bounds` uses `Width` / `Height` (not `W` / `H`) — same shape as
  `monitors list`.

**User-visible formatting (family default):**
- For lists, group by `App`, show `<App> — <count>` lines, then 1–3
  sample titles per app. With ~5–15 entries the full list is fine
  to show inline.
- For single-window output, show: `ID  App  Title  Monitor  WxH+X+Y`.

---

## WIN-L-1: List every open window

**When to use:** "list windows", "show every open window", "what's
open".

**Command:**
```bash
windowctl windows list --json
```

**Expected response:** JSON array per the schema above. Typically
5–15 entries (server-side filtered to real application windows).

**Common errors:** None — `windows list` does not require AX
permission and never returns `Accessibility permission denied`.

**User-visible formatting:** Group by `App`. Print
`Total: <N> windows across <M> apps`, then a sorted-by-count list.
At 5–15 entries the full list fits inline.

---

## WIN-L-2: Filter by title substring

**When to use:** "find my chrome windows", "any window with 'jira'
in the title".

**Command:**
```bash
windowctl windows list --title "<substring>" --json
```

**Expected response:** JSON array filtered to titles containing
`<substring>` (case-insensitive). Empty array means no matches.

**Common errors:**
- `windowctl windows list: --title cannot be empty` (exit 2) →
  you passed an empty string. Don't.
- Empty array → user's substring matches nothing. Re-run without
  `--title` and offer the closest app guesses based on what IS open.

**User-visible formatting:** If 1 match, print `ID App Title Monitor
WxH+X+Y`. If 2–10, table. If >10, summarize and ask the user to
narrow.

---

## WIN-L-3: Filter by application (exact app name)

**When to use:** "all chrome windows", "every slack window".

**Command:**
```bash
windowctl windows list --app "<App Name>" --json
```

**Expected response:** JSON array of windows owned by exactly that
app. `--app` is **exact** (not substring) — `chrome` will NOT match
`Google Chrome`.

**Common errors:**
- `--app cannot be empty` (exit 2).
- Empty array → wrong app name. Run `windowctl windows list --json`
  and read `.[].App` to see canonical names. macOS gotchas:
  `Google Chrome` (not `Chrome`), `Visual Studio Code` reports as
  `Code`, `iTerm2` (not `iTerm`).

**User-visible formatting:** Same table as WIN-L-2.

---

## WIN-L-4: Filter by title AND app

**When to use:** "my chrome window with the github tab", "the vscode
window for the windowctl repo".

**Command:**
```bash
windowctl windows list --title "<substring>" --app "<App Name>" --json
```

**Expected response:** Intersection of WIN-L-2 and WIN-L-3.

**Common errors:** Empty array — drop `--app` first to see if the
app name is wrong; drop `--title` next.

**User-visible formatting:** Single line per match.

---

## WIN-L-5: Pre-flight before move / resize / focus

**When to use:** Internal — every recipe in `move.md`, `resize.md`,
and `focus.md` should run this first.

**Command:**
```bash
windowctl windows list --title "<filter>" --json
```

**Expected response:**
- 1 match → proceed with the action.
- 0 matches → ask the user; do NOT run the action with a filter
  that the CLI will reject as `no window matched filter --title=...`.
- N matches → list them with ID + Title + Monitor + Bounds and ask
  which. The CLI itself silently picks the first match — rarely
  what the user wants.

**Common errors:** None — pure read.

**User-visible formatting:** Skip when match count is 1; just
proceed. On 0 or many, present and stop.

---

## WIN-L-6: Find the focused window

**When to use:** "this window", "the one I'm using", "the current
one", "what's frontmost". Any time the user says "this" without
naming an app.

**Command:**
```bash
windowctl windows list --json
```

Then filter client-side for `Focused: true`. Exactly one window
should be focused at any time.

**Expected response:** Exactly one match. Use its `Title` (most
stable across re-enumerations) as the filter for the next
`move` / `resize` / `focus` call.

**Common errors:**
- Zero focused windows during fullscreen / Spaces transitions
  (rare). Re-poll after a moment.
- Multiple windows with `Focused: true` — should not happen; if it
  does, surface the IDs and ask.

**User-visible formatting:** Don't print the resolution step. Just
act on the focused window: *"Snapping <App> — <Title>."*
