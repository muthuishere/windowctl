## ADDED Requirements

### Requirement: A public product site
The project SHALL publish a public site at `winctl.deemwar.com` that explains
what windowctl does, shows the zone model visually, and gives a developer a
working first command without leaving the page.

#### Scenario: A visitor can install and act within the first screen
- **WHEN** a visitor loads the landing page
- **THEN** they see the install command and a runnable `windowctl move` example above the fold, and a Quickstart action in the hero

#### Scenario: The zone vocabulary is shown, not described
- **WHEN** a visitor reads about zones
- **THEN** the halves, quarters and N:M splits are rendered as labelled rectangles with the matching flag beneath each, rather than described only in prose

#### Scenario: Documentation is searchable
- **WHEN** a visitor searches the site
- **THEN** the built-in search returns results across every documentation page

### Requirement: Claims on the site are true and dated
The site SHALL state the implementation status of each capability honestly,
including capabilities that are designed but not built, and MUST NOT present a
planned surface as a working one. Performance figures MUST come from a recorded
spike, not an estimate.

#### Scenario: Unbuilt surfaces are labelled
- **WHEN** a capability is designed but not implemented (virtual camera; the device layer on Windows)
- **THEN** the site labels it as not built and links the decision that scoped it, rather than omitting it or implying it works

#### Scenario: Partially working surfaces are labelled as such
- **WHEN** a capability works only under a condition (speaker loopback needs a system-audio grant; the virtual audio driver needs a notarized build)
- **THEN** the site states the condition next to the capability

#### Scenario: Numbers are attributable
- **WHEN** the site quotes a latency or timing figure
- **THEN** the figure names the spike it came from

### Requirement: Visual identity is shared with the deemwar family but distinct
The site SHALL use the deemwar design system — Inter for text, JetBrains Mono
for code, charcoal dark as the default theme, a single accent colour — and MUST
use an accent distinct from the one already used by `toolnexus`, so sibling
products are recognisable without being confusable.

#### Scenario: Both themes are legible
- **WHEN** the visitor's system prefers light or dark
- **THEN** the page renders in that theme with tokens defined for it, and the theme toggle overrides it in both directions

#### Scenario: No horizontal page scroll on a phone
- **WHEN** the page is viewed at a narrow viewport
- **THEN** the body does not scroll horizontally; wide tables and diagrams reflow or scroll within their own container

### Requirement: Deployment is reproducible from the repository
The site SHALL be built and deployed from `site/` in this repository by a
documented task, to Cloudflare Pages, independently of the binary release
pipeline.

#### Scenario: Deploy without a version bump
- **WHEN** a maintainer changes site copy and runs the deploy task
- **THEN** the site updates without a version bump, tag, or npm publish

#### Scenario: Release without a site deploy
- **WHEN** a maintainer runs the release pipeline
- **THEN** the Go build and npm publish complete without requiring the site to build
