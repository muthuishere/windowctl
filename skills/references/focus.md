---
name: window-ctl-skill
---

# Focus — recipes

Covers `windowctl focus` — raising a window to the front and giving
its app keyboard focus.

**What "focus" does per platform:**
- **macOS:** `AXUIElementPerformAction(kAXRaiseAction)` then
  `NSRunningApplication activateWithOptions:` so the window comes
  to the front AND the app gains key-window focus.
- **Windows:** `SetForegroundWindow` (subject to Windows' foreground
  lock rules — may flash the taskbar instead of activating in some
  cases).
- **Linux X11:** `wmctrl -i -a <id>`.

**Pre-flight:** Same as `move` — run `windowctl windows list --title
"<filter>" --json` first to confirm exactly one match. Empty filter
strings are rejected.

**Matcher:** `--title` is case-insensitive substring; `--app` is
case-insensitive exact match. At least one is required.

**Already-focused:** `windows list --json` exposes a per-window
`Focused: true` flag. If the user asks to focus the window that's
already focused, this is a no-op — surface that and skip the call.

**User-visible formatting (family default):**
- On success, `windowctl focus` prints nothing. Confirm with:
  *"Focused <App> — <Title>."*
- On error, surface stderr verbatim. The CLI now echoes the filter
  in no-match errors: `no window matched filter --title="..."`.

---

## FOC-1: Focus by title substring

**When to use:** "focus jira", "bring my github tab to the front",
"raise the windowctl window".

**Command:**
```bash
windowctl focus --title "<substring>"
```

**Expected response:** Empty stdout, exit 0.

**Common errors:**
- `no window matched filter --title="..."` → no window with that
  substring. Run `windowctl windows list --json` and offer the
  closest matches.
- `Accessibility permission denied` (macOS) → `permissions.md`.
- `window <id> is gone from the AX tree` → AX-bridge edge case.
  Retry with `WCTL_AX_DEBUG=1` (see `permissions.md` PERM-3) and
  surface the dump.

**User-visible formatting:** *"Focused <App> — <full Title>."*

---

## FOC-2: Focus by exact app name

**When to use:** "focus slack", "bring chrome to the front",
"activate VS Code".

**Command:**
```bash
windowctl focus --app "<App Name>"
```

**Expected response:** Empty stdout, exit 0. If the app has multiple
windows, the first in OS-enumeration order wins.

**Common errors:**
- `no window matched filter --app="..."` → wrong app name (substring
  won't work — `--app` is exact). Common gotchas in `windows.md`
  WIN-L-3. macOS apps frequently differ from binary names: `Code`
  not `Visual Studio Code`, `Google Chrome` not `Chrome`.
- AX denied on macOS.

**User-visible formatting:** *"Focused <App Name>."* If the app has
multiple windows, append *"(first window — <Title>)"*.

---

## FOC-3: Focus the most recently moved / resized window

**When to use:** Follow-ups like "focus it", "bring that one to the
front" referring to a window the skill just operated on.

**Call sequence:**
1. Read `last_match` from session context (set by every successful
   `move` / `resize` / `focus`).
2. `windowctl focus --title "<last_match.Title>"` — title is more
   stable than ID across re-enumerations.

**Expected response:** Empty stdout, exit 0.

**Common errors:** `last_match` empty → no prior action this
session; ask the user to name the window.

**User-visible formatting:** *"Focused <App>."*

---

## FOC-4: Focus is a no-op if already focused

**When to use:** Internal — before issuing `focus`, check whether
the target window is already frontmost.

**Call sequence:**
1. `windowctl windows list --json`.
2. Find the target window in the list.
3. If `target.Focused == true`, skip the `focus` call and tell the
   user *"<App> is already focused."*
4. Otherwise, run `windowctl focus --title "<target.Title>"`.

**Expected response:** Either no command run (already focused) or
exit 0.

**Common errors:** None new.

**User-visible formatting:** *"<App> is already focused."* / *"Focused
<App>."*
