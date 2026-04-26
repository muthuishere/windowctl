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
