# Spec: `windows-list`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented
- **Source:** FR-WIN-01, FR-OUT-01, FR-OUT-02, §11
- **Summary:** List currently open windows, optionally filtered by title or
  application, in either a human-readable table or JSON.

## Requirement: List all windows

### Scenario: Default table output

- **WHEN** the user runs `windowctl windows list` with no flags
- **THEN** all currently open windows are printed as a column-aligned table
- **AND** the columns include `ID`, `Title`, `App`, `PID`, `Monitor`,
  `Bounds`

## Requirement: Filter by title

### Scenario: Partial title match

- **WHEN** the user runs `windowctl windows list --title chrome`
- **THEN** only windows whose title contains the substring `chrome` are
  returned
- **AND** the match is case-insensitive (per README example using lowercase
  `chrome`)

## Requirement: Filter by application

### Scenario: Filter by app name

- **WHEN** the user runs `windowctl windows list --app Firefox`
- **THEN** only windows belonging to the `Firefox` application are returned

## Requirement: JSON output

### Scenario: `--json` flag emits machine-readable output

- **WHEN** the user runs `windowctl windows list --json`
- **THEN** output is a JSON array of window objects suitable for piping
  into `jq` or other tools
- **AND** each object exposes the same fields as the table view

## Requirement: Performance

### Scenario: Listing returns quickly on a standard desktop

- **WHEN** `windowctl windows list` is invoked on a standard desktop
- **THEN** the command completes within 500 ms (per §8.1)
- **AND** no polling loop is involved — the OS API is queried directly

## Tasks

| Task | Purpose |
|------|---------|
| `task build` | Builds the CLI binary that exposes `windowctl windows list` |
| `task test` | Runs unit tests for filter parsing and output formatting |
