# Spec: `release-pipeline`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented
- **Source:** §9.1, §9.2
- **Summary:** Cross-compile, checksum, and publish GitHub releases for
  macOS, Windows, and Linux on both `amd64` and `arm64`, orchestrated by
  Taskfile and GoReleaser.

## Requirement: Taskfile-driven build orchestration

### Scenario: Required tasks exist

- **WHEN** a developer runs `task --list`
- **THEN** the available tasks include `build`, `test`, `lint`,
  `snapshot`, and `release` with the behavior described in §9.1

## Requirement: Cross-compiled binaries

### Scenario: GoReleaser produces all target binaries

- **WHEN** `task release` (or `task snapshot`) runs
- **THEN** GoReleaser produces binaries for `darwin`, `windows`, and
  `linux` on both `amd64` and `arm64`

## Requirement: macOS CGO linker flags

### Scenario: macOS build links the required frameworks

- **WHEN** the macOS binary is built
- **THEN** CGO is enabled and the linker flags include
  `-framework ApplicationServices` and `-framework CoreGraphics`

## Requirement: Checksums and GitHub release

### Scenario: SHA-256 checksums are published with the binaries

- **WHEN** a release is published
- **THEN** SHA-256 checksums are generated for every binary and the
  binaries are attached to a GitHub release
