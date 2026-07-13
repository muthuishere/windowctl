# window-state

## ADDED Requirements

### Requirement: Window state control

`windowctl minimize|maximize|fullscreen|close (--title <s> | --app <s>)` SHALL change the
matched window's state via the platform window API (AX on macOS), resolving the target the
same way Move/Focus do and returning `ErrAccessibilityDenied` without trust,
`ErrNoMatch` when nothing matches.

#### Scenario: Minimize and close

- **WHEN** `windowctl minimize --title "Notes"` is run
- **THEN** the matched window is minimized to the Dock (AX `kAXMinimizedAttribute=true`).
- **WHEN** `windowctl close --title "Notes"` is run
- **THEN** the matched window's close control is pressed (AX close button).

#### Scenario: Maximize and fullscreen

- **WHEN** `windowctl maximize --title "Notes"` is run
- **THEN** the window is resized to its monitor's visible frame.
- **WHEN** `windowctl fullscreen --title "Notes"` is run
- **THEN** native fullscreen is toggled for the window.
