# Spec: `ci-test-matrix`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Partial — three-OS build & test matrix is live and green;
  Windows and macOS real-window smoke tests are wired into the workflow
  via the `task smoke-windows` / `task smoke-darwin` targets in
  `Taskfile.yml` (which delegate to `scripts/smoke-windows.ps1` and
  `scripts/smoke-darwin.sh` respectively); a Linux real-window smoke
  test is still TBD.
- **Source:** §10.1, §10.2, §10.3, §10.4
- **Summary:** GitHub Actions runs `go build ./...`, `go vet ./...`, and
  `go test ./...` on Ubuntu, macOS, and Windows on every push/PR, with
  Windows and macOS real-window smoke tests as follow-up workflow steps.
  No Docker is used for window tests.

## Requirement: Build & test on three OSes

### Scenario: Workflow runs on every push and PR

- **WHEN** a push or pull request lands
- **THEN** `go vet ./...`, `go build ./...`, and `go test ./...` run on
  `ubuntu-latest`, `macos-latest`, and `windows-latest` (per
  `.github/workflows/ci.yaml`)
- **AND** all three jobs must pass for the workflow to succeed

## Requirement: Windows integration smoke test *(Implemented)*

### Scenario: Real-window smoke test on Windows runner

- **WHEN** the workflow runs on `windows-latest`
- **THEN** it builds the CLI, launches a real application (Notepad),
  detects it via `windowctl windows list --json`, executes a `move`,
  and verifies no errors are returned
- **AND** the step is wired in `.github/workflows/ci.yaml` as
  `task smoke-windows` (gated on `matrix.os == 'windows-latest'`),
  with the Task runner installed via `arduino/setup-task@v2`

## Requirement: macOS integration smoke test *(Implemented)*

### Scenario: Real-window smoke test on macOS runner (default — AX not granted)

- **WHEN** the workflow runs on `macos-latest` without
  Accessibility permission granted to the runner process
- **THEN** the step builds the CLI, launches TextEdit via
  LaunchServices (`open -a TextEdit`), detects it via
  `windowctl windows list --json`, runs `windowctl move`, and asserts
  that `move` exits non-zero with `core.ErrAccessibilityDenied` —
  matching the macOS adapter contract in
  [`01-os-spikes.md`](./01-os-spikes.md)
- **AND** the step is wired in `.github/workflows/ci.yaml` as
  `task smoke-darwin` (gated on `matrix.os == 'macos-latest'`),
  with the Task runner installed via `arduino/setup-task@v2`

### Scenario: Real-window smoke test on macOS runner (opt-in — AX granted)

- **WHEN** the same script runs locally with `WCTL_SMOKE_AX=1` after
  the developer has granted Accessibility access in System Settings
- **THEN** `windowctl move` is asserted to exit 0 AND a follow-up
  `windows list` reports TextEdit's bounds matching the requested
  rectangle within `WCTL_AX_TOLERANCE` pixels (default 10, to absorb
  the CG/AX title-bar and shadow geometry skew)

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
| `task smoke-windows` | Drives `scripts/smoke-windows.ps1` (CI step on `windows-latest`) |
| `task smoke-darwin` | Drives `scripts/smoke-darwin.sh` (CI step on `macos-latest`) |
