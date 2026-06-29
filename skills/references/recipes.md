---
name: window-ctl-skill
---

# Composite recipes

Multi-step layouts that compose primitives from `move.md`, `focus.md`,
`monitors.md`, and `windows.md`. Each recipe is a sequence — run them
in order, surface intermediate failures, stop on first error.

**Convention:** A composite is a recipe whose value comes from the
sequence, not from any single step. They live here (not in the
per-family files) so a search for "split chrome and slack 50/50"
finds the whole flow, not just `MOV-Z-1`.

---

## CMP-1: Split two apps 50/50 on the current monitor

**When to use:** "split chrome and slack 50/50", "put terminal on the
left and editor on the right", "side by side".

**Sequence:**
1. `windowctl windows list --app "<App A>" --json` — confirm match (WIN-L-5).
2. `windowctl windows list --app "<App B>" --json` — confirm match.
3. `windowctl move --app "<App A>" --zone 1A`
4. `windowctl move --app "<App B>" --zone 1B`

**Synthesis:** *"Split <App A> (left) and <App B> (right) on your
current monitor."*

**Common errors:**
- Either app not running → tell the user which is missing and stop.
  Don't move only one.
- AX denied on macOS → route to `permissions.md` (PERM-2). After
  grant, retry from step 3.

---

## CMP-2: Three-way IDE layout (editor + terminal + browser)

**When to use:** "set up my IDE layout", "editor middle, terminal
right, browser left", "three-way split with code in the middle".

**Sequence:**
1. Confirm all three apps have at least one window each (WIN-L-5 ×3).
2. `windowctl move --app "<Editor>" --zone 3:2`   # middle third
3. `windowctl move --app "<Terminal>" --zone 3:3` # right third
4. `windowctl move --app "<Browser>" --zone 3:1`  # left third

**Synthesis:** *"Three-way split: <Browser> | <Editor> | <Terminal>."*

**Common errors:**
- Any one filter returns 0 → list what's open and ask. Don't apply a
  partial layout.

---

## CMP-3: Send an app to the external display, full screen-equivalent

**When to use:** "throw chrome on my external", "send slack to the
big monitor and maximize".

**Sequence:**
1. `windowctl monitors list --json` (cache).
2. Resolve external monitor `id` via MON-R-2 in `monitors.md` (uses
   1-indexed IDs; the leftmost display is always `1`, so anything
   with `id >= 2` is a candidate "external").
3. `windowctl move --app "<App>" --monitor <id> --zone 1A` (full
   width if combined with the next step) OR use coords to fill the
   whole monitor:
   `windowctl move --app "<App>" --monitor <id> --x 0 --y 0 --w <mon.width> --h <mon.height>`

**Synthesis:** *"Sent <App> to your external (<WxH>) and filled the
display."*

**Common errors:**
- 0 external monitors (only `id: 1` exists) → see MON-R-2.
- `windowctl` does not have a "maximize" zone — use coords to fill,
  or accept that `1A` only covers the left half.

---

## CMP-4: Stack windows on monitor 1, focus the top one

**When to use:** "bring all my chrome windows to the laptop screen
and focus the github one".

**Sequence:**
1. `windowctl windows list --app "<App>" --json` — get all windows of
   that app.
2. For each window: `windowctl move --title "<window.title>"
   --monitor 1 --zone 1A` (or whatever zone makes sense). Monitor IDs
   are 1-indexed; `1` is always the leftmost display.
3. `windowctl focus --title "<the one to focus>"`.

**Synthesis:** *"Moved <N> <App> windows to monitor 1 and focused
the <X> one."*

**Common errors:**
- Step 2 moves them all into the same zone, so they stack. If the
  user wanted them tiled, switch to per-window splits (`N:1`,
  `N:2`, ...) where N is the window count.

---

## CMP-5: Pull every window off the unplugged monitor

**When to use:** "I just unplugged my external — move everything to
the laptop", "the second monitor is gone, rescue my windows".

**Sequence:**
1. `windowctl monitors list --json` — confirm only one monitor now
   (it will be `id: 1`, the surviving display).
2. `windowctl windows list --json` — find any window whose `monitor`
   field references an ID that no longer exists, OR whose `bounds.x`
   is outside the surviving monitor.
3. For each: `windowctl move --title "<title>" --monitor 1 --zone 1A`
   (or some sensible default).

**Synthesis:** *"Recovered <N> stranded windows to monitor 1."*

**Common errors:**
- Some windows may report `monitor: 1` already but still have
  off-screen `bounds` — the OS can lag updating the field. Use
  `bounds` as the source of truth.
- AX denied → PERM-2.

---

## CMP-6: Mirror a layout across a monitor swap

**When to use:** "put my work layout on the external instead of the
laptop" — same relative arrangement, different monitor.

**Sequence:**
1. `windowctl monitors list --json`.
2. Resolve target monitor `id` (MON-R-2).
3. For each app in the layout, re-run its zone command with
   `--monitor <id>`.

**Synthesis:** *"Re-applied your layout on monitor <id>."*

**Common errors:**
- The new monitor's aspect ratio differs — a layout that looked good
  on 16:9 may look squished on 16:10. Zones are proportional, so
  this is usually fine, but warn for extreme ratios.

---

## CMP-7: Snap THIS window (the focused one)

**When to use:** "snap this to the right half", "send this to
monitor 2", "make the current window full-screen-ish" — anything
where the user says "this", "the current window", or "the one
in front" without naming an app.

**Sequence:**
1. `windowctl windows list --json` — find the entry with
   `Focused: true`. That is "this window".
2. `windowctl move --title "<that window's title>" --zone <z>`
   (or add `--monitor <id>` for a cross-monitor snap).

**Synthesis:** *"Snapped <App> — <Title> to <zone>."*

**Common errors:**
- Zero windows with `Focused: true` — can happen mid-Spaces or
  Mission Control transition. Re-poll `windows list --json` once;
  if still empty, ask the user which window they meant.
- User said "this" while no window is frontmost (e.g. Finder desktop
  in focus) → ask them to name the window or click into it first.

---

## CMP-8: Resize THIS window in place

**When to use:** "resize this to 900x700", "make this smaller",
"shrink the current window to 1200x800" — in-place size change,
keep X/Y where they are.

**Sequence:**
1. `windowctl windows list --json` — find the entry with
   `Focused: true`. Capture its current `bounds.w` and `bounds.h`
   so the synthesis can report old vs new.
2. `windowctl resize --title "<that window's title>" --w <W> --h <H>`.
   `resize` keeps X/Y untouched — only Width/Height change.

**Synthesis:** *"Resized <App> from <oldWxH> to <newWxH>."*

**Common errors:**
- OS clamp (`exit 1` with a descriptive message) — Chrome refuses
  widths below ~500 px, some apps enforce min heights, etc. Surface
  the clamp error verbatim, then re-run `windows list --json` to
  show the user what size the window actually ended up at (often
  unchanged) so they can pick a valid target.
- Zero windows with `Focused: true` → same fallback as CMP-7
  (re-poll once, then ask).

---

## CMP-9: Apply a multi-window layout in one call

**When to use:** the user names multiple placements in a single turn —
"split chrome and slack 50/50, put terminal on monitor 2 bottom-half,
send my browser to the external", "set up these four windows for me",
or any "do all of these placements" instruction. Prefer `batch` over a
sequence of `move` calls: one CLI invocation, one summary, partial
failures don't abort the rest of the layout, less terminal noise.

**Sequence:**
1. Pre-flight every named app:
   `windowctl windows list --app "<App>" --json` for each. If any
   returns 0 windows, fail-fast — list which apps are missing and
   ask. Do not send a partial batch.
2. Build a JSON array of entries. **Field names are lowercase**
   (`app`, `title`, `monitor`, `zone`, `x`, `y`, `w`, `h`) — this is
   different from the PascalCase used in `windows list --json`
   output. Per-entry rules: at least one of `title`/`app` required;
   target is EITHER `zone` OR all four `x/y/w/h`; `monitor` is
   1-indexed and optional.
3. Pipe to `windowctl batch` (heredoc) or write to a temp file and
   pass `--file`. Add `--json` so each entry is reported as
   `{ "entry": ..., "ok": true }` or `{ "entry": ..., "error": "..." }`.

   ```sh
   windowctl batch --json <<'EOF'
   [
     { "app": "Google Chrome", "zone": "1A" },
     { "app": "Slack",         "zone": "1B" },
     { "app": "Terminal",      "monitor": 2, "zone": "2C" },
     { "app": "Arc",           "monitor": 2, "zone": "1A" }
   ]
   EOF
   ```
4. Parse per-entry results. Exit code 0 = all passed, 1 = at least
   one failed (rest still applied), 2 = invalid input (fix the JSON
   and retry — nothing was applied).

**Synthesis:** *"Placed <N>/<total>: chrome left, slack right,
terminal bottom-half of monitor 2, browser to external. Failed:
<name> (<per-entry error>)."* Name each entry — don't dump raw
stdout.

**Common errors:**
- Pre-flight finds a missing app → ask before sending the batch;
  don't apply a partial layout silently.
- Per-entry OS clamp (Chrome's min width, etc.) → surface that one
  entry's error from the `--json` output, leave the rest as placed.
- Exit code 2 (invalid input) → the JSON failed validation (missing
  `app`/`title`, both `zone` and coords, etc.). Fix and retry.

---

## CMP-10: Save and restore a named layout

**When to use:** "save my current arrangement as <name>", "restore my
<name> layout", "what does my work layout look like", "snapshot this
setup". Two-phase recipe — save once, restore many times.

**Sequence (save):**
1. `windowctl windows list --json` — capture every window.
2. For each window, project the PascalCase fields into the lowercase
   batch schema. **The case translation is the gotcha:**
   `{App, Bounds: {X, Y, Width, Height}}` → `{app, x, y, w, h}`.
   Use absolute coords (not zones) so the layout is exact. Skip
   `monitor` — coords already pin the window to a display.
3. Write the resulting JSON array to
   `~/.windowctl/layouts/<name>.json` (mkdir -p the parent first).

**Sequence (restore):**
1. `windowctl batch --file ~/.windowctl/layouts/<name>.json --json`.
2. Parse per-entry output the same way as CMP-9.

**Synthesis:**
- Save → *"Saved <N> windows to <name>."*
- Restore → *"Restored <name>: <N>/<total> placed. Failed: <names>."*

**Common errors:**
- On restore, layout includes apps that aren't running → that entry
  reports a no-match error in the `--json` output. Surface it and
  continue; the rest still apply.
- Window minimum-size clamps on restore (Chrome etc.) — same as
  CMP-9, surface per-entry, leave the rest as placed.
- Layout file missing or malformed → `batch` exits 2; tell the user
  the named layout doesn't exist (or got corrupted) and offer to
  re-save from the current arrangement.
