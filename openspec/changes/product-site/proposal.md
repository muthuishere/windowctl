## Why

`windowctl` has a README and nothing a human lands on. The product is visual —
zones are rectangles, multi-monitor placement is spatial, and the device layer
is a claim you have to see evidenced — and a README renders none of that. There
is also no honest, public statement of what works today versus what is designed
but unbuilt, which is exactly what a developer evaluating it needs first.

Decisions and rejected alternatives: `docs/adr/0012-marketing-site.md`.
Deploy path de-risked by spike S9 (`spikes/s9_site_deploy.sh`, 9/9) before any
live resource was created.

## What Changes

- **New `site/`** — Astro + Starlight, the same stack and design system as
  `toolnexus`, with the accent forked from deemwar blue to a tiling teal so the
  two products read as siblings rather than the same page.
- **A splash landing page** carrying the hero, a fact strip of measured numbers
  (not adjectives), a CSS zone diagram, a multi-monitor strip, and an honest
  per-surface status table for the device layer including the parts that do not
  work yet.
- **Twelve documentation pages** covering quickstart, zones, monitors, listing,
  move/resize/focus, batch layouts, the device layer, input, capture, the agent
  skill, the Go library, permissions, and how it is built.
- **Cloudflare Pages deployment** as project `winctl-deemwar` at
  `winctl.deemwar.com`, matching the shape of the existing deemwar Pages sites.
- **`task site:*`** — dev, build, deploy from the repo's Taskfile.

## Impact

- Affected specs: `product-site` (new).
- Affected code: `site/` (new), `Taskfile.yml`, `.gitignore`.
- No effect on the Go module: `go build ./...` and `task release` never see
  `site/`. Site deploys and version releases are independent pipelines.
- New live resources: one Cloudflare Pages project and one proxied CNAME in the
  `deemwar.com` zone.
- The site now states the CLI surface a third time (after the README and the
  embedded skill). A flag change is a three-place change; the honest status
  table in particular must be re-checked whenever a device surface graduates.
