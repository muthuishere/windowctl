# Spec: `os-spikes`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Partial (macOS Supported, Windows Supported, Linux best-effort,
  Wayland limited per README platform table)
- **Source:** §7, §7.1, §10.3
- **Summary:** Per-OS tracer bullet proving that the chosen native API on
  each supported platform can enumerate windows and monitors, move/resize a
  real window, and bring it to the foreground — end-to-end through a thin
  platform adapter behind a clean Go interface.

This spec is the foundation every later capability depends on. It is not a
user-facing CLI command on its own; it is the per-OS proof that the adapter
contract is real.

## Requirement: macOS adapter spike *(Partial — list paths real, move/focus stubbed)*

### Scenario: Enumerate windows and monitors on macOS

- **WHEN** the macOS adapter is built (CGO enabled, `-framework
  CoreGraphics -framework CoreFoundation`)
- **THEN** `ListWindows` enumerates real windows via
  `CGWindowListCopyWindowInfo` and `ListMonitors` enumerates displays
  via `CGGetActiveDisplayList`, exposed through the same Go interface
  used by the other platforms

### Scenario: Move and Focus on macOS

- **WHEN** the macOS adapter is asked to move or focus a window
- **THEN** the call returns `core.ErrNotImplemented` until the
  Accessibility (AX) wiring lands as a follow-up slice
- **AND** the macOS smoke test (`scripts/smoke-darwin.sh`) asserts
  this explicitly so the gap stays loud rather than silent

### Scenario: Accessibility permission denied on macOS *(future work)*

- **WHEN** the macOS adapter (post-AX wiring) is invoked without
  Accessibility permission
- **THEN** the operation will exit non-zero with an instruction to
  grant Accessibility access

## Requirement: Windows adapter spike

### Scenario: Enumerate, move, and focus a real window on Windows

- **WHEN** the Windows adapter is built against `user32.dll` (via `syscall`
  or `x/sys/windows`)
- **THEN** it can list windows, and move/resize/focus the matched window
  using Win32 APIs
- **AND** the operations are exposed through the shared Go interface

### Scenario: Operation requires elevated privileges on Windows

- **WHEN** a Windows operation cannot be completed at the current privilege
  level
- **THEN** it fails gracefully with a descriptive error rather than
  crashing

## Requirement: Linux adapter spike

### Scenario: Enumerate, move, and focus a real window under X11

- **WHEN** the Linux adapter runs under an X11 session
- **THEN** it can list windows and move/focus the matched window using X11
  as the primary backend, with `wmctrl` / `xdotool` available as fallback

### Scenario: Wayland session

- **WHEN** the Linux adapter detects a Wayland session
- **THEN** behavior is best-effort only; failures must produce clear error
  messages rather than silent no-ops

## Requirement: Adapter isolation

### Scenario: Core layer has no OS-specific code

- **WHEN** the core/business-logic packages are inspected
- **THEN** they contain no Win32, Cocoa, or Xlib references
- **AND** all OS calls go through a Go interface implemented by the
  platform adapter modules

## Tasks

| Task | Purpose |
|------|---------|
| `task build` | Builds the per-OS CLI binary that exercises the platform adapter |
| `task test` | Runs the OS-mockable core unit tests |

> Per-OS adapter validation against real windows runs in
> [`ci-test-matrix`](./10-ci-test-matrix.md), not via a standalone
> Taskfile target.
