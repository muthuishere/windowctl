---
name: window-ctl-skill
---

# Visual write rail — find (OCR), scroll, drag, clipboard, window state, recipes

The visual loop's WRITE half. `screenshot` + `mouse`/`type` let you see and
act; these verbs let you **find what to click by its text** and **capture a
flow so you never re-derive it**. This is the visual twin of browser-bridge's
DOM recipes: browser-bridge reads the DOM, windowctl drives ANY app's UI like a
human — so it writes where the DOM (DraftJS/contentEditable/native/canvas)
resists scripting.

## find — OCR the screen, get click coordinates

`windowctl find --text "Submit"` runs native macOS Vision OCR over a monitor,
window, or region and returns every match with a **click point in the same
global point space `mouse click` consumes** — no pixel math, retina-safe, even
across monitors with negative origins (a display left of the primary).

```bash
# Locate on-screen text (rows: clickX  clickY  confidence  text)
windowctl find --text "Add a comment" --app "Google Chrome"

# Just the top match's click point, for piping:
read CX CY < <(windowctl find --text "Submit" --first)
windowctl mouse click --x "$CX" --y "$CY"

# All matches as JSON (bounds + click + confidence); empty --text = every line
windowctl find --text "" --app "Google Chrome" --json
```

Scope flags mirror `screenshot`: `--app`/`--title` (a window), `--monitor N`,
`--x --y --w --h` (a region), or nothing (the focused monitor). `--first`
exits non-zero when nothing matched, so scripts can gate on it.

**The loop, no hardcoded pixels:** `find --text "X" --first` → `mouse click` →
`type` → `find --text` again to VERIFY the state changed.

## scroll and drag

```bash
windowctl scroll --dy -10 --x 800 --y 500     # wheel down at a point (monitor-relative with --monitor)
windowctl drag --from-x 100 --from-y 200 --to-x 400 --to-y 200   # slider / select / drag-drop
```

## clipboard — the layout-proof way to insert text

Paste beats synthetic typing (no keycode/layout/IME mismatch, no first-responder
drops):

```bash
windowctl clipboard set --text "long text with émojis 🚀"   # or: … set  (reads stdin)
windowctl focus --app TextEdit
windowctl key --combo cmd+v --app TextEdit                  # paste
windowctl clipboard get                                     # read it back
```

## window state

```bash
windowctl minimize   --app Slack
windowctl maximize   --app "Google Chrome"     # fill its monitor (OS clamps to the menu-bar area)
windowctl fullscreen --title "Keynote"         # toggle native fullscreen
windowctl close      --app Preview
```

## recipes — capture a flow once, replay it forever

A recipe is a named, parameterized (`$VAR`) sequence of the primitives, scoped
to a window, saved in `~/.config/windowctl/recipes.json`. Two are seeded
(`youtube-comment`, `x-compose`).

```bash
windowctl recipe list
windowctl recipe run youtube-comment --var TEXT="great video"     # fills the box, STOPS before posting
windowctl recipe run youtube-comment --var TEXT="great video" --confirm   # actually posts
cat flow.json | windowctl recipe save my-flow                     # author from JSON (stdin or --file)
```

**Safe by default:** the final submit/post click is `confirm`-gated — a bare
run fills the compose box and stops; only `--confirm` publishes. **Allow all
ways:** `find-click` steps are vision-first with a coordinate fallback; `type`
steps take `method: keystroke|paste|auto` (clipboard-paste fallback).

Recipe body shape:

```json
{
  "match": {"app": "Google Chrome", "title": "YouTube"},
  "vars": ["TEXT"],
  "steps": [
    {"action": "focus"},
    {"action": "find-click", "text": "Add a comment"},
    {"action": "type", "text": "$TEXT", "method": "auto"},
    {"action": "find-click", "text": "Comment", "confirm": true}
  ]
}
```

Step actions: `focus`, `launch`, `wait-text`, `find-click`, `click`, `type`,
`paste`, `key`, `scroll`, `drag`.

## Hardening notes (learned from live runs)

- **Focusing a window ≠ focusing its text field.** Click INTO the field
  (`find-click`) before typing, or the keystrokes hit a non-first-responder and
  drop. Recipes and the focus guard handle this; ad-hoc scripts should click first.
- **Never re-raise an already-focused window before typing** — it resets first
  responder and drops the keystrokes. The guard already skips the redundant raise.
- **Blinking text carets split words in OCR** (`Anchor` → `Anc|or`). Verify
  with a substring away from the caret, or read back via clipboard.
- Tunables: `WCTL_FOCUS_SETTLE_MS` (post-focus settle), `WCTL_RECIPE_STEP_DELAY_MS`
  (inter-step pace), `WCTL_SCREEN_*`/coordinate contract in `automation.md`.
