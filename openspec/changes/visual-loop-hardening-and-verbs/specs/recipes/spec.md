# recipes

## ADDED Requirements

### Requirement: Reusable, parameterized visual automation recipes

`windowctl` SHALL persist and replay named, parameterized sequences of its visual
primitives (focus, launch, wait-for-text, find-click, click, type, paste, key, scroll,
drag), scoped to a window match, so that once a flow is worked out (comment on YouTube,
compose on X, fill any app's form) it is captured as a recipe and replayable — the visual
twin of a browser-bridge DOM recipe. Recipes live in `~/.config/windowctl/recipes.json`
(directory overridable via `WINDOWCTL_CONFIG_DIR`); a few real recipes are embedded as
seeds and overridden by user recipes of the same name.

#### Scenario: Save, list, run

- **WHEN** `windowctl recipe save <name>` is given a recipe body (steps + match + vars) on
  stdin or `--file`
- **THEN** it is validated and stored; `windowctl recipe list` shows it; `windowctl recipe
  run <name> --var TEXT=...` executes its steps in order, substituting `$VAR` / `${VAR}`.

#### Scenario: Vision-first with fallbacks ("allow all ways")

- **WHEN** a `find-click` step runs
- **THEN** it OCRs the scope and clicks the top text match; if OCR finds nothing it falls
  back to the step's `(x,y)` coordinate. A `type` step's `method` may be `keystroke`,
  `paste` (clipboard + cmd+v, layout-proof), or `auto` (keystroke, paste on failure).

#### Scenario: Safe-by-default — never publishes without confirmation

- **WHEN** a recipe contains a `confirm: true` step (the final submit/post/publish click)
  and is run WITHOUT `--confirm`
- **THEN** that step is SKIPPED, the box is left filled but unposted, and the run result
  flags `held_for_confirm`. The seeds (`youtube-comment`, `x-compose`) put the submit
  button click behind `confirm`, so a bare run fills the compose box and stops.

#### Scenario: Replay reliability — no first-responder race

- **WHEN** a recipe issues a `find-click` immediately followed by `key`/`type` against the
  same window
- **THEN** the focus guard does NOT re-raise the already-focused window (which would reset
  first responder and drop keystrokes), and a bounded inter-step settle
  (`WCTL_RECIPE_STEP_DELAY_MS`) paces the actions — verified live: a recipe that clicks a
  field, selects-all, and types replaced the document content exactly (confirmed by
  clipboard readback, OCR-independent).
