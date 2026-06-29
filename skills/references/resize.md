---
name: window-ctl-skill
---

# Resize — recipes

Covers `windowctl resize` — changing a window's `Width` / `Height` in
place. The window's current `X` / `Y` (top-left position) is
preserved.

**When to prefer over `move`:**
- The user wants to keep the window where it is and only change its
  size ("make chrome smaller", "resize this to 900x700").
- For "place at top-left of monitor 2 with size WxH" use `move
  --monitor 2 --x 0 --y 0 --w W --h H` instead.

**Pre-flight:** Same as `move` — run `windowctl windows list --title
"<filter>" --json` first to confirm exactly one match
(see `windows.md` WIN-L-5).

**Validation:**
- `--w` and `--h` are required and must be `> 0`. Empty / zero /
  negative → `--w and --h are required and must be > 0`, exit 2.
- `--title` or `--app` is required. Empty values rejected.

**Matcher:** `--title` is case-insensitive substring; `--app` is
case-insensitive exact match.

**OS clamps:** Same surface as `move` — when the requested size hits
a minimum-window-size limit (Chrome ≈ 500 px wide etc.), `resize`
exits non-zero with:
```
windowctl: requested 300x300 at (0,25), OS clamped to 500x350 ...
  (likely a minimum-window-size constraint)
```
Trust `Width` / `Height` in the message; re-read with
`windows list` for the authoritative position.

**User-visible formatting (family default):**
- On exit 0, `windowctl resize` prints nothing. Confirm with:
  *"Resized <app> to <W>x<H>."*
- On clamp, surface the CLI message and the re-read bounds.

---

## RES-1: Resize by app name

**When to use:** "resize chrome to 900x700", "make slack 800 wide
and 600 tall".

**Command:**
```bash
windowctl resize --app "<App Name>" --w <px> --h <px>
```

**Expected response:** Empty stdout, exit 0. Window's `X` / `Y`
unchanged; only `Width` / `Height` set to the requested values
(modulo OS clamps).

**Common errors:**
- `--w and --h are required and must be > 0` → user passed zero,
  negative, or omitted a dim.
- `no window matched filter --app="..."` → wrong app name. See
  `windows.md` WIN-L-3.
- `Accessibility permission denied` (macOS) → `permissions.md`.
- Clamp message — surface and re-read.

**User-visible formatting:** *"Resized <App Name> to <W>x<H>."*

---

## RES-2: Resize by title substring

**When to use:** "resize the github tab to 1200x800", "shrink the
jira window to 600x400".

**Command:**
```bash
windowctl resize --title "<substring>" --w <px> --h <px>
```

**Expected response:** Empty stdout, exit 0.

**Common errors:**
- Multiple matches → CLI silently resizes the first. Pre-flight
  with WIN-L-5 and ask which.
- Same validation / AX / clamp errors as RES-1.

**User-visible formatting:** *"Resized <App> — <Title> to <W>x<H>."*

---

## RES-3: Resize the focused window

**When to use:** "resize this to 900x700", "make this smaller — 600
by 400" — any "this window" / "the current one" intent paired with
new dimensions.

**Call sequence:**
1. `windowctl windows list --json`.
2. Pick the window with `Focused: true` (see `windows.md` WIN-L-6).
3. `windowctl resize --title "<focused.Title>" --w <px> --h <px>`.

**Expected response:** Empty stdout, exit 0.

**Common errors:** Same as RES-1, plus zero-focused edge case from
WIN-L-6.

**User-visible formatting:** *"Resized <App> to <W>x<H>."*

---

## RES-4: Shrink-to-readable defaults

**When to use:** "make this smaller", "shrink this", "cut it in
half" — any vague-shrink intent without specific dimensions.

**Call sequence:**
1. Read current bounds: `windowctl windows list --title "<filter>"
   --json` and grab `.[0].Bounds.Width / Height`.
2. Compute target: `W' = Width / 2`, `H' = Height / 2` (or whatever
   ratio the user implied). Floor to a sensible minimum (e.g.
   400×300 to avoid OS-clamp).
3. `windowctl resize --title "<filter>" --w <W'> --h <H'>`.

**Expected response:** Exit 0 (or clamp, if user asked for too
small).

**Common errors:**
- Hitting OS minimum → surface clamp message; ask if the user wants
  a larger minimum.

**User-visible formatting:** *"Shrunk <App> from <Wx H> to <W'xH'>."*
