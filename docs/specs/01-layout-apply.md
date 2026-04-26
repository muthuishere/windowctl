# Spec: `layout-apply`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Planned
- **Source:** §4.4, FR-LAY-01, §11, OI-02
- **Summary:** Apply a declarative YAML layout file that arranges multiple
  windows across monitors in a single command.

> The shape of the layout YAML is **not yet defined** (OI-02). This spec
> captures the apply command's surface and error contract only; field-level
> scenarios will be added when the schema is decided.

## Requirement: Apply a layout file

The CLI reads a YAML file passed as a positional argument and applies the
window arrangement it describes.

### Scenario: User applies an existing layout file

- **WHEN** the user runs `windowctl apply layout.yaml` and the file exists
  and parses as YAML
- **THEN** windowctl reads the file and arranges windows according to its
  contents
- **AND** the process exits with status `0` on success

### Scenario: Layout file does not exist

- **WHEN** the user runs `windowctl apply missing.yaml` and the path does
  not resolve to a file
- **THEN** the process exits non-zero with a descriptive "layout file not
  found" message

### Scenario: Layout file is not valid YAML

- **WHEN** the user runs `windowctl apply broken.yaml` and the file fails
  YAML parsing
- **THEN** the process exits non-zero with parse error details
