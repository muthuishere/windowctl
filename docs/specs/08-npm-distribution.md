# Spec: `npm-distribution`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented — `@muthuishere/windowctl` ships as a wrapper
  package whose `optionalDependencies` resolve to the matching platform
  sub-package (`@muthuishere/windowctl-<os>-<arch>`); npm itself selects
  the right one at install time. No postinstall, no network call at
  install time, no GitHub Releases dependency.
- **Source:** §9.3, OI-05
- **Summary:** Make the CLI installable and runnable via npm so that
  `npx @muthuishere/windowctl <command>` and
  `npm install -g @muthuishere/windowctl` both work, with the right
  prebuilt binary for the host shipped inside an npm sub-package.

## Strategy decision (resolves OI-05)

Multi-package `optionalDependencies` pattern. The main package
`@muthuishere/windowctl` ships only the JS launcher
(`npm/bin/windowctl.js`) and declares five platform sub-packages as
`optionalDependencies`:

```json
"optionalDependencies": {
  "@muthuishere/windowctl-darwin-arm64": "<version>",
  "@muthuishere/windowctl-darwin-x64":   "<version>",
  "@muthuishere/windowctl-linux-arm64":  "<version>",
  "@muthuishere/windowctl-linux-x64":    "<version>",
  "@muthuishere/windowctl-windows-x64":  "<version>"
}
```

Each sub-package has its own `package.json` with `"os"` and `"cpu"`
fields constraining install to the matching host. npm only fetches the
sub-package whose OS+CPU match the installer; the others are silently
skipped.

The launcher (`require.resolve(`${pkg}/bin/${binName}`)`) finds the
native binary inside the installed sub-package and execs it. No
`postinstall`, no tar download, no GitHub Releases dependency at
install time.

Windows arm64 is excluded for now (matches the goreleaser `ignore`
rule) — niche platform; revisit when there's demand.

## Requirement: `npx @muthuishere/windowctl` runs the native binary

### Scenario: First-run via npx

- **WHEN** a user runs `npx @muthuishere/windowctl windows list`
- **THEN** the launcher (`npm/bin/windowctl.js`) maps
  `${process.platform}-${process.arch}` to the matching sub-package
  name (e.g. `darwin-arm64` → `@muthuishere/windowctl-darwin-arm64`),
  resolves the binary inside it via `require.resolve`, and exec's it,
  forwarding `argv`, stdio, and the exit code

### Scenario: Unsupported host platform

- **WHEN** `process.platform` / `process.arch` does not appear in the
  launcher's `SUPPORTED` map (e.g. windows-arm64, freebsd, openbsd)
- **THEN** the launcher prints the unsupported key, lists the supported
  keys, points at the GitHub Releases page for manual downloads, and
  exits non-zero

### Scenario: Platform sub-package was skipped

- **WHEN** the launcher runs but `require.resolve` for the matching
  sub-package fails (e.g. `npm install --no-optional`, or the optional
  dep was filtered out by an enterprise registry)
- **THEN** it prints a diagnostic explaining the cause and the fix
  (`npm install -g @muthuishere/windowctl`), and exits non-zero

## Requirement: Global install

### Scenario: `npm install -g @muthuishere/windowctl` exposes the CLI on PATH

- **WHEN** a user runs `npm install -g @muthuishere/windowctl`
- **THEN** the `bin` field in the main `package.json` causes a
  `windowctl` shim to be placed on the system PATH that delegates to
  the JS launcher

## Requirement: Version sync across all 6 packages

### Scenario: Bumping the version

- **WHEN** the developer runs
  `node scripts/bump-npm-version.js <version>` (or
  `task npm:bump -- <version>`)
- **THEN** the same `<version>` is written into all 6 `package.json`
  files (main + 5 sub-packages) and into the main package's
  `optionalDependencies` pins, so npm's resolution stays consistent

## Requirement: Local-only publish flow

### Scenario: Publishing to npm without GitHub Actions

- **WHEN** the developer runs `task release -- <version>` (which
  invokes `scripts/release-local.sh`)
- **THEN** the pipeline runs entirely on the local machine: bumps
  versions, runs `goreleaser release --snapshot --clean --skip=publish`
  to cross-compile and stage binaries into
  `npm/platforms/<os-arch>/bin/`, then `npm publish --access public`
  for each of the 6 packages (sub-packages first, main last), then
  commits the version bump, tags `v<version>`, pushes, and creates the
  matching GitHub release with the snapshot tarballs and checksums
- **AND** no `NPM_TOKEN` is required in CI; auth is whatever
  `npm whoami` already trusts on the developer's machine

## Tasks

| Task | Purpose |
|------|---------|
| `task snapshot` | `goreleaser release --snapshot --clean --skip=publish` — staging-only; populates `npm/platforms/<os-arch>/bin/` |
| `task npm:bump -- <version>` | Sync all 6 `package.json` versions to `<version>` |
| `task npm:publish` | Publish all 6 packages non-interactively (assumes binaries already staged) |
| `task release -- <version>` | Full one-shot release: bump → build → publish → commit → tag → push → GH release |
