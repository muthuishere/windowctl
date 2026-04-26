# Spec: `npm-distribution`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Partial (download vs. bundle strategy is OI-05)
- **Source:** §9.3, OI-05
- **Summary:** Make the CLI installable and runnable via npm so that
  `npx windowctl <command>` and `npm install -g windowctl` both work.

## Requirement: `npx windowctl` runs the native binary

### Scenario: First-run via npx

- **WHEN** a user runs `npx windowctl windows list`
- **THEN** the npm package's JS wrapper detects OS and architecture at
  runtime, invokes the correct native binary, and forwards arguments and
  exit code

## Requirement: Global install

### Scenario: `npm install -g windowctl` exposes the CLI on PATH

- **WHEN** a user runs `npm install -g windowctl`
- **THEN** a `windowctl` executable is available on the system PATH

## Requirement: Binary acquisition strategy

### Scenario: Strategy is undecided

- **WHEN** the npm package is built
- **THEN** the choice between bundling all platform binaries vs.
  downloading the matching binary on first use is **TBD** (OI-05) and
  must be resolved before this spec moves to `Implemented`
