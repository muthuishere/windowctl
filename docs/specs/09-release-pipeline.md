# Spec: `release-pipeline`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Partial — pipeline is wired and CGO-disabled cross-compiles
  succeed; the macOS CGO linker-flag requirement is staged in
  `.goreleaser.yaml` as a documented toggle and activates when the
  macOS adapter starts using CGO.
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

## Requirement: macOS CGO linker flags *(Pending — activates with macOS adapter CGO)*

### Scenario: macOS build links the required frameworks

- **WHEN** the macOS binary is built **and the macOS adapter uses CGO**
- **THEN** CGO is enabled and the linker flags include
  `-framework ApplicationServices` and `-framework CoreGraphics`

> The macOS adapter currently ships as a non-CGO scaffold (every method
> returns `core.ErrNotImplemented`), so the release pipeline builds it
> with `CGO_ENABLED=0` like the other platforms. `.goreleaser.yaml`
> contains a commented-out CGO-enabled darwin build with the required
> framework flags; flipping that block from a comment to a live build
> entry is the work that moves this requirement to `Implemented`.

## Requirement: Checksums and GitHub release

### Scenario: SHA-256 checksums are published with the binaries

- **WHEN** a release is published
- **THEN** SHA-256 checksums are generated for every binary and the
  binaries are attached to a GitHub release

## Tasks

| Task | Purpose |
|------|---------|
| `task build` | Builds the CLI binary for the current platform |
| `task test` | Runs all tests prior to release |
| `task lint` | Runs the linter prior to release |
| `task snapshot` | Builds a snapshot release for all platforms |
| `task release` | Publishes a GitHub release via GoReleaser |

> This spec **is** the Taskfile contract — every entry above is one of
> the required tasks listed in §9.1.
