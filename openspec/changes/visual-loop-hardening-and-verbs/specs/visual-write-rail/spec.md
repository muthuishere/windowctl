# visual-write-rail

## ADDED Requirements

### Requirement: App-agnostic visual write rail

`windowctl` SHALL support a DOM-free write path — locate an input by its on-screen label
via `find --text`, click it, and insert text (via clipboard-paste or typing) — that works
in ANY app or website, including rich editors (DraftJS/contenteditable) that break
selector/DOM-based automation. This complements browser-bridge's DOM *reads* with a
vision-driven *write* rail.

#### Scenario: Fill a web compose box without DOM knowledge

- **WHEN** a logged-in site with a comment/compose box is on screen
- **AND** `find --text` locates the box's placeholder/label and a click + insert is issued
- **THEN** the text appears IN the box (screenshot-verified), with no site-specific
  selector or API recipe.

#### Scenario: Safety — fill, never publish

- **WHEN** the write-rail proof runs against a real logged-in site
- **THEN** it SHALL stop before any submit/post control (fill-and-verify only), or use a
  fully reversible target that is cleared immediately — it SHALL never publish to a public
  feed.
