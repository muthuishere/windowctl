# input-automation

## MODIFIED Requirements

### Requirement: Focus-guarded input is reliable across a focus change

`TypeInto`/`PressKeyInto` (the `type`/`key` paths with a `--title`/`--app` filter) SHALL
focus the target, verify it became focused, AND wait a short bounded settle before
injecting, so keystrokes issued immediately after a focus/first-responder change are not
dropped. The darwin type path SHALL pace synthetic keyboard events (a small delay between
key-down and key-up, matching the chord path) to reduce dropped characters.

#### Scenario: Type immediately after focusing a different window

- **WHEN** focus is switched to a target window and `type --title` injects with no manual
  delay
- **THEN** the full text lands in the target (the post-focus settle covers first-responder
  readiness).

#### Scenario: Settle is bounded and configurable

- **WHEN** `WCTL_FOCUS_SETTLE_MS` is set
- **THEN** the settle uses that many milliseconds (0 disables it); the default is a small
  non-zero value.

#### Scenario: Reliable insertion alternative is documented

- **WHEN** a caller needs guaranteed insertion of long/complex text
- **THEN** the documented path is `clipboard set` + paste, not synthetic typing.
