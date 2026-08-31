---
title: Permissions
description: What macOS asks for, when it asks, and how to check without prompting.
---

```sh
windowctl permissions            # prompt for Accessibility
windowctl permissions --status   # read-only check, never prompts
windowctl permissions --status --json
```

## Accessibility

Needed for `move`, `resize`, `focus`, `batch`, and input injection — because
those go through the Accessibility API. **Not** needed to list windows or
monitors.

The prompt fires from exactly two places: `windowctl permissions`, and lazily on
the first write operation. It never fires from `windows list`, so a read-only
script cannot surprise a user with a system dialog.

Grant it to the process that *runs* windowctl — your terminal application, or
your agent's host process — not to windowctl itself.

## Screen Recording

Needed for `screenshot` and screen capture. macOS prompts on first use.

## System audio

Needed for non-silent speaker loopback. Without it, macOS delivers frames of
silence rather than an error — so a capture that "works" can still be recording
nothing. Check levels, not just frame counts.

## Scripting around it

`--status` exits with a clean boolean answer and never blocks, which is what you
want at the top of a script:

```sh
if ! windowctl permissions --status >/dev/null; then
  echo "grant Accessibility first: windowctl permissions" >&2
  exit 1
fi
```

On Linux and Windows the permission calls are no-ops that report success — the
same script runs everywhere.
