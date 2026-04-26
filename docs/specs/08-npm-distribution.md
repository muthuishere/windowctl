# Spec: `npm-distribution`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Partial — strategy resolved, package skeleton shipped;
  blocked from `Implemented` until at least one matching GitHub release
  has been published so the postinstall download has something to fetch.
- **Source:** §9.3, OI-05
- **Summary:** Make the CLI installable and runnable via npm so that
  `npx windowctl <command>` and `npm install -g windowctl` both work.

## Strategy decision (resolves OI-05)

The package downloads the matching native binary on `postinstall` rather
than bundling every platform's binary. The npm tarball stays small (one
JS wrapper plus a download script), and the binaries are sourced from
the GitHub release that matches `package.json#version`. The download
extracts a `tar.gz` into `npm/native/`, where the wrapper looks for it.

## Requirement: `npx windowctl` runs the native binary

### Scenario: First-run via npx

- **WHEN** a user runs `npx windowctl windows list`
- **THEN** the npm package's JS wrapper (`npm/bin/windowctl.js`)
  resolves the binary at `npm/native/windowctl` (or `windowctl.exe` on
  Windows) and exec's it, forwarding `argv`, stdio, and the exit code

### Scenario: Binary missing because postinstall failed

- **WHEN** the wrapper is invoked but the binary is not present
- **THEN** it prints a diagnostic pointing at
  <https://github.com/muthuishere/windowctl/releases> and exits non-zero,
  so the failure is visible rather than silently exec'ing nothing

## Requirement: Global install

### Scenario: `npm install -g windowctl` exposes the CLI on PATH

- **WHEN** a user runs `npm install -g windowctl`
- **THEN** the `bin` field in `npm/package.json` causes a `windowctl`
  shim to be placed on the system PATH that delegates to the JS wrapper

## Requirement: Postinstall binary acquisition

### Scenario: Postinstall downloads the right binary for the host

- **WHEN** `npm install` runs the package's `postinstall` step
- **THEN** `npm/scripts/install.js` maps `process.platform` /
  `process.arch` to the GoReleaser asset name
  (`windowctl_<version>_<os>_<arch>.tar.gz`), downloads it from
  `https://github.com/muthuishere/windowctl/releases/download/v<version>/<asset>`,
  and extracts it into `npm/native/`

### Scenario: Release for this version does not yet exist

- **WHEN** the GitHub release for `v<package.json#version>` is missing
- **THEN** the download script fails with `HTTP 404` and the user sees
  a message pointing them at the releases page; npm install exits
  non-zero so the gap is loud rather than silent

## Tasks

| Task | Purpose |
|------|---------|
| `task snapshot` | Produces the per-platform binaries that the npm postinstall downloads |
| `task release` | Publishes the GitHub release whose URL `npm/scripts/install.js` fetches |

> The npm package itself does not have a `task` target yet — it is
> built and tested with `npm install` / `npm publish` directly. Adding
> a wrapper task once a real publishing flow exists is a small follow-up.
