# Spec: `monitors-list`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented
- **Source:** FR-MON-01, FR-OUT-01, FR-OUT-02
- **Summary:** Enumerate the physical displays attached to the system.

## Requirement: List all monitors

### Scenario: Default table output

- **WHEN** the user runs `windowctl monitors list`
- **THEN** all attached monitors are printed as a column-aligned table
- **AND** the columns include `ID`, `X`, `Y`, `Width`, `Height`, `Primary`

## Requirement: JSON output

### Scenario: `--json` flag emits machine-readable output

- **WHEN** the user runs `windowctl monitors list --json`
- **THEN** output is a JSON array of monitor objects with the same fields
  as the table view

## ADDED Requirements

### Requirement: Multi-display enumeration on every supported OS

The adapter SHALL enumerate every attached physical display, not just
the primary one, on macOS, Linux, and Windows. The previous Windows
caveat ("primary only via `GetSystemMetrics`") that was flagged in
`CLAUDE.md` is now closed: `internal/adapter/windows/adapter.go` uses
`EnumDisplayMonitors` + `GetMonitorInfoW` (user32.dll) and assigns
sequential IDs (0..N-1) in enumeration order, mirroring the darwin
adapter's `wctl_collect_monitors` shape.

#### Scenario: Windows reports every attached display

- **GIVEN** a Windows host with two or more displays connected
- **WHEN** the user runs `windowctl monitors list`
- **THEN** the output contains one row per display (not just the primary)
- **AND** exactly one row has `Primary=true` (the display whose
  `MONITORINFO.dwFlags` includes `MONITORINFOF_PRIMARY = 0x1`)
- **AND** each row's `X`, `Y`, `Width`, `Height` come from the
  display's `rcMonitor` rectangle in virtual-screen coordinates

#### Scenario: macOS reports every active display

- **GIVEN** a macOS host with two or more displays connected
- **WHEN** the user runs `windowctl monitors list`
- **THEN** the output contains one row per active display
  (`CGGetActiveDisplayList`)
- **AND** the row whose display ID equals `CGMainDisplayID()` has
  `Primary=true`

## Tasks

| Task | Purpose |
|------|---------|
| `task build` | Builds the CLI binary that exposes `windowctl monitors list` |
| `task test` | Runs unit tests for monitor enumeration and output formatting |
