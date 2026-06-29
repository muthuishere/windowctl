---
name: window-ctl-skill
---

# Batch — recipes

Covers `windowctl batch` — applying many window placements in one
call. Prefer `batch` over a sequence of N `move` calls whenever the
agent already knows the full layout up front: the user sees one
rendered result instead of N intermediate flickers, and a single
clamp on one window does NOT abort the rest. The CLI reports per
entry, so partial success is first-class.

**Entry shape (JSON):**
```json
{
  "app":     "<string, optional>",
  "title":   "<string, optional>",
  "monitor": <int, optional, 1-indexed>,
  "zone":    "<string, optional>",
  "x": <int>, "y": <int>, "w": <int>, "h": <int>
}
```

Per-entry rules (same as `move`):
- At least one of `title` / `app` is required.
- Target is EITHER `zone` OR all four of `x`/`y`/`w`/`h` — not both,
  not a partial coord set.
- `monitor` is optional; with `monitor`, `x`/`y` are relative to that
  monitor's origin (same semantics as `move --monitor`).

**Case mismatch gotcha (load-bearing):** `batch` JSON uses
**lowercase** keys (`app`, `title`, `monitor`, `zone`, `x`, `y`, `w`,
`h`). But `windowctl windows list --json` and `windowctl monitors
list --json` return **PascalCase** keys (`App`, `Title`, `Monitor`,
`Bounds.Width`, ...). Anything that pipes the output of `windows
list` / `monitors list` into `batch` has to translate keys —
including `Bounds.Width` → `w` and `Bounds.Height` → `h`. Don't
forward PascalCase blindly; `batch` will reject the entry.

**Exit codes:**
| Code | Meaning |
| --- | --- |
| `0` | Every entry succeeded. |
| `1` | At least one entry failed (others may have succeeded — check per-entry results). |
| `2` | Invalid input: JSON parse error, missing `--file`, malformed entry shape. |

**Invocation forms:**
```bash
windowctl batch < layout.json
windowctl batch --file layout.json
windowctl batch --json < layout.json
```

Entries run **sequentially** in array order. One failure does not
stop the rest — `batch` always processes every entry and reports
the full per-entry result.

**User-visible formatting (family default):** Always render a
per-entry summary, never a raw stdout dump. Format:
*"5/6 placed; failed: Slack (no window matched filter --app=\"Slack\")"*.
On full success, *"6/6 placed."* Don't auto-retry failed entries —
surface them and ask.

**macOS:** Same Accessibility (AX) rule as `move` / `focus` /
`resize` — the first call from a new parent process prompts for AX
trust. If `batch` returns `Accessibility permission denied` for
every entry, route to `permissions.md` (PERM-2) before doing
anything else.

---

## BAT-1: Apply a JSON layout from a file

**When to use:** User has a saved layout file they want to apply —
*"apply my work-mode layout"*, *"load layout.json"*, *"restore my
saved arrangement"*.

**Command:**
```bash
windowctl batch --file layout.json
```

**Expected response:** Human-readable per-entry text on stdout, one
line per entry. Exit `0` if every entry placed cleanly; exit `1` if
any entry failed (others may still have moved).

**Common errors:**
- Exit `2` with a parse error → the JSON itself is malformed or the
  file is missing. Surface the parser message verbatim and stop;
  don't try to "fix" the file silently.
- Exit `2` with a shape error (`entry N: zone and x/y/w/h are
  mutually exclusive`, `entry N: at least one of title/app
  required`) → one entry's shape is wrong. Name the index and the
  field; do not retry.
- Exit `1` with per-entry failures (clamp messages, `no window
  matched filter ...`) → these are PER-ENTRY, not batch-wide.
  Report each failed entry with its error; the successful entries
  did move.
- `Accessibility permission denied` on every entry (macOS) → AX
  trust is missing. Route to `permissions.md` (PERM-2), then re-run.

**Synthesis:** *"`<N>/<total>` placed from `layout.json`."* If any
failed, append *"Failed: `<entry-key>` — `<per-entry error>`"* per
failure on its own line. Don't claim the whole batch failed when
only one entry clamped.

---

## BAT-2: Apply a layout from stdin (heredoc / inline)

**When to use:** The agent is constructing the layout on the fly
from natural-language ("split chrome and slack 50/50, put terminal
on monitor 2") and doesn't want to write a temp file. Same
exit-code and per-entry semantics as BAT-1.

**Command:**
```bash
windowctl batch <<'EOF'
[
  { "app": "Google Chrome",  "zone": "1A" },
  { "app": "Slack",          "zone": "1B" },
  { "app": "Ghostty",        "monitor": 2, "zone": "1A" }
]
EOF
```

The single-quoted heredoc tag (`'EOF'`) is intentional — it
disables shell interpolation inside the JSON so `$` and backticks
are passed through literally.

**Expected response:** Same as BAT-1 — human-readable per-entry
text, exit `0` on full success or `1` on any per-entry failure.

**Common errors:** Same as BAT-1. The most common heredoc-specific
mistake is forgetting the quotes around `EOF` and having the shell
mangle a `$` inside a title.

**Synthesis:** Same as BAT-1 — *"`<N>/<total>` placed."* plus a
per-failure list when any entry failed.

---

## BAT-3: Apply a layout with structured per-entry results (`--json`)

**When to use:** The agent needs to programmatically diff requested
vs landed bounds, report partial successes, or feed the result
into another step (e.g. retry only the clamped entries with
adjusted geometry).

**Command:**
```bash
windowctl batch --json --file layout.json
```

**Expected response:** stdout is one JSON object per entry,
shaped:
```json
{ "entry": { "app": "Ghostty", "monitor": 2, "zone": "1A" }, "ok": true }
{ "entry": { "app": "Slack",   "zone": "1B" },               "error": "no window matched filter --app=\"Slack\"" }
```

Each entry echoes back the original entry plus either `"ok": true`
or `"error": "<message>"`. Exit codes are unchanged from BAT-1
(`0` / `1` / `2`). Branch on the per-entry `ok` field, not on
`$?` alone — `$?` only tells you "any failure", not which.

**Common errors:**
- Same per-entry errors as BAT-1, but easier to parse — match on
  the `error` substring (`no window matched`, `OS clamped to`,
  `Accessibility permission denied`) to classify.
- Exit `2` (parse / shape error) → no per-entry JSON is emitted.
  Treat as BAT-1's exit-2 path.

**Synthesis:** Parse the per-entry stream, then render the same
human summary as BAT-1: *"`<N>/<total>` placed; failed: `<app>`
(`<error>`)"*. The `--json` form is for the agent's benefit; the
user still gets the plain summary.

---

## BAT-4: Build a layout from "this window" + named apps

**When to use:** Any multi-window placement instruction the user
gives in one turn — *"put chrome left, slack right, terminal
bottom"*, *"editor middle, browser left, slack right on monitor
2"*. The point of going through `batch` (instead of N sequential
`move` calls) is **one rendered result** for the user, plus
partial-failure tolerance.

**Call sequence:**
1. `windowctl windows list --json` — see `windows.md` WIN-L-1 /
   WIN-L-6. Use this to confirm each named app actually has a
   window, and to resolve "this window" via `Focused: true`.
2. Build the JSON array in memory. Per entry:
   - Use `app` for unambiguous app-level placements (`"app": "Slack"`).
   - Use `title` (substring) when the user named a specific window
     ("the github tab"), or to disambiguate multi-window apps
     (Chrome, VS Code).
   - Add `monitor` when the user named a display.
   - Choose `zone` for grammar that fits the zone enum / `N:M`
     split; choose `x/y/w/h` for pixel-exact placements.
3. `windowctl batch <<EOF` (BAT-2) or write a temp file and use
   `--file` (BAT-1).

**Expected response:** A single per-entry result stream from
`batch`. One AX prompt at most (only on the first AX-touching call
from this parent process).

**Common errors:**
- A named app isn't running → step 1 returns `[]` for that filter.
  Don't fabricate an entry that will fail; tell the user which app
  is missing and ask whether to skip or launch it.
- "This window" lookup returns zero `Focused: true` rows (rare,
  during Spaces transitions) — re-poll once before giving up
  (WIN-L-6).
- One entry clamps, the rest succeed → BAT-1 partial-success
  handling. Don't roll back the successful entries.

**Synthesis:** Name the layout you applied and the entries placed:
*"Applied 3-way layout: Chrome (left) · Slack (right) · Ghostty
(bottom). 3/3 placed."* On partial success, drop the count and
list the failure: *"2/3 placed. Failed: Slack — no window matched
filter --app=\"Slack\"."*

---

## BAT-5: Save the current arrangement as a reusable layout

**When to use:** *"save my current layout as work-mode"*,
*"remember this arrangement"*, *"snapshot my windows so I can
restore later"*.

**Call sequence:**
1. `windowctl windows list --json` — start from the full list. By
   default, filter to real app windows (drop menubar agents /
   helper UI per the `windows.md` group-by-App heuristic). If the
   user wants a specific subset, ask before snapshotting.
2. Project each window into a batch-entry shape:
   - `app` ← `App` (PascalCase → lowercase key).
   - `title` ← `Title` only when `App` alone would be ambiguous
     (multiple windows of the same app you want to disambiguate).
   - `monitor` ← `Monitor` (already 1-indexed and lowercase-keyed
     in the new file).
   - `x` ← `Bounds.X`, `y` ← `Bounds.Y`, `w` ← `Bounds.Width`,
     `h` ← `Bounds.Height`. Use absolute coords (omit `monitor`)
     so the layout is portable to a re-saved arrangement; or keep
     `monitor` and translate to monitor-relative coords if the
     user wants the layout to follow a specific display.
3. Write the JSON array to disk at the caller-provided path
   (e.g. `~/layouts/work-mode.json`).

**Note:** This recipe does not call `windowctl batch` — it
*produces* the file that BAT-1 / BAT-3 will later consume. The
key-translation step (PascalCase → lowercase, `Width` → `w`,
`Height` → `h`) is the part that breaks if you skip it.

**Expected response:** A file written to disk containing a
`batch`-shaped JSON array.

**Common errors:**
- Forgetting to translate keys → the file looks right but BAT-1
  rejects every entry with `entry N: at least one of title/app
  required` because the lowercase reader didn't see `App`. Always
  translate.
- Snapshotting the raw 40-row macOS window list including menubar
  / helper UI → on restore, `batch` tries to place windows that
  don't exist in a meaningful sense. Filter first.
- A window with `Monitor: 0` (off-screen / minimized) → don't emit
  it; restoring would place it at coords on no display.

**Synthesis:** *"Captured `<N>` windows to `<path>`. Restore with
`windowctl batch --file <path>`."*
