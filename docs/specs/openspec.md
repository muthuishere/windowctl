# windowctl — OpenSpec Specifications

**Source of truth:** [`docs/requirements.md`](../requirements.md) v0.1.

This document slices the windowctl requirements into vertical, user-facing
capability specs in OpenSpec style. Each slice stands alone (input → behavior →
output) and references back to the requirement IDs it covers.

Slices are ordered by **delivery value**, not by document order:

1. `layout-apply` is the headline capability — the *why* of the tool.
2. `os-spikes` is the tracer-bullet foundation that proves each target OS can
   actually drive a real window before any higher-level capability is built.

Everything else builds on those two.

### Architectural ground truth

From v1, the CLI has always been a thin wrapper over an internal library
package — every CLI command in this document is implemented by delegating to
the library, not by reimplementing logic in the CLI layer. What was **not**
done initially was **exporting** that library as a public, importable Go
module. That export is a later capability and is captured in
[`go-library-api`](#8-spec-go-library-api), whose status reflects this split:
the wrapper-shape and OS-mockable core have been in place from the beginning,
the public Go-module surface and dependency-isolation guarantee came later.

Individual CLI specs below therefore do not restate "the CLI delegates to the
library" in every scenario — that is a global invariant covered once, here.

---

## Status Index

| # | Spec | Status | Source requirements |
|---|------|--------|---------------------|
| 1 | [`layout-apply`](#1-spec-layout-apply) | Planned | §4.4, FR-LAY-01, §11, OI-02 |
| 2 | [`os-spikes`](#2-spec-os-spikes) | Partial | §7, §7.1, §10.3 |
| 3 | [`windows-list`](#3-spec-windows-list) | Implemented | FR-WIN-01, FR-OUT-01, FR-OUT-02, §11 |
| 4 | [`monitors-list`](#4-spec-monitors-list) | Implemented | FR-MON-01, FR-OUT-01, FR-OUT-02 |
| 5 | [`window-move-zone`](#5-spec-window-move-zone) | Implemented | FR-MOV-01, FR-MOV-03, FR-MOV-04, §4.3.1, §4.3.2, §11 |
| 6 | [`window-move-coords`](#6-spec-window-move-coords) | Implemented | FR-MOV-02, FR-MOV-04, §4.3.3, §11 |
| 7 | [`window-focus`](#7-spec-window-focus) | Implemented | FR-FOC-01, §11 |
| 8 | [`go-library-api`](#8-spec-go-library-api) | Partial | LIB-01, LIB-02, LIB-03, §8.3 |
| 9 | [`npm-distribution`](#9-spec-npm-distribution) | Partial | §9.3, OI-05 |
| 10 | [`release-pipeline`](#10-spec-release-pipeline) | Implemented | §9.1, §9.2 |
| 11 | [`ci-test-matrix`](#11-spec-ci-test-matrix) | Implemented | §10.1, §10.2, §10.3, §10.4 |

**Status legend:** `Implemented` · `Partial` · `Planned` · `Not planned`.

---

## 1. Spec: `layout-apply`

- **Status:** Planned
- **Source:** §4.4, FR-LAY-01, §11, OI-02
- **Summary:** Apply a declarative YAML layout file that arranges multiple
  windows across monitors in a single command.

> The shape of the layout YAML is **not yet defined** (OI-02). This spec captures
> the apply command's surface and error contract only; field-level scenarios
> will be added when the schema is decided.

### Requirement: Apply a layout file

The CLI reads a YAML file passed as a positional argument and applies the
window arrangement it describes.

#### Scenario: User applies an existing layout file

- **WHEN** the user runs `windowctl apply layout.yaml` and the file exists and
  parses as YAML
- **THEN** windowctl reads the file and arranges windows according to its
  contents
- **AND** the process exits with status `0` on success

#### Scenario: Layout file does not exist

- **WHEN** the user runs `windowctl apply missing.yaml` and the path does not
  resolve to a file
- **THEN** the process exits non-zero with a descriptive "layout file not
  found" message

#### Scenario: Layout file is not valid YAML

- **WHEN** the user runs `windowctl apply broken.yaml` and the file fails YAML
  parsing
- **THEN** the process exits non-zero with parse error details

---

## 2. Spec: `os-spikes`

- **Status:** Partial (macOS Supported, Windows Supported, Linux best-effort,
  Wayland limited per README platform table)
- **Source:** §7, §7.1, §10.3
- **Summary:** Per-OS tracer bullet proving that the chosen native API on each
  supported platform can enumerate windows and monitors, move/resize a real
  window, and bring it to the foreground — end-to-end through a thin platform
  adapter behind a clean Go interface.

This spec is the foundation every later capability depends on. It is not a
user-facing CLI command on its own; it is the per-OS proof that the adapter
contract is real.

### Requirement: macOS adapter spike

#### Scenario: Enumerate, move, and focus a real window on macOS

- **WHEN** the macOS adapter is built and Accessibility permission is granted
- **THEN** it can list windows via CoreGraphics, and move/resize/focus the
  matched window via the Accessibility API
- **AND** the operations are exposed through the same Go interface used by the
  other platforms

#### Scenario: Accessibility permission denied on macOS

- **WHEN** the macOS adapter is invoked without Accessibility permission
- **THEN** the operation exits non-zero with an instruction to grant
  Accessibility access

### Requirement: Windows adapter spike

#### Scenario: Enumerate, move, and focus a real window on Windows

- **WHEN** the Windows adapter is built against `user32.dll` (via `syscall` or
  `x/sys/windows`)
- **THEN** it can list windows, and move/resize/focus the matched window using
  Win32 APIs
- **AND** the operations are exposed through the shared Go interface

#### Scenario: Operation requires elevated privileges on Windows

- **WHEN** a Windows operation cannot be completed at the current privilege
  level
- **THEN** it fails gracefully with a descriptive error rather than crashing

### Requirement: Linux adapter spike

#### Scenario: Enumerate, move, and focus a real window under X11

- **WHEN** the Linux adapter runs under an X11 session
- **THEN** it can list windows and move/focus the matched window using X11 as
  the primary backend, with `wmctrl` / `xdotool` available as fallback

#### Scenario: Wayland session

- **WHEN** the Linux adapter detects a Wayland session
- **THEN** behavior is best-effort only; failures must produce clear error
  messages rather than silent no-ops

### Requirement: Adapter isolation

#### Scenario: Core layer has no OS-specific code

- **WHEN** the core/business-logic packages are inspected
- **THEN** they contain no Win32, Cocoa, or Xlib references
- **AND** all OS calls go through a Go interface implemented by the platform
  adapter modules

---

## 3. Spec: `windows-list`

- **Status:** Implemented
- **Source:** FR-WIN-01, FR-OUT-01, FR-OUT-02, §11
- **Summary:** List currently open windows, optionally filtered by title or
  application, in either a human-readable table or JSON.

### Requirement: List all windows

#### Scenario: Default table output

- **WHEN** the user runs `windowctl windows list` with no flags
- **THEN** all currently open windows are printed as a column-aligned table
- **AND** the columns include `ID`, `Title`, `App`, `PID`, `Monitor`, `Bounds`

### Requirement: Filter by title

#### Scenario: Partial title match

- **WHEN** the user runs `windowctl windows list --title chrome`
- **THEN** only windows whose title contains the substring `chrome` are
  returned
- **AND** the match is case-insensitive (per README example using lowercase
  `chrome`)

### Requirement: Filter by application

#### Scenario: Filter by app name

- **WHEN** the user runs `windowctl windows list --app Firefox`
- **THEN** only windows belonging to the `Firefox` application are returned

### Requirement: JSON output

#### Scenario: `--json` flag emits machine-readable output

- **WHEN** the user runs `windowctl windows list --json`
- **THEN** output is a JSON array of window objects suitable for piping into
  `jq` or other tools
- **AND** each object exposes the same fields as the table view

### Requirement: Performance

#### Scenario: Listing returns quickly on a standard desktop

- **WHEN** `windowctl windows list` is invoked on a standard desktop
- **THEN** the command completes within 500 ms (per §8.1)
- **AND** no polling loop is involved — the OS API is queried directly

---

## 4. Spec: `monitors-list`

- **Status:** Implemented
- **Source:** FR-MON-01, FR-OUT-01, FR-OUT-02
- **Summary:** Enumerate the physical displays attached to the system.

### Requirement: List all monitors

#### Scenario: Default table output

- **WHEN** the user runs `windowctl monitors list`
- **THEN** all attached monitors are printed as a column-aligned table
- **AND** the columns include `ID`, `X`, `Y`, `Width`, `Height`, `Primary`

### Requirement: JSON output

#### Scenario: `--json` flag emits machine-readable output

- **WHEN** the user runs `windowctl monitors list --json`
- **THEN** output is a JSON array of monitor objects with the same fields as
  the table view

---

## 5. Spec: `window-move-zone`

- **Status:** Implemented
- **Source:** FR-MOV-01, FR-MOV-03, FR-MOV-04, §4.3.1, §4.3.2, §11
- **Summary:** Move a matched window into a logical zone (enum like `2B` or
  split like `3:2`) on a chosen or auto-resolved monitor.

### Requirement: Move into an enum zone

#### Scenario: Place a window in a predefined quadrant

- **WHEN** the user runs `windowctl move --title chrome --monitor 2 --zone 2B`
- **THEN** the matched window is resized to the top-right quarter of monitor 2
- **AND** the supported enum zones are `1A`, `1B`, `2A`, `2B`, `2C`, `2D` per
  §4.3.1

### Requirement: Move into a split zone

#### Scenario: Place a window in column M of an N-way split

- **WHEN** the user runs `windowctl move --title Slack --zone 3:1`
- **THEN** the window is placed in the first of three equal vertical columns
  spanning the full monitor height
- **AND** the rectangle is computed as `cellWidth = monitorWidth / N`,
  `x = monitor.X + (M-1) * cellWidth`, `y = monitor.Y`, `w = cellWidth`,
  `h = monitorHeight` (per §4.3.2)

### Requirement: Monitor auto-resolution when `--monitor` is omitted

#### Scenario: Resolve monitor by majority overlap

- **WHEN** the user runs `windowctl move --title Terminal --zone 1A` without
  `--monitor`
- **THEN** the target monitor is the one containing the majority of the
  window's visible area (per FR-MOV-03)

### Requirement: Window matching

#### Scenario: At least one matcher must be provided

- **WHEN** the user runs `windowctl move --zone 1A` with neither `--title` nor
  `--app`
- **THEN** the command exits non-zero with a descriptive matcher-required
  error

#### Scenario: Multiple windows match the filter

- **WHEN** the user's filter matches more than one window
- **THEN** the first match is used (v0.1 behavior; deterministic ordering is
  TBD per OI-01)

### Requirement: Error contract

#### Scenario: No matching window

- **WHEN** the filter matches zero windows
- **THEN** the command exits non-zero with a descriptive message

#### Scenario: Invalid zone string

- **WHEN** `--zone` is neither a recognized enum nor a valid `N:M` split
- **THEN** the command exits non-zero with a descriptive message

#### Scenario: Invalid monitor ID

- **WHEN** `--monitor` references a monitor that does not exist
- **THEN** the command exits non-zero with a descriptive message

---

## 6. Spec: `window-move-coords`

- **Status:** Implemented
- **Source:** FR-MOV-02, FR-MOV-04, §4.3.3, §11
- **Summary:** Move a matched window using raw `--x/--y/--w/--h` coordinates,
  interpreted as absolute global coordinates or as monitor-relative depending
  on whether `--monitor` is supplied.

### Requirement: Absolute coordinate move

#### Scenario: Coordinates without `--monitor` are global

- **WHEN** the user runs `windowctl move --title chrome --x 0 --y 0 --w 960 --h 1080`
- **THEN** the window is placed at the absolute screen coordinates `(0, 0)`
  with size `960×1080`

### Requirement: Monitor-relative coordinate move

#### Scenario: Coordinates with `--monitor` are relative to that monitor's origin

- **WHEN** the user runs `windowctl move --title chrome --monitor 2 --x 0 --y 0 --w 960 --h 1080`
- **THEN** the coordinates are interpreted relative to monitor 2's origin
  before being applied

### Requirement: Window matching and error contract

#### Scenario: Same matcher and error contract as zone moves

- **WHEN** the user invokes `move` with manual coordinates
- **THEN** the matcher rules and error scenarios from `window-move-zone`
  apply identically (no matcher → error, no match → error, invalid monitor →
  error)

---

## 7. Spec: `window-focus`

- **Status:** Implemented
- **Source:** FR-FOC-01, §11
- **Summary:** Bring a matched window to the foreground.

### Requirement: Focus a matched window

#### Scenario: Focus by title

- **WHEN** the user runs `windowctl focus --title jira`
- **THEN** the matched window is brought to the foreground

#### Scenario: No matching window

- **WHEN** the title or app filter matches no window
- **THEN** the command exits non-zero with a descriptive message

---

## 8. Spec: `go-library-api`

- **Status:** Partial
- **Source:** LIB-01, LIB-02, LIB-03, §8.3
- **Summary:** Make the library that already powers the CLI usable as a
  first-class, importable Go module at `github.com/muthuishere/windowctl`,
  with no CLI dependencies leaking into consumers.

> **Timeline note.** From v1 the CLI has been a thin wrapper around an
> internal library package, and core logic has been OS-mockable from the
> start. What was **not** in place initially is the exported, importable
> public surface — that is the work this spec primarily tracks.

### Requirement: CLI is a thin wrapper *(Implemented from v1)*

#### Scenario: All business logic lives in the library

- **WHEN** the CLI binary handles any subcommand (`windows list`,
  `monitors list`, `move`, `focus`, `apply`)
- **THEN** it parses flags and delegates to the library
- **AND** it contains no zone-math, monitor-resolution, window-matching, or
  layout-parsing logic of its own

### Requirement: Core logic is testable without OS dependencies *(Implemented from v1)*

#### Scenario: Core packages mock the OS adapter

- **WHEN** the core packages' unit tests run
- **THEN** they execute without a real window manager, using mock adapters
  satisfying the OS interface (per §8.3)

### Requirement: Public API surface *(Planned — not exported initially)*

#### Scenario: Library exposes the documented types and functions

- **WHEN** a Go consumer imports `github.com/muthuishere/windowctl`
- **THEN** the package exposes the types `Window`, `Monitor`, `Rect`,
  `Filter`, `Match`, `Target` and the functions `ListWindows(filter Filter)`,
  `ListMonitors()`, `Move(match Match, target Target)`, `Focus(match Match)`
  with the signatures defined in LIB-01

> Initially the package providing this surface lived in an `internal/` path
> and could not be imported by third parties. Promoting it to a public
> import path is the work that moves this requirement to `Implemented`.

### Requirement: Library importability without CLI deps *(Planned — not exported initially)*

#### Scenario: Importing the library does not pull in CLI deps

- **WHEN** a third-party Go module depends only on `github.com/muthuishere/windowctl`
- **THEN** transitive dependencies do not include the CLI's flag/parsing
  packages (per LIB-03)

---

## 9. Spec: `npm-distribution`

- **Status:** Partial (download vs. bundle strategy is OI-05)
- **Source:** §9.3, OI-05
- **Summary:** Make the CLI installable and runnable via npm so that
  `npx windowctl <command>` and `npm install -g windowctl` both work.

### Requirement: `npx windowctl` runs the native binary

#### Scenario: First-run via npx

- **WHEN** a user runs `npx windowctl windows list`
- **THEN** the npm package's JS wrapper detects OS and architecture at
  runtime, invokes the correct native binary, and forwards arguments and exit
  code

### Requirement: Global install

#### Scenario: `npm install -g windowctl` exposes the CLI on PATH

- **WHEN** a user runs `npm install -g windowctl`
- **THEN** a `windowctl` executable is available on the system PATH

### Requirement: Binary acquisition strategy

#### Scenario: Strategy is undecided

- **WHEN** the npm package is built
- **THEN** the choice between bundling all platform binaries vs. downloading
  the matching binary on first use is **TBD** (OI-05) and must be resolved
  before this spec moves to `Implemented`

---

## 10. Spec: `release-pipeline`

- **Status:** Implemented
- **Source:** §9.1, §9.2
- **Summary:** Cross-compile, checksum, and publish GitHub releases for
  macOS, Windows, and Linux on both `amd64` and `arm64`, orchestrated by
  Taskfile and GoReleaser.

### Requirement: Taskfile-driven build orchestration

#### Scenario: Required tasks exist

- **WHEN** a developer runs `task --list`
- **THEN** the available tasks include `build`, `test`, `lint`, `snapshot`,
  and `release` with the behavior described in §9.1

### Requirement: Cross-compiled binaries

#### Scenario: GoReleaser produces all target binaries

- **WHEN** `task release` (or `task snapshot`) runs
- **THEN** GoReleaser produces binaries for `darwin`, `windows`, and `linux`
  on both `amd64` and `arm64`

### Requirement: macOS CGO linker flags

#### Scenario: macOS build links the required frameworks

- **WHEN** the macOS binary is built
- **THEN** CGO is enabled and the linker flags include
  `-framework ApplicationServices` and `-framework CoreGraphics`

### Requirement: Checksums and GitHub release

#### Scenario: SHA-256 checksums are published with the binaries

- **WHEN** a release is published
- **THEN** SHA-256 checksums are generated for every binary and the binaries
  are attached to a GitHub release

---

## 11. Spec: `ci-test-matrix`

- **Status:** Implemented
- **Source:** §10.1, §10.2, §10.3, §10.4
- **Summary:** GitHub Actions runs `go build ./...` and `go test ./...` on
  Ubuntu, macOS, and Windows on every push/PR, plus a Windows smoke test that
  drives a real window. No Docker is used for window tests.

### Requirement: Build & test on three OSes

#### Scenario: Workflow runs on every push and PR

- **WHEN** a push or pull request lands
- **THEN** `go build ./...` and `go test ./...` run on `ubuntu-latest`,
  `macos-latest`, and `windows-latest`
- **AND** all three jobs must pass for the workflow to succeed

### Requirement: Windows integration smoke test

#### Scenario: Real-window smoke test on Windows runner

- **WHEN** the workflow runs on `windows-latest`
- **THEN** it builds the CLI, launches a real application (e.g. Notepad),
  detects it via `windowctl windows list`, executes a `move`, and verifies no
  errors are returned

### Requirement: Testing constraints

#### Scenario: No Docker for window-management tests

- **WHEN** any window-management test is added to CI
- **THEN** it must not run inside Docker (no real window manager available)
- **AND** multi-monitor behavior is acknowledged as not integration-tested in
  CI (GitHub runners are single-display)
