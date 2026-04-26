# Spec: `window-focus`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented
- **Source:** FR-FOC-01, §11
- **Summary:** Bring a matched window to the foreground.

## Requirement: Focus a matched window

### Scenario: Focus by title

- **WHEN** the user runs `windowctl focus --title jira`
- **THEN** the matched window is brought to the foreground

### Scenario: No matching window

- **WHEN** the title or app filter matches no window
- **THEN** the command exits non-zero with a descriptive message
