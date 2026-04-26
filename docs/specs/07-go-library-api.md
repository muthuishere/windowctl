# Spec: `go-library-api`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Partial
- **Source:** LIB-01, LIB-02, LIB-03, §8.3
- **Summary:** Make the library that already powers the CLI usable as a
  first-class, importable Go module at `github.com/muthuishere/windowctl`,
  with no CLI dependencies leaking into consumers.

> **Timeline note.** From v1 the CLI has been a thin wrapper around an
> internal library package, and core logic has been OS-mockable from the
> start. What was **not** in place initially is the exported, importable
> public surface — that is the work this spec primarily tracks.

## Requirement: CLI is a thin wrapper *(Implemented from v1)*

### Scenario: All business logic lives in the library

- **WHEN** the CLI binary handles any subcommand (`windows list`,
  `monitors list`, `move`, `focus`)
- **THEN** it parses flags and delegates to the library
- **AND** it contains no zone-math, monitor-resolution, or
  window-matching logic of its own

## Requirement: Core logic is testable without OS dependencies *(Implemented from v1)*

### Scenario: Core packages mock the OS adapter

- **WHEN** the core packages' unit tests run
- **THEN** they execute without a real window manager, using mock
  adapters satisfying the OS interface (per §8.3)

## Requirement: Public API surface *(Planned — not exported initially)*

### Scenario: Library exposes the documented types and functions

- **WHEN** a Go consumer imports `github.com/muthuishere/windowctl`
- **THEN** the package exposes the types `Window`, `Monitor`, `Rect`,
  `Filter`, `Match`, `Target` and the functions
  `ListWindows(filter Filter)`, `ListMonitors()`,
  `Move(match Match, target Target)`, `Focus(match Match)` with the
  signatures defined in LIB-01

> Initially the package providing this surface lived in an `internal/`
> path and could not be imported by third parties. Promoting it to a
> public import path is the work that moves this requirement to
> `Implemented`.

## Requirement: Library importability without CLI deps *(Planned — not exported initially)*

### Scenario: Importing the library does not pull in CLI deps

- **WHEN** a third-party Go module depends only on
  `github.com/muthuishere/windowctl`
- **THEN** transitive dependencies do not include the CLI's flag/parsing
  packages (per LIB-03)

## Tasks

| Task | Purpose |
|------|---------|
| `task build` | Builds the CLI, which already delegates to the internal library |
| `task test` | Runs the library's OS-mockable unit tests |

> No current Taskfile target verifies the **exported** public surface or
> the no-CLI-deps guarantee. Closing that gap (e.g. an importer-smoke
> task) is part of the work that moves the `Public API surface` and
> `Library importability` requirements from `Planned` to `Implemented`.
