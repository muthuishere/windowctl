# Spec: `window-move-coords`

> Part of the [windowctl OpenSpec capability set](./README.md).

- **Status:** Implemented
- **Source:** FR-MOV-02, FR-MOV-04, §4.3.3, §11
- **Summary:** Move a matched window using raw `--x/--y/--w/--h`
  coordinates, interpreted as absolute global coordinates or as
  monitor-relative depending on whether `--monitor` is supplied.

## Requirement: Absolute coordinate move

### Scenario: Coordinates without `--monitor` are global

- **WHEN** the user runs
  `windowctl move --title chrome --x 0 --y 0 --w 960 --h 1080`
- **THEN** the window is placed at the absolute screen coordinates
  `(0, 0)` with size `960×1080`

## Requirement: Monitor-relative coordinate move

### Scenario: Coordinates with `--monitor` are relative to that monitor's origin

- **WHEN** the user runs
  `windowctl move --title chrome --monitor 2 --x 0 --y 0 --w 960 --h 1080`
- **THEN** the coordinates are interpreted relative to monitor 2's origin
  before being applied

## Requirement: Window matching and error contract

### Scenario: Same matcher and error contract as zone moves

- **WHEN** the user invokes `move` with manual coordinates
- **THEN** the matcher rules and error scenarios from
  [`window-move-zone`](./04-window-move-zone.md) apply identically (no
  matcher → error, no match → error, invalid monitor → error)
