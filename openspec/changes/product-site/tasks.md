## 1. De-risk

- [x] 1.1 Spike S9 — token active, `deemwar.com` zone resolves, `winctl.deemwar.com` unclaimed, wrangler present, site builds with root-relative links (9/9 PASS)

## 2. Build

- [x] 2.1 Scaffold Astro + Starlight at `site/`, base `/` (not a project path — the one config that must differ from toolnexus)
- [x] 2.2 Fork the deemwar design tokens into `src/styles/windowctl.css`; Inter Variable + JetBrains Mono; accent teal `#0FB9A8`
- [x] 2.3 Zone-grid, monitor-strip, fact-strip and status-table components in pure CSS — no JS to draw a rectangle
- [x] 2.4 Landing page: hero, quickstart command, fact strip, zone diagram, batch layout, read-the-desktop tabs, agent skill, device layer, why-built-this-way
- [x] 2.5 Twelve documentation pages
- [x] 2.6 `llms.txt` via starlight-llms-txt, matching toolnexus

## 3. Ship

- [x] 3.1 Cloudflare Pages project `winctl-deemwar`
- [x] 3.2 Deploy `site/dist`
- [x] 3.3 Attach the custom domain + proxied CNAME for `winctl.deemwar.com`
- [x] 3.4 Verify the live URL serves the built page over HTTPS
- [x] 3.5 `task site:dev` / `site:build` / `site:deploy`
- [x] 3.6 Gitignore `site/node_modules` and `site/dist`

## 4. Follow-up

- [ ] 4.1 An explainer video in the hero (toolnexus ships one; the placeholder pattern is there)
- [ ] 4.2 Real screenshots or a recorded terminal for the batch-layout section
- [ ] 4.3 A CI check that the status table matches `RESUME-STATE.md` — the table is a promise, and stale promises are the failure mode of an honest status page
