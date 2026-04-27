# Spec: `permissions-subcommand`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented
- **Source:** §8.2, requirements §11, huddle decision 1777287341695 (Suna +
  Shaama round, locked by Muthukumaran)
- **Summary:** Add a `windowctl permissions` subcommand so macOS users have a
  discoverable, opt-in way to request Accessibility (AX) trust up front
  instead of hitting the lazy guard on first `move` / `focus`. On Linux and
  Windows the subcommand exists but is a no-op that prints a friendly "no
  permission setup required on this OS" message — the CLI surface stays
  symmetric across platforms.

This slice exists because the alternative — calling
`AXIsProcessTrustedWithOptions` at startup on every macOS invocation — was
explicitly rejected in the huddle: TCC trust is keyed per parent process, so
read-only `windowctl windows list` invocations would get ambushed by a
"wants to control your computer" dialog whose stated scope doesn't match
user intent. The marker-file alternative (`~/.config/windowctl/ax-prompted`)
was also rejected: the marker lies the moment the user invokes from a
different shell because TCC state is per-parent-process. The discoverable
opt-in subcommand is the only design that respects both the OS permission
model and the principle of least surprise.

## Requirement: `windowctl permissions` exists as a top-level subcommand

### Scenario: Subcommand listed in CLI usage

- **WHEN** the user runs `windowctl --help` or `windowctl help`
- **THEN** the usage text lists `windowctl permissions` alongside
  `windows`, `monitors`, `move`, `focus`

### Scenario: Subcommand routed in `cmd/windowctl/main.go`

- **WHEN** the user invokes `windowctl permissions`
- **THEN** `main.go` dispatches to a `permissionsCmd` handler that calls
  the public `windowctl.RequestAccessibility()` function and prints a
  human-readable result line to stdout

## Requirement: macOS — invoke the AX trust check with prompt enabled

### Scenario: AX already granted

- **WHEN** the macOS process is already trusted by the Accessibility
  subsystem (`AXIsProcessTrustedWithOptions` returns true)
- **THEN** `windowctl permissions` prints
  `Accessibility permission: granted` to stdout and exits 0
- **AND** no system dialog appears

### Scenario: AX not granted — prompt fires once per parent process

- **WHEN** the macOS process is not trusted by the Accessibility
  subsystem
- **AND** `windowctl permissions` is invoked
- **THEN** the adapter calls `AXIsProcessTrustedWithOptions` with
  `kAXTrustedCheckOptionPrompt = true`, which causes macOS to display
  the system "wants to control your computer" dialog (subject to the
  per-parent-process TCC policy)
- **AND** the command prints
  `Accessibility permission: denied — grant access in System Settings → Privacy & Security → Accessibility, then re-run`
  to stderr
- **AND** the command exits non-zero

### Scenario: No other windowctl command triggers the prompt

- **WHEN** the user invokes any subcommand other than `permissions`
  (`windows list`, `monitors list`, `move`, `focus`)
- **THEN** the AX prompt option (`kAXTrustedCheckOptionPrompt = true`)
  is NOT passed; the existing lazy guard inside `Move` / `Focus`
  continues to use prompt=false (delivered in slice 01-os-spikes,
  task 001)

## Requirement: Linux and Windows — no-op surface

### Scenario: Linux

- **WHEN** the user invokes `windowctl permissions` on Linux
- **THEN** the command prints
  `Accessibility permission: not required on linux`
  to stdout and exits 0

### Scenario: Windows

- **WHEN** the user invokes `windowctl permissions` on Windows
- **THEN** the command prints
  `Accessibility permission: not required on windows`
  to stdout and exits 0

## Requirement: Move / Focus error message points users at the subcommand

### Scenario: AX-denied error mentions the new subcommand

- **WHEN** macOS `Move` or `Focus` returns `core.ErrAccessibilityDenied`
- **THEN** the error message contains the substring
  `windowctl permissions` so users have a one-line, copy-pasteable
  remediation
- **AND** the message still mentions
  `System Settings → Privacy & Security → Accessibility` for users
  who prefer the manual route

## Requirement: Adapter contract extension

### Scenario: `core.Adapter` exposes `RequestAccessibility`

- **WHEN** the `core.Adapter` interface is inspected
- **THEN** it declares
  `RequestAccessibility() error` alongside the existing `ListWindows`,
  `ListMonitors`, `Move`, `Focus`
- **AND** all three platform adapters (darwin, linux, windows)
  implement it
- **AND** the public `windowctl.RequestAccessibility()` function
  delegates to `defaultAdapter.RequestAccessibility()`, mirroring the
  existing wrapper pattern for `Focus` / `Move`

## Tasks

| Task | Purpose |
|------|---------|
| `task build` | Builds the CLI binary that exposes `windowctl permissions` |
| `task test` | Runs unit tests for the subcommand wiring, the adapter contract, and the updated `ErrAccessibilityDenied` message |
| `task smoke-darwin` | Smoke harness still passes — `windowctl permissions` is not part of the smoke loop, but the existing AX-denied assertion against `move` continues to hold against the updated error message |
