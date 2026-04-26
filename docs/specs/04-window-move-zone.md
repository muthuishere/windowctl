# Spec: `window-move-zone`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented
- **Source:** FR-MOV-01, FR-MOV-03, FR-MOV-04, §4.3.1, §4.3.2, §11
- **Summary:** Move a matched window into a logical zone (enum like `2B` or
  split like `3:2`) on a chosen or auto-resolved monitor.

## Requirement: Move into an enum zone

### Scenario: Place a window in a predefined quadrant

- **WHEN** the user runs `windowctl move --title chrome --monitor 2 --zone 2B`
- **THEN** the matched window is resized to the top-right quarter of
  monitor 2
- **AND** the supported enum zones are `1A`, `1B`, `2A`, `2B`, `2C`, `2D`
  per §4.3.1

## Requirement: Move into a split zone

### Scenario: Place a window in column M of an N-way split

- **WHEN** the user runs `windowctl move --title Slack --zone 3:1`
- **THEN** the window is placed in the first of three equal vertical
  columns spanning the full monitor height
- **AND** the rectangle is computed as `cellWidth = monitorWidth / N`,
  `x = monitor.X + (M-1) * cellWidth`, `y = monitor.Y`, `w = cellWidth`,
  `h = monitorHeight` (per §4.3.2)

## Requirement: Monitor auto-resolution when `--monitor` is omitted

### Scenario: Resolve monitor by majority overlap

- **WHEN** the user runs `windowctl move --title Terminal --zone 1A`
  without `--monitor`
- **THEN** the target monitor is the one containing the majority of the
  window's visible area (per FR-MOV-03)

## Requirement: Window matching

### Scenario: At least one matcher must be provided

- **WHEN** the user runs `windowctl move --zone 1A` with neither `--title`
  nor `--app`
- **THEN** the command exits non-zero with a descriptive matcher-required
  error

### Scenario: Multiple windows match the filter

- **WHEN** the user's filter matches more than one window
- **THEN** the first match is used (v0.1 behavior; deterministic ordering
  is TBD per OI-01)

## Requirement: Error contract

### Scenario: No matching window

- **WHEN** the filter matches zero windows
- **THEN** the command exits non-zero with a descriptive message

### Scenario: Invalid zone string

- **WHEN** `--zone` is neither a recognized enum nor a valid `N:M` split
- **THEN** the command exits non-zero with a descriptive message

### Scenario: Invalid monitor ID

- **WHEN** `--monitor` references a monitor that does not exist
- **THEN** the command exits non-zero with a descriptive message

## Tasks

| Task | Purpose |
|------|---------|
| `task build` | Builds the CLI binary that exposes `windowctl move` |
| `task test` | Runs unit tests for zone math, monitor auto-resolution, matcher rules, and the error contract |
