# Spec: `ci-test-matrix`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented
- **Source:** §10.1, §10.2, §10.3, §10.4
- **Summary:** GitHub Actions runs `go build ./...` and `go test ./...` on
  Ubuntu, macOS, and Windows on every push/PR, plus a Windows smoke test
  that drives a real window. No Docker is used for window tests.

## Requirement: Build & test on three OSes

### Scenario: Workflow runs on every push and PR

- **WHEN** a push or pull request lands
- **THEN** `go build ./...` and `go test ./...` run on `ubuntu-latest`,
  `macos-latest`, and `windows-latest`
- **AND** all three jobs must pass for the workflow to succeed

## Requirement: Windows integration smoke test

### Scenario: Real-window smoke test on Windows runner

- **WHEN** the workflow runs on `windows-latest`
- **THEN** it builds the CLI, launches a real application (e.g. Notepad),
  detects it via `windowctl windows list`, executes a `move`, and
  verifies no errors are returned

## Requirement: Testing constraints

### Scenario: No Docker for window-management tests

- **WHEN** any window-management test is added to CI
- **THEN** it must not run inside Docker (no real window manager
  available)
- **AND** multi-monitor behavior is acknowledged as not
  integration-tested in CI (GitHub runners are single-display)

## Tasks

| Task | Purpose |
|------|---------|
| `task build` | Runs in CI on each OS to verify cross-platform build |
| `task test` | Runs the unit-test matrix on each OS |
| `task lint` | Runs the linter |

> The Windows real-window smoke test is a CI-workflow step, not a
> standalone Taskfile target.
