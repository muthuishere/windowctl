# ADR 0013 — publish to npm with OIDC, from CI

- Status: Accepted
- Date: 2026-07-26

## Context
Releases have been local-only by design (spec 09): `task release -- <version>`
runs everything from the maintainer's Mac and publishes with whatever npm
credential that machine holds. That has blocked v0.6.0 for days — `npm whoami`
returns E401, and the release script correctly aborts at preflight. A release
path that depends on one machine's login state is a release path that stops
working the moment the token expires.

## Decision
**Publish from GitHub Actions using npm OIDC trusted publishing**, triggered by
pushing a `v*` tag. No `NPM_TOKEN` secret, no `npm login`, no long-lived
credential anywhere: npm exchanges the workflow's OIDC token for short-lived
publish rights against a trusted publisher configured per package.

- **Runner is `macos-latest`.** The darwin build needs CGO against
  CoreGraphics/ApplicationServices; a macOS host also cross-compiles the
  CGO-free linux and windows targets. Ubuntu would need osxcross.
- **goreleaser runs `--skip=publish`** and the workflow creates the GitHub
  release itself, because the goreleaser config's release target
  (`muthuishere/windowctl`) predates the repo living under
  `deemwar-products`.
- **Provenance is off.** npm turns provenance on by default under trusted
  publishing, but generating an attestation requires a **public** source
  repository. This repo is private, so leaving it on fails every publish.
  `NPM_CONFIG_PROVENANCE: false` is set with a comment saying to flip it if the
  repo is ever made public.
- **The local path stays.** `task release` is untouched and still works for
  anyone with a valid npm login — this adds a credential-free route, it does not
  remove the existing one.

## The manual step this cannot avoid
Trusted publishing must be configured **once per package** in the npmjs.com UI
(Settings → Publishing access → Trusted publisher: repository
`deemwar-products/windowctl`, workflow `release.yaml`). There is no public API
for it. Six packages: the main one plus the five platform sub-packages. Until
that is done, the workflow's publish step will fail with a permissions error —
which is the correct failure, not a bug in the workflow.

## Alternatives rejected
- **An `NPM_TOKEN` repository secret.** Works today, but it is a long-lived
  credential in a private repo's secret store, and it is exactly the thing OIDC
  exists to delete. Rejected unless trusted publishing turns out to be blocked.
- **Keep local-only publishing and just fix the login.** Fixes this release, not
  the next one; the failure recurs every time the token rotates.
- **Publish from a public mirror repo to also get provenance.** Real benefit
  (signed attestations), but it requires a mirror pipeline and keeping two repos
  in sync — deliberately out of scope here.

## Consequences
- A release becomes `git tag v0.6.0 && git push origin v0.6.0`.
- The workflow commits the version bump back to `main`, so the repo and the
  published packages agree.
- Spec 09 ("No GitHub Actions release path — everything runs from the
  developer's Mac") is now partially superseded; it should be updated when this
  workflow has completed one successful run, not before.
- Nothing here changes the npm package layout, the platform matrix, or the Go
  module path.
