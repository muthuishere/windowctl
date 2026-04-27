# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`windowctl` is a cross-platform CLI **and** Go library for window/monitor management on macOS, Windows, and Linux. The CLI is a thin shell over the importable Go package — never put business logic in `cmd/windowctl`.

Authoritative specs live in `docs/requirements.md` and `docs/specs/` (OpenSpec-style slices indexed by `docs/specs/README.md`). When adding or changing behavior, locate the matching slice (FR-WIN-01, FR-MOV-03, etc.) and keep code, tests, and the spec aligned.

## Common commands

Uses [Taskfile](https://taskfile.dev):

```sh
task build              # go build -o bin/windowctl ./cmd/windowctl
task test               # go test ./...
task lint               # go vet ./...
task snapshot           # goreleaser release --snapshot --skip=publish; stages binaries into npm/platforms/<os-arch>/bin/
task release:dry-run    # same as snapshot, with --verbose for goreleaser config debugging
task npm:bump -- 0.2.0  # sync version across main + 5 platform sub-packages
task npm:publish        # npm publish all 6 packages non-interactively (binaries must already be staged)
task release -- 0.2.0   # full local release: bump → snapshot+stage → publish 6 packages → commit → tag → push → GH release
```

Single test: `go test ./... -run TestApplyFilterByTitleIsPartialAndCaseInsensitive`. Tests are in the repo root package (`windowctl`) and `cmd/windowctl`; the per-OS adapter packages have no unit tests — they're validated via the smoke scripts in CI.

Real-window smoke tests (run automatically by `.github/workflows/ci.yaml` via `task smoke-darwin` / `task smoke-windows` on the matching runner; the workflow installs `task` via `arduino/setup-task@v2`):

- `task smoke-darwin` — builds CLI, launches TextEdit via LaunchServices (`open -a`), polls `windowctl windows list` until detected (10s budget), then runs `windowctl move` and auto-detects the AX state from the actual outcome — exit 0 means trust is inherited (asserts bounds within `WCTL_AX_TOLERANCE`, default 50px to absorb title-bar/chrome variance), exit !=0 with "Accessibility permission denied" means a fresh runner (asserts the denied error). `WCTL_SMOKE_AX=1` is an opt-in strict mode that REQUIRES the granted path, useful for catching TCC regressions on a known-good local setup.
- `task smoke-windows` — builds CLI, launches Notepad, verifies `windows list` detects it and `move` succeeds.

Module is Go 1.24, single dep `golang.org/x/sys`.

## Architecture

Three layers, with build-tag-isolated platform code:

1. **`cmd/windowctl/`** — flag parsing, table/JSON formatting. Calls only the public package. No OS code, no business logic.
2. **Public package `github.com/muthuishere/windowctl`** (root `*.go` files) — `ListWindows`, `ListMonitors`, `Move`, `Focus`, `MoveZone`, `MoveCoords`. Re-exports `core` types. Holds shared logic that must remain OS-agnostic and unit-testable: filter matching (`applyFilter`), monitor auto-resolution (`resolveCurrentMonitor` — majority-overlap per FR-MOV-03), zone parsing (`zone.go`), and the relative-vs-absolute coord rule for `MoveCoords` (relative when `--monitor` is set).
3. **`internal/core`** — OS-agnostic `Window`/`Monitor`/`Rect`/`Filter`/`Match`/`Target` types and the `Adapter` interface. Exists *only* to break the import cycle between the public package and the per-OS adapters; nothing else should live here.
4. **`internal/adapter/{darwin,linux,windows}`** — one `Adapter` implementation per OS. Selected at link time by `adapter_darwin.go` / `adapter_linux.go` / `adapter_windows.go` (each with the matching `//go:build` tag) which call `newPlatformAdapter()`.

The adapter interface is intentionally tiny (`ListWindows`, `ListMonitors`, `Move`, `Focus`, `RequestAccessibility`). Anything that can be expressed as composition of those primitives belongs in the public package, not the adapters. `RequestAccessibility` is a no-op on linux/windows and the only AX-prompting call site on darwin — surfaced through the `windowctl permissions` subcommand.

### Per-OS notes

- **darwin** (`internal/adapter/darwin/adapter.go`): CGO against CoreGraphics + CoreFoundation + ApplicationServices + AppKit. ListWindows / ListMonitors use CG; Move / Focus / RequestAccessibility use the Accessibility (AX) API via `AXUIElementSetAttributeValue` (position+size sandwich for cross-display moves) and `AXUIElementPerformAction(kAXRaiseAction)` + `NSRunningApplication activateWithOptions:` for focus (the latter reached via `objc_msgSend` so the file stays pure C — no `.m` or Swift). All CFRetain/CFRelease and AX lifetime is owned by the C helpers (`wctl_collect_windows`, `wctl_collect_monitors`, `wctl_ax_check`, `wctl_ax_request`, `wctl_window_for_id`, `wctl_ax_resolve`, `wctl_ax_set_bounds`, `wctl_ax_focus`); Go only iterates POD struct arrays via `unsafe.Slice`. The CGWindowID → AXUIElementRef bridge is a per-call AX walk under the owner PID, disambiguated by title + bounds (no caching, no `_AXUIElementGetWindow` private SPI). Move/Focus return `core.ErrAccessibilityDenied` when AX permission is missing — `windowctl permissions` is the discoverable opt-in for granting it. AX prompt only fires from `windowctl permissions` or the lazy guard inside Move/Focus, never from `windows list`.
- **windows** (`internal/adapter/windows/adapter.go`): `user32.dll` via `golang.org/x/sys/windows`. `EnumWindows` callback for window listing; `ListMonitors` enumerates every attached display via `EnumDisplayMonitors` + `GetMonitorInfoW`, assigning sequential IDs (0..N-1) and setting `Primary` from `MONITORINFOF_PRIMARY`.
- **linux** (`internal/adapter/linux/adapter.go`): shells out to `wmctrl` (list/move/focus) and `xrandr` (monitors). Native X11 via Xlib is planned. Wayland is best-effort.

### Build & release

`.goreleaser.yaml` has two build entries: `windowctl-nondarwin` (linux/windows, `CGO_ENABLED=0`, ignores `windows/arm64`) and `windowctl-darwin` (`CGO_ENABLED=1`). Cross-compiling to darwin from a non-darwin host requires a darwin C toolchain (osxcross); on macOS the system clang handles it. Each build entry has a per-build post hook (`scripts/stage-npm-binary.sh`) that copies the freshly-built binary into the matching `npm/platforms/<os>-<arch>/bin/` directory.

The npm distribution uses the multi-package `optionalDependencies` pattern: `@muthuishere/windowctl` is a thin JS launcher (`npm/bin/windowctl.js`) that `require.resolve`s the matching `@muthuishere/windowctl-<os>-<arch>` sub-package and execs the native binary. No postinstall, no GitHub Releases dependency at install time. Five sub-packages: darwin-{arm64,x64}, linux-{arm64,x64}, windows-x64. Each has `os`+`cpu` fields so npm only installs the matching one.

Local release flow (no GitHub Actions involved): `task release -- <version>` runs the full pipeline — `node scripts/bump-npm-version.js` syncs versions across all 6 `package.json` files, `goreleaser release --snapshot --clean --skip=publish` cross-compiles + stages binaries via the post hooks, `scripts/publish-npm-local.sh` publishes all 6 packages non-interactively (`npm publish --access public` per package; assumes the configured token bypasses 2FA), then commit + tag + push + `gh release create`. See `docs/specs/09-release-pipeline.md`.

## Conventions

- `--title` is a case-insensitive substring match; `--app` is case-insensitive **exact** match. Don't change one without the other (covered by `adapter_test.go`).
- When multiple windows match a filter, the first match is used (FR-MOV-04 / OI-01 in `docs/requirements.md` — TBD for v0.1; flag any change as a spec change).
- Errors that flow to the user must be descriptive enough to act on (see §11 of `docs/requirements.md`); use `core.ErrNotImplemented` / `core.ErrNoMatch` rather than ad-hoc strings where they apply.
- CI runs on `ubuntu-latest`, `macos-latest`, `windows-latest`. Anything that won't compile/test on all three breaks the build — keep OS-specific code behind build tags.
