---
name: window-ctl-skill
---

# Permissions — recipes (macOS Accessibility)

`windowctl permissions` is the only command in the CLI that triggers
the macOS Accessibility (AX) permission prompt. Every other command
that needs AX (`move`, `focus`) returns
`core.ErrAccessibilityDenied` without prompting — the user has to
opt in via `windowctl permissions` or the system grant flow.

**Per-platform behavior:**
- **macOS:** Real AX flow — see PERM-1 and PERM-S-1.
- **Linux / Windows:** Both subcommands are no-ops. PERM-S-1 always
  reports "granted". The recipes below still work; they just have no
  side effect.

**The TCC parent-process gotcha (load-bearing on macOS):**
AX trust is keyed per **parent process**, not per binary. Granting
trust to `windowctl` invoked from `Ghostty.app` does not grant
trust to `windowctl` invoked from `iTerm.app` or VS Code's
integrated terminal — each parent needs its own grant. If the user
runs the agent in a different terminal than the one they ran
`windowctl permissions` in, expect a fresh "denied".

---

## PERM-S-1: Check AX status (no prompt)

**When to use:** Before assuming AX is granted, or when diagnosing
"every move returns denied". Read-only — never opens the system
dialog. For a machine-readable signal (e.g. piping into a
conditional), use PERM-S-2 instead.

**Command:**
```bash
windowctl permissions --status
```

**Expected response:**
- Exit 0 + stdout `granted` → AX is trusted for this parent process.
- Exit non-zero + stdout `denied` → not trusted. Route to PERM-1.

**Common errors:** None — this command is pure read.

**User-visible formatting:** *"Accessibility: granted"* /
*"Accessibility: denied — needs grant."*

---

## PERM-S-2: Check AX status as JSON (machine-readable)

**When to use:** Scripts or programmatic checks that need a parseable
signal rather than grep-matching the human-readable text from
PERM-S-1. Same read-only behavior — never opens the system dialog.

**Command:**
```bash
windowctl permissions --status --json
```

**Expected response:**
- `{"trusted":true}` → AX is trusted for this parent process.
- `{"trusted":false}` → not trusted. Route to PERM-1.
- Exit code is **0 in both cases** — branch on the JSON, not on
  `$?`.

**Common errors:** None — this command is pure read.

**User-visible formatting:** Usually invisible — feed the JSON into
a downstream conditional. If you must surface it, render as
*"Accessibility: trusted"* / *"Accessibility: not trusted"*.

---

## PERM-1: Request AX (triggers system prompt)

**When to use:** First-time setup, or after PERM-S-1 reports
`denied`. **This is the only `windowctl` command that opens the
macOS prompt** — running it kicks off the system dialog the first
time per parent process.

**Command:**
```bash
windowctl permissions
```

**Expected response:**
- First run, no prior grant → exits non-zero with
  `Accessibility permission denied`. The system dialog opens; user
  approves in `System Settings > Privacy & Security > Accessibility`.
- Subsequent runs after grant → exit 0. No output.

**Common errors:**
- User clicks Deny → re-running this command will **not** re-open
  the dialog (TCC remembers the deny). They must enable the parent
  process manually under `System Settings > Privacy & Security >
  Accessibility` and tick the box.

**User-visible formatting:** Walk the user through:
1. *"Run `windowctl permissions` from the same terminal you launch
   the agent in."* (TCC is keyed per parent process — don't run it
   in a different terminal.)
2. *"Approve the prompt that pops up — your terminal / IDE will
   appear in `System Settings > Privacy & Security >
   Accessibility`. Tick the box."*
3. *"Re-run the original command."*

---

## PERM-2: Recover from "denied" mid-session

**When to use:** A `move` or `focus` returned `Accessibility
permission denied` mid-session.

**Call sequence:**
1. Tell the user *"macOS needs Accessibility permission for this
   terminal. Run `windowctl permissions` and approve in System
   Settings."*
2. Wait for confirmation.
3. Verify with `windowctl permissions --status` (PERM-S-1).
4. Retry the original `move` / `focus`.

**Expected response:** Status flips `granted`, retry succeeds.

**Common errors:**
- Status still `denied` after the user says "done" — they granted to
  the wrong parent process (e.g. ran `windowctl permissions` in
  Terminal but launched the agent in iTerm). Tell them to re-run
  from the agent's parent terminal.
- They restarted the parent process without re-granting — sometimes
  TCC drops the cache. Re-run PERM-1.

**User-visible formatting:** Don't auto-retry until the user
confirms they approved. A silent retry that hits another denial
doubles the noise.

---

## PERM-3: Diagnose "window is gone from the AX tree" with WCTL_AX_DEBUG

**When to use:** A `move` / `focus` / `resize` returned
`window <id> is gone from the AX tree (its app may have quit before
move)` **and** `windowctl permissions --status` (PERM-S-1) reports
`granted`. The error means AX trust is fine but the AX walker
couldn't re-resolve the target window — usually a multi-process
app (Chrome / Electron / Slack) where the AX walk landed on the
wrong PID, or the target window genuinely vanished between the
list and the move. Don't run this for plain `denied` errors —
that's PERM-1 / PERM-2.

**Call sequence:**
1. Re-run the failing command **verbatim** with `WCTL_AX_DEBUG=1`
   prepended. The dump goes to stderr — capture it.
   ```bash
   WCTL_AX_DEBUG=1 windowctl move --app "Google Chrome" --x 100 --y 100 --w 800 --h 600
   ```
2. Read the stderr dump. It lists the per-PID AX window walk and
   which match rule (title / geometry / single-window) was tried
   for each candidate.
3. Surface the **entire stderr block verbatim** to the user inside
   a fenced code block. Don't paraphrase — the per-PID detail is
   the diagnostic.
4. Ask the user whether the AX window list looks right (correct
   app PID, expected window title visible, etc.).

**Expected response:** The stderr dump shows which PIDs were walked
and which match rule fired (or failed) for each. From there:
- If the AX walked the **wrong PID** for a multi-process app, the
  next attempt should disambiguate with `--title "<exact window
  title>"` instead of (or in addition to) `--app`.
- If the target window is genuinely missing from every PID's list,
  the window really did close between resolve and move — re-list
  windows and retry.

**Common errors:**
- Multi-process apps (Chrome, Electron, Slack, VS Code) often have
  a helper PID that owns no visible windows — the AX walk lands
  there and reports an empty list. Switch to `--title` to force
  match by window title across all PIDs of the app.
- User pastes the dump back with PIDs but no titles → the target
  window had no AX title at the moment of the walk (common for
  splash / loading windows). Wait for the window to finish loading
  and retry.

**User-visible formatting:** Paste the `WCTL_AX_DEBUG` stderr
output inside a fenced code block exactly as captured, then ask
*"Does the AX window list above look right — is the target window
and its PID in there?"* Don't try to interpret the dump for the
user before they've seen it; the raw walk is the point.
