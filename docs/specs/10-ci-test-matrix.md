# Spec: `ci-test-matrix`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Partial — three-OS build & test matrix is live and green;
  the Windows real-window smoke test exists as a runnable script
  (`scripts/smoke-windows.ps1`) but is not yet wired into the workflow.
- **Source:** §10.1, §10.2, §10.3, §10.4
- **Summary:** GitHub Actions runs `go build ./...`, `go vet ./...`, and
  `go test ./...` on Ubuntu, macOS, and Windows on every push/PR, with a
  Windows real-window smoke test as a follow-up workflow step. No Docker
  is used for window tests.

## Requirement: Build & test on three OSes

### Scenario: Workflow runs on every push and PR

- **WHEN** a push or pull request lands
- **THEN** `go vet ./...`, `go build ./...`, and `go test ./...` run on
  `ubuntu-latest`, `macos-latest`, and `windows-latest` (per
  `.github/workflows/ci.yaml`)
- **AND** all three jobs must pass for the workflow to succeed

## Requirement: Windows integration smoke test *(Pending workflow integration)*

### Scenario: Real-window smoke test on Windows runner

- **WHEN** the workflow runs on `windows-latest`
- **THEN** it builds the CLI, launches a real application (Notepad),
  detects it via `windowctl windows list --json`, executes a `move`,
  and verifies no errors are returned

> The smoke logic is implemented as `scripts/smoke-windows.ps1`. To
> activate it, add this step to the `windows-latest` job in
> `.github/workflows/ci.yaml`:
>
> ```yaml
> - name: Windows real-window smoke test
>   if: matrix.os == 'windows-latest'
>   run: pwsh ./scripts/smoke-windows.ps1
> ```

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

> The Windows real-window smoke test is intentionally a workflow step
> driving `scripts/smoke-windows.ps1`, not a standalone Taskfile
> target — it depends on a clean Windows GUI runner that doesn't exist
> on a developer's macOS or Linux box.
